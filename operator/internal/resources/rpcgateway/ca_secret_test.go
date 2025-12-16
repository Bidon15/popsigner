package rpcgateway

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
)

func TestCASecretName(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *popsignerv1.POPSignerCluster
		expected string
	}{
		{
			name: "default name",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "prod"},
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{},
				},
			},
			expected: "prod-rpc-gateway-ca",
		},
		{
			name: "custom name",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "prod"},
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{
						MTLS: popsignerv1.MTLSConfig{
							CASecretName: "my-ca",
						},
					},
				},
			},
			expected: "my-ca",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CASecretName(tt.cluster); got != tt.expected {
				t.Errorf("CASecretName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCASecretKey(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *popsignerv1.POPSignerCluster
		expected string
	}{
		{
			name: "default key",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "prod"},
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{},
				},
			},
			expected: "ca.crt",
		},
		{
			name: "custom key",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "prod"},
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{
						MTLS: popsignerv1.MTLSConfig{
							CASecretKey: "custom-ca.pem",
						},
					},
				},
			},
			expected: "custom-ca.pem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CASecretKey(tt.cluster); got != tt.expected {
				t.Errorf("CASecretKey() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClientAuthType(t *testing.T) {
	tests := []struct {
		name     string
		cluster  *popsignerv1.POPSignerCluster
		expected string
	}{
		{
			name: "default auth type",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "prod"},
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{},
				},
			},
			expected: "VerifyClientCertIfGiven",
		},
		{
			name: "custom auth type",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "prod"},
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{
						MTLS: popsignerv1.MTLSConfig{
							ClientAuthType: "RequireAndVerifyClientCert",
						},
					},
				},
			},
			expected: "RequireAndVerifyClientCert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientAuthType(tt.cluster); got != tt.expected {
				t.Errorf("ClientAuthType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetMTLSEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		mtlsEnabled bool
		wantLen     int
	}{
		{
			name:        "disabled",
			mtlsEnabled: false,
			wantLen:     0,
		},
		{
			name:        "enabled",
			mtlsEnabled: true,
			wantLen:     3, // MTLS_ENABLED, MTLS_CA_CERT_PATH, MTLS_CLIENT_AUTH_TYPE
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &popsignerv1.POPSignerCluster{
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{
						MTLS: popsignerv1.MTLSConfig{
							Enabled: tt.mtlsEnabled,
						},
					},
				},
			}

			envVars := GetMTLSEnvVars(cluster)
			if len(envVars) != tt.wantLen {
				t.Errorf("GetMTLSEnvVars() returned %d vars, want %d", len(envVars), tt.wantLen)
			}
		})
	}
}

func TestGetMTLSEnvVarsValues(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		Spec: popsignerv1.POPSignerClusterSpec{
			RPCGateway: popsignerv1.RPCGatewaySpec{
				MTLS: popsignerv1.MTLSConfig{
					Enabled:        true,
					ClientAuthType: "RequireAndVerifyClientCert",
				},
			},
		},
	}

	envVars := GetMTLSEnvVars(cluster)

	envMap := make(map[string]string)
	for _, env := range envVars {
		envMap[env.Name] = env.Value
	}

	if envMap["MTLS_ENABLED"] != "true" {
		t.Errorf("expected MTLS_ENABLED to be 'true', got %q", envMap["MTLS_ENABLED"])
	}
	if envMap["MTLS_CA_CERT_PATH"] != "/etc/popsigner/ca/ca.crt" {
		t.Errorf("expected MTLS_CA_CERT_PATH to be '/etc/popsigner/ca/ca.crt', got %q", envMap["MTLS_CA_CERT_PATH"])
	}
	if envMap["MTLS_CLIENT_AUTH_TYPE"] != "RequireAndVerifyClientCert" {
		t.Errorf("expected MTLS_CLIENT_AUTH_TYPE to be 'RequireAndVerifyClientCert', got %q", envMap["MTLS_CLIENT_AUTH_TYPE"])
	}
}

func TestGetCASecretVolume(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: popsignerv1.POPSignerClusterSpec{
			RPCGateway: popsignerv1.RPCGatewaySpec{
				MTLS: popsignerv1.MTLSConfig{
					Enabled: true,
				},
			},
		},
	}

	volume := GetCASecretVolume(cluster)

	if volume.Name != CAVolumeName {
		t.Errorf("expected volume name %q, got %q", CAVolumeName, volume.Name)
	}
	if volume.Secret == nil {
		t.Fatal("expected Secret volume source to be set")
	}
	if volume.Secret.SecretName != "test-cluster-rpc-gateway-ca" {
		t.Errorf("expected secret name %q, got %q", "test-cluster-rpc-gateway-ca", volume.Secret.SecretName)
	}
	if len(volume.Secret.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(volume.Secret.Items))
	}
	if volume.Secret.Items[0].Key != "ca.crt" {
		t.Errorf("expected key 'ca.crt', got %q", volume.Secret.Items[0].Key)
	}
	if volume.Secret.Items[0].Path != "ca.crt" {
		t.Errorf("expected path 'ca.crt', got %q", volume.Secret.Items[0].Path)
	}
}

