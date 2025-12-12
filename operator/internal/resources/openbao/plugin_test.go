package openbao

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/constants"
)

func TestPluginDownloadURL(t *testing.T) {
	tests := []struct {
		version string
		os      string
		arch    string
		wantURL string
	}{
		{
			version: "1.0.0",
			os:      "linux",
			arch:    "amd64",
			wantURL: "https://github.com/Bidon15/popsigner/releases/download/v1.0.0/banhbaoring-secp256k1_linux_amd64",
		},
		{
			version: "2.0.0",
			os:      "linux",
			arch:    "arm64",
			wantURL: "https://github.com/Bidon15/popsigner/releases/download/v2.0.0/banhbaoring-secp256k1_linux_arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_"+tt.os+"_"+tt.arch, func(t *testing.T) {
			url := PluginDownloadURL(tt.version, tt.os, tt.arch)
			if url != tt.wantURL {
				t.Errorf("PluginDownloadURL() = %v, want %v", url, tt.wantURL)
			}
		})
	}
}

func TestGetPluginInfo(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *popsignerv1.POPSignerCluster
		wantVersion string
	}{
		{
			name: "default version",
			cluster: &popsignerv1.POPSignerCluster{
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						Plugin: popsignerv1.PluginSpec{},
					},
				},
			},
			wantVersion: constants.DefaultPluginVersion,
		},
		{
			name: "custom version",
			cluster: &popsignerv1.POPSignerCluster{
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						Plugin: popsignerv1.PluginSpec{
							Version: "2.0.0",
						},
					},
				},
			},
			wantVersion: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := GetPluginInfo(tt.cluster)

			if info.Name != PluginName {
				t.Errorf("PluginInfo.Name = %v, want %v", info.Name, PluginName)
			}
			if info.Command != PluginBinaryName {
				t.Errorf("PluginInfo.Command = %v, want %v", info.Command, PluginBinaryName)
			}
			if info.Version != tt.wantVersion {
				t.Errorf("PluginInfo.Version = %v, want %v", info.Version, tt.wantVersion)
			}
		})
	}
}

func TestInitContainer(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *popsignerv1.POPSignerCluster
		wantVersion string
	}{
		{
			name: "default version",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{},
				},
			},
			wantVersion: constants.DefaultPluginVersion,
		},
		{
			name: "custom version",
			cluster: &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerClusterSpec{
					OpenBao: popsignerv1.OpenBaoSpec{
						Plugin: popsignerv1.PluginSpec{
							Version: "2.0.0",
						},
					},
				},
			},
			wantVersion: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := InitContainer(tt.cluster)

			// Check container name
			if container.Name != "download-plugin" {
				t.Errorf("Container name = %v, want download-plugin", container.Name)
			}

			// Check image
			if !strings.HasPrefix(container.Image, "alpine:") {
				t.Errorf("Container image = %v, should start with alpine:", container.Image)
			}

			// Check command
			if len(container.Command) == 0 {
				t.Fatal("Container command is empty")
			}

			// Check volume mounts
			if len(container.VolumeMounts) != 1 {
				t.Fatalf("Expected 1 volume mount, got %d", len(container.VolumeMounts))
			}
			if container.VolumeMounts[0].Name != "plugins" {
				t.Errorf("VolumeMount name = %v, want plugins", container.VolumeMounts[0].Name)
			}
			if container.VolumeMounts[0].MountPath != PluginDir {
				t.Errorf("VolumeMount path = %v, want %v", container.VolumeMounts[0].MountPath, PluginDir)
			}

			// Check that the download script contains the expected version
			script := strings.Join(container.Command, " ")
			expectedURL := PluginDownloadURL(tt.wantVersion, "linux", "amd64")
			if !strings.Contains(script, expectedURL) {
				t.Errorf("Download script should contain URL %v", expectedURL)
			}
		})
	}
}

func TestRegisterPluginScript(t *testing.T) {
	script := RegisterPluginScript()

	// Check that script contains expected commands
	expectedContents := []string{
		"VAULT_ADDR",
		"bao status",
		"bao plugin register",
		"bao secrets enable",
		PluginName,
	}

	for _, expected := range expectedContents {
		if !strings.Contains(script, expected) {
			t.Errorf("Script should contain %q", expected)
		}
	}
}

func TestPluginRegistrationJob(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			OpenBao: popsignerv1.OpenBaoSpec{
				Version: "2.0.0",
			},
		},
	}

	container := PluginRegistrationJob(cluster)

	// Check container name
	if container.Name != "register-plugin" {
		t.Errorf("Container name = %v, want register-plugin", container.Name)
	}

	// Check image
	expectedImage := OpenBaoImage + ":2.0.0"
	if container.Image != expectedImage {
		t.Errorf("Container image = %v, want %v", container.Image, expectedImage)
	}

	// Check env vars
	envNames := make(map[string]bool)
	for _, env := range container.Env {
		envNames[env.Name] = true
	}
	if !envNames["VAULT_ADDR"] {
		t.Error("Container should have VAULT_ADDR env var")
	}
	if !envNames["VAULT_SKIP_VERIFY"] {
		t.Error("Container should have VAULT_SKIP_VERIFY env var")
	}

	// Check volume mounts
	hasPluginsMount := false
	for _, vm := range container.VolumeMounts {
		if vm.Name == "plugins" && vm.MountPath == PluginDir {
			hasPluginsMount = true
			break
		}
	}
	if !hasPluginsMount {
		t.Error("Container should have plugins volume mount")
	}
}
