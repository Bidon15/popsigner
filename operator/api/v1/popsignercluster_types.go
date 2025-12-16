package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// POPSignerClusterSpec defines the desired state
type POPSignerClusterSpec struct {
	// Domain for the cluster endpoints (e.g., keys.mycompany.com)
	// +kubebuilder:validation:Required
	Domain string `json:"domain"`

	// OpenBao configuration
	// +kubebuilder:default={}
	OpenBao OpenBaoSpec `json:"openbao,omitempty"`

	// Control Plane API configuration
	// +kubebuilder:default={}
	API APISpec `json:"api,omitempty"`

	// Web Dashboard configuration
	// +kubebuilder:default={}
	Dashboard DashboardSpec `json:"dashboard,omitempty"`

	// Database configuration
	// +kubebuilder:default={}
	Database DatabaseSpec `json:"database,omitempty"`

	// Redis configuration
	// +kubebuilder:default={}
	Redis RedisSpec `json:"redis,omitempty"`

	// Monitoring configuration
	// +kubebuilder:default={}
	Monitoring MonitoringSpec `json:"monitoring,omitempty"`

	// Backup configuration
	// +kubebuilder:default={}
	Backup BackupSpec `json:"backup,omitempty"`

	// Billing configuration
	// +kubebuilder:default={}
	Billing BillingSpec `json:"billing,omitempty"`

	// RPCGateway configures the JSON-RPC gateway for OP Stack integration.
	// +optional
	RPCGateway RPCGatewaySpec `json:"rpcGateway,omitempty"`
}

// RPCGatewaySpec defines the configuration for the JSON-RPC gateway.
type RPCGatewaySpec struct {
	// Enabled controls whether the RPC gateway is deployed.
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Replicas is the number of gateway replicas.
	// +optional
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas,omitempty"`

	// Image is the container image for the gateway.
	// +optional
	Image string `json:"image,omitempty"`

	// Version is the image tag/version.
	// +optional
	Version string `json:"version,omitempty"`

	// Resources defines the CPU/memory resources for the gateway.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// RateLimit configures per-address rate limiting.
	// +optional
	RateLimit RateLimitConfig `json:"rateLimit,omitempty"`

	// TLS configures TLS/mTLS settings.
	// +optional
	TLS RPCGatewayTLSConfig `json:"tls,omitempty"`

	// MTLS configures mutual TLS for Arbitrum Nitro integration.
	// +optional
	MTLS MTLSConfig `json:"mtls,omitempty"`
}

// RateLimitConfig defines rate limiting configuration.
type RateLimitConfig struct {
	// RequestsPerSecond is the maximum requests per second per address.
	// +optional
	// +kubebuilder:default=100
	RequestsPerSecond int `json:"requestsPerSecond,omitempty"`

	// BurstSize is the maximum burst size.
	// +optional
	// +kubebuilder:default=200
	BurstSize int `json:"burstSize,omitempty"`
}

// RPCGatewayTLSConfig defines TLS settings for the gateway.
type RPCGatewayTLSConfig struct {
	// Enabled controls whether TLS is enabled.
	// +optional
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// SecretName is the name of the TLS secret.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// MutualTLS enables mutual TLS (client certificate verification).
	// +optional
	// +kubebuilder:default=false
	MutualTLS bool `json:"mutualTLS,omitempty"`
}

// MTLSConfig configures mutual TLS for Arbitrum Nitro integration.
type MTLSConfig struct {
	// Enabled enables mTLS client certificate authentication.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// CASecretName is the name of the Secret containing the CA certificate.
	// If not specified, uses "{cluster-name}-rpc-gateway-ca".
	// +optional
	CASecretName string `json:"caSecretName,omitempty"`

	// CASecretKey is the key in the Secret containing the CA certificate PEM.
	// Defaults to "ca.crt".
	// +optional
	CASecretKey string `json:"caSecretKey,omitempty"`

	// ClientAuthType specifies the client authentication policy.
	// Options: "NoClientCert", "RequestClientCert", "RequireAnyClientCert", "VerifyClientCertIfGiven", "RequireAndVerifyClientCert"
	// Defaults to "VerifyClientCertIfGiven" (allows both API key and mTLS).
	// +optional
	// +kubebuilder:validation:Enum=NoClientCert;RequestClientCert;RequireAnyClientCert;VerifyClientCertIfGiven;RequireAndVerifyClientCert
	ClientAuthType string `json:"clientAuthType,omitempty"`
}

// OpenBaoSpec configures the OpenBao cluster
type OpenBaoSpec struct {
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:default="2.0.0"
	Version string `json:"version,omitempty"`

	// Storage configuration
	Storage StorageSpec `json:"storage,omitempty"`

	// Resource requirements
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Auto-unseal configuration
	AutoUnseal AutoUnsealSpec `json:"autoUnseal,omitempty"`

	// TLS configuration
	TLS TLSSpec `json:"tls,omitempty"`

	// Plugin configuration
	Plugin PluginSpec `json:"plugin,omitempty"`
}

