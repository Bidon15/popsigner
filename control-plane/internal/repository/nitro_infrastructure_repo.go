// Package repository provides data access layer implementations.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NitroInfrastructure represents deployed Nitro infrastructure on a parent chain.
type NitroInfrastructure struct {
	ParentChainID                   int64
	RollupCreatorAddress            string
	BridgeCreatorAddress            *string
	Version                         string
	DeployedAt                      time.Time
	DeployedBy                      *uuid.UUID
	DeploymentTxHash                *string
	OSPEntryAddress                 *string
	ChallengeManagerTemplatesAddress *string
	CreatedAt                       time.Time
	UpdatedAt                       time.Time
}

// NitroInfrastructureRepository defines the interface for Nitro infrastructure data operations.
type NitroInfrastructureRepository interface {
	// Get retrieves infrastructure for a parent chain.
	Get(ctx context.Context, parentChainID int64) (*NitroInfrastructure, error)
	// Create inserts new infrastructure record.
	Create(ctx context.Context, infra *NitroInfrastructure) error
	// Update updates an existing infrastructure record.
	Update(ctx context.Context, infra *NitroInfrastructure) error
	// Upsert creates or updates infrastructure.
	Upsert(ctx context.Context, infra *NitroInfrastructure) error
	// List returns all infrastructure records.
	List(ctx context.Context) ([]*NitroInfrastructure, error)
	// Delete removes infrastructure for a parent chain.
	Delete(ctx context.Context, parentChainID int64) error
}

type nitroInfrastructureRepo struct {
	pool *pgxpool.Pool
}

// NewNitroInfrastructureRepository creates a new Nitro infrastructure repository.
func NewNitroInfrastructureRepository(pool *pgxpool.Pool) NitroInfrastructureRepository {
	return &nitroInfrastructureRepo{pool: pool}
}

// Get retrieves infrastructure for a parent chain.
func (r *nitroInfrastructureRepo) Get(ctx context.Context, parentChainID int64) (*NitroInfrastructure, error) {
	query := `
		SELECT parent_chain_id, rollup_creator_address, bridge_creator_address, version,
		       deployed_at, deployed_by, deployment_tx_hash, osp_entry_address,
		       challenge_manager_templates_address, created_at, updated_at
		FROM nitro_infrastructure
		WHERE parent_chain_id = $1`

	var infra NitroInfrastructure
	err := r.pool.QueryRow(ctx, query, parentChainID).Scan(
		&infra.ParentChainID,
		&infra.RollupCreatorAddress,
		&infra.BridgeCreatorAddress,
		&infra.Version,
		&infra.DeployedAt,
		&infra.DeployedBy,
		&infra.DeploymentTxHash,
		&infra.OSPEntryAddress,
		&infra.ChallengeManagerTemplatesAddress,
		&infra.CreatedAt,
		&infra.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &infra, nil
}

// Create inserts new infrastructure record.
func (r *nitroInfrastructureRepo) Create(ctx context.Context, infra *NitroInfrastructure) error {
	query := `
		INSERT INTO nitro_infrastructure (
			parent_chain_id, rollup_creator_address, bridge_creator_address, version,
			deployed_at, deployed_by, deployment_tx_hash, osp_entry_address,
			challenge_manager_templates_address
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	if infra.DeployedAt.IsZero() {
		infra.DeployedAt = time.Now()
	}

	return r.pool.QueryRow(ctx, query,
		infra.ParentChainID,
		infra.RollupCreatorAddress,
		infra.BridgeCreatorAddress,
		infra.Version,
		infra.DeployedAt,
		infra.DeployedBy,
		infra.DeploymentTxHash,
		infra.OSPEntryAddress,
		infra.ChallengeManagerTemplatesAddress,
	).Scan(&infra.CreatedAt, &infra.UpdatedAt)
}

// Update updates an existing infrastructure record.
func (r *nitroInfrastructureRepo) Update(ctx context.Context, infra *NitroInfrastructure) error {
	query := `
		UPDATE nitro_infrastructure
		SET rollup_creator_address = $2,
		    bridge_creator_address = $3,
		    version = $4,
		    deployed_at = $5,
		    deployed_by = $6,
		    deployment_tx_hash = $7,
		    osp_entry_address = $8,
		    challenge_manager_templates_address = $9,
		    updated_at = NOW()
		WHERE parent_chain_id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		infra.ParentChainID,
		infra.RollupCreatorAddress,
		infra.BridgeCreatorAddress,
		infra.Version,
		infra.DeployedAt,
		infra.DeployedBy,
		infra.DeploymentTxHash,
		infra.OSPEntryAddress,
		infra.ChallengeManagerTemplatesAddress,
	).Scan(&infra.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return pgx.ErrNoRows
	}
	return err
}

