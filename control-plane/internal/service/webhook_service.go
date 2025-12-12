// Package service provides business logic implementations.
package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	apierrors "github.com/Bidon15/popsigner/control-plane/internal/pkg/errors"
	"github.com/Bidon15/popsigner/control-plane/internal/repository"
)

// WebhookService defines the interface for webhook operations.
type WebhookService interface {
	// CRUD operations
	Create(ctx context.Context, orgID uuid.UUID, req CreateWebhookRequest) (*models.Webhook, error)
	Get(ctx context.Context, orgID, webhookID uuid.UUID) (*models.Webhook, error)
	List(ctx context.Context, orgID uuid.UUID) ([]*models.Webhook, error)
	Update(ctx context.Context, orgID, webhookID uuid.UUID, req UpdateWebhookRequest) (*models.Webhook, error)
	Delete(ctx context.Context, orgID, webhookID uuid.UUID) error

	// Delivery operations
	Deliver(ctx context.Context, orgID uuid.UUID, event models.WebhookEvent, payload any) error
	GetDeliveries(ctx context.Context, orgID, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error)
	GetDelivery(ctx context.Context, orgID, deliveryID uuid.UUID) (*models.WebhookDelivery, error)
	RetryDelivery(ctx context.Context, orgID, webhookID, deliveryID uuid.UUID) error

	// Signature verification helper
	VerifySignature(secret, signature string, body []byte) (bool, error)

	// Rotate webhook secret
	RotateSecret(ctx context.Context, orgID, webhookID uuid.UUID) (*models.Webhook, error)
}

// CreateWebhookRequest is the request to create a webhook.
type CreateWebhookRequest struct {
	URL    string                 `json:"url" validate:"required,url"`
	Events []models.WebhookEvent  `json:"events" validate:"required,min=1"`
}

// UpdateWebhookRequest is the request to update a webhook.
type UpdateWebhookRequest struct {
	URL     *string                `json:"url,omitempty" validate:"omitempty,url"`
	Events  []models.WebhookEvent  `json:"events,omitempty"`
	Enabled *bool                  `json:"enabled,omitempty"`
}

// WebhookServiceConfig holds configuration for the webhook service.
type WebhookServiceConfig struct {
	// Timeout for webhook HTTP requests
	Timeout time.Duration
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// RetryBackoff is the base backoff duration between retries
	RetryBackoff time.Duration
	// MaxFailures before disabling a webhook
	MaxFailures int
	// UserAgent for webhook requests
	UserAgent string
}

// DefaultWebhookServiceConfig returns sensible defaults.
func DefaultWebhookServiceConfig() WebhookServiceConfig {
	return WebhookServiceConfig{
		Timeout:      10 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 5 * time.Second,
		MaxFailures:  10,
		UserAgent:    "BanhBaoRing-Webhook/1.0",
	}
}

type webhookService struct {
	webhookRepo repository.WebhookRepository
	httpClient  *http.Client
	config      WebhookServiceConfig
}

// NewWebhookService creates a new webhook service.
func NewWebhookService(webhookRepo repository.WebhookRepository, config WebhookServiceConfig) WebhookService {
	return &webhookService{
		webhookRepo: webhookRepo,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}
}

// validWebhookEvents contains all valid webhook events.
var validWebhookEvents = map[models.WebhookEvent]bool{
	models.WebhookEventKeyCreated:         true,
	models.WebhookEventKeyDeleted:         true,
	models.WebhookEventSignatureCompleted: true,
	models.WebhookEventQuotaWarning:       true,
	models.WebhookEventQuotaExceeded:      true,
	models.WebhookEventPaymentSucceeded:   true,
	models.WebhookEventPaymentFailed:      true,
}

