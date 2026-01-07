package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for deployment data operations.
type Repository interface {
	// Deployment operations
	CreateDeployment(ctx context.Context, d *Deployment) error
	GetDeployment(ctx context.Context, id uuid.UUID) (*Deployment, error)
	GetDeploymentByChainID(ctx context.Context, chainID int64) (*Deployment, error)
	// GetDeploymentByChainIDAndOrg retrieves a deployment by chain ID scoped to an organization.
	GetDeploymentByChainIDAndOrg(ctx context.Context, chainID int64, orgID uuid.UUID) (*Deployment, error)
	UpdateDeploymentStatus(ctx context.Context, id uuid.UUID, status Status, stage *string) error
	UpdateDeploymentConfig(ctx context.Context, id uuid.UUID, config json.RawMessage) error
	SetDeploymentError(ctx context.Context, id uuid.UUID, errMsg string) error
	ClearDeploymentError(ctx context.Context, id uuid.UUID) error
	ListDeploymentsByStatus(ctx context.Context, status Status) ([]*Deployment, error)
	ListAllDeployments(ctx context.Context) ([]*Deployment, error)
	// ListDeploymentsByOrg lists all deployments for a specific organization.
	ListDeploymentsByOrg(ctx context.Context, orgID uuid.UUID) ([]*Deployment, error)
	// ListDeploymentsByOrgAndStatus lists deployments filtered by org and status.
	ListDeploymentsByOrgAndStatus(ctx context.Context, orgID uuid.UUID, status Status) ([]*Deployment, error)

	// MarkStaleDeploymentsFailed marks deployments that have been "running" for longer
	// than the timeout as "failed". This handles cases where the deployment pod crashed
	// without updating the status. Returns the number of deployments marked as failed.
	// HIGH-028: Scoped to a specific organization to prevent cross-org data access.
	MarkStaleDeploymentsFailed(ctx context.Context, orgID uuid.UUID, timeout time.Duration) (int, error)

	// Transaction operations
	RecordTransaction(ctx context.Context, tx *Transaction) error
	GetTransactionsByDeployment(ctx context.Context, deploymentID uuid.UUID) ([]Transaction, error)
	GetTransactionByHash(ctx context.Context, hash string) (*Transaction, error)

	// Artifact operations
	SaveArtifact(ctx context.Context, a *Artifact) error
	GetArtifact(ctx context.Context, deploymentID uuid.UUID, artifactType string) (*Artifact, error)
	GetAllArtifacts(ctx context.Context, deploymentID uuid.UUID) ([]Artifact, error)
}

