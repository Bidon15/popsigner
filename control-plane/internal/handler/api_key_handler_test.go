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

	"github.com/Bidon15/popsigner/control-plane/internal/middleware"
	"github.com/Bidon15/popsigner/control-plane/internal/models"
	apierrors "github.com/Bidon15/popsigner/control-plane/internal/pkg/errors"
	"github.com/Bidon15/popsigner/control-plane/internal/service"
)

// mockAPIKeyService is a mock implementation of APIKeyService for testing.
type mockAPIKeyService struct {
	createFunc func(ctx context.Context, orgID uuid.UUID, req service.CreateAPIKeyRequest) (*models.APIKey, string, error)
	validateFunc func(ctx context.Context, rawKey string) (*models.APIKey, error)
	listFunc     func(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error)
	getFunc      func(ctx context.Context, orgID, keyID uuid.UUID) (*models.APIKey, error)
	revokeFunc   func(ctx context.Context, orgID, keyID uuid.UUID) error
	deleteFunc   func(ctx context.Context, orgID, keyID uuid.UUID) error
}

func (m *mockAPIKeyService) Create(ctx context.Context, orgID uuid.UUID, req service.CreateAPIKeyRequest) (*models.APIKey, string, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, orgID, req)
	}
	return nil, "", nil
}

func (m *mockAPIKeyService) Validate(ctx context.Context, rawKey string) (*models.APIKey, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, rawKey)
	}
	return nil, nil
}

func (m *mockAPIKeyService) List(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, orgID)
	}
	return nil, nil
}

func (m *mockAPIKeyService) Get(ctx context.Context, orgID, keyID uuid.UUID) (*models.APIKey, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, orgID, keyID)
	}
	return nil, nil
}

func (m *mockAPIKeyService) Revoke(ctx context.Context, orgID, keyID uuid.UUID) error {
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, orgID, keyID)
	}
	return nil
}

func (m *mockAPIKeyService) Delete(ctx context.Context, orgID, keyID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, orgID, keyID)
	}
	return nil
}

// createTestRequest creates a request with org ID in context
func createTestRequest(t *testing.T, method, path string, body interface{}, orgID uuid.UUID) *http.Request {
	t.Helper()

	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Add org ID to context
	ctx := context.WithValue(req.Context(), middleware.OrgIDKey, orgID.String())
	return req.WithContext(ctx)
}

func TestAPIKeyHandler_Create(t *testing.T) {
	orgID := uuid.New()
	keyID := uuid.New()

	tests := []struct {
		name           string
		body           interface{}
		mockService    *mockAPIKeyService
		expectedStatus int
		checkResponse  func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "creates key successfully",
			body: CreateAPIKeyRequest{
				Name:   "Test Key",
				Scopes: []string{"keys:read", "keys:write"},
			},
			mockService: &mockAPIKeyService{
				createFunc: func(ctx context.Context, oID uuid.UUID, req service.CreateAPIKeyRequest) (*models.APIKey, string, error) {
					return &models.APIKey{
						ID:        keyID,
						OrgID:     oID,
						Name:      req.Name,
						KeyPrefix: "bbr_live_abcd1234",
						Scopes:    req.Scopes,
						CreatedAt: time.Now(),
					}, "bbr_live_abcd1234567890", nil
				},
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp struct {
					Data models.CreateAPIKeyResponse `json:"data"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp.Data.Key == "" {
					t.Error("Expected key in response")
				}
				if resp.Data.Warning == "" {
					t.Error("Expected warning in response")
				}
			},
		},
		{
			name: "rejects missing name",
			body: CreateAPIKeyRequest{
				Scopes: []string{"keys:read"},
			},
			mockService:    &mockAPIKeyService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "rejects empty scopes",
			body: CreateAPIKeyRequest{
				Name:   "Test Key",
				Scopes: []string{},
			},
			mockService:    &mockAPIKeyService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "rejects invalid scopes",
			body: CreateAPIKeyRequest{
				Name:   "Test Key",
				Scopes: []string{"invalid:scope"},
			},
			mockService:    &mockAPIKeyService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "rejects invalid JSON",
			body:           "not json",
			mockService:    &mockAPIKeyService{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIKeyHandler(tt.mockService)

			var reqBody []byte
			if str, ok := tt.body.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/api-keys", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.OrgIDKey, orgID.String())
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.Create(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d. Body: %s", rec.Code, tt.expectedStatus, rec.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestAPIKeyHandler_List(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name           string
		mockService    *mockAPIKeyService
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "lists keys successfully",
			mockService: &mockAPIKeyService{
				listFunc: func(ctx context.Context, oID uuid.UUID) ([]*models.APIKey, error) {
					return []*models.APIKey{
						{
							ID:        uuid.New(),
							OrgID:     oID,
							Name:      "Key 1",
							KeyPrefix: "bbr_live_key1",
							Scopes:    []string{"keys:read"},
							CreatedAt: time.Now(),
						},
						{
							ID:        uuid.New(),
							OrgID:     oID,
							Name:      "Key 2",
							KeyPrefix: "bbr_live_key2",
							Scopes:    []string{"keys:write"},
							CreatedAt: time.Now(),
						},
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "returns empty list",
			mockService: &mockAPIKeyService{
				listFunc: func(ctx context.Context, oID uuid.UUID) ([]*models.APIKey, error) {
					return []*models.APIKey{}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name: "handles service error",
			mockService: &mockAPIKeyService{
				listFunc: func(ctx context.Context, oID uuid.UUID) ([]*models.APIKey, error) {
					return nil, apierrors.ErrInternal
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIKeyHandler(tt.mockService)

			req := createTestRequest(t, http.MethodGet, "/v1/api-keys", nil, orgID)
			rec := httptest.NewRecorder()
			handler.List(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp struct {
					Data []*models.APIKeyResponse `json:"data"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != tt.expectedCount {
					t.Errorf("Response count = %d, want %d", len(resp.Data), tt.expectedCount)
				}
			}
		})
	}
}

