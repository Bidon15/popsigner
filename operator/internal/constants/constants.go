// Package constants provides shared constants for the POPSigner operator.
package constants

const (
	// Labels
	LabelApp       = "app.kubernetes.io/name"
	LabelInstance  = "app.kubernetes.io/instance"
	LabelVersion   = "app.kubernetes.io/version"
	LabelComponent = "app.kubernetes.io/component"
	LabelManagedBy = "app.kubernetes.io/managed-by"

	// Component names
	ComponentOpenBao   = "openbao"
	ComponentAPI       = "api"
	ComponentDashboard = "dashboard"
	ComponentPostgres  = "postgres"
	ComponentRedis     = "redis"

	// Phases
	PhasePending      = "Pending"
	PhaseInitializing = "Initializing"
	PhaseRunning      = "Running"
	PhaseDegraded     = "Degraded"
	PhaseFailed       = "Failed"

	// Tenant phases
	TenantPhaseActive    = "Active"
	TenantPhaseSuspended = "Suspended"
	TenantPhaseDeleted   = "Deleted"

	// Backup phases
	BackupPhaseRunning   = "Running"
	BackupPhaseCompleted = "Completed"
	BackupPhaseFailed    = "Failed"

	// Restore phases
	RestorePhaseStopping  = "Stopping"
	RestorePhaseRestoring = "Restoring"
	RestorePhaseStarting  = "Starting"
	RestorePhaseCompleted = "Completed"
	RestorePhaseFailed    = "Failed"

	// Finalizer
	Finalizer = "popsigner.com/finalizer"

	// Manager name
	ManagedBy = "popsigner-operator"

	// Default versions
	DefaultOpenBaoVersion   = "2.0.0"
	DefaultPluginVersion    = "1.0.0"
	DefaultPostgresVersion  = "16"
	DefaultRedisVersion     = "7"
	DefaultAPIVersion       = "1.0.0"
	DefaultDashboardVersion = "1.0.0"

	// Default resources
	DefaultOpenBaoReplicas   = 3
	DefaultAPIReplicas       = 2
	DefaultDashboardReplicas = 2
	DefaultDatabaseReplicas  = 1
	DefaultRedisReplicas     = 1

	// Default storage sizes
	DefaultOpenBaoStorageSize    = "10Gi"
	DefaultDatabaseStorageSize   = "10Gi"
	DefaultRedisStorageSize      = "5Gi"
	DefaultPrometheusStorageSize = "50Gi"

	// Port numbers
	PortOpenBao        = 8200
	PortOpenBaoCluster = 8201
	PortAPI            = 8080
	PortDashboard      = 3000
	PortPostgres       = 5432
	PortRedis          = 6379
	PortPrometheus     = 9090
	PortGrafana        = 3000
)

// Labels returns standard labels for a component
func Labels(clusterName, component, version string) map[string]string {
	return map[string]string{
		LabelApp:       "popsigner",
		LabelInstance:  clusterName,
		LabelComponent: component,
		LabelVersion:   version,
		LabelManagedBy: ManagedBy,
	}
}

// SelectorLabels returns labels for pod selection
func SelectorLabels(clusterName, component string) map[string]string {
	return map[string]string{
		LabelApp:       "popsigner",
		LabelInstance:  clusterName,
		LabelComponent: component,
	}
}
