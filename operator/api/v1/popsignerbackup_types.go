package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// POPSignerBackupSpec defines the desired state
type POPSignerBackupSpec struct {
	// +kubebuilder:validation:Required
	ClusterRef ClusterReference `json:"clusterRef"`

	// full or incremental
	// +kubebuilder:validation:Enum=full;incremental
	// +kubebuilder:default="full"
	Type string `json:"type,omitempty"`

	// Components to backup
	// +kubebuilder:default={"openbao","database","secrets"}
	Components []string `json:"components,omitempty"`

	// Override cluster backup destination
	Destination *BackupDestination `json:"destination,omitempty"`
}

// POPSignerBackupStatus defines the observed state
type POPSignerBackupStatus struct {
	// Pending, Running, Completed, Failed
	Phase string `json:"phase,omitempty"`

	StartedAt   *metav1.Time `json:"startedAt,omitempty"`
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	Components []BackupComponentStatus `json:"components,omitempty"`
	TotalSize  string                  `json:"totalSize,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// BackupComponentStatus represents the backup status of a component
type BackupComponentStatus struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Size     string `json:"size,omitempty"`
	Location string `json:"location,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.status.totalSize`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// POPSignerBackup is the Schema for the popsignerbackups API
type POPSignerBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   POPSignerBackupSpec   `json:"spec,omitempty"`
	Status POPSignerBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// POPSignerBackupList contains a list of POPSignerBackup
type POPSignerBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []POPSignerBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&POPSignerBackup{}, &POPSignerBackupList{})
}

