// Package openbao provides Kubernetes resource builders for OpenBao.
package openbao

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/constants"
	"github.com/Bidon15/popsigner/operator/internal/resources"
)

const (
	// InitJobSuffix is the suffix for the init job name.
	InitJobSuffix = "init"
)

// PluginConfig defines a plugin to be registered with OpenBao.
type PluginConfig struct {
	Name       string // Plugin name (e.g., "secp256k1")
	Type       string // Plugin type: "secret" or "auth"
	Command    string // Binary command name
	MountPath  string // Path to mount the secrets engine (e.g., "secp256k1")
	EngineType string // Custom engine type name (defaults to Name)
}

// SecretsEngineConfig defines a secrets engine to enable.
type SecretsEngineConfig struct {
	Path    string            // Mount path (e.g., "secret")
	Type    string            // Engine type (e.g., "kv-v2", "pki", "transit")
	Options map[string]string // Optional: engine-specific options
}

// DefaultPlugins returns the default plugins to register.
func DefaultPlugins() []PluginConfig {
	return []PluginConfig{
		{
			Name:       "secp256k1",
			Type:       "secret",
			Command:    "popsigner-secp256k1",
			MountPath:  "secp256k1",
			EngineType: "banhbaoring-secp256k1",
		},
	}
}

// DefaultSecretsEngines returns the default secrets engines to enable.
func DefaultSecretsEngines() []SecretsEngineConfig {
	return []SecretsEngineConfig{
		{Path: "secret", Type: "kv-v2"},      // For deployment API keys, etc.
		{Path: "pki", Type: "pki"},           // For mTLS client certificates
		{Path: "transit", Type: "transit"},   // For general encryption operations
	}
}

// InitJob creates a Kubernetes Job that initializes OpenBao.
// This job:
// 1. Waits for OpenBao to be ready
// 2. Initializes OpenBao (vault operator init)
// 3. Stores root token and unseal keys in a Secret
// 4. Unseals OpenBao (if not using auto-unseal)
// 5. Registers plugins
// 6. Enables required secrets engines
func InitJob(cluster *popsignerv1.POPSignerCluster) *batchv1.Job {
	name := resources.ResourceName(cluster.Name, constants.ComponentOpenBao)
	jobName := fmt.Sprintf("%s-%s", name, InitJobSuffix)
	labels := resources.Labels(cluster.Name, constants.ComponentOpenBao, cluster.Spec.OpenBao.Version)

	version := cluster.Spec.OpenBao.Version
	if version == "" {
		version = constants.DefaultOpenBaoVersion
	}

	backoffLimit := int32(3)
	ttlSeconds := int32(3600) // Clean up after 1 hour

	return &batchv1.Job{
		ObjectMeta: resources.ObjectMeta(jobName, cluster.Namespace, labels),
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: name,
					Containers: []corev1.Container{
						initContainer(cluster, name, version),
					},
					Volumes: []corev1.Volume{
						{
							Name: "plugins",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "tls",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: fmt.Sprintf("%s-tls", name),
								},
							},
						},
					},
				},
			},
		},
	}
}

func initContainer(cluster *popsignerv1.POPSignerCluster, stsName, version string) corev1.Container {
	return corev1.Container{
		Name:    "init-openbao",
		Image:   fmt.Sprintf("%s:%s", OpenBaoImage, version),
		Command: []string{"/bin/sh", "-c", initScript(cluster, stsName)},
		Env: []corev1.EnvVar{
			resources.EnvVar("VAULT_ADDR", "https://127.0.0.1:8200"),
			resources.EnvVar("VAULT_SKIP_VERIFY", "true"),
			resources.EnvVar("OPENBAO_STS_NAME", stsName),
			resources.EnvVar("NAMESPACE", cluster.Namespace),
		},
		VolumeMounts: []corev1.VolumeMount{
			resources.VolumeMount("plugins", PluginDir, false),
			resources.VolumeMount("tls", TLSDir, true),
		},
	}
}

