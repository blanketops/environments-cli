module github.com/blanketops/environments-cli

go 1.26.0

toolchain go1.26.10

require (
	cloud.google.com/go/secretmanager v1.23.1
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.15.0
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets v0.12.0
	github.com/aws/aws-sdk-go-v2 v1.47.5
	github.com/aws/aws-sdk-go-v2/config v1.33.12
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.50.7
	github.com/hashicorp/vault/api v1.23.0
	github.com/spf13/cobra v1.10.2
	k8s.io/apimachinery v0.37.1
	k8s.io/client-go v0.37.1
)