package openbao

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

func TestStatefulSet(t *testing.T) {
	tests := []struct {
		name            string
		cluster         *banhbaoringv1.BanhBaoRingCluster
		wantReplicas    int32
		wantVersion     string
		wantStorageSize string
	}{
		{
			name: "default values",
			cluster: &banhbaoringv1.BanhBaoRingCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: banhbaoringv1.BanhBaoRingClusterSpec{
					OpenBao: banhbaoringv1.OpenBaoSpec{},
				},
			},
			wantReplicas:    3,
			wantVersion:     constants.DefaultOpenBaoVersion,
			wantStorageSize: constants.DefaultOpenBaoStorageSize,
		},
		{
			name: "custom replicas and version",
			cluster: &banhbaoringv1.BanhBaoRingCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "production",
				},
				Spec: banhbaoringv1.BanhBaoRingClusterSpec{
					OpenBao: banhbaoringv1.OpenBaoSpec{
						Replicas: 5,
						Version:  "2.1.0",
						Storage: banhbaoringv1.StorageSpec{
							Size:         resource.MustParse("20Gi"),
							StorageClass: "fast-ssd",
						},
					},
				},
			},
			wantReplicas:    5,
			wantVersion:     "2.1.0",
			wantStorageSize: "20Gi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sts := StatefulSet(tt.cluster)

			// Check name
			expectedName := tt.cluster.Name + "-openbao"
			if sts.Name != expectedName {
				t.Errorf("StatefulSet name = %v, want %v", sts.Name, expectedName)
			}

			// Check namespace
			if sts.Namespace != tt.cluster.Namespace {
				t.Errorf("StatefulSet namespace = %v, want %v", sts.Namespace, tt.cluster.Namespace)
			}

			// Check replicas
			if *sts.Spec.Replicas != tt.wantReplicas {
				t.Errorf("StatefulSet replicas = %v, want %v", *sts.Spec.Replicas, tt.wantReplicas)
			}

			// Check service name
			if sts.Spec.ServiceName != expectedName {
				t.Errorf("StatefulSet serviceName = %v, want %v", sts.Spec.ServiceName, expectedName)
			}

			// Check container image
			if len(sts.Spec.Template.Spec.Containers) != 1 {
				t.Fatalf("expected 1 container, got %d", len(sts.Spec.Template.Spec.Containers))
			}
			container := sts.Spec.Template.Spec.Containers[0]
			expectedImage := OpenBaoImage + ":" + tt.wantVersion
			if container.Image != expectedImage {
				t.Errorf("Container image = %v, want %v", container.Image, expectedImage)
			}

			// Check container name
			if container.Name != "openbao" {
				t.Errorf("Container name = %v, want openbao", container.Name)
			}

			// Check ports
			if len(container.Ports) != 2 {
				t.Errorf("expected 2 ports, got %d", len(container.Ports))
			}

			// Check volume claim templates
			if len(sts.Spec.VolumeClaimTemplates) != 1 {
				t.Fatalf("expected 1 VolumeClaimTemplate, got %d", len(sts.Spec.VolumeClaimTemplates))
			}
			pvc := sts.Spec.VolumeClaimTemplates[0]
			if pvc.Name != "data" {
				t.Errorf("VolumeClaimTemplate name = %v, want data", pvc.Name)
			}

			// Check labels
			if sts.Labels[constants.LabelComponent] != constants.ComponentOpenBao {
				t.Errorf("Label component = %v, want %v", sts.Labels[constants.LabelComponent], constants.ComponentOpenBao)
			}
		})
	}
}

func TestBuildEnv(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			OpenBao: banhbaoringv1.OpenBaoSpec{},
		},
	}

	env := buildEnv(cluster)

	// Check required env vars
	requiredEnvs := map[string]bool{
		"VAULT_ADDR":         false,
		"VAULT_CLUSTER_ADDR": false,
		"VAULT_API_ADDR":     false,
		"VAULT_SKIP_VERIFY":  false,
		"HOSTNAME":           false,
	}

	for _, e := range env {
		if _, ok := requiredEnvs[e.Name]; ok {
			requiredEnvs[e.Name] = true
		}
	}

	for name, found := range requiredEnvs {
		if !found {
			t.Errorf("Required env var %s not found", name)
		}
	}
}