// Create creates a new webhook.
func (s *webhookService) Create(ctx context.Context, orgID uuid.UUID, req CreateWebhookRequest) (*models.Webhook, error) {
	// Validate URL
	if req.URL == "" {
		return nil, apierrors.NewValidationError("url", "URL is required")
	}
	if _, err := url.ParseRequestURI(req.URL); err != nil {
		return nil, apierrors.NewValidationError("url", "Invalid URL format")
	}

	// Validate events
	if len(req.Events) == 0 {
		return nil, apierrors.NewValidationError("events", "At least one event is required")
	}
	for _, event := range req.Events {
		if !validWebhookEvents[event] {
			return nil, apierrors.NewValidationError("events", fmt.Sprintf("Invalid event: %s", event))
		}
	}

	// Generate secret
	secret, err := generateWebhookSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	webhook := &models.Webhook{
		ID:           uuid.New(),
		OrgID:        orgID,
		URL:          req.URL,
		Secret:       secret,
		Events:       req.Events,
		Enabled:      true,
		FailureCount: 0,
	}

	if err := s.webhookRepo.Create(ctx, webhook); err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	return webhook, nil
}

// Get retrieves a webhook by ID.
func (s *webhookService) Get(ctx context.Context, orgID, webhookID uuid.UUID) (*models.Webhook, error) {
	webhook, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	if webhook == nil {
		return nil, apierrors.NewNotFoundError("Webhook")
	}

	// Verify ownership
	if webhook.OrgID != orgID {
		return nil, apierrors.NewNotFoundError("Webhook")
	}

	return webhook, nil
}

// List retrieves all webhooks for an organization.
func (s *webhookService) List(ctx context.Context, orgID uuid.UUID) ([]*models.Webhook, error) {
	webhooks, err := s.webhookRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	return webhooks, nil
}

// Update updates a webhook.
func (s *webhookService) Update(ctx context.Context, orgID, webhookID uuid.UUID, req UpdateWebhookRequest) (*models.Webhook, error) {
	webhook, err := s.Get(ctx, orgID, webhookID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.URL != nil {
		if _, err := url.ParseRequestURI(*req.URL); err != nil {
			return nil, apierrors.NewValidationError("url", "Invalid URL format")
		}
		webhook.URL = *req.URL
	}

	if len(req.Events) > 0 {
		for _, event := range req.Events {
			if !validWebhookEvents[event] {
				return nil, apierrors.NewValidationError("events", fmt.Sprintf("Invalid event: %s", event))
			}
		}
		webhook.Events = req.Events
	}

	if req.Enabled != nil {
		webhook.Enabled = *req.Enabled
		// Reset failure count when re-enabling
		if *req.Enabled && webhook.FailureCount > 0 {
			webhook.FailureCount = 0
		}
	}

	if err := s.webhookRepo.Update(ctx, webhook); err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	return webhook, nil
}

// Delete removes a webhook.
func (s *webhookService) Delete(ctx context.Context, orgID, webhookID uuid.UUID) error {
	// Verify ownership first
	_, err := s.Get(ctx, orgID, webhookID)
	if err != nil {
		return err
	}

	if err := s.webhookRepo.Delete(ctx, webhookID); err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}

// Deliver sends a webhook payload to all matching webhooks for an org.
func (s *webhookService) Deliver(ctx context.Context, orgID uuid.UUID, event models.WebhookEvent, payload any) error {
	// Get all webhooks that subscribe to this event
	webhooks, err := s.webhookRepo.ListByOrgAndEvent(ctx, orgID, event)
	if err != nil {
		return fmt.Errorf("failed to list webhooks: %w", err)
	}

	// Deliver to each webhook asynchronously
	for _, webhook := range webhooks {
		go s.deliverToWebhook(context.Background(), webhook, event, payload)
	}

	return nil
}

// deliverToWebhook sends the payload to a single webhook.
func (s *webhookService) deliverToWebhook(ctx context.Context, webhook *models.Webhook, event models.WebhookEvent, payload any) {
	// Build webhook payload
	webhookPayload := models.WebhookPayload{
		ID:        uuid.New().String(),
		Event:     event,
		OrgID:     webhook.OrgID.String(),
		Timestamp: time.Now().UTC(),
		Data:      payload,
	}

	body, err := json.Marshal(webhookPayload)
	if err != nil {
		s.recordDeliveryFailure(ctx, webhook, event, "", err)
		return
	}

	// Attempt delivery with retries
	var lastErr error
	var lastStatusCode int
	var lastResponse string
	var duration time.Duration

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := s.config.RetryBackoff * time.Duration(1<<uint(attempt-1))
			time.Sleep(backoff)
		}

		statusCode, response, dur, err := s.sendWebhook(ctx, webhook, event, body)
		duration = dur
		lastStatusCode = statusCode
		lastResponse = response
		lastErr = err

		// Success if we got a 2xx response
		if err == nil && statusCode >= 200 && statusCode < 300 {
			s.recordDeliverySuccess(ctx, webhook, event, string(body), statusCode, response, duration)
			return
		}

		// Don't retry on 4xx errors (except 429)
		if statusCode >= 400 && statusCode < 500 && statusCode != 429 {
			break
		}
	}

	// All retries failed
	s.recordDeliveryFailure(ctx, webhook, event, string(body), fmt.Errorf("delivery failed: status=%d, response=%s, err=%v", lastStatusCode, lastResponse, lastErr))
	
	// Update failure count
	_ = s.webhookRepo.IncrementFailureCount(ctx, webhook.ID)

	// Disable webhook if too many failures
	webhook, _ = s.webhookRepo.GetByID(ctx, webhook.ID)
	if webhook != nil && webhook.FailureCount >= s.config.MaxFailures {
		webhook.Enabled = false
		_ = s.webhookRepo.Update(ctx, webhook)
	}
}