func TestGetCASecretVolumeCustomSecret(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
		Spec: popsignerv1.POPSignerClusterSpec{
			RPCGateway: popsignerv1.RPCGatewaySpec{
				MTLS: popsignerv1.MTLSConfig{
					Enabled:      true,
					CASecretName: "custom-ca-secret",
					CASecretKey:  "custom.crt",
				},
			},
		},
	}

	volume := GetCASecretVolume(cluster)

	if volume.Secret.SecretName != "custom-ca-secret" {
		t.Errorf("expected secret name %q, got %q", "custom-ca-secret", volume.Secret.SecretName)
	}
	if volume.Secret.Items[0].Key != "custom.crt" {
		t.Errorf("expected key 'custom.crt', got %q", volume.Secret.Items[0].Key)
	}
}

func TestGetCASecretVolumeMount(t *testing.T) {
	mount := GetCASecretVolumeMount()

	if mount.Name != CAVolumeName {
		t.Errorf("expected mount name %q, got %q", CAVolumeName, mount.Name)
	}
	if mount.MountPath != CAMountPath {
		t.Errorf("expected mount path %q, got %q", CAMountPath, mount.MountPath)
	}
	if !mount.ReadOnly {
		t.Error("expected mount to be read-only")
	}
}

func TestEnsureCASecret(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = popsignerv1.AddToScheme(scheme)

	tests := []struct {
		name           string
		cluster        *popsignerv1.POPSignerCluster
		existingSecret *corev1.Secret
		wantCreated    bool
	}{
		{
			name: "mTLS disabled - no secret created",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{
						MTLS: popsignerv1.MTLSConfig{
							Enabled: false,
						},
					},
				},
			},
			wantCreated: false,
		},
		{
			name: "mTLS enabled - creates secret",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{
						MTLS: popsignerv1.MTLSConfig{
							Enabled: true,
						},
					},
				},
			},
			wantCreated: true,
		},
		{
			name: "mTLS enabled - secret already exists",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec: popsignerv1.POPSignerClusterSpec{
					RPCGateway: popsignerv1.RPCGatewaySpec{
						MTLS: popsignerv1.MTLSConfig{
							Enabled: true,
						},
					},
				},
			},
			existingSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-rpc-gateway-ca",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"ca.crt": []byte("existing-ca-cert"),
				},
			},
			wantCreated: false, // Should not overwrite existing secret
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objs []client.Object
			if tt.existingSecret != nil {
				objs = append(objs, tt.existingSecret)
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				Build()

			err := EnsureCASecret(context.Background(), fakeClient, tt.cluster)
			if err != nil {
				t.Fatalf("EnsureCASecret() error = %v", err)
			}

			// Check if secret exists
			secret := &corev1.Secret{}
			secretName := CASecretName(tt.cluster)
			err = fakeClient.Get(context.Background(), client.ObjectKey{
				Name:      secretName,
				Namespace: tt.cluster.Namespace,
			}, secret)

			if tt.wantCreated {
				if err != nil {
					t.Errorf("expected secret to be created, got error: %v", err)
				}
				// Verify labels
				if secret.Labels["app.kubernetes.io/component"] != "rpc-gateway" {
					t.Error("expected component label to be 'rpc-gateway'")
				}
				if secret.Labels["popsigner.com/cluster"] != tt.cluster.Name {
					t.Errorf("expected cluster label to be %q", tt.cluster.Name)
				}
			}

			// If existing secret was provided, verify it wasn't overwritten
			if tt.existingSecret != nil {
				if string(secret.Data["ca.crt"]) != "existing-ca-cert" {
					t.Error("existing secret should not be overwritten")
				}
			}
		})
	}
}

func TestEnsureCASecretWithCustomName(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = popsignerv1.AddToScheme(scheme)

	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: popsignerv1.POPSignerClusterSpec{
			RPCGateway: popsignerv1.RPCGatewaySpec{
				MTLS: popsignerv1.MTLSConfig{
					Enabled:      true,
					CASecretName: "my-custom-ca",
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	err := EnsureCASecret(context.Background(), fakeClient, cluster)
	if err != nil {
		t.Fatalf("EnsureCASecret() error = %v", err)
	}

	// Verify secret was created with custom name
	secret := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{
		Name:      "my-custom-ca",
		Namespace: "default",
	}, secret)
	if err != nil {
		t.Errorf("expected secret with custom name to be created, got error: %v", err)
	}
}

