package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// MockWebhookRepository is a mock implementation of repository.WebhookRepository.
type MockWebhookRepository struct {
	mock.Mock
}

func (m *MockWebhookRepository) Create(ctx context.Context, webhook *models.Webhook) error {
	args := m.Called(ctx, webhook)
	return args.Error(0)
}

func (m *MockWebhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

func (m *MockWebhookRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.Webhook, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Webhook), args.Error(1)
}

func (m *MockWebhookRepository) ListByOrgAndEvent(ctx context.Context, orgID uuid.UUID, event models.WebhookEvent) ([]*models.Webhook, error) {
	args := m.Called(ctx, orgID, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Webhook), args.Error(1)
}

func (m *MockWebhookRepository) Update(ctx context.Context, webhook *models.Webhook) error {
	args := m.Called(ctx, webhook)
	return args.Error(0)
}

func (m *MockWebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWebhookRepository) CreateDelivery(ctx context.Context, delivery *models.WebhookDelivery) error {
	args := m.Called(ctx, delivery)
	return args.Error(0)
}

func (m *MockWebhookRepository) GetDeliveryByID(ctx context.Context, id uuid.UUID) (*models.WebhookDelivery, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookRepository) ListDeliveriesByWebhook(ctx context.Context, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error) {
	args := m.Called(ctx, webhookID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookRepository) UpdateLastTriggered(ctx context.Context, webhookID uuid.UUID) error {
	args := m.Called(ctx, webhookID)
	return args.Error(0)
}

func (m *MockWebhookRepository) IncrementFailureCount(ctx context.Context, webhookID uuid.UUID) error {
	args := m.Called(ctx, webhookID)
	return args.Error(0)
}

func (m *MockWebhookRepository) ResetFailureCount(ctx context.Context, webhookID uuid.UUID) error {
	args := m.Called(ctx, webhookID)
	return args.Error(0)
}

func TestWebhookService_Create(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()

	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Webhook")).Return(nil)

	webhook, err := svc.Create(ctx, orgID, CreateWebhookRequest{
		URL:    "https://example.com/webhook",
		Events: []models.WebhookEvent{models.WebhookEventKeyCreated, models.WebhookEventKeyDeleted},
	})

	require.NoError(t, err)
	assert.NotNil(t, webhook)
	assert.Equal(t, orgID, webhook.OrgID)
	assert.Equal(t, "https://example.com/webhook", webhook.URL)
	assert.Len(t, webhook.Events, 2)
	assert.True(t, webhook.Enabled)
	assert.NotEmpty(t, webhook.Secret)
	assert.Contains(t, webhook.Secret, "whsec_")
	mockRepo.AssertExpectations(t)
}

func TestWebhookService_Create_InvalidURL(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()

	_, err := svc.Create(ctx, orgID, CreateWebhookRequest{
		URL:    "not-a-valid-url",
		Events: []models.WebhookEvent{models.WebhookEventKeyCreated},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid URL")
}

func TestWebhookService_Create_EmptyURL(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()

	_, err := svc.Create(ctx, orgID, CreateWebhookRequest{
		URL:    "",
		Events: []models.WebhookEvent{models.WebhookEventKeyCreated},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestWebhookService_Create_NoEvents(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()

	_, err := svc.Create(ctx, orgID, CreateWebhookRequest{
		URL:    "https://example.com/webhook",
		Events: []models.WebhookEvent{},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "At least one event")
}

func TestWebhookService_Create_InvalidEvent(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()

	_, err := svc.Create(ctx, orgID, CreateWebhookRequest{
		URL:    "https://example.com/webhook",
		Events: []models.WebhookEvent{"invalid.event"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid event")
}

func TestWebhookService_Get(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()
	webhookID := uuid.New()

	expectedWebhook := &models.Webhook{
		ID:      webhookID,
		OrgID:   orgID,
		URL:     "https://example.com/webhook",
		Secret:  "whsec_test",
		Events:  []models.WebhookEvent{models.WebhookEventKeyCreated},
		Enabled: true,
	}

	mockRepo.On("GetByID", ctx, webhookID).Return(expectedWebhook, nil)

	webhook, err := svc.Get(ctx, orgID, webhookID)

	require.NoError(t, err)
	assert.Equal(t, expectedWebhook, webhook)
	mockRepo.AssertExpectations(t)
}

func TestWebhookService_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()
	webhookID := uuid.New()

	mockRepo.On("GetByID", ctx, webhookID).Return(nil, nil)

	_, err := svc.Get(ctx, orgID, webhookID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestWebhookService_Get_WrongOrg(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()
	otherOrgID := uuid.New()
	webhookID := uuid.New()

	webhook := &models.Webhook{
		ID:    webhookID,
		OrgID: otherOrgID, // Different org
	}

	mockRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)

	_, err := svc.Get(ctx, orgID, webhookID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestWebhookService_List(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()

	webhooks := []*models.Webhook{
		{ID: uuid.New(), OrgID: orgID, URL: "https://example.com/hook1"},
		{ID: uuid.New(), OrgID: orgID, URL: "https://example.com/hook2"},
	}

	mockRepo.On("ListByOrg", ctx, orgID).Return(webhooks, nil)

	result, err := svc.List(ctx, orgID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestWebhookService_Update(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()
	webhookID := uuid.New()

	existingWebhook := &models.Webhook{
		ID:      webhookID,
		OrgID:   orgID,
		URL:     "https://example.com/old",
		Events:  []models.WebhookEvent{models.WebhookEventKeyCreated},
		Enabled: true,
	}

	mockRepo.On("GetByID", ctx, webhookID).Return(existingWebhook, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Webhook")).Return(nil)

	newURL := "https://example.com/new"
	enabled := false

	webhook, err := svc.Update(ctx, orgID, webhookID, UpdateWebhookRequest{
		URL:     &newURL,
		Events:  []models.WebhookEvent{models.WebhookEventKeyDeleted},
		Enabled: &enabled,
	})

	require.NoError(t, err)
	assert.Equal(t, newURL, webhook.URL)
	assert.Equal(t, []models.WebhookEvent{models.WebhookEventKeyDeleted}, webhook.Events)
	assert.False(t, webhook.Enabled)
	mockRepo.AssertExpectations(t)
}

func TestWebhookService_Delete(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()
	webhookID := uuid.New()

	webhook := &models.Webhook{
		ID:    webhookID,
		OrgID: orgID,
	}

	mockRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	mockRepo.On("Delete", ctx, webhookID).Return(nil)

	err := svc.Delete(ctx, orgID, webhookID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestWebhookService_GetDeliveries(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()
	webhookID := uuid.New()

	webhook := &models.Webhook{
		ID:    webhookID,
		OrgID: orgID,
	}

	deliveries := []*models.WebhookDelivery{
		{ID: uuid.New(), WebhookID: webhookID, Event: models.WebhookEventKeyCreated, Success: true},
		{ID: uuid.New(), WebhookID: webhookID, Event: models.WebhookEventKeyDeleted, Success: false},
	}

	mockRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	mockRepo.On("ListDeliveriesByWebhook", ctx, webhookID, 50).Return(deliveries, nil)

	result, err := svc.GetDeliveries(ctx, orgID, webhookID, 50)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestWebhookService_VerifySignature(t *testing.T) {
	mockRepo := new(MockWebhookRepository)
	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig()).(*webhookService)

	secret := "whsec_test_secret"
	body := []byte(`{"event":"key.created","data":{}}`)
	timestamp := time.Now().Unix()

	// Generate signature
	signature := svc.calculateSignature(secret, timestamp, body)

	// Verify it
	valid, err := svc.VerifySignature(secret, signature, body)

	require.NoError(t, err)
	assert.True(t, valid)
}

func TestWebhookService_VerifySignature_Invalid(t *testing.T) {
	mockRepo := new(MockWebhookRepository)
	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig()).(*webhookService)

	secret := "whsec_test_secret"
	body := []byte(`{"event":"key.created","data":{}}`)

	// Use a current timestamp but wrong signature
	timestamp := time.Now().Unix()
	wrongSignature := fmt.Sprintf("t=%d,v1=invalid_signature_that_does_not_match", timestamp)

	valid, err := svc.VerifySignature(secret, wrongSignature, body)

	require.NoError(t, err)
	assert.False(t, valid) // Should return false for HMAC mismatch
}

func TestWebhookService_VerifySignature_ExpiredTimestamp(t *testing.T) {
	mockRepo := new(MockWebhookRepository)
	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig()).(*webhookService)

	secret := "whsec_test_secret"
	body := []byte(`{"event":"key.created","data":{}}`)

	// Old timestamp (more than 5 minutes ago)
	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
	signature := svc.calculateSignature(secret, oldTimestamp, body)

	_, err := svc.VerifySignature(secret, signature, body)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestWebhookService_Deliver(t *testing.T) {
	// Create a test server that will receive the webhook
	received := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("X-Webhook-Signature"), "t=")
		assert.Equal(t, string(models.WebhookEventKeyCreated), r.Header.Get("X-Webhook-Event"))

		// Parse body
		var payload models.WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		assert.NoError(t, err)
		assert.Equal(t, models.WebhookEventKeyCreated, payload.Event)

		w.WriteHeader(http.StatusOK)
		received <- true
	}))
	defer server.Close()

	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	config := DefaultWebhookServiceConfig()
	config.MaxRetries = 0 // No retries for test
	svc := NewWebhookService(mockRepo, config)

	orgID := uuid.New()
	webhookID := uuid.New()

	webhook := &models.Webhook{
		ID:      webhookID,
		OrgID:   orgID,
		URL:     server.URL,
		Secret:  "whsec_test",
		Events:  []models.WebhookEvent{models.WebhookEventKeyCreated},
		Enabled: true,
	}

	mockRepo.On("ListByOrgAndEvent", ctx, orgID, models.WebhookEventKeyCreated).Return([]*models.Webhook{webhook}, nil)
	mockRepo.On("CreateDelivery", mock.Anything, mock.AnythingOfType("*models.WebhookDelivery")).Return(nil)
	mockRepo.On("UpdateLastTriggered", mock.Anything, webhookID).Return(nil)
	mockRepo.On("ResetFailureCount", mock.Anything, webhookID).Return(nil)

	err := svc.Deliver(ctx, orgID, models.WebhookEventKeyCreated, map[string]string{"key": "value"})

	require.NoError(t, err)

	// Wait for delivery
	select {
	case <-received:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Webhook not received within timeout")
	}
}

func TestWebhookService_Deliver_Failure(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	config := DefaultWebhookServiceConfig()
	config.MaxRetries = 0 // No retries for faster test
	config.MaxFailures = 5
	svc := NewWebhookService(mockRepo, config)

	orgID := uuid.New()
	webhookID := uuid.New()

	webhook := &models.Webhook{
		ID:           webhookID,
		OrgID:        orgID,
		URL:          server.URL,
		Secret:       "whsec_test",
		Events:       []models.WebhookEvent{models.WebhookEventKeyCreated},
		Enabled:      true,
		FailureCount: 0,
	}

	mockRepo.On("ListByOrgAndEvent", ctx, orgID, models.WebhookEventKeyCreated).Return([]*models.Webhook{webhook}, nil)
	mockRepo.On("CreateDelivery", mock.Anything, mock.MatchedBy(func(d *models.WebhookDelivery) bool {
		return !d.Success
	})).Return(nil)
	mockRepo.On("UpdateLastTriggered", mock.Anything, webhookID).Return(nil)
	mockRepo.On("IncrementFailureCount", mock.Anything, webhookID).Return(nil)
	mockRepo.On("GetByID", mock.Anything, webhookID).Return(&models.Webhook{
		ID:           webhookID,
		OrgID:        orgID,
		FailureCount: 1,
	}, nil)

	err := svc.Deliver(ctx, orgID, models.WebhookEventKeyCreated, map[string]string{"key": "value"})

	require.NoError(t, err)

	// Wait for delivery attempt to complete
	time.Sleep(100 * time.Millisecond)
}

func TestWebhookService_RotateSecret(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockWebhookRepository)

	svc := NewWebhookService(mockRepo, DefaultWebhookServiceConfig())

	orgID := uuid.New()
	webhookID := uuid.New()
	oldSecret := "whsec_old_secret"

	webhook := &models.Webhook{
		ID:     webhookID,
		OrgID:  orgID,
		Secret: oldSecret,
	}

	mockRepo.On("GetByID", ctx, webhookID).Return(webhook, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*models.Webhook")).Return(nil)

	result, err := svc.RotateSecret(ctx, orgID, webhookID)

	require.NoError(t, err)
	assert.NotEqual(t, oldSecret, result.Secret)
	assert.Contains(t, result.Secret, "whsec_")
	mockRepo.AssertExpectations(t)
}

func TestGenerateWebhookSecret(t *testing.T) {
	secret1, err1 := generateWebhookSecret()
	secret2, err2 := generateWebhookSecret()

	require.NoError(t, err1)
	require.NoError(t, err2)

	assert.NotEqual(t, secret1, secret2)
	assert.Contains(t, secret1, "whsec_")
	assert.Contains(t, secret2, "whsec_")
}

