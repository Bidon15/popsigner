// Package repository provides data access layer implementations.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// WebhookRepository defines the interface for webhook operations.
type WebhookRepository interface {
	// Webhook CRUD
	Create(ctx context.Context, webhook *models.Webhook) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.Webhook, error)
	ListByOrgAndEvent(ctx context.Context, orgID uuid.UUID, event models.WebhookEvent) ([]*models.Webhook, error)
	Update(ctx context.Context, webhook *models.Webhook) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Delivery tracking
	CreateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error
	GetDeliveryByID(ctx context.Context, id uuid.UUID) (*models.WebhookDelivery, error)
	ListDeliveriesByWebhook(ctx context.Context, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error)

	// Status management
	UpdateLastTriggered(ctx context.Context, webhookID uuid.UUID) error
	IncrementFailureCount(ctx context.Context, webhookID uuid.UUID) error
	ResetFailureCount(ctx context.Context, webhookID uuid.UUID) error
}

type webhookRepo struct {
	pool *pgxpool.Pool
}

// NewWebhookRepository creates a new webhook repository.
func NewWebhookRepository(pool *pgxpool.Pool) WebhookRepository {
	return &webhookRepo{pool: pool}
}

// Create inserts a new webhook.
func (r *webhookRepo) Create(ctx context.Context, webhook *models.Webhook) error {
	if webhook.ID == uuid.Nil {
		webhook.ID = uuid.New()
	}
	now := time.Now()
	webhook.CreatedAt = now
	webhook.UpdatedAt = now

	// Convert events slice to JSON for storage
	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	query := `
		INSERT INTO webhooks (id, org_id, url, secret, events, enabled, failure_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = r.pool.Exec(ctx, query,
		webhook.ID,
		webhook.OrgID,
		webhook.URL,
		webhook.Secret,
		eventsJSON,
		webhook.Enabled,
		webhook.FailureCount,
		webhook.CreatedAt,
		webhook.UpdatedAt,
	)
	return err
}

// GetByID retrieves a webhook by ID.
func (r *webhookRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	query := `
		SELECT id, org_id, url, secret, events, enabled, last_triggered_at, failure_count, created_at, updated_at
		FROM webhooks WHERE id = $1`

	var webhook models.Webhook
	var eventsJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&webhook.ID,
		&webhook.OrgID,
		&webhook.URL,
		&webhook.Secret,
		&eventsJSON,
		&webhook.Enabled,
		&webhook.LastTriggeredAt,
		&webhook.FailureCount,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse events JSON
	if err := json.Unmarshal(eventsJSON, &webhook.Events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}

	return &webhook, nil
}

// ListByOrg retrieves all webhooks for an organization.
func (r *webhookRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.Webhook, error) {
	query := `
		SELECT id, org_id, url, secret, events, enabled, last_triggered_at, failure_count, created_at, updated_at
		FROM webhooks 
		WHERE org_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanWebhooks(rows)
}

// ListByOrgAndEvent retrieves all enabled webhooks that subscribe to a specific event.
func (r *webhookRepo) ListByOrgAndEvent(ctx context.Context, orgID uuid.UUID, event models.WebhookEvent) ([]*models.Webhook, error) {
	// Use JSON containment to check if the event is in the events array
	query := `
		SELECT id, org_id, url, secret, events, enabled, last_triggered_at, failure_count, created_at, updated_at
		FROM webhooks 
		WHERE org_id = $1 
		  AND enabled = true 
		  AND events @> $2
		ORDER BY created_at`

	eventJSON, _ := json.Marshal([]string{string(event)})

	rows, err := r.pool.Query(ctx, query, orgID, eventJSON)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanWebhooks(rows)
}

