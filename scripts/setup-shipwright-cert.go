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

package scripts

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"
)

const (
	shipwrightCSRName        = "shipwright-build-webhook-csr"
	shipwrightNamespace      = "shipwright-build"
	shipwrightSecretName     = "shipwright-build-webhook-cert"
	shipwrightDeployment     = "shipwright-build-webhook"
	shipwrightCertPollPeriod = 1 * time.Second
)

var shipwrightSANDNSNames = []string{
	"shp-build-webhook",
	"shp-build-webhook.shipwright-build",
	"shp-build-webhook.shipwright-build.svc",
	"shp-build-webhook.shipwright-build.svc.cluster.local",
}

var shipwrightConversionCRDs = []string{
	"clusterbuildstrategies.shipwright.io",
	"buildstrategies.shipwright.io",
	"builds.shipwright.io",
	"buildruns.shipwright.io",
}

// SetupShipwrightCert issues a serving cert for the Shipwright Build webhook
// and patches it into the CRDs' conversion webhook caBundle.
//
// The original bash script shelled out to openssl (to build the key/CSR) and
// jq + base64 (to read the signed cert back and encode the CA bundle). openssl
// isn't on Windows by default, and macOS's BSD base64 takes different flags
// than Linux's GNU base64 — both real cross-platform footguns. Here the key
// pair and CSR are generated with Go's crypto/x509 library and base64/JSON use
// encoding/base64 + encoding/json, so the only external dependency left is
// kubectl.
func SetupShipwrightCert() error {
	if err := requireBinary("kubectl"); err != nil {
		return err
	}

	fmt.Println("[INFO] Generating key and signing request for Shipwright Build Webhook")

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating RSA key: %w", err)
	}
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			Organization: []string{"system:nodes"},
			CommonName:   "system:node:shp-build-webhook.shipwright-build.svc.cluster.local",
		},
		DNSNames:           shipwrightSANDNSNames,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, key)
	if err != nil {
		return fmt.Errorf("creating CSR: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	fmt.Println("[INFO] Deleting previous CertificateSigningRequest")
	_ = run("kubectl", "delete", "csr", shipwrightCSRName, "--ignore-not-found")

	fmt.Println("[INFO] Create a CertificateSigningRequest")
	csrManifest, _ := json.Marshal(map[string]any{
		"apiVersion": "certificates.k8s.io/v1",
		"kind":       "CertificateSigningRequest",
		"metadata":   map[string]any{"name": shipwrightCSRName},
		"spec": map[string]any{
			"groups":     []string{"system:authenticated"},
			"request":    base64.StdEncoding.EncodeToString(csrPEM),
			"signerName": "kubernetes.io/kubelet-serving",
			"usages":     []string{"digital signature", "key encipherment", "server auth"},
		},
	})
	if err := runStdin(csrManifest, "kubectl", "create", "-f", "-"); err != nil {
		return fmt.Errorf("creating CSR object: %w", err)
	}

	fmt.Println("[INFO] Approve the CertificateSigningRequest")
	if err := run("kubectl", "certificate", "approve", shipwrightCSRName); err != nil {
		return fmt.Errorf("approving CSR: %w", err)
	}

	var certB64 string
	deadline := time.Now().Add(5 * time.Minute)
	for {
		out, err := capture("kubectl", "get", "csr", shipwrightCSRName, "-o", "jsonpath={.status.certificate}")
		if err == nil && out != "" && out != "null" {
			certB64 = out
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for CSR %s to be signed", shipwrightCSRName)
		}
		fmt.Println("[INFO] Waiting for certificate to be ready")
		time.Sleep(shipwrightCertPollPeriod)
	}

	certPEM, err := base64.StdEncoding.DecodeString(certB64)
	if err != nil {
		return fmt.Errorf("decoding signed certificate: %w", err)
	}

	fmt.Println("[INFO] Deleting the CertificateSigningRequest")
	_ = run("kubectl", "delete", "csr", shipwrightCSRName, "--ignore-not-found")

	fmt.Printf("[INFO] Creating TLS secret %s\n", shipwrightSecretName)
	_ = run("kubectl", "-n", shipwrightNamespace, "delete", "secret", shipwrightSecretName, "--ignore-not-found")
	secretManifest, _ := json.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"type":       "kubernetes.io/tls",
		"metadata":   map[string]any{"name": shipwrightSecretName, "namespace": shipwrightNamespace},
		"data": map[string]any{
			"tls.crt": base64.StdEncoding.EncodeToString(certPEM),
			"tls.key": base64.StdEncoding.EncodeToString(keyPEM),
		},
	})
	if err := runStdin(secretManifest, "kubectl", "create", "-f", "-"); err != nil {
		return fmt.Errorf("creating TLS secret: %w", err)
	}

	fmt.Println("[INFO] Retrieving CABundle")
	caData, err := capture("kubectl", "get", "configmap", "-n", "kube-system",
		"extension-apiserver-authentication", "-o", "jsonpath={.data.client-ca-file}")
	if err != nil {
		return fmt.Errorf("retrieving CA bundle: %w", err)
	}
	caBundle := base64.StdEncoding.EncodeToString([]byte(caData))

	fmt.Println("[INFO] Patching caBundle into CustomResourceDefinitions")
	for _, crd := range shipwrightConversionCRDs {
		patch, _ := json.Marshal(map[string]any{
			"spec": map[string]any{
				"conversion": map[string]any{
					"webhook": map[string]any{
						"clientConfig": map[string]any{"caBundle": caBundle},
					},
				},
			},
		})
		if err := run("kubectl", "patch", "crd", crd, "-p", string(patch)); err != nil {
			return fmt.Errorf("patching CRD %s: %w", crd, err)
		}
	}

	fmt.Println("[INFO] Restarting shipwright-build-webhook")
	if err := run("kubectl", "-n", shipwrightNamespace, "rollout", "restart", "deployment", shipwrightDeployment); err != nil {
		return fmt.Errorf("rollout restart failed: %w", err)
	}
	if err := run("kubectl", "-n", shipwrightNamespace, "rollout", "status", "deployment", shipwrightDeployment); err != nil {
		return fmt.Errorf("rollout status failed: %w", err)
	}
	return nil
}