// AutoUnsealSpec configures auto-unseal
type AutoUnsealSpec struct {
	// Enable auto-unseal
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// Provider type: awskms, gcpkms, azurekv, transit
	// +kubebuilder:validation:Enum=awskms;gcpkms;azurekv;transit
	Provider string `json:"provider,omitempty"`

	// AWS KMS configuration
	AWSKMS *AWSKMSSpec `json:"awskms,omitempty"`

	// GCP Cloud KMS configuration
	GCPKMS *GCPKMSSpec `json:"gcpkms,omitempty"`

	// Azure Key Vault configuration
	AzureKV *AzureKVSpec `json:"azurekv,omitempty"`

	// Transit (another Vault) configuration
	Transit *TransitSpec `json:"transit,omitempty"`
}

// AWSKMSSpec configures AWS KMS auto-unseal
type AWSKMSSpec struct {
	KeyID  string `json:"keyId"`
	Region string `json:"region,omitempty"`
	// Reference to secret containing access credentials
	Credentials *SecretKeyRef `json:"credentials,omitempty"`
}

// GCPKMSSpec configures GCP Cloud KMS auto-unseal
type GCPKMSSpec struct {
	Project   string `json:"project"`
	Location  string `json:"location"`
	KeyRing   string `json:"keyRing"`
	CryptoKey string `json:"cryptoKey"`
	// Reference to secret containing service account key
	Credentials *SecretKeyRef `json:"credentials,omitempty"`
}

// AzureKVSpec configures Azure Key Vault auto-unseal
type AzureKVSpec struct {
	TenantID  string `json:"tenantId"`
	VaultName string `json:"vaultName"`
	KeyName   string `json:"keyName"`
	// Reference to secret containing Azure credentials
	Credentials *SecretKeyRef `json:"credentials,omitempty"`
}

// TransitSpec configures Transit auto-unseal
type TransitSpec struct {
	Address   string       `json:"address"`
	MountPath string       `json:"mountPath,omitempty"`
	KeyName   string       `json:"keyName"`
	Token     SecretKeyRef `json:"token"`
}

// TLSSpec configures TLS settings
type TLSSpec struct {
	// cert-manager issuer name
	Issuer string `json:"issuer,omitempty"`
	// Or use existing secret
	SecretName string `json:"secretName,omitempty"`
}

// PluginSpec configures the secp256k1 plugin
type PluginSpec struct {
	// +kubebuilder:default="1.0.0"
	Version string `json:"version,omitempty"`
}

// StorageSpec configures persistent storage
type StorageSpec struct {
	// +kubebuilder:default="10Gi"
	Size resource.Quantity `json:"size,omitempty"`
	// StorageClass name
	StorageClass string `json:"storageClass,omitempty"`
}

// APISpec configures the Control Plane API
type APISpec struct {
	// +kubebuilder:default=2
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:default="1.0.0"
	Version string `json:"version,omitempty"`

	// Image overrides the default image (e.g., "ghcr.io/bidon15/popsigner-control-plane")
	Image string `json:"image,omitempty"`

	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	Autoscaling AutoscalingSpec `json:"autoscaling,omitempty"`
}

// AutoscalingSpec configures horizontal pod autoscaling
type AutoscalingSpec struct {
	Enabled     bool  `json:"enabled,omitempty"`
	MinReplicas int32 `json:"minReplicas,omitempty"`
	MaxReplicas int32 `json:"maxReplicas,omitempty"`
	TargetCPU   int32 `json:"targetCPU,omitempty"`
}