// sendWebhook performs the actual HTTP request.
func (s *webhookService) sendWebhook(ctx context.Context, webhook *models.Webhook, event models.WebhookEvent, body []byte) (statusCode int, response string, duration time.Duration, err error) {
	// Calculate signature
	timestamp := time.Now().Unix()
	signature := s.calculateSignature(webhook.Secret, timestamp, body)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(body))
	if err != nil {
		return 0, "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.config.UserAgent)
	req.Header.Set("X-Webhook-ID", webhook.ID.String())
	req.Header.Set("X-Webhook-Event", string(event))
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", timestamp))

	// Send request
	start := time.Now()
	resp, err := s.httpClient.Do(req)
	duration = time.Since(start)

	if err != nil {
		return 0, "", duration, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body (limited)
	respBody := make([]byte, 1024)
	n, _ := io.ReadFull(resp.Body, respBody)
	response = string(respBody[:n])

	return resp.StatusCode, response, duration, nil
}

// calculateSignature generates the HMAC signature for a webhook payload.
func (s *webhookService) calculateSignature(secret string, timestamp int64, body []byte) string {
	// Signed payload format: timestamp.body
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(body))

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signedPayload))
	signature := hex.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("t=%d,v1=%s", timestamp, signature)
}

// VerifySignature verifies a webhook signature.
func (s *webhookService) VerifySignature(secret, signature string, body []byte) (bool, error) {
	// Parse signature header: t=timestamp,v1=signature
	var timestamp int64
	var sig string

	_, err := fmt.Sscanf(signature, "t=%d,v1=%s", &timestamp, &sig)
	if err != nil {
		return false, fmt.Errorf("invalid signature format")
	}

	// Check timestamp is within 5 minutes
	now := time.Now().Unix()
	if now-timestamp > 300 || timestamp-now > 300 {
		return false, fmt.Errorf("signature timestamp expired")
	}

	// Recalculate signature
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(body))
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expectedSig)), nil
}

