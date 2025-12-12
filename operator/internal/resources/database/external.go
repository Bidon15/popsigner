package database

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
)

// ExternalConfig holds external database configuration
type ExternalConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
	URL      string
}

// GetExternalConfig retrieves external database config from secret
func GetExternalConfig(ctx context.Context, c client.Client, cluster *popsignerv1.POPSignerCluster) (*ExternalConfig, error) {
	if cluster.Spec.Database.Managed {
		return nil, fmt.Errorf("database is managed, not external")
	}

	connRef := cluster.Spec.Database.ConnectionString
	if connRef == nil {
		return nil, fmt.Errorf("connectionString not configured for external database")
	}

	secret := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{
		Name:      connRef.Name,
		Namespace: cluster.Namespace,
	}, secret); err != nil {
		return nil, fmt.Errorf("failed to get connection string secret: %w", err)
	}

	connString, ok := secret.Data[connRef.Key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret", connRef.Key)
	}

	// Return the raw URL for simplicity
	// In production, properly parse the URL
	return &ExternalConfig{
		URL: string(connString),
	}, nil
}

// ConnectionStringSecret returns the connection string for applications
func ConnectionStringSecret(cluster *popsignerv1.POPSignerCluster, connString string) *corev1.Secret {
	name := fmt.Sprintf("%s-database-url", cluster.Name)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"url": connString,
		},
	}
}
