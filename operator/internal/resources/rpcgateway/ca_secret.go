// Package rpcgateway provides resource builders for the JSON-RPC gateway.
package rpcgateway

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
)

const (
	// DefaultCASecretKey is the default key for CA certificate in the secret.
	DefaultCASecretKey = "ca.crt"

	// DefaultClientAuthType allows both API key and mTLS.
	DefaultClientAuthType = "VerifyClientCertIfGiven"

	// CAVolumeName is the name of the volume for CA certificate.
	CAVolumeName = "ca-cert"

	// CAMountPath is the path where CA certificate is mounted.
	CAMountPath = "/etc/popsigner/ca"
)

// CASecretName returns the name of the CA secret for the cluster.
func CASecretName(cluster *popsignerv1.POPSignerCluster) string {
	if cluster.Spec.RPCGateway.MTLS.CASecretName != "" {
		return cluster.Spec.RPCGateway.MTLS.CASecretName
	}
	return fmt.Sprintf("%s-rpc-gateway-ca", cluster.Name)
}

// CASecretKey returns the key in the secret containing the CA certificate.
func CASecretKey(cluster *popsignerv1.POPSignerCluster) string {
	if cluster.Spec.RPCGateway.MTLS.CASecretKey != "" {
		return cluster.Spec.RPCGateway.MTLS.CASecretKey
	}
	return DefaultCASecretKey
}

// ClientAuthType returns the TLS client auth type.
func ClientAuthType(cluster *popsignerv1.POPSignerCluster) string {
	if cluster.Spec.RPCGateway.MTLS.ClientAuthType != "" {
		return cluster.Spec.RPCGateway.MTLS.ClientAuthType
	}
	return DefaultClientAuthType
}

// EnsureCASecret ensures the CA secret exists.
// Note: The actual CA certificate is managed by the control plane's certificate service.
// This function creates a placeholder secret that should be populated externally or
// by the control plane during initialization.
func EnsureCASecret(ctx context.Context, c client.Client, cluster *popsignerv1.POPSignerCluster) error {
	if !cluster.Spec.RPCGateway.MTLS.Enabled {
		return nil
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CASecretName(cluster),
			Namespace: cluster.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "popsigner",
				"app.kubernetes.io/component":  "rpc-gateway",
				"app.kubernetes.io/managed-by": "popsigner-operator",
				"popsigner.com/cluster":        cluster.Name,
			},
		},
		Type: corev1.SecretTypeOpaque,
	}

	// Check if secret already exists
	existing := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}, existing)

	if err == nil {
		// Secret exists, don't overwrite
		return nil
	}

	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("checking CA secret: %w", err)
	}

	// Create placeholder secret with instructions
	secret.StringData = map[string]string{
		DefaultCASecretKey: `# POPSigner CA Certificate
# This secret should contain the POPSigner CA certificate.
# 
# Option 1: Let the control plane populate this during initialization
# Option 2: Manually add your CA certificate:
#   kubectl create secret generic <name> --from-file=ca.crt=path/to/ca.crt
#
# The CA certificate is used to verify client certificates during mTLS authentication.
`,
	}

	if err := c.Create(ctx, secret); err != nil {
		return fmt.Errorf("creating CA secret: %w", err)
	}

	return nil
}

// GetCASecretVolume returns the volume definition for mounting the CA secret.
func GetCASecretVolume(cluster *popsignerv1.POPSignerCluster) corev1.Volume {
	return corev1.Volume{
		Name: CAVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: CASecretName(cluster),
				Items: []corev1.KeyToPath{
					{
						Key:  CASecretKey(cluster),
						Path: "ca.crt",
					},
				},
			},
		},
	}
}

// GetCASecretVolumeMount returns the volume mount for the CA certificate.
func GetCASecretVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      CAVolumeName,
		MountPath: CAMountPath,
		ReadOnly:  true,
	}
}

// GetMTLSEnvVars returns environment variables for mTLS configuration.
func GetMTLSEnvVars(cluster *popsignerv1.POPSignerCluster) []corev1.EnvVar {
	if !cluster.Spec.RPCGateway.MTLS.Enabled {
		return nil
	}

	return []corev1.EnvVar{
		{
			Name:  "MTLS_ENABLED",
			Value: "true",
		},
		{
			Name:  "MTLS_CA_CERT_PATH",
			Value: fmt.Sprintf("%s/ca.crt", CAMountPath),
		},
		{
			Name:  "MTLS_CLIENT_AUTH_TYPE",
			Value: ClientAuthType(cluster),
		},
	}
}

