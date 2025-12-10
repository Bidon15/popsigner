package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Bidon15/banhbaoring/control-plane/internal/middleware"
	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
	"github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
	"github.com/Bidon15/banhbaoring/control-plane/internal/service"
)

// MockWebhookService is a mock implementation of service.WebhookService.
type MockWebhookService struct {
	mock.Mock
}

func (m *MockWebhookService) Create(ctx context.Context, orgID uuid.UUID, req service.CreateWebhookRequest) (*models.Webhook, error) {
	args := m.Called(ctx, orgID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

func (m *MockWebhookService) Get(ctx context.Context, orgID, webhookID uuid.UUID) (*models.Webhook, error) {
	args := m.Called(ctx, orgID, webhookID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

func (m *MockWebhookService) List(ctx context.Context, orgID uuid.UUID) ([]*models.Webhook, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Webhook), args.Error(1)
}

func (m *MockWebhookService) Update(ctx context.Context, orgID, webhookID uuid.UUID, req service.UpdateWebhookRequest) (*models.Webhook, error) {
	args := m.Called(ctx, orgID, webhookID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

func (m *MockWebhookService) Delete(ctx context.Context, orgID, webhookID uuid.UUID) error {
	args := m.Called(ctx, orgID, webhookID)
	return args.Error(0)
}

func (m *MockWebhookService) Deliver(ctx context.Context, orgID uuid.UUID, event models.WebhookEvent, payload any) error {
	args := m.Called(ctx, orgID, event, payload)
	return args.Error(0)
}

func (m *MockWebhookService) GetDeliveries(ctx context.Context, orgID, webhookID uuid.UUID, limit int) ([]*models.WebhookDelivery, error) {
	args := m.Called(ctx, orgID, webhookID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookService) GetDelivery(ctx context.Context, orgID, deliveryID uuid.UUID) (*models.WebhookDelivery, error) {
	args := m.Called(ctx, orgID, deliveryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookService) RetryDelivery(ctx context.Context, orgID, webhookID, deliveryID uuid.UUID) error {
	args := m.Called(ctx, orgID, webhookID, deliveryID)
	return args.Error(0)
}

func (m *MockWebhookService) VerifySignature(secret, signature string, body []byte) (bool, error) {
	args := m.Called(secret, signature, body)
	return args.Bool(0), args.Error(1)
}

func (m *MockWebhookService) RotateSecret(ctx context.Context, orgID, webhookID uuid.UUID) (*models.Webhook, error) {
	args := m.Called(ctx, orgID, webhookID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Webhook), args.Error(1)
}

// Helper to create a request with org context
func createWebhookRequest(method, path string, orgID uuid.UUID, body interface{}) *http.Request {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.OrgIDKey, orgID.String())
	return req.WithContext(ctx)
}

func TestWebhookHandler_Create(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()
	webhookID := uuid.New()
	now := time.Now()

	webhook := &models.Webhook{
		ID:        webhookID,
		OrgID:     orgID,
		URL:       "https://example.com/webhook",
		Secret:    "whsec_test_secret",
		Events:    []models.WebhookEvent{models.WebhookEventKeyCreated},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mockService.On("Create", mock.Anything, orgID, mock.AnythingOfType("service.CreateWebhookRequest")).Return(webhook, nil)

	req := createWebhookRequest(http.MethodPost, "/", orgID, CreateWebhookHTTPRequest{
		URL:    "https://example.com/webhook",
		Events: []string{"key.created"},
	})
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var resp response.Response
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotNil(t, resp.Data)

	// Check that secret is included in create response
	dataBytes, _ := json.Marshal(resp.Data)
	var webhookResp WebhookResponse
	json.Unmarshal(dataBytes, &webhookResp)
	assert.NotEmpty(t, webhookResp.Secret)

	mockService.AssertExpectations(t)
}

func TestWebhookHandler_Create_InvalidBody(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()

	req := createWebhookRequest(http.MethodPost, "/", orgID, nil)
	req.Body = http.NoBody
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestWebhookHandler_Create_MissingURL(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()

	req := createWebhookRequest(http.MethodPost, "/", orgID, CreateWebhookHTTPRequest{
		URL:    "",
		Events: []string{"key.created"},
	})
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestWebhookHandler_Create_MissingEvents(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()

	req := createWebhookRequest(http.MethodPost, "/", orgID, CreateWebhookHTTPRequest{
		URL:    "https://example.com/webhook",
		Events: []string{},
	})
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestWebhookHandler_List(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()
	now := time.Now()

	webhooks := []*models.Webhook{
		{ID: uuid.New(), OrgID: orgID, URL: "https://example.com/hook1", Events: []models.WebhookEvent{models.WebhookEventKeyCreated}, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), OrgID: orgID, URL: "https://example.com/hook2", Events: []models.WebhookEvent{models.WebhookEventKeyDeleted}, CreatedAt: now, UpdatedAt: now},
	}

	mockService.On("List", mock.Anything, orgID).Return(webhooks, nil)

	req := createWebhookRequest(http.MethodGet, "/", orgID, nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp response.Response
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)

	// Secrets should not be included in list response
	dataBytes, _ := json.Marshal(resp.Data)
	var webhookResps []WebhookResponse
	json.Unmarshal(dataBytes, &webhookResps)
	for _, wr := range webhookResps {
		assert.Empty(t, wr.Secret)
	}

	mockService.AssertExpectations(t)
}

func TestWebhookHandler_Get(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()
	webhookID := uuid.New()
	now := time.Now()

	webhook := &models.Webhook{
		ID:        webhookID,
		OrgID:     orgID,
		URL:       "https://example.com/webhook",
		Secret:    "whsec_secret",
		Events:    []models.WebhookEvent{models.WebhookEventKeyCreated},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mockService.On("Get", mock.Anything, orgID, webhookID).Return(webhook, nil)

	req := createWebhookRequest(http.MethodGet, "/"+webhookID.String(), orgID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", webhookID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	handler.Get(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Secret should not be included in get response
	var resp response.Response
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	dataBytes, _ := json.Marshal(resp.Data)
	var webhookResp WebhookResponse
	json.Unmarshal(dataBytes, &webhookResp)
	assert.Empty(t, webhookResp.Secret)

	mockService.AssertExpectations(t)
}

func TestWebhookHandler_Update(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()
	webhookID := uuid.New()
	now := time.Now()

	updatedWebhook := &models.Webhook{
		ID:        webhookID,
		OrgID:     orgID,
		URL:       "https://example.com/new-webhook",
		Events:    []models.WebhookEvent{models.WebhookEventKeyDeleted},
		Enabled:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mockService.On("Update", mock.Anything, orgID, webhookID, mock.AnythingOfType("service.UpdateWebhookRequest")).Return(updatedWebhook, nil)

	enabled := false
	req := createWebhookRequest(http.MethodPatch, "/"+webhookID.String(), orgID, UpdateWebhookHTTPRequest{
		URL:     ptrString("https://example.com/new-webhook"),
		Events:  []string{"key.deleted"},
		Enabled: &enabled,
	})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", webhookID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	handler.Update(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestWebhookHandler_Delete(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()
	webhookID := uuid.New()

	mockService.On("Delete", mock.Anything, orgID, webhookID).Return(nil)

	req := createWebhookRequest(http.MethodDelete, "/"+webhookID.String(), orgID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", webhookID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	handler.Delete(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	mockService.AssertExpectations(t)
}

func TestWebhookHandler_RotateSecret(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()
	webhookID := uuid.New()
	now := time.Now()

	webhook := &models.Webhook{
		ID:        webhookID,
		OrgID:     orgID,
		URL:       "https://example.com/webhook",
		Secret:    "whsec_new_secret",
		Events:    []models.WebhookEvent{models.WebhookEventKeyCreated},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	mockService.On("RotateSecret", mock.Anything, orgID, webhookID).Return(webhook, nil)

	req := createWebhookRequest(http.MethodPost, "/"+webhookID.String()+"/rotate-secret", orgID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", webhookID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	handler.RotateSecret(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// New secret should be included in response
	var resp response.Response
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	dataBytes, _ := json.Marshal(resp.Data)
	var webhookResp WebhookResponse
	json.Unmarshal(dataBytes, &webhookResp)
	assert.NotEmpty(t, webhookResp.Secret)

	mockService.AssertExpectations(t)
}

func TestWebhookHandler_ListDeliveries(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()
	webhookID := uuid.New()
	now := time.Now()

	deliveries := []*models.WebhookDelivery{
		{ID: uuid.New(), WebhookID: webhookID, Event: models.WebhookEventKeyCreated, Success: true, AttemptedAt: now},
		{ID: uuid.New(), WebhookID: webhookID, Event: models.WebhookEventKeyDeleted, Success: false, AttemptedAt: now},
	}

	mockService.On("GetDeliveries", mock.Anything, orgID, webhookID, 50).Return(deliveries, nil)

	req := createWebhookRequest(http.MethodGet, "/"+webhookID.String()+"/deliveries", orgID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", webhookID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	handler.ListDeliveries(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	mockService.AssertExpectations(t)
}

func TestWebhookHandler_RetryDelivery(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	orgID := uuid.New()
	webhookID := uuid.New()
	deliveryID := uuid.New()

	mockService.On("RetryDelivery", mock.Anything, orgID, webhookID, deliveryID).Return(nil)

	req := createWebhookRequest(http.MethodPost, "/"+webhookID.String()+"/deliveries/"+deliveryID.String()+"/retry", orgID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", webhookID.String())
	rctx.URLParams.Add("deliveryId", deliveryID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	handler.RetryDelivery(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	mockService.AssertExpectations(t)
}

func TestWebhookHandler_Unauthorized(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	// Request without org ID in context
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestWebhookHandler_Routes(t *testing.T) {
	mockService := new(MockWebhookService)
	handler := NewWebhookHandler(mockService)

	router := handler.Routes()
	assert.NotNil(t, router)
}

// Helper function
func ptrString(s string) *string {
	return &s
}