// scanWebhooks scans multiple webhook rows.
func (r *webhookRepo) scanWebhooks(rows pgx.Rows) ([]*models.Webhook, error) {
	var webhooks []*models.Webhook
	for rows.Next() {
		var webhook models.Webhook
		var eventsJSON []byte

		if err := rows.Scan(
			&webhook.ID,
			&webhook.OrgID,
			&webhook.URL,
			&webhook.Secret,
			&eventsJSON,
			&webhook.Enabled,
			&webhook.LastTriggeredAt,
			&webhook.FailureCount,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(eventsJSON, &webhook.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		webhooks = append(webhooks, &webhook)
	}
	return webhooks, rows.Err()
}

// Update updates a webhook.
func (r *webhookRepo) Update(ctx context.Context, webhook *models.Webhook) error {
	webhook.UpdatedAt = time.Now()

	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	query := `
		UPDATE webhooks 
		SET url = $2, events = $3, enabled = $4, updated_at = $5
		WHERE id = $1`

	_, err = r.pool.Exec(ctx, query,
		webhook.ID,
		webhook.URL,
		eventsJSON,
		webhook.Enabled,
		webhook.UpdatedAt,
	)
	return err
}

// Delete removes a webhook.
func (r *webhookRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CreateDelivery records a webhook delivery attempt.
func (r *webhookRepo) CreateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	if delivery.ID == uuid.Nil {
		delivery.ID = uuid.New()
	}
	delivery.AttemptedAt = time.Now()

	query := `
		INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status_code, response_body, duration_ms, success, error, attempted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		delivery.ID,
		delivery.WebhookID,
		delivery.Event,
		delivery.Payload,
		delivery.StatusCode,
		delivery.ResponseBody,
		int64(delivery.Duration/time.Millisecond),
		delivery.Success,
		delivery.Error,
		delivery.AttemptedAt,
	)
	return err
}

// GetDeliveryByID retrieves a delivery by ID.
func (r *webhookRepo) GetDeliveryByID(ctx context.Context, id uuid.UUID) (*models.WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event, payload, status_code, response_body, duration_ms, success, error, attempted_at
		FROM webhook_deliveries WHERE id = $1`

	var delivery models.WebhookDelivery
	var durationMs int64

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&delivery.ID,
		&delivery.WebhookID,
		&delivery.Event,
		&delivery.Payload,
		&delivery.StatusCode,
		&delivery.ResponseBody,
		&durationMs,
		&delivery.Success,
		&delivery.Error,
		&delivery.AttemptedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	delivery.Duration = time.Duration(durationMs) * time.Millisecond
	return &delivery, nil
}

// ListDeliveriesByWebhook retrieves recent deliveries for a webhook.
func (r *webhookRepo) ListDeliveriesByWebhook(ctx context.Context, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, webhook_id, event, payload, status_code, response_body, duration_ms, success, error, attempted_at
		FROM webhook_deliveries 
		WHERE webhook_id = $1
		ORDER BY attempted_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*models.WebhookDelivery
	for rows.Next() {
		var delivery models.WebhookDelivery
		var durationMs int64

		if err := rows.Scan(
			&delivery.ID,
			&delivery.WebhookID,
			&delivery.Event,
			&delivery.Payload,
			&delivery.StatusCode,
			&delivery.ResponseBody,
			&durationMs,
			&delivery.Success,
			&delivery.Error,
			&delivery.AttemptedAt,
		); err != nil {
			return nil, err
		}

		delivery.Duration = time.Duration(durationMs) * time.Millisecond
		deliveries = append(deliveries, &delivery)
	}
	return deliveries, rows.Err()
}

// UpdateLastTriggered updates the last triggered timestamp.
func (r *webhookRepo) UpdateLastTriggered(ctx context.Context, webhookID uuid.UUID) error {
	query := `UPDATE webhooks SET last_triggered_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, webhookID)
	return err
}

// IncrementFailureCount increments the failure counter.
func (r *webhookRepo) IncrementFailureCount(ctx context.Context, webhookID uuid.UUID) error {
	query := `UPDATE webhooks SET failure_count = failure_count + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, webhookID)
	return err
}

// ResetFailureCount resets the failure counter to zero.
func (r *webhookRepo) ResetFailureCount(ctx context.Context, webhookID uuid.UUID) error {
	query := `UPDATE webhooks SET failure_count = 0, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, webhookID)
	return err
}

// Compile-time check to ensure webhookRepo implements WebhookRepository.
var _ WebhookRepository = (*webhookRepo)(nil)