// Upsert creates or updates infrastructure.
func (r *nitroInfrastructureRepo) Upsert(ctx context.Context, infra *NitroInfrastructure) error {
	query := `
		INSERT INTO nitro_infrastructure (
			parent_chain_id, rollup_creator_address, bridge_creator_address, version,
			deployed_at, deployed_by, deployment_tx_hash, osp_entry_address,
			challenge_manager_templates_address
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (parent_chain_id) DO UPDATE SET
			rollup_creator_address = EXCLUDED.rollup_creator_address,
			bridge_creator_address = EXCLUDED.bridge_creator_address,
			version = EXCLUDED.version,
			deployed_at = EXCLUDED.deployed_at,
			deployed_by = EXCLUDED.deployed_by,
			deployment_tx_hash = EXCLUDED.deployment_tx_hash,
			osp_entry_address = EXCLUDED.osp_entry_address,
			challenge_manager_templates_address = EXCLUDED.challenge_manager_templates_address,
			updated_at = NOW()
		RETURNING created_at, updated_at`

	if infra.DeployedAt.IsZero() {
		infra.DeployedAt = time.Now()
	}

	return r.pool.QueryRow(ctx, query,
		infra.ParentChainID,
		infra.RollupCreatorAddress,
		infra.BridgeCreatorAddress,
		infra.Version,
		infra.DeployedAt,
		infra.DeployedBy,
		infra.DeploymentTxHash,
		infra.OSPEntryAddress,
		infra.ChallengeManagerTemplatesAddress,
	).Scan(&infra.CreatedAt, &infra.UpdatedAt)
}

// List returns all infrastructure records.
func (r *nitroInfrastructureRepo) List(ctx context.Context) ([]*NitroInfrastructure, error) {
	query := `
		SELECT parent_chain_id, rollup_creator_address, bridge_creator_address, version,
		       deployed_at, deployed_by, deployment_tx_hash, osp_entry_address,
		       challenge_manager_templates_address, created_at, updated_at
		FROM nitro_infrastructure
		ORDER BY parent_chain_id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var infraList []*NitroInfrastructure
	for rows.Next() {
		var infra NitroInfrastructure
		if err := rows.Scan(
			&infra.ParentChainID,
			&infra.RollupCreatorAddress,
			&infra.BridgeCreatorAddress,
			&infra.Version,
			&infra.DeployedAt,
			&infra.DeployedBy,
			&infra.DeploymentTxHash,
			&infra.OSPEntryAddress,
			&infra.ChallengeManagerTemplatesAddress,
			&infra.CreatedAt,
			&infra.UpdatedAt,
		); err != nil {
			return nil, err
		}
		infraList = append(infraList, &infra)
	}
	return infraList, rows.Err()
}

// Delete removes infrastructure for a parent chain.
func (r *nitroInfrastructureRepo) Delete(ctx context.Context, parentChainID int64) error {
	query := `DELETE FROM nitro_infrastructure WHERE parent_chain_id = $1`
	result, err := r.pool.Exec(ctx, query, parentChainID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Compile-time check to ensure nitroInfrastructureRepo implements NitroInfrastructureRepository.
var _ NitroInfrastructureRepository = (*nitroInfrastructureRepo)(nil)
