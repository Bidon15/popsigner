package openbao

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
	"github.com/Bidon15/banhbaoring/operator/internal/resources"
)

const (
	// PluginName is the name of the secp256k1 plugin.
	PluginName = "secp256k1"
	// PluginBinaryName is the binary name for the plugin.
	PluginBinaryName = "banhbaoring-secp256k1"
	// PluginDownloadBaseURL is the base URL for downloading plugins.
	PluginDownloadBaseURL = "https://github.com/Bidon15/banhbaoring/releases/download"
)

// PluginInfo contains information needed to register the plugin.
type PluginInfo struct {
	Name    string
	Command string
	SHA256  string
	Version string
}

// PluginDownloadURL returns the URL for downloading the plugin binary.
func PluginDownloadURL(version, os, arch string) string {
	return fmt.Sprintf(
		"%s/v%s/%s_%s_%s",
		PluginDownloadBaseURL, version, PluginBinaryName, os, arch,
	)
}

// GetPluginInfo returns plugin registration info.
func GetPluginInfo(cluster *banhbaoringv1.BanhBaoRingCluster) PluginInfo {
	version := cluster.Spec.OpenBao.Plugin.Version
	if version == "" {
		version = constants.DefaultPluginVersion
	}

	return PluginInfo{
		Name:    PluginName,
		Command: PluginBinaryName,
		Version: version,
	}
}

// InitContainer returns an init container that downloads the plugin.
func InitContainer(cluster *banhbaoringv1.BanhBaoRingCluster) corev1.Container {
	version := cluster.Spec.OpenBao.Plugin.Version
	if version == "" {
		version = constants.DefaultPluginVersion
	}

	downloadScript := fmt.Sprintf(`#!/bin/sh
set -e
PLUGIN_URL="%s"
PLUGIN_PATH="%s/%s"

echo "Downloading plugin from $PLUGIN_URL"

# Download plugin
wget -q -O "$PLUGIN_PATH" "$PLUGIN_URL" || curl -sSL -o "$PLUGIN_PATH" "$PLUGIN_URL"

chmod +x "$PLUGIN_PATH"

# Calculate SHA256
sha256sum "$PLUGIN_PATH" | cut -d' ' -f1 > %s/sha256.txt

echo "Plugin downloaded successfully"
echo "SHA256: $(cat %s/sha256.txt)"
`, PluginDownloadURL(version, "linux", "amd64"), PluginDir, PluginBinaryName, PluginDir, PluginDir)

	return corev1.Container{
		Name:    "download-plugin",
		Image:   "alpine:3.19",
		Command: []string{"/bin/sh", "-c", downloadScript},
		VolumeMounts: []corev1.VolumeMount{
			resources.VolumeMount("plugins", PluginDir, false),
		},
	}
}

// RegisterPluginScript returns a script to register the plugin with OpenBao.
func RegisterPluginScript() string {
	return fmt.Sprintf(`#!/bin/sh
set -e

export VAULT_ADDR="https://127.0.0.1:8200"
export VAULT_SKIP_VERIFY=true

# Wait for Vault to be ready and unsealed
echo "Waiting for OpenBao to be ready..."
until bao status 2>/dev/null | grep -q "Sealed.*false"; do
    echo "OpenBao is not ready yet..."
    sleep 5
done

echo "OpenBao is ready"

# Get plugin SHA256
PLUGIN_SHA=$(cat %s/sha256.txt)
echo "Plugin SHA256: $PLUGIN_SHA"

# Login with root token
bao login "$VAULT_ROOT_TOKEN"

# Register the plugin
echo "Registering plugin..."
bao plugin register -sha256="$PLUGIN_SHA" secret %s

# Enable the secrets engine
echo "Enabling secrets engine..."
bao secrets enable -path=keys %s || echo "Secrets engine already enabled"

echo "Plugin registered and enabled successfully"
`, PluginDir, PluginName, PluginName)
}

// PluginRegistrationJob returns a Job spec for registering the plugin.
// This is called after the cluster is initialized and unsealed.
func PluginRegistrationJob(cluster *banhbaoringv1.BanhBaoRingCluster) corev1.Container {
	return corev1.Container{
		Name:    "register-plugin",
		Image:   fmt.Sprintf("%s:%s", OpenBaoImage, cluster.Spec.OpenBao.Version),
		Command: []string{"/bin/sh", "-c", RegisterPluginScript()},
		Env: []corev1.EnvVar{
			resources.EnvVar("VAULT_ADDR", "https://127.0.0.1:8200"),
			resources.EnvVar("VAULT_SKIP_VERIFY", "true"),
		},
		VolumeMounts: []corev1.VolumeMount{
			resources.VolumeMount("plugins", PluginDir, false),
		},
	}
}
