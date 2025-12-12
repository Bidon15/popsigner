package unseal

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
)

// AWSKMSProvider implements the Provider interface for AWS KMS.
type AWSKMSProvider struct{}

// Name returns the provider name.
func (p *AWSKMSProvider) Name() string {
	return "awskms"
}

// Validate checks if the configuration is valid.
func (p *AWSKMSProvider) Validate(spec *popsignerv1.AutoUnsealSpec) error {
	if spec.AWSKMS == nil {
		return fmt.Errorf("awskms configuration required")
	}
	if spec.AWSKMS.KeyID == "" {
		return fmt.Errorf("awskms.keyId is required")
	}
	return nil
}

// GetConfig returns the HCL configuration for the seal stanza.
func (p *AWSKMSProvider) GetConfig(spec *popsignerv1.AutoUnsealSpec) string {
	if spec.AWSKMS == nil {
		return ""
	}

	region := spec.AWSKMS.Region
	if region == "" {
		region = "us-east-1"
	}

	return fmt.Sprintf(`seal "awskms" {
  region     = "%s"
  kms_key_id = "%s"
}`, region, spec.AWSKMS.KeyID)
}

// GetEnvVars returns environment variables needed by the provider.
func (p *AWSKMSProvider) GetEnvVars(ctx context.Context, spec *popsignerv1.AutoUnsealSpec, namespace string) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar

	if spec.AWSKMS == nil {
		return envVars, nil
	}

	if spec.AWSKMS.Region != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "AWS_REGION",
			Value: spec.AWSKMS.Region,
		})
	}

	// If credentials secret is specified, add env vars from secret
	if spec.AWSKMS.Credentials != nil {
		envVars = append(envVars,
			corev1.EnvVar{
				Name: "AWS_ACCESS_KEY_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: spec.AWSKMS.Credentials.Name,
						},
						Key: "access-key-id",
					},
				},
			},
			corev1.EnvVar{
				Name: "AWS_SECRET_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: spec.AWSKMS.Credentials.Name,
						},
						Key: "secret-access-key",
					},
				},
			},
		)
	}

	return envVars, nil
}

// GetVolumes returns additional volumes needed by the provider.
func (p *AWSKMSProvider) GetVolumes(spec *popsignerv1.AutoUnsealSpec) []corev1.Volume {
	// AWS KMS doesn't require additional volumes
	return nil
}

// GetVolumeMounts returns additional volume mounts needed by the provider.
func (p *AWSKMSProvider) GetVolumeMounts(spec *popsignerv1.AutoUnsealSpec) []corev1.VolumeMount {
	// AWS KMS doesn't require additional volume mounts
	return nil
}
