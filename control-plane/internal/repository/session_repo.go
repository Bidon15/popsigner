package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// SessionRepository defines methods for session data access.
type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	Get(ctx context.Context, id string) (*models.Session, error)
	Delete(ctx context.Context, id string) error
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
	CleanupExpired(ctx context.Context) (int64, error)
}

type sessionRepo struct {
	pool *pgxpool.Pool
}

// NewSessionRepository creates a new SessionRepository instance.
func NewSessionRepository(pool *pgxpool.Pool) SessionRepository {
	return &sessionRepo{pool: pool}
}

func (r *sessionRepo) Create(ctx context.Context, session *models.Session) error {
	session.CreatedAt = time.Now()

	// Convert Data map to JSON bytes
	dataJSON, err := json.Marshal(session.Data)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO sessions (id, user_id, data, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err = r.pool.Exec(ctx, query,
		session.ID,
		session.UserID,
		dataJSON,
		session.ExpiresAt,
		session.CreatedAt,
	)

	return err
}

func (r *sessionRepo) Get(ctx context.Context, id string) (*models.Session, error) {
	query := `
		SELECT id, user_id, data, expires_at, created_at
		FROM sessions WHERE id = $1`

	var session models.Session
	var dataJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&dataJSON,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse JSON data into map
	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &session.Data); err != nil {
			return nil, err
		}
	}

	return &session, nil
}

func (r *sessionRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *sessionRepo) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

func (r *sessionRepo) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < NOW()`
	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// Compile-time check to ensure sessionRepo implements SessionRepository.
var _ SessionRepository = (*sessionRepo)(nil)