func TestAPIKeyHandler_Get(t *testing.T) {
	orgID := uuid.New()
	keyID := uuid.New()

	tests := []struct {
		name           string
		keyIDParam     string
		mockService    *mockAPIKeyService
		expectedStatus int
	}{
		{
			name:       "gets key successfully",
			keyIDParam: keyID.String(),
			mockService: &mockAPIKeyService{
				getFunc: func(ctx context.Context, oID, kID uuid.UUID) (*models.APIKey, error) {
					return &models.APIKey{
						ID:        kID,
						OrgID:     oID,
						Name:      "Test Key",
						KeyPrefix: "bbr_live_test",
						Scopes:    []string{"keys:read"},
						CreatedAt: time.Now(),
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "returns 404 for nonexistent key",
			keyIDParam: uuid.New().String(),
			mockService: &mockAPIKeyService{
				getFunc: func(ctx context.Context, oID, kID uuid.UUID) (*models.APIKey, error) {
					return nil, apierrors.NewNotFoundError("API key")
				},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "rejects invalid UUID",
			keyIDParam:     "not-a-uuid",
			mockService:    &mockAPIKeyService{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIKeyHandler(tt.mockService)

			req := createTestRequest(t, http.MethodGet, "/v1/api-keys/"+tt.keyIDParam, nil, orgID)

			// Use chi router to get URL params
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.keyIDParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Get(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

func TestAPIKeyHandler_Delete(t *testing.T) {
	orgID := uuid.New()
	keyID := uuid.New()

	tests := []struct {
		name           string
		keyIDParam     string
		mockService    *mockAPIKeyService
		expectedStatus int
	}{
		{
			name:       "deletes key successfully",
			keyIDParam: keyID.String(),
			mockService: &mockAPIKeyService{
				deleteFunc: func(ctx context.Context, oID, kID uuid.UUID) error {
					return nil
				},
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "returns 404 for nonexistent key",
			keyIDParam: uuid.New().String(),
			mockService: &mockAPIKeyService{
				deleteFunc: func(ctx context.Context, oID, kID uuid.UUID) error {
					return apierrors.NewNotFoundError("API key")
				},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "rejects invalid UUID",
			keyIDParam:     "not-a-uuid",
			mockService:    &mockAPIKeyService{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIKeyHandler(tt.mockService)

			req := createTestRequest(t, http.MethodDelete, "/v1/api-keys/"+tt.keyIDParam, nil, orgID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.keyIDParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Delete(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

func TestAPIKeyHandler_Revoke(t *testing.T) {
	orgID := uuid.New()
	keyID := uuid.New()

	tests := []struct {
		name           string
		keyIDParam     string
		mockService    *mockAPIKeyService
		expectedStatus int
	}{
		{
			name:       "revokes key successfully",
			keyIDParam: keyID.String(),
			mockService: &mockAPIKeyService{
				revokeFunc: func(ctx context.Context, oID, kID uuid.UUID) error {
					return nil
				},
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "returns 404 for nonexistent key",
			keyIDParam: uuid.New().String(),
			mockService: &mockAPIKeyService{
				revokeFunc: func(ctx context.Context, oID, kID uuid.UUID) error {
					return apierrors.NewNotFoundError("API key")
				},
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "returns 409 for already revoked key",
			keyIDParam: keyID.String(),
			mockService: &mockAPIKeyService{
				revokeFunc: func(ctx context.Context, oID, kID uuid.UUID) error {
					return apierrors.NewConflictError("API key is already revoked")
				},
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "rejects invalid UUID",
			keyIDParam:     "not-a-uuid",
			mockService:    &mockAPIKeyService{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIKeyHandler(tt.mockService)

			req := createTestRequest(t, http.MethodPost, "/v1/api-keys/"+tt.keyIDParam+"/revoke", nil, orgID)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.keyIDParam)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			handler.Revoke(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

func TestAPIKeyHandler_Routes(t *testing.T) {
	mockService := &mockAPIKeyService{}
	handler := NewAPIKeyHandler(mockService)
	router := handler.Routes()

	if router == nil {
		t.Error("Routes() returned nil router")
	}
}

func TestAPIKeyHandler_Unauthorized(t *testing.T) {
	handler := NewAPIKeyHandler(&mockAPIKeyService{})

	// Request without org ID in context
	req := httptest.NewRequest(http.MethodGet, "/v1/api-keys", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

