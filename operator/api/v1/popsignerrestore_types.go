package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// POPSignerRestoreSpec defines the desired state
type POPSignerRestoreSpec struct {
	// +kubebuilder:validation:Required
	ClusterRef ClusterReference `json:"clusterRef"`

	// Reference to backup resource
	BackupRef *BackupReference `json:"backupRef,omitempty"`

	// Or restore from specific location
	Source *BackupDestination `json:"source,omitempty"`

	// Components to restore (default: all from backup)
	Components []string `json:"components,omitempty"`

	Options RestoreOptions `json:"options,omitempty"`
}

// BackupReference references a POPSignerBackup
type BackupReference struct {
	Name string `json:"name"`
}

// RestoreOptions configures restore behavior
type RestoreOptions struct {
	// +kubebuilder:default=true
	StopApplications bool `json:"stopApplications,omitempty"`

	// +kubebuilder:default=true
	VerifyBackup bool `json:"verifyBackup,omitempty"`
}

// POPSignerRestoreStatus defines the observed state
type POPSignerRestoreStatus struct {
	// Pending, Stopping, Restoring, Starting, Completed, Failed
	Phase string `json:"phase,omitempty"`

	StartedAt   *metav1.Time `json:"startedAt,omitempty"`
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	Steps []RestoreStep `json:"steps,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// RestoreStep represents a step in the restore process
type RestoreStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Backup",type=string,JSONPath=`.spec.backupRef.name`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// POPSignerRestore is the Schema for the popsignerrestores API
type POPSignerRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   POPSignerRestoreSpec   `json:"spec,omitempty"`
	Status POPSignerRestoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// POPSignerRestoreList contains a list of POPSignerRestore
type POPSignerRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []POPSignerRestore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&POPSignerRestore{}, &POPSignerRestoreList{})
}

