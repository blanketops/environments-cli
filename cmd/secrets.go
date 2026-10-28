/*
Copyright 2026 The BlanketOps Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/hashicorp/vault/api"
)

// SecretSet represents the canonical set of BlanketOps secrets that must exist
// for the Environments operator to function. Each provider implementation
// creates these same logical secrets using provider-native naming conventions.
type SecretSet struct {
	GitSSHPrivateKey      string
	GitSSHPublicKey       string
	GitKnownHosts         string
	RegistryConfig        string
	GitHubWebhookSecret   string
	GitHubAPIToken        string
	CrossplaneGitHubToken string
}

// DefaultSecretSet returns the standard BlanketOps secret set with values
// sourced from the local environment (~/.ssh, etc.) and placeholders for
// secrets that must be provided by the user.
func DefaultSecretSet() (*SecretSet, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	sshPrivateKey, err := os.ReadFile(filepath.Join(home, ".ssh", "id_ed25519"))
	if err != nil {
		return nil, fmt.Errorf("read ssh private key: %w", err)
	}

	sshPublicKey, err := os.ReadFile(filepath.Join(home, ".ssh", "id_ed25519.pub"))
	if err != nil {
		return nil, fmt.Errorf("read ssh public key: %w", err)
	}

	knownHosts := sshKeyscan("github.com")

	return &SecretSet{
		GitSSHPrivateKey:      string(sshPrivateKey),
		GitSSHPublicKey:       string(sshPublicKey),
		GitKnownHosts:         knownHosts,
		RegistryConfig:        placeholderRegistryConfig(),
		GitHubWebhookSecret:   "<REPLACE_WITH_WEBHOOK_SECRET>",
		GitHubAPIToken:        "<REPLACE_WITH_GITHUB_TOKEN>",
		CrossplaneGitHubToken: "<REPLACE_WITH_CROSSPLANE_GITHUB_TOKEN>",
	}, nil
}

func sshKeyscan(host string) string {
	// Best effort; known_hosts is not strictly required for operation
	data, _ := execCommand("ssh-keyscan", host)
	return strings.TrimSpace(string(data))
}

func placeholderRegistryConfig() string {
	auth := base64.StdEncoding.EncodeToString([]byte("<USERNAME>:<TOKEN>"))
	return fmt.Sprintf("{\"auths\":{\"https://index.docker.io/v1/\":{\"auth\":\"%s\"}}}", auth)
}

// execCommand runs a command and returns stdout. Used for ssh-keyscan.
func execCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// ---------------------------------------------------------------------------
// Provider interface
// ---------------------------------------------------------------------------

// SecretBootstrapper defines the contract for bootstrapping BlanketOps secrets
// into a provider's secret store.
type SecretBootstrapper interface {
	// Bootstrap creates all BlanketOps secrets in the provider.
	Bootstrap(ctx context.Context, secrets *SecretSet) error
	// Verify lists the created secrets to confirm they exist.
	Verify(ctx context.Context) error
	// ProviderName returns the human-readable provider name.
	ProviderName() string
}

// ---------------------------------------------------------------------------
// AWS Secrets Manager
// ---------------------------------------------------------------------------

type AWSBootstrapper struct {
	client     *secretsmanager.Client
	region     string
	pathPrefix string
}

func NewAWSBootstrapper(ctx context.Context, region, pathPrefix string) (*AWSBootstrapper, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &AWSBootstrapper{
		client:     secretsmanager.NewFromConfig(cfg),
		region:     region,
		pathPrefix: strings.TrimPrefix(pathPrefix, "/"),
	}, nil
}

func (b *AWSBootstrapper) ProviderName() string { return "AWS Secrets Manager" }

func (b *AWSBootstrapper) fullName(name string) string {
	if b.pathPrefix == "" {
		return name
	}
	return fmt.Sprintf("%s/%s", b.pathPrefix, name)
}

func (b *AWSBootstrapper) createSecret(ctx context.Context, name, description, value string) error {
	fullName := b.fullName(name)
	_, err := b.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(fullName),
		Description:  aws.String(description),
		SecretString: aws.String(value),
	})
	var alreadyExists *types.ResourceExistsException
	if errors.As(err, &alreadyExists) {
		// Update existing
		_, err = b.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
			SecretId:     aws.String(fullName),
			SecretString: aws.String(value),
		})
	}
	return err
}

func (b *AWSBootstrapper) Bootstrap(ctx context.Context, s *SecretSet) error {
	fmt.Printf("Bootstrapping BlanketOps Environments secrets in %s (region: %s)\n", b.ProviderName(), b.region)

	secrets := []struct {
		name        string
		description string
		value       string
	}{
		{name: "environments/git/ssh-privatekey", description: "BlanketOps Environments git SSH private key for source cloning", value: s.GitSSHPrivateKey},
		{name: "environments/git/ssh-publickey", description: "BlanketOps  Environmentsgit SSH public key", value: s.GitSSHPublicKey},
		{name: "environments/git/known-hosts", description: "BlanketOps Environments git SSH known hosts (GitHub)", value: s.GitKnownHosts},
		{name: "environments/registry/config", description: "BlanketOps Environments Docker registry auth config", value: s.RegistryConfig},
		{name: "environments/github/webhook/secret", description: "BlanketOps Environments GitHub webhook HMAC secret", value: s.GitHubWebhookSecret},
		{name: "environments/github/api/token", description: "BlanketOps Environments GitHub API token for environments controller", value: s.GitHubAPIToken},
		{name: "environments/crossplane/github/token", description: "BlanketOps Environments Crossplane GitHub provider token", value: s.CrossplaneGitHubToken},
	}

	for _, sec := range secrets {
		fmt.Printf("  Creating %s...\n", b.fullName(sec.name))
		if err := b.createSecret(ctx, sec.name, sec.description, sec.value); err != nil {
			return fmt.Errorf("create secret %s: %w", sec.name, err)
		}
	}
	return nil
}

func (b *AWSBootstrapper) Verify(ctx context.Context) error {
	out, err := b.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
		Filters: []types.Filter{{
			Key:    types.FilterNameStringType("begins-with"),
			Values: []string{b.fullName("")},
		}},
	})
	if err != nil {
		return err
	}
	fmt.Printf("Verified %d secrets under prefix %s\n", len(out.SecretList), b.fullName(""))
	for _, sec := range out.SecretList {
		fmt.Printf("  - %s\n", aws.ToString(sec.Name))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Azure Key Vault
// ---------------------------------------------------------------------------

type AzureBootstrapper struct {
	client   *azsecrets.Client
	vaultURL string
}

func NewAzureBootstrapper(ctx context.Context, vaultURL string) (*AzureBootstrapper, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("azure credential: %w", err)
	}
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create keyvault client: %w", err)
	}
	return &AzureBootstrapper{client: client, vaultURL: vaultURL}, nil
}

func (b *AzureBootstrapper) ProviderName() string { return "Azure Key Vault" }

func (b *AzureBootstrapper) flatName(name string) string {
	return strings.ReplaceAll(name, "/", "--")
}

func (b *AzureBootstrapper) setSecret(ctx context.Context, name, value string) error {
	flatName := b.flatName(fmt.Sprintf("blanketops-environments--%s", name))
	_, err := b.client.SetSecret(ctx, flatName, azsecrets.SetSecretParameters{
		Value: &value,
	}, nil)
	return err
}

func (b *AzureBootstrapper) Bootstrap(ctx context.Context, s *SecretSet) error {
	fmt.Printf("Bootstrapping BlanketOps Environments secrets in %s\n", b.ProviderName())

	secrets := []struct {
		name  string
		value string
	}{
		{name: "environments--git--ssh-privatekey", value: s.GitSSHPrivateKey},
		{name: "environments--git--ssh-publickey", value: s.GitSSHPublicKey},
		{name: "environments--git--known-hosts", value: s.GitKnownHosts},
		{name: "environments--registry--config", value: s.RegistryConfig},
		{name: "environments--github--webhook--secret", value: s.GitHubWebhookSecret},
		{name: "environments--github--api--token", value: s.GitHubAPIToken},
		{name: "environments--crossplane--github--token", value: s.CrossplaneGitHubToken},
	}

	for _, sec := range secrets {
		fmt.Printf("  Creating %s...\n", sec.name)
		if err := b.setSecret(ctx, sec.name, sec.value); err != nil {
			return fmt.Errorf("set secret %s: %w", sec.name, err)
		}
	}
	return nil
}

func (b *AzureBootstrapper) Verify(ctx context.Context) error {
	pager := b.client.NewListSecretsPager(nil)
	fmt.Printf("Verified secrets in %s:\n", b.vaultURL)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, sec := range page.Value {
			if sec.ID != nil {
				idStr := string(*sec.ID)
				if strings.Contains(idStr, "blanketops") {
					fmt.Printf("  - %s\n", idStr)
				}
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// GCP Secret Manager
// ---------------------------------------------------------------------------

type GCPBootstrapper struct {
	client    *secretmanager.Client
	projectID string
}

func NewGCPBootstrapper(ctx context.Context, projectID string) (*GCPBootstrapper, error) {
	// Use Application Default Credentials
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create secretmanager client: %w", err)
	}
	return &GCPBootstrapper{client: client, projectID: projectID}, nil
}

func (b *GCPBootstrapper) ProviderName() string { return "GCP Secret Manager" }

func (b *GCPBootstrapper) flatName(name string) string {
	return strings.ReplaceAll(name, "/", "_")
}

func (b *GCPBootstrapper) createSecret(ctx context.Context, name, value string) error {
	flatName := b.flatName(fmt.Sprintf("blanketops_environments_%s", name))
	req := &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", b.projectID),
		SecretId: flatName,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}
	secret, err := b.client.CreateSecret(ctx, req)
	if err != nil {
		// If exists, add version
		if strings.Contains(err.Error(), "already exists") {
			secret = &secretmanagerpb.Secret{Name: fmt.Sprintf("projects/%s/secrets/%s", b.projectID, flatName)}
		} else {
			return err
		}
	}
	_, err = b.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	})
	return err
}

func (b *GCPBootstrapper) Bootstrap(ctx context.Context, s *SecretSet) error {
	fmt.Printf("Bootstrapping BlanketOps Environments secrets in %s (project: %s)\n", b.ProviderName(), b.projectID)

	secrets := []struct {
		name  string
		value string
	}{
		{name: "git_ssh-privatekey", value: s.GitSSHPrivateKey},
		{name: "git_ssh-publickey", value: s.GitSSHPublicKey},
		{name: "git_known-hosts", value: s.GitKnownHosts},
		{name: "registry_config", value: s.RegistryConfig},
		{name: "github_webhook_secret", value: s.GitHubWebhookSecret},
		{name: "github_api_token", value: s.GitHubAPIToken},
		{name: "crossplane_github_token", value: s.CrossplaneGitHubToken},
	}

	for _, sec := range secrets {
		fmt.Printf("  Creating %s...\n", b.flatName(fmt.Sprintf("blanketops_environments_%s", sec.name)))
		if err := b.createSecret(ctx, sec.name, sec.value); err != nil {
			return fmt.Errorf("create secret %s: %w", sec.name, err)
		}
	}
	return nil
}

func (b *GCPBootstrapper) Verify(ctx context.Context) error {
	req := &secretmanagerpb.ListSecretsRequest{
		Parent: fmt.Sprintf("projects/%s", b.projectID),
		Filter: "name:blanketops",
	}
	it := b.client.ListSecrets(ctx, req)
	fmt.Printf("Verified secrets in %s:\n", b.projectID)
	for {
		secret, err := it.Next()
		if err != nil {
			break
		}
		fmt.Printf("  - %s\n", secret.Name)
	}
	return nil
}

// ---------------------------------------------------------------------------
// HashiCorp Vault
// ---------------------------------------------------------------------------

type VaultBootstrapper struct {
	client *api.Client
	mount  string
}

func NewVaultBootstrapper(addr, token, mount string) (*VaultBootstrapper, error) {
	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create vault client: %w", err)
	}
	client.SetToken(token)
	return &VaultBootstrapper{client: client, mount: mount}, nil
}

func (b *VaultBootstrapper) ProviderName() string { return "HashiCorp Vault" }

func (b *VaultBootstrapper) writeSecret(ctx context.Context, path, value string) error {
	fullPath := fmt.Sprintf("%s/data/%s", strings.TrimPrefix(b.mount, "/"), path)
	data := map[string]interface{}{"data": map[string]interface{}{"value": value}}
	_, err := b.client.Logical().Write(fullPath, data)
	return err
}

func (b *VaultBootstrapper) Bootstrap(ctx context.Context, s *SecretSet) error {
	fmt.Printf("Bootstrapping BlanketOps Environments secrets in %s\n", b.ProviderName())

	secrets := []struct {
		path  string
		value string
	}{
		{path: "blanketops/environments/git/ssh-privatekey", value: s.GitSSHPrivateKey},
		{path: "blanketops/environments/git/ssh-publickey", value: s.GitSSHPublicKey},
		{path: "blanketops/environments/git/known-hosts", value: s.GitKnownHosts},
		{path: "blanketops/environments/registry/config", value: s.RegistryConfig},
		{path: "blanketops/environments/github/webhook/secret", value: s.GitHubWebhookSecret},
		{path: "blanketops/environments/github/api/token", value: s.GitHubAPIToken},
		{path: "blanketops/environments/crossplane/github/token", value: s.CrossplaneGitHubToken},
	}

	for _, sec := range secrets {
		fmt.Printf("  Creating %s...\n", sec.path)
		if err := b.writeSecret(ctx, sec.path, sec.value); err != nil {
			return fmt.Errorf("write secret %s: %w", sec.path, err)
		}
	}
	return nil
}

func (b *VaultBootstrapper) Verify(ctx context.Context) error {
	fullPath := fmt.Sprintf("%s/metadata", strings.TrimPrefix(b.mount, "/"))
	secret, err := b.client.Logical().List(fullPath)
	if err != nil {
		return err
	}
	if secret == nil || secret.Data == nil {
		fmt.Println("No secrets found")
		return nil
	}
	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return nil
	}
	fmt.Printf("Verified secrets in %s:\n", b.mount)
	for _, k := range keys {
		fmt.Printf("  - %s\n", k)
	}
	return nil
}

// ---------------------------------------------------------------------------
// CLI command wiring (called from main.go)
// ---------------------------------------------------------------------------

func SecretsBootstrap(ctx context.Context, provider string, args []string) error {
	var bootstrapper SecretBootstrapper
	var err error

	switch provider {
	case "aws":
		if len(args) < 1 {
			return errors.New("usage: bops-env secrets bootstrap aws <region> [path-prefix]")
		}
		region := args[0]
		pathPrefix := "/blanketops"
		if len(args) >= 2 {
			pathPrefix = args[1]
		}
		bootstrapper, err = NewAWSBootstrapper(ctx, region, pathPrefix)
	case "azure":
		if len(args) < 1 {
			return errors.New("usage: bops-env secrets bootstrap azure <vault-url>")
		}
		bootstrapper, err = NewAzureBootstrapper(ctx, args[0])
	case "gcp":
		if len(args) < 1 {
			return errors.New("usage: bops-env secrets bootstrap gcp <project-id>")
		}
		bootstrapper, err = NewGCPBootstrapper(ctx, args[0])
	case "vault":
		if len(args) < 2 {
			return errors.New("usage: bops-env secrets bootstrap vault <addr> <token> [mount]")
		}
		mount := "secret"
		if len(args) >= 3 {
			mount = args[2]
		}
		bootstrapper, err = NewVaultBootstrapper(args[0], args[1], mount)
	default:
		return fmt.Errorf("unknown provider: %s (supported: aws, azure, gcp, vault)", provider)
	}

	if err != nil {
		return fmt.Errorf("create %s bootstrapper: %w", provider, err)
	}

	secrets, err := DefaultSecretSet()
	if err != nil {
		return fmt.Errorf("prepare secret set: %w", err)
	}

	if err := bootstrapper.Bootstrap(ctx, secrets); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	fmt.Println("\n✅ Bootstrap complete!")
	return bootstrapper.Verify(ctx)
}

func SecretsVerify(ctx context.Context, provider string, args []string) error {
	var bootstrapper SecretBootstrapper
	var err error

	switch provider {
	case "aws":
		if len(args) < 1 {
			return errors.New("usage: bops-env secrets verify aws <region> [path-prefix]")
		}
		region := args[0]
		pathPrefix := "/blanketops"
		if len(args) >= 2 {
			pathPrefix = args[1]
		}
		bootstrapper, err = NewAWSBootstrapper(ctx, region, pathPrefix)
	case "azure":
		if len(args) < 1 {
			return errors.New("usage: bops-env secrets verify azure <vault-url>")
		}
		bootstrapper, err = NewAzureBootstrapper(ctx, args[0])
	case "gcp":
		if len(args) < 1 {
			return errors.New("usage: bops-env secrets verify gcp <project-id>")
		}
		bootstrapper, err = NewGCPBootstrapper(ctx, args[0])
	case "vault":
		if len(args) < 2 {
			return errors.New("usage: bops-env secrets verify vault <addr> <token> [mount]")
		}
		mount := "secret"
		if len(args) >= 3 {
			mount = args[2]
		}
		bootstrapper, err = NewVaultBootstrapper(args[0], args[1], mount)
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	if err != nil {
		return fmt.Errorf("create %s bootstrapper: %w", provider, err)
	}

	return bootstrapper.Verify(ctx)
}
