package openbao

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

func TestConfigMap(t *testing.T) {
	tests := []struct {
		name         string
		cluster      *popsignerv1.POPSignerCluster
		wantReplicas int
		wantAutoSeal bool
		wantSealType string
	}{
		{
			name: "default configuration",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{},
				},
			},
			wantReplicas: 3,
			wantAutoSeal: false,
		},
		{
			name: "custom replicas",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "production",
				},
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						Replicas: 5,
						Version:  "2.1.0",
					},
				},
			},
			wantReplicas: 5,
			wantAutoSeal: false,
		},
		{
			name: "with AWS KMS auto-unseal",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						AutoUnseal: popsignerv1.AutoUnsealSpec{
							Enabled:  true,
							Provider: "awskms",
							AWSKMS: &popsignerv1.AWSKMSSpec{
								KeyID:  "key-123",
								Region: "us-west-2",
							},
						},
					},
				},
			},
			wantReplicas: 3,
			wantAutoSeal: true,
			wantSealType: "awskms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := ConfigMap(tt.cluster)
			if err != nil {
				t.Fatalf("ConfigMap() error = %v", err)
			}

			// Check name
			expectedName := tt.cluster.Name + "-openbao-config"
			if cm.Name != expectedName {
				t.Errorf("ConfigMap name = %v, want %v", cm.Name, expectedName)
			}

			// Check namespace
			if cm.Namespace != tt.cluster.Namespace {
				t.Errorf("ConfigMap namespace = %v, want %v", cm.Namespace, tt.cluster.Namespace)
			}

			// Check config.hcl exists
			config, ok := cm.Data["config.hcl"]
			if !ok {
				t.Fatal("config.hcl not found in ConfigMap data")
			}

			// Check basic config entries
			if !strings.Contains(config, "ui = true") {
				t.Error("Config should contain 'ui = true'")
			}
			if !strings.Contains(config, "storage \"raft\"") {
				t.Error("Config should contain Raft storage configuration")
			}
			if !strings.Contains(config, "listener \"tcp\"") {
				t.Error("Config should contain TCP listener configuration")
			}
			if !strings.Contains(config, "plugin_directory") {
				t.Error("Config should contain plugin_directory")
			}

			// Check retry_join entries match replica count
			joinCount := strings.Count(config, "retry_join")
			if joinCount != tt.wantReplicas {
				t.Errorf("Expected %d retry_join entries, got %d", tt.wantReplicas, joinCount)
			}

			// Check auto-unseal configuration
			if tt.wantAutoSeal {
				if !strings.Contains(config, "seal \""+tt.wantSealType+"\"") {
					t.Errorf("Config should contain seal configuration for %s", tt.wantSealType)
				}
			}

			// Check labels
			if cm.Labels[constants.LabelComponent] != constants.ComponentOpenBao {
				t.Errorf("Label component = %v, want %v", cm.Labels[constants.LabelComponent], constants.ComponentOpenBao)
			}
		})
	}
}

func TestBuildSealConfig(t *testing.T) {
	tests := []struct {
		name         string
		cluster      *popsignerv1.POPSignerCluster
		wantEmpty    bool
		wantSealType string
		wantContains []string
	}{
		{
			name: "no auto-unseal",
			cluster: &popsignerv1.POPSignerCluster{
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						AutoUnseal: popsignerv1.AutoUnsealSpec{
							Enabled: false,
						},
					},
				},
			},
			wantEmpty: true,
		},
		{
			name: "aws kms",
			cluster: &popsignerv1.POPSignerCluster{
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						AutoUnseal: popsignerv1.AutoUnsealSpec{
							Enabled:  true,
							Provider: "awskms",
							AWSKMS: &popsignerv1.AWSKMSSpec{
								KeyID:  "alias/my-key",
								Region: "eu-west-1",
							},
						},
					},
				},
			},
			wantEmpty:    false,
			wantSealType: "awskms",
			wantContains: []string{"region", "eu-west-1", "kms_key_id", "alias/my-key"},
		},
		{
			name: "gcp kms",
			cluster: &popsignerv1.POPSignerCluster{
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						AutoUnseal: popsignerv1.AutoUnsealSpec{
							Enabled:  true,
							Provider: "gcpkms",
							GCPKMS: &popsignerv1.GCPKMSSpec{
								Project:   "my-project",
								Location:  "global",
								KeyRing:   "my-keyring",
								CryptoKey: "my-key",
							},
						},
					},
				},
			},
			wantEmpty:    false,
			wantSealType: "gcpckms",
			wantContains: []string{"project", "my-project", "key_ring", "my-keyring"},
		},
		{
			name: "azure key vault",
			cluster: &popsignerv1.POPSignerCluster{
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						AutoUnseal: popsignerv1.AutoUnsealSpec{
							Enabled:  true,
							Provider: "azurekv",
							AzureKV: &popsignerv1.AzureKVSpec{
								TenantID:  "my-tenant",
								VaultName: "my-vault",
								KeyName:   "my-key",
							},
						},
					},
				},
			},
			wantEmpty:    false,
			wantSealType: "azurekeyvault",
			wantContains: []string{"tenant_id", "my-tenant", "vault_name", "my-vault"},
		},
		{
			name: "transit",
			cluster: &popsignerv1.POPSignerCluster{
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						AutoUnseal: popsignerv1.AutoUnsealSpec{
							Enabled:  true,
							Provider: "transit",
							Transit: &popsignerv1.TransitSpec{
								Address:   "https://vault.example.com:8200",
								MountPath: "transit",
								KeyName:   "autounseal",
							},
						},
					},
				},
			},
			wantEmpty:    false,
			wantSealType: "transit",
			wantContains: []string{"address", "vault.example.com", "key_name", "autounseal"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := buildSealConfig(tt.cluster)

			if tt.wantEmpty {
				if config != "" {
					t.Errorf("Expected empty config, got: %s", config)
				}
				return
			}

			if config == "" {
				t.Fatal("Expected non-empty config")
			}

			if !strings.Contains(config, "seal \""+tt.wantSealType+"\"") {
				t.Errorf("Config should contain seal type %s, got: %s", tt.wantSealType, config)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(config, want) {
					t.Errorf("Config should contain %q, got: %s", want, config)
				}
			}
		})
	}
}
