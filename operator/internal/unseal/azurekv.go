package unseal

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
)

// AzureKVProvider implements the Provider interface for Azure Key Vault.
type AzureKVProvider struct{}

// Name returns the provider name.
func (p *AzureKVProvider) Name() string {
	return "azurekv"
}

// Validate checks if the configuration is valid.
func (p *AzureKVProvider) Validate(spec *popsignerv1.AutoUnsealSpec) error {
	if spec.AzureKV == nil {
		return fmt.Errorf("azurekv configuration required")
	}
	if spec.AzureKV.TenantID == "" {
		return fmt.Errorf("azurekv.tenantId is required")
	}
	if spec.AzureKV.VaultName == "" {
		return fmt.Errorf("azurekv.vaultName is required")
	}
	if spec.AzureKV.KeyName == "" {
		return fmt.Errorf("azurekv.keyName is required")
	}
	return nil
}

// GetConfig returns the HCL configuration for the seal stanza.
func (p *AzureKVProvider) GetConfig(spec *popsignerv1.AutoUnsealSpec) string {
	if spec.AzureKV == nil {
		return ""
	}

	return fmt.Sprintf(`seal "azurekeyvault" {
  tenant_id  = "%s"
  vault_name = "%s"
  key_name   = "%s"
}`, spec.AzureKV.TenantID, spec.AzureKV.VaultName, spec.AzureKV.KeyName)
}

// GetEnvVars returns environment variables needed by the provider.
func (p *AzureKVProvider) GetEnvVars(ctx context.Context, spec *popsignerv1.AutoUnsealSpec, namespace string) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	if spec.AzureKV == nil {
		return envVars, nil
	}

	envVars = append(envVars, corev1.EnvVar{
		Name:  "AZURE_TENANT_ID",
		Value: spec.AzureKV.TenantID,
	})

	// If credentials secret is specified, add env vars from secret
	if spec.AzureKV.Credentials != nil {
		envVars = append(envVars,
			corev1.EnvVar{
				Name: "AZURE_CLIENT_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: spec.AzureKV.Credentials.Name,
						},
						Key: "client-id",
					},
				},
			},
			corev1.EnvVar{
				Name: "AZURE_CLIENT_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: spec.AzureKV.Credentials.Name,
						},
						Key: "client-secret",
					},
				},
			},
		)
	}

	return envVars, nil
}

// GetVolumes returns additional volumes needed by the provider.
func (p *AzureKVProvider) GetVolumes(spec *popsignerv1.AutoUnsealSpec) []corev1.Volume {
	// Azure Key Vault doesn't require additional volumes
	return nil
}

// GetVolumeMounts returns additional volume mounts needed by the provider.
func (p *AzureKVProvider) GetVolumeMounts(spec *popsignerv1.AutoUnsealSpec) []corev1.VolumeMount {
	// Azure Key Vault doesn't require additional volume mounts
	return nil
}
