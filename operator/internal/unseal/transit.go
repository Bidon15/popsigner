package unseal

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
)

// TransitProvider implements the Provider interface for Transit (Vault-to-Vault) unseal.
type TransitProvider struct{}

// Name returns the provider name.
func (p *TransitProvider) Name() string {
	return "transit"
}

// Validate checks if the configuration is valid.
func (p *TransitProvider) Validate(spec *popsignerv1.AutoUnsealSpec) error {
	if spec.Transit == nil {
		return fmt.Errorf("transit configuration required")
	}
	if spec.Transit.Address == "" {
		return fmt.Errorf("transit.address is required")
	}
	if spec.Transit.KeyName == "" {
		return fmt.Errorf("transit.keyName is required")
	}
	return nil
}

// GetConfig returns the HCL configuration for the seal stanza.
func (p *TransitProvider) GetConfig(spec *popsignerv1.AutoUnsealSpec) string {
	if spec.Transit == nil {
		return ""
	}

	mountPath := spec.Transit.MountPath
	if mountPath == "" {
		mountPath = "transit"
	}

	return fmt.Sprintf(`seal "transit" {
  address         = "%s"
  disable_renewal = "false"
  mount_path      = "%s"
  key_name        = "%s"
  tls_skip_verify = "true"
}`, spec.Transit.Address, mountPath, spec.Transit.KeyName)
}

// GetEnvVars returns environment variables needed by the provider.
func (p *TransitProvider) GetEnvVars(ctx context.Context, spec *popsignerv1.AutoUnsealSpec, namespace string) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	if spec.Transit == nil {
		return envVars, nil
	}

	// Add the transit token from secret
	envVars = append(envVars, corev1.EnvVar{
		Name: "VAULT_TOKEN",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: spec.Transit.Token.Name,
				},
				Key: spec.Transit.Token.Key,
			},
		},
	})

	return envVars, nil
}

// GetVolumes returns additional volumes needed by the provider.
func (p *TransitProvider) GetVolumes(spec *popsignerv1.AutoUnsealSpec) []corev1.Volume {
	// Transit provider doesn't require additional volumes
	return nil
}

// GetVolumeMounts returns additional volume mounts needed by the provider.
func (p *TransitProvider) GetVolumeMounts(spec *popsignerv1.AutoUnsealSpec) []corev1.VolumeMount {
	// Transit provider doesn't require additional volume mounts
	return nil
}
