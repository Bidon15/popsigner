package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// POPSignerTenantSpec defines the desired state
type POPSignerTenantSpec struct {
	// Reference to the parent cluster
	// +kubebuilder:validation:Required
	ClusterRef ClusterReference `json:"clusterRef"`

	// Display name for the tenant
	DisplayName string `json:"displayName,omitempty"`

	// Plan: free, starter, pro, enterprise
	// +kubebuilder:validation:Enum=free;starter;pro;enterprise
	// +kubebuilder:default="free"
	Plan string `json:"plan,omitempty"`

	// Resource quotas
	Quotas TenantQuotas `json:"quotas,omitempty"`

	// Initial admin user
	Admin TenantAdmin `json:"admin,omitempty"`

	// Custom settings
	Settings TenantSettings `json:"settings,omitempty"`
}

// ClusterReference references a POPSignerCluster
type ClusterReference struct {
	Name string `json:"name"`
}

// TenantQuotas defines resource limits for a tenant
type TenantQuotas struct {
	// +kubebuilder:default=5
	Keys int32 `json:"keys,omitempty"`

	// +kubebuilder:default=10000
	SignaturesPerMonth int64 `json:"signaturesPerMonth,omitempty"`

	// +kubebuilder:default=1
	Namespaces int32 `json:"namespaces,omitempty"`

	// +kubebuilder:default=1
	TeamMembers int32 `json:"teamMembers,omitempty"`

	// +kubebuilder:default=2
	APIKeys int32 `json:"apiKeys,omitempty"`
}

// TenantAdmin defines the initial admin user
type TenantAdmin struct {
	Email    string        `json:"email"`
	Password *SecretKeyRef `json:"password,omitempty"`
}

// TenantSettings contains tenant-specific settings
type TenantSettings struct {
	// +kubebuilder:default=30
	AuditRetentionDays int32 `json:"auditRetentionDays,omitempty"`

	AllowExportableKeys bool     `json:"allowExportableKeys,omitempty"`
	AllowedIPRanges     []string `json:"allowedIPRanges,omitempty"`

	Webhooks []WebhookConfig `json:"webhooks,omitempty"`
}

// WebhookConfig defines a webhook endpoint
type WebhookConfig struct {
	URL    string       `json:"url"`
	Events []string     `json:"events,omitempty"`
	Secret SecretKeyRef `json:"secret"`
}

// POPSignerTenantStatus defines the observed state
type POPSignerTenantStatus struct {
	// Phase: Pending, Active, Suspended, Deleted
	// +kubebuilder:default="Pending"
	Phase string `json:"phase,omitempty"`

	// OpenBao namespace for this tenant
	OpenBaoNamespace string `json:"openbaoNamespace,omitempty"`

	// Current usage
	Usage TenantUsage `json:"usage,omitempty"`

	CreatedAt    *metav1.Time `json:"createdAt,omitempty"`
	LastActiveAt *metav1.Time `json:"lastActiveAt,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// TenantUsage represents current resource usage
type TenantUsage struct {
	Keys                int32 `json:"keys,omitempty"`
	SignaturesThisMonth int64 `json:"signaturesThisMonth,omitempty"`
	APIKeys             int32 `json:"apiKeys,omitempty"`
	TeamMembers         int32 `json:"teamMembers,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Plan",type=string,JSONPath=`.spec.plan`
// +kubebuilder:printcolumn:name="Keys",type=integer,JSONPath=`.status.usage.keys`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// POPSignerTenant is the Schema for the popsignertenants API
type POPSignerTenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   POPSignerTenantSpec   `json:"spec,omitempty"`
	Status POPSignerTenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// POPSignerTenantList contains a list of POPSignerTenant
type POPSignerTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []POPSignerTenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&POPSignerTenant{}, &POPSignerTenantList{})
}

