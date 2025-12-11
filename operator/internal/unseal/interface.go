// Package unseal provides auto-unseal provider implementations for OpenBao.
package unseal

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
)

// Provider defines the interface for auto-unseal providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// Validate checks if the configuration is valid.
	Validate(spec *banhbaoringv1.AutoUnsealSpec) error

	// GetConfig returns the HCL configuration for the seal stanza.
	GetConfig(spec *banhbaoringv1.AutoUnsealSpec) string

	// GetEnvVars returns environment variables needed by the provider.
	GetEnvVars(ctx context.Context, spec *banhbaoringv1.AutoUnsealSpec, namespace string) ([]corev1.EnvVar, error)

	// GetVolumes returns additional volumes needed by the provider.
	GetVolumes(spec *banhbaoringv1.AutoUnsealSpec) []corev1.Volume

	// GetVolumeMounts returns additional volume mounts needed by the provider.
	GetVolumeMounts(spec *banhbaoringv1.AutoUnsealSpec) []corev1.VolumeMount
}

// NewProvider creates the appropriate unseal provider.
func NewProvider(providerType string) (Provider, error) {
	switch providerType {
	case "awskms":
		return &AWSKMSProvider{}, nil
	case "gcpkms":
		return &GCPKMSProvider{}, nil
	case "azurekv":
		return &AzureKVProvider{}, nil
	case "transit":
		return &TransitProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetProviderForCluster returns the appropriate provider for a cluster.
func GetProviderForCluster(cluster *banhbaoringv1.BanhBaoRingCluster) (Provider, error) {
	if !cluster.Spec.OpenBao.AutoUnseal.Enabled {
		return nil, nil
	}

	return NewProvider(cluster.Spec.OpenBao.AutoUnseal.Provider)
}
