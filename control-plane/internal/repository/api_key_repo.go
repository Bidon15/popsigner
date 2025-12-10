// Package repository provides data access layer implementations.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// APIKeyRepository defines the interface for API key data operations.
type APIKeyRepository interface {
	Create(ctx context.Context, key *models.APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error)
	GetByPrefix(ctx context.Context, prefix string) (*models.APIKey, error)
	GetByHash(ctx context.Context, hash string) (*models.APIKey, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type apiKeyRepo struct {
	pool *pgxpool.Pool
}

// NewAPIKeyRepository creates a new API key repository.
func NewAPIKeyRepository(pool *pgxpool.Pool) APIKeyRepository {
	return &apiKeyRepo{pool: pool}
}

// Create inserts a new API key into the database.
func (r *apiKeyRepo) Create(ctx context.Context, key *models.APIKey) error {
	query := `
		INSERT INTO api_keys (id, org_id, user_id, name, key_prefix, key_hash, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`

	key.ID = uuid.New()
	return r.pool.QueryRow(ctx, query,
		key.ID,
		key.OrgID,
		key.UserID,
		key.Name,
		key.KeyPrefix,
		key.KeyHash,
		key.Scopes,
		key.ExpiresAt,
	).Scan(&key.CreatedAt)
}

// GetByID retrieves an API key by its UUID.
func (r *apiKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	query := `
		SELECT id, org_id, user_id, name, key_prefix, key_hash, scopes,
		       last_used_at, expires_at, revoked_at, created_at
		FROM api_keys WHERE id = $1`

	var key models.APIKey
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&key.ID,
		&key.OrgID,
		&key.UserID,
		&key.Name,
		&key.KeyPrefix,
		&key.KeyHash,
		&key.Scopes,
		&key.LastUsedAt,
		&key.ExpiresAt,
		&key.RevokedAt,
		&key.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// GetByPrefix retrieves an API key by its prefix.
// Used for quick lookup during validation.
func (r *apiKeyRepo) GetByPrefix(ctx context.Context, prefix string) (*models.APIKey, error) {
	query := `
		SELECT id, org_id, user_id, name, key_prefix, key_hash, scopes,
		       last_used_at, expires_at, revoked_at, created_at
		FROM api_keys WHERE key_prefix = $1`

	var key models.APIKey
	err := r.pool.QueryRow(ctx, query, prefix).Scan(
		&key.ID,
		&key.OrgID,
		&key.UserID,
		&key.Name,
		&key.KeyPrefix,
		&key.KeyHash,
		&key.Scopes,
		&key.LastUsedAt,
		&key.ExpiresAt,
		&key.RevokedAt,
		&key.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// GetByHash retrieves an API key by its hash.
// Used for exact key validation.
func (r *apiKeyRepo) GetByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	query := `
		SELECT id, org_id, user_id, name, key_prefix, key_hash, scopes,
		       last_used_at, expires_at, revoked_at, created_at
		FROM api_keys WHERE key_hash = $1`

	var key models.APIKey
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&key.ID,
		&key.OrgID,
		&key.UserID,
		&key.Name,
		&key.KeyPrefix,
		&key.KeyHash,
		&key.Scopes,
		&key.LastUsedAt,
		&key.ExpiresAt,
		&key.RevokedAt,
		&key.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// ListByOrg retrieves all API keys for an organization.
// Does not return the key hash for security.
func (r *apiKeyRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error) {
	query := `
		SELECT id, org_id, user_id, name, key_prefix, scopes,
		       last_used_at, expires_at, revoked_at, created_at
		FROM api_keys WHERE org_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		var key models.APIKey
		if err := rows.Scan(
			&key.ID,
			&key.OrgID,
			&key.UserID,
			&key.Name,
			&key.KeyPrefix,
			&key.Scopes,
			&key.LastUsedAt,
			&key.ExpiresAt,
			&key.RevokedAt,
			&key.CreatedAt,
		); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}
	return keys, rows.Err()
}

// UpdateLastUsed updates the last_used_at timestamp for an API key.
func (r *apiKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Revoke marks an API key as revoked.
func (r *apiKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Delete removes an API key from the database.
func (r *apiKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