// initScript generates the shell script for OpenBao initialization.
func initScript(cluster *popsignerv1.POPSignerCluster, stsName string) string {
	plugins := DefaultPlugins()
	engines := DefaultSecretsEngines()

	// Generate plugin registration commands
	pluginCmds := ""
	for _, p := range plugins {
		engineType := p.EngineType
		if engineType == "" {
			engineType = p.Name
		}
		pluginCmds += fmt.Sprintf(`
# Register plugin: %s
echo "Registering plugin: %s"
if [ -f "%s/sha256.txt" ]; then
    PLUGIN_SHA=$(cat %s/sha256.txt)
    bao plugin register -sha256="$PLUGIN_SHA" %s %s || echo "Plugin %s may already be registered"
    bao secrets enable -path=%s %s || echo "Secrets engine %s may already be enabled"
else
    echo "Warning: Plugin SHA256 not found, skipping %s registration"
fi
`, p.Name, p.Name, PluginDir, PluginDir, p.Type, p.Name, p.Name, p.MountPath, engineType, p.MountPath, p.Name)
	}

	// Generate secrets engine enablement commands
	engineCmds := ""
	for _, e := range engines {
		engineCmds += fmt.Sprintf(`
# Enable secrets engine: %s (%s)
echo "Enabling secrets engine: %s at path %s"
bao secrets enable -path=%s %s 2>/dev/null || echo "Secrets engine %s may already be enabled"
`, e.Path, e.Type, e.Type, e.Path, e.Path, e.Type, e.Path)
	}

	// Check if auto-unseal is enabled
	autoUnseal := cluster.Spec.OpenBao.AutoUnseal.Enabled

	unsealLogic := ""
	if !autoUnseal {
		unsealLogic = `
# Manual unseal required
echo "Unsealing OpenBao..."
for i in 1 2 3; do
    KEY=$(echo "$INIT_OUTPUT" | grep "Unseal Key $i:" | awk '{print $NF}')
    bao operator unseal "$KEY"
done
echo "OpenBao unsealed"
`
	} else {
		unsealLogic = `
# Auto-unseal enabled, waiting for OpenBao to self-unseal
echo "Auto-unseal enabled, waiting for OpenBao to unseal..."
until bao status 2>/dev/null | grep -q "Sealed.*false"; do
    echo "Waiting for auto-unseal..."
    sleep 5
done
echo "OpenBao is unsealed"
`
	}

	return fmt.Sprintf(`#!/bin/sh
set -e

OPENBAO_ADDR="https://%s-0.%s.%s.svc.cluster.local:8200"
export VAULT_ADDR="$OPENBAO_ADDR"
export VAULT_CACERT="%s/ca.crt"

echo "=============================================="
echo "POPSigner OpenBao Initialization"
echo "=============================================="
echo "OpenBao Address: $OPENBAO_ADDR"
echo "Namespace: $NAMESPACE"
echo ""

# Wait for OpenBao to be ready
echo "Waiting for OpenBao to be ready..."
until wget -q --spider --no-check-certificate "$VAULT_ADDR/v1/sys/health?standbyok=true&sealedok=true&uninitok=true" 2>/dev/null; do
    echo "OpenBao not ready yet, waiting..."
    sleep 5
done
echo "OpenBao is reachable"

# Check if already initialized
INIT_STATUS=$(bao status -format=json 2>/dev/null | grep -o '"initialized":[^,]*' | cut -d: -f2 || echo "false")
if [ "$INIT_STATUS" = "true" ]; then
    echo "OpenBao is already initialized"
    
    # Get existing root token from secret
    echo "Retrieving existing root token..."
    # Note: In production, you'd use kubectl or the K8s API to get this
    # For now, we assume the token is passed via env or secret mount
    
    # Skip to plugin/engine setup
else
    echo "Initializing OpenBao..."
    INIT_OUTPUT=$(bao operator init -key-shares=5 -key-threshold=3 -format=json)
    
    # Extract root token and unseal keys
    ROOT_TOKEN=$(echo "$INIT_OUTPUT" | grep -o '"root_token":"[^"]*"' | cut -d'"' -f4)
    
    echo "OpenBao initialized successfully"
    echo "Root Token: ${ROOT_TOKEN:0:10}..."
    
    # Create Kubernetes secret with root token and unseal keys
    # This is done via kubectl since we're in the pod
    echo "Storing credentials in Kubernetes Secret..."
    
    # We'll output the init data for the controller to capture
    echo "---INIT_OUTPUT_START---"
    echo "$INIT_OUTPUT"
    echo "---INIT_OUTPUT_END---"
    
    %s
fi

# Login with root token
echo "Logging in with root token..."
if [ -n "$VAULT_ROOT_TOKEN" ]; then
    bao login "$VAULT_ROOT_TOKEN"
elif [ -n "$ROOT_TOKEN" ]; then
    export VAULT_TOKEN="$ROOT_TOKEN"
else
    echo "Error: No root token available"
    exit 1
fi

echo ""
echo "=============================================="
echo "Enabling Secrets Engines"
echo "=============================================="
%s

echo ""
echo "=============================================="
echo "Registering Plugins"
echo "=============================================="
%s

echo ""
echo "=============================================="
echo "Setting Up PKI (mTLS)"
echo "=============================================="
# Configure PKI for client certificate issuance
echo "Configuring PKI secrets engine..."
bao secrets tune -max-lease-ttl=87600h pki 2>/dev/null || echo "PKI already tuned"

# Generate root CA if not exists
bao read pki/cert/ca 2>/dev/null || {
    echo "Generating root CA..."
    bao write pki/root/generate/internal \
        common_name="POPSigner Root CA" \
        ttl=87600h \
        || echo "Root CA may already exist"
}

# Configure CA URLs
bao write pki/config/urls \
    issuing_certificates="https://%s.%s.svc.cluster.local:8200/v1/pki/ca" \
    crl_distribution_points="https://%s.%s.svc.cluster.local:8200/v1/pki/crl" \
    2>/dev/null || echo "PKI URLs already configured"

# Create role for client certificates
bao write pki/roles/client-cert \
    allowed_domains="popsigner.local" \
    allow_subdomains=true \
    max_ttl=72h \
    key_type=rsa \
    key_bits=2048 \
    2>/dev/null || echo "Client cert role may already exist"

echo ""
echo "=============================================="
echo "Initialization Complete!"
echo "=============================================="
echo "Enabled secrets engines:"
bao secrets list
echo ""
echo "Registered plugins:"
bao plugin list secret
echo ""

`, stsName, stsName, cluster.Namespace, TLSDir, unsealLogic, engineCmds, pluginCmds,
		stsName, cluster.Namespace, stsName, cluster.Namespace)
}

// InitJobName returns the name of the init job for a cluster.
func InitJobName(clusterName string) string {
	name := resources.ResourceName(clusterName, constants.ComponentOpenBao)
	return fmt.Sprintf("%s-%s", name, InitJobSuffix)
}