// DashboardSpec configures the Web Dashboard
type DashboardSpec struct {
	// +kubebuilder:default=2
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:default="1.0.0"
	Version string `json:"version,omitempty"`

	// Image overrides the default image (e.g., "ghcr.io/bidon15/popsigner-dashboard")
	Image string `json:"image,omitempty"`

	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// DatabaseSpec configures PostgreSQL
type DatabaseSpec struct {
	// Deploy managed PostgreSQL (true) or use external (false)
	// +kubebuilder:default=true
	Managed bool `json:"managed,omitempty"`

	// +kubebuilder:default="16"
	Version string `json:"version,omitempty"`

	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	Storage StorageSpec `json:"storage,omitempty"`

	// External database connection string
	ConnectionString *SecretKeyRef `json:"connectionString,omitempty"`
}

// RedisSpec configures Redis
type RedisSpec struct {
	// +kubebuilder:default=true
	Managed bool `json:"managed,omitempty"`

	// +kubebuilder:default="7"
	Version string `json:"version,omitempty"`

	// standalone or cluster
	// +kubebuilder:default="standalone"
	// +kubebuilder:validation:Enum=standalone;cluster
	Mode string `json:"mode,omitempty"`

	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	Storage StorageSpec `json:"storage,omitempty"`
}

// MonitoringSpec configures observability stack
type MonitoringSpec struct {
	Enabled bool `json:"enabled,omitempty"`

	Prometheus PrometheusSpec `json:"prometheus,omitempty"`
	Grafana    GrafanaSpec    `json:"grafana,omitempty"`
	Alerting   AlertingSpec   `json:"alerting,omitempty"`
}

// PrometheusSpec configures Prometheus
type PrometheusSpec struct {
	// +kubebuilder:default="15d"
	Retention string      `json:"retention,omitempty"`
	Storage   StorageSpec `json:"storage,omitempty"`
}

// GrafanaSpec configures Grafana
type GrafanaSpec struct {
	Enabled       bool          `json:"enabled,omitempty"`
	AdminPassword *SecretKeyRef `json:"adminPassword,omitempty"`
}

// AlertingSpec configures alerting
type AlertingSpec struct {
	Enabled   bool            `json:"enabled,omitempty"`
	Slack     *SlackAlertSpec `json:"slack,omitempty"`
	PagerDuty *PagerDutySpec  `json:"pagerduty,omitempty"`
}

// SlackAlertSpec configures Slack alerting
type SlackAlertSpec struct {
	WebhookURL SecretKeyRef `json:"webhookUrl"`
}

// PagerDutySpec configures PagerDuty alerting
type PagerDutySpec struct {
	RoutingKey SecretKeyRef `json:"routingKey"`
}

// BackupSpec configures automatic backups
type BackupSpec struct {
	Enabled bool `json:"enabled,omitempty"`

	// Cron schedule (default: daily at 2 AM UTC)
	// +kubebuilder:default="0 2 * * *"
	Schedule string `json:"schedule,omitempty"`

	// Retention in days
	// +kubebuilder:default=30
	Retention int32 `json:"retention,omitempty"`

	Destination BackupDestination `json:"destination,omitempty"`
}

// BackupDestination configures backup storage location
type BackupDestination struct {
	S3  *S3Destination  `json:"s3,omitempty"`
	GCS *GCSDestination `json:"gcs,omitempty"`
}

// S3Destination configures S3-compatible storage
type S3Destination struct {
	Bucket      string       `json:"bucket"`
	Region      string       `json:"region,omitempty"`
	Prefix      string       `json:"prefix,omitempty"`
	Credentials SecretKeyRef `json:"credentials"`
}

// GCSDestination configures Google Cloud Storage
type GCSDestination struct {
	Bucket      string       `json:"bucket"`
	Prefix      string       `json:"prefix,omitempty"`
	Credentials SecretKeyRef `json:"credentials"`
}

// BillingSpec configures payment integrations
type BillingSpec struct {
	Stripe StripeSpec `json:"stripe,omitempty"`
}

// StripeSpec configures Stripe integration
type StripeSpec struct {
	Enabled          bool         `json:"enabled,omitempty"`
	SecretKeyRef     SecretKeyRef `json:"secretKeyRef,omitempty"`
	WebhookSecretRef SecretKeyRef `json:"webhookSecretRef,omitempty"`
}

// SecretKeyRef references a key in a Secret
type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// POPSignerClusterStatus defines the observed state
type POPSignerClusterStatus struct {
	// Current phase: Pending, Initializing, Running, Degraded, Failed
	// +kubebuilder:default="Pending"
	Phase string `json:"phase,omitempty"`

	// Component statuses
	OpenBao    ComponentStatus `json:"openbao,omitempty"`
	API        ComponentStatus `json:"api,omitempty"`
	Dashboard  ComponentStatus `json:"dashboard,omitempty"`
	Database   ComponentStatus `json:"database,omitempty"`
	Redis      ComponentStatus `json:"redis,omitempty"`
	RPCGateway ComponentStatus `json:"rpcGateway,omitempty"`

	// Access endpoints
	Endpoints EndpointsStatus `json:"endpoints,omitempty"`

	// Conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ComponentStatus represents the status of a cluster component
type ComponentStatus struct {
	Ready   bool   `json:"ready,omitempty"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
	// For StatefulSets: "2/3" format
	Replicas string `json:"replicas,omitempty"`
}

// EndpointsStatus contains the access endpoints for the cluster
type EndpointsStatus struct {
	API        string `json:"api,omitempty"`
	Dashboard  string `json:"dashboard,omitempty"`
	OpenBao    string `json:"openbao,omitempty"`
	Grafana    string `json:"grafana,omitempty"`
	RPCGateway string `json:"rpcGateway,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="OpenBao",type=string,JSONPath=`.status.openbao.replicas`
// +kubebuilder:printcolumn:name="API",type=string,JSONPath=`.status.api.replicas`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// POPSignerCluster is the Schema for the popsignerclusters API
type POPSignerCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   POPSignerClusterSpec   `json:"spec,omitempty"`
	Status POPSignerClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// POPSignerClusterList contains a list of POPSignerCluster
type POPSignerClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []POPSignerCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&POPSignerCluster{}, &POPSignerClusterList{})
}

