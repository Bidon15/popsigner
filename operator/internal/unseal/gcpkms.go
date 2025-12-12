package unseal

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
)

// GCPKMSProvider implements the Provider interface for GCP Cloud KMS.
type GCPKMSProvider struct{}

// Name returns the provider name.
func (p *GCPKMSProvider) Name() string {
	return "gcpkms"
}

// Validate checks if the configuration is valid.
func (p *GCPKMSProvider) Validate(spec *popsignerv1.AutoUnsealSpec) error {
	if spec.GCPKMS == nil {
		return fmt.Errorf("gcpkms configuration required")
	}
	if spec.GCPKMS.Project == "" {
		return fmt.Errorf("gcpkms.project is required")
	}
	if spec.GCPKMS.Location == "" {
		return fmt.Errorf("gcpkms.location is required")
	}
	if spec.GCPKMS.KeyRing == "" {
		return fmt.Errorf("gcpkms.keyRing is required")
	}
	if spec.GCPKMS.CryptoKey == "" {
		return fmt.Errorf("gcpkms.cryptoKey is required")
	}
	return nil
}

// GetConfig returns the HCL configuration for the seal stanza.
func (p *GCPKMSProvider) GetConfig(spec *popsignerv1.AutoUnsealSpec) string {
	if spec.GCPKMS == nil {
		return ""
	}

	return fmt.Sprintf(`seal "gcpckms" {
  project     = "%s"
  region      = "%s"
  key_ring    = "%s"
  crypto_key  = "%s"
}`, spec.GCPKMS.Project, spec.GCPKMS.Location, spec.GCPKMS.KeyRing, spec.GCPKMS.CryptoKey)
}

// GetEnvVars returns environment variables needed by the provider.
func (p *GCPKMSProvider) GetEnvVars(ctx context.Context, spec *popsignerv1.AutoUnsealSpec, namespace string) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	if spec.GCPKMS == nil {
		return envVars, nil
	}

	envVars = append(envVars, corev1.EnvVar{
		Name:  "GOOGLE_PROJECT",
		Value: spec.GCPKMS.Project,
	})

	// If credentials secret is specified, point to the mounted credentials file
	if spec.GCPKMS.Credentials != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "GOOGLE_APPLICATION_CREDENTIALS",
			Value: "/vault/gcp/credentials.json",
		})
	}

	return envVars, nil
}

// GetVolumes returns additional volumes needed by the provider.
func (p *GCPKMSProvider) GetVolumes(spec *popsignerv1.AutoUnsealSpec) []corev1.Volume {
	if spec.GCPKMS == nil || spec.GCPKMS.Credentials == nil {
		return nil
	}

	return []corev1.Volume{
		{
			Name: "gcp-credentials",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: spec.GCPKMS.Credentials.Name,
					Items: []corev1.KeyToPath{
						{
							Key:  spec.GCPKMS.Credentials.Key,
							Path: "credentials.json",
						},
					},
				},
			},
		},
	}
}

// GetVolumeMounts returns additional volume mounts needed by the provider.
func (p *GCPKMSProvider) GetVolumeMounts(spec *popsignerv1.AutoUnsealSpec) []corev1.VolumeMount {
	if spec.GCPKMS == nil || spec.GCPKMS.Credentials == nil {
		return nil
	}

	return []corev1.VolumeMount{
		{
			Name:      "gcp-credentials",
			MountPath: "/vault/gcp",
			ReadOnly:  true,
		},
	}
}