func TestAutoUnsealEnv(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *banhbaoringv1.BanhBaoRingCluster
		wantEnvVars []string
	}{
		{
			name: "aws kms with credentials",
			cluster: &banhbaoringv1.BanhBaoRingCluster{
				Spec: banhbaoringv1.BanhBaoRingClusterSpec{
					OpenBao: banhbaoringv1.OpenBaoSpec{
						AutoUnseal: banhbaoringv1.AutoUnsealSpec{
							Enabled:  true,
							Provider: "awskms",
							AWSKMS: &banhbaoringv1.AWSKMSSpec{
								KeyID:  "key-123",
								Region: "us-west-2",
								Credentials: &banhbaoringv1.SecretKeyRef{
									Name: "aws-creds",
									Key:  "credentials",
								},
							},
						},
					},
				},
			},
			wantEnvVars: []string{"AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
		},
		{
			name: "gcp kms",
			cluster: &banhbaoringv1.BanhBaoRingCluster{
				Spec: banhbaoringv1.BanhBaoRingClusterSpec{
					OpenBao: banhbaoringv1.OpenBaoSpec{
						AutoUnseal: banhbaoringv1.AutoUnsealSpec{
							Enabled:  true,
							Provider: "gcpkms",
							GCPKMS: &banhbaoringv1.GCPKMSSpec{
								Project:   "my-project",
								Location:  "us-central1",
								KeyRing:   "my-keyring",
								CryptoKey: "my-key",
								Credentials: &banhbaoringv1.SecretKeyRef{
									Name: "gcp-creds",
									Key:  "key.json",
								},
							},
						},
					},
				},
			},
			wantEnvVars: []string{"GOOGLE_PROJECT", "GOOGLE_APPLICATION_CREDENTIALS"},
		},
		{
			name: "azure key vault",
			cluster: &banhbaoringv1.BanhBaoRingCluster{
				Spec: banhbaoringv1.BanhBaoRingClusterSpec{
					OpenBao: banhbaoringv1.OpenBaoSpec{
						AutoUnseal: banhbaoringv1.AutoUnsealSpec{
							Enabled:  true,
							Provider: "azurekv",
							AzureKV: &banhbaoringv1.AzureKVSpec{
								TenantID:  "tenant-123",
								VaultName: "my-vault",
								KeyName:   "my-key",
								Credentials: &banhbaoringv1.SecretKeyRef{
									Name: "azure-creds",
									Key:  "credentials",
								},
							},
						},
					},
				},
			},
			wantEnvVars: []string{"AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := autoUnsealEnv(tt.cluster)

			envNames := make(map[string]bool)
			for _, e := range env {
				envNames[e.Name] = true
			}

			for _, want := range tt.wantEnvVars {
				if !envNames[want] {
					t.Errorf("Expected env var %s not found", want)
				}
			}
		})
	}
}

func TestBuildVolumes(t *testing.T) {
	volumes := buildVolumes("test-cluster-openbao")

	expectedVolumes := map[string]bool{
		"config":  false,
		"plugins": false,
		"tls":     false,
	}

	for _, v := range volumes {
		if _, ok := expectedVolumes[v.Name]; ok {
			expectedVolumes[v.Name] = true
		}
	}

	for name, found := range expectedVolumes {
		if !found {
			t.Errorf("Expected volume %s not found", name)
		}
	}
}

func TestSecurityContext(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			OpenBao: banhbaoringv1.OpenBaoSpec{},
		},
	}

	sts := StatefulSet(cluster)

	// Check pod security context
	if sts.Spec.Template.Spec.SecurityContext == nil {
		t.Fatal("Pod security context is nil")
	}
	if sts.Spec.Template.Spec.SecurityContext.FSGroup == nil || *sts.Spec.Template.Spec.SecurityContext.FSGroup != 1000 {
		t.Error("FSGroup should be 1000")
	}

	// Check container security context
	container := sts.Spec.Template.Spec.Containers[0]
	if container.SecurityContext == nil {
		t.Fatal("Container security context is nil")
	}
	if container.SecurityContext.Capabilities == nil {
		t.Fatal("Container capabilities is nil")
	}

	hasIPCLock := false
	for _, cap := range container.SecurityContext.Capabilities.Add {
		if cap == corev1.Capability("IPC_LOCK") {
			hasIPCLock = true
			break
		}
	}
	if !hasIPCLock {
		t.Error("Container should have IPC_LOCK capability")
	}
}

func TestProbes(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			OpenBao: banhbaoringv1.OpenBaoSpec{},
		},
	}

	sts := StatefulSet(cluster)
	container := sts.Spec.Template.Spec.Containers[0]

	// Check readiness probe
	if container.ReadinessProbe == nil {
		t.Fatal("Readiness probe is nil")
	}
	if container.ReadinessProbe.HTTPGet == nil {
		t.Fatal("Readiness probe HTTPGet is nil")
	}
	if container.ReadinessProbe.HTTPGet.Path != "/v1/sys/health?standbyok=true" {
		t.Errorf("Readiness probe path = %v, want /v1/sys/health?standbyok=true", container.ReadinessProbe.HTTPGet.Path)
	}
	if container.ReadinessProbe.HTTPGet.Scheme != corev1.URISchemeHTTPS {
		t.Errorf("Readiness probe scheme = %v, want HTTPS", container.ReadinessProbe.HTTPGet.Scheme)
	}

	// Check liveness probe
	if container.LivenessProbe == nil {
		t.Fatal("Liveness probe is nil")
	}
	if container.LivenessProbe.HTTPGet == nil {
		t.Fatal("Liveness probe HTTPGet is nil")
	}
}
