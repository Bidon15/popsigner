// Package repository provides database access layer implementations.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// UserRepository defines methods for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByOAuth(ctx context.Context, provider, providerID string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateOAuth(ctx context.Context, userID uuid.UUID, provider, providerID string) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	SetEmailVerified(ctx context.Context, id uuid.UUID) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

type userRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	user.ID = uuid.New()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (id, email, password_hash, name, avatar_url, email_verified, oauth_provider, oauth_provider_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.AvatarURL,
		user.EmailVerified,
		user.OAuthProvider,
		user.OAuthProviderID,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, avatar_url, email_verified,
		       oauth_provider, oauth_provider_id, last_login_at, created_at, updated_at
		FROM users WHERE id = $1`

	var user models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.AvatarURL,
		&user.EmailVerified,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, avatar_url, email_verified,
		       oauth_provider, oauth_provider_id, last_login_at, created_at, updated_at
		FROM users WHERE email = $1`

	var user models.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.AvatarURL,
		&user.EmailVerified,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByOAuth(ctx context.Context, provider, providerID string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, avatar_url, email_verified,
		       oauth_provider, oauth_provider_id, last_login_at, created_at, updated_at
		FROM users WHERE oauth_provider = $1 AND oauth_provider_id = $2`

	var user models.User
	err := r.pool.QueryRow(ctx, query, provider, providerID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.AvatarURL,
		&user.EmailVerified,
		&user.OAuthProvider,
		&user.OAuthProviderID,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) Update(ctx context.Context, user *models.User) error {
	query := `UPDATE users SET name = $2, avatar_url = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, user.ID, user.Name, user.AvatarURL)
	return err
}

func (r *userRepo) UpdateOAuth(ctx context.Context, userID uuid.UUID, provider, providerID string) error {
	query := `UPDATE users SET oauth_provider = $2, oauth_provider_id = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, userID, provider, providerID)
	return err
}

func (r *userRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, hash)
	return err
}

func (r *userRepo) SetEmailVerified(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET email_verified = true, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// Compile-time check to ensure userRepo implements UserRepository.
var _ UserRepository = (*userRepo)(nil)