// recordDeliverySuccess records a successful webhook delivery.
func (s *webhookService) recordDeliverySuccess(ctx context.Context, webhook *models.Webhook, event models.WebhookEvent, payload string, statusCode int, response string, duration time.Duration) {
	delivery := &models.WebhookDelivery{
		ID:           uuid.New(),
		WebhookID:    webhook.ID,
		Event:        event,
		Payload:      payload,
		StatusCode:   statusCode,
		ResponseBody: response,
		Duration:     duration,
		Success:      true,
	}

	_ = s.webhookRepo.CreateDelivery(ctx, delivery)
	_ = s.webhookRepo.UpdateLastTriggered(ctx, webhook.ID)
	_ = s.webhookRepo.ResetFailureCount(ctx, webhook.ID)
}

// recordDeliveryFailure records a failed webhook delivery.
func (s *webhookService) recordDeliveryFailure(ctx context.Context, webhook *models.Webhook, event models.WebhookEvent, payload string, err error) {
	delivery := &models.WebhookDelivery{
		ID:        uuid.New(),
		WebhookID: webhook.ID,
		Event:     event,
		Payload:   payload,
		Success:   false,
		Error:     err.Error(),
	}

	_ = s.webhookRepo.CreateDelivery(ctx, delivery)
	_ = s.webhookRepo.UpdateLastTriggered(ctx, webhook.ID)
}

// GetDeliveries retrieves delivery history for a webhook.
func (s *webhookService) GetDeliveries(ctx context.Context, orgID, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error) {
	// Verify ownership
	_, err := s.Get(ctx, orgID, webhookID)
	if err != nil {
		return nil, err
	}

	deliveries, err := s.webhookRepo.ListDeliveriesByWebhook(ctx, webhookID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}

	return deliveries, nil
}

// GetDelivery retrieves a specific delivery.
func (s *webhookService) GetDelivery(ctx context.Context, orgID, deliveryID uuid.UUID) (*models.WebhookDelivery, error) {
	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}
	if delivery == nil {
		return nil, apierrors.NewNotFoundError("Delivery")
	}

	// Verify ownership by checking the parent webhook
	webhook, err := s.webhookRepo.GetByID(ctx, delivery.WebhookID)
	if err != nil || webhook == nil || webhook.OrgID != orgID {
		return nil, apierrors.NewNotFoundError("Delivery")
	}

	return delivery, nil
}

// RetryDelivery retries a failed webhook delivery.
func (s *webhookService) RetryDelivery(ctx context.Context, orgID, webhookID, deliveryID uuid.UUID) error {
	// Get the original delivery
	delivery, err := s.webhookRepo.GetDeliveryByID(ctx, deliveryID)
	if err != nil {
		return fmt.Errorf("failed to get delivery: %w", err)
	}
	if delivery == nil {
		return apierrors.NewNotFoundError("Delivery")
	}

	// Verify delivery belongs to the webhook
	if delivery.WebhookID != webhookID {
		return apierrors.NewNotFoundError("Delivery")
	}

	// Get the webhook
	webhook, err := s.Get(ctx, orgID, webhookID)
	if err != nil {
		return err
	}

	// Parse the original payload
	var payload models.WebhookPayload
	if err := json.Unmarshal([]byte(delivery.Payload), &payload); err != nil {
		return fmt.Errorf("failed to parse original payload: %w", err)
	}

	// Retry delivery asynchronously
	go s.deliverToWebhook(context.Background(), webhook, delivery.Event, payload.Data)

	return nil
}

// RotateSecret generates a new secret for a webhook.
func (s *webhookService) RotateSecret(ctx context.Context, orgID, webhookID uuid.UUID) (*models.Webhook, error) {
	webhook, err := s.Get(ctx, orgID, webhookID)
	if err != nil {
		return nil, err
	}

	// Generate new secret
	newSecret, err := generateWebhookSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	webhook.Secret = newSecret
	webhook.UpdatedAt = time.Now()

	// Update in database (need to use a raw update for the secret)
	if err := s.webhookRepo.Update(ctx, webhook); err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	return webhook, nil
}

// generateWebhookSecret generates a secure random webhook secret.
func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("whsec_%s", base64.URLEncoding.EncodeToString(b)), nil
}

// Compile-time check to ensure webhookService implements WebhookService.
var _ WebhookService = (*webhookService)(nil)

