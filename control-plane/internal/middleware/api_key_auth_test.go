package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// mockAPIKeyService is a mock implementation of APIKeyService for testing.
type mockAPIKeyService struct {
	validateFunc func(ctx context.Context, rawKey string) (*models.APIKey, error)
}

func (m *mockAPIKeyService) Create(ctx context.Context, orgID uuid.UUID, req interface{}) (*models.APIKey, string, error) {
	return nil, "", nil
}

func (m *mockAPIKeyService) Validate(ctx context.Context, rawKey string) (*models.APIKey, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, rawKey)
	}
	return nil, nil
}

func (m *mockAPIKeyService) List(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error) {
	return nil, nil
}

func (m *mockAPIKeyService) Get(ctx context.Context, orgID, keyID uuid.UUID) (*models.APIKey, error) {
	return nil, nil
}

func (m *mockAPIKeyService) Revoke(ctx context.Context, orgID, keyID uuid.UUID) error {
	return nil
}

func (m *mockAPIKeyService) Delete(ctx context.Context, orgID, keyID uuid.UUID) error {
	return nil
}

func TestAPIKeyAuth(t *testing.T) {
	testOrgID := uuid.New()
	testKeyID := uuid.New()
	testKey := &models.APIKey{
		ID:        testKeyID,
		OrgID:     testOrgID,
		Name:      "Test Key",
		Scopes:    []string{"keys:read", "keys:write"},
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name           string
		authHeader     string
		xAPIKeyHeader  string
		validateFunc   func(ctx context.Context, rawKey string) (*models.APIKey, error)
		expectedStatus int
		expectContext  bool
	}{
		{
			name:       "valid Bearer token",
			authHeader: "Bearer bbr_live_validkey12345678",
			validateFunc: func(ctx context.Context, rawKey string) (*models.APIKey, error) {
				return testKey, nil
			},
			expectedStatus: http.StatusOK,
			expectContext:  true,
		},
		{
			name:       "valid ApiKey token",
			authHeader: "ApiKey bbr_live_validkey12345678",
			validateFunc: func(ctx context.Context, rawKey string) (*models.APIKey, error) {
				return testKey, nil
			},
			expectedStatus: http.StatusOK,
			expectContext:  true,
		},
		{
			name:          "valid X-API-Key header",
			xAPIKeyHeader: "bbr_live_validkey12345678",
			validateFunc: func(ctx context.Context, rawKey string) (*models.APIKey, error) {
				return testKey, nil
			},
			expectedStatus: http.StatusOK,
			expectContext:  true,
		},
		{
			name:           "missing authentication",
			expectedStatus: http.StatusUnauthorized,
			expectContext:  false,
		},
		{
			name:       "invalid key",
			authHeader: "Bearer bbr_live_invalidkey",
			validateFunc: func(ctx context.Context, rawKey string) (*models.APIKey, error) {
				return nil, &mockError{message: "unauthorized"}
			},
			expectedStatus: http.StatusUnauthorized,
			expectContext:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockAPIKeyService{validateFunc: tt.validateFunc}
			middleware := APIKeyAuth(mockService)

			var contextChecked bool
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contextChecked = true
				if tt.expectContext {
					apiKey := GetAPIKeyFromContext(r.Context())
					if apiKey == nil {
						t.Error("Expected API key in context")
					}
					orgID := GetOrgIDFromContext(r.Context())
					if orgID == uuid.Nil {
						t.Error("Expected org ID in context")
					}
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.xAPIKeyHeader != "" {
				req.Header.Set("X-API-Key", tt.xAPIKeyHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			if tt.expectContext && !contextChecked {
				t.Error("Handler was not called")
			}
		})
	}
}

func TestRequireAPIKeyScope(t *testing.T) {
	testOrgID := uuid.New()
	testKeyID := uuid.New()

	tests := []struct {
		name           string
		keyScopes      []string
		requiredScope  string
		expectedStatus int
	}{
		{
			name:           "has required scope",
			keyScopes:      []string{"keys:read", "keys:write"},
			requiredScope:  "keys:read",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing required scope",
			keyScopes:      []string{"keys:read"},
			requiredScope:  "keys:write",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "wildcard scope grants access",
			keyScopes:      []string{"*"},
			requiredScope:  "billing:write",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testKey := &models.APIKey{
				ID:        testKeyID,
				OrgID:     testOrgID,
				Name:      "Test Key",
				Scopes:    tt.keyScopes,
				CreatedAt: time.Now(),
			}

			middleware := RequireAPIKeyScope(tt.requiredScope)
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			// Add API key to context
			ctx := context.WithValue(req.Context(), APIKeyContextKey, testKey)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRequireAPIKeyScopes(t *testing.T) {
	testOrgID := uuid.New()
	testKeyID := uuid.New()

	tests := []struct {
		name           string
		keyScopes      []string
		requiredScopes []string
		expectedStatus int
	}{
		{
			name:           "has one of required scopes",
			keyScopes:      []string{"keys:read"},
			requiredScopes: []string{"keys:read", "keys:write"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing all required scopes",
			keyScopes:      []string{"audit:read"},
			requiredScopes: []string{"keys:read", "keys:write"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "wildcard grants access",
			keyScopes:      []string{"*"},
			requiredScopes: []string{"keys:read", "billing:write"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testKey := &models.APIKey{
				ID:        testKeyID,
				OrgID:     testOrgID,
				Name:      "Test Key",
				Scopes:    tt.keyScopes,
				CreatedAt: time.Now(),
			}

			middleware := RequireAPIKeyScopes(tt.requiredScopes...)
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := context.WithValue(req.Context(), APIKeyContextKey, testKey)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

func TestGetAPIKeyFromContext(t *testing.T) {
	t.Run("returns key when present", func(t *testing.T) {
		testKey := &models.APIKey{
			ID:    uuid.New(),
			OrgID: uuid.New(),
			Name:  "Test Key",
		}

		ctx := context.WithValue(context.Background(), APIKeyContextKey, testKey)
		result := GetAPIKeyFromContext(ctx)

		if result == nil {
			t.Error("Expected API key, got nil")
		}
		if result.ID != testKey.ID {
			t.Errorf("ID = %v, want %v", result.ID, testKey.ID)
		}
	})

	t.Run("returns nil when not present", func(t *testing.T) {
		ctx := context.Background()
		result := GetAPIKeyFromContext(ctx)

		if result != nil {
			t.Errorf("Expected nil, got %v", result)
		}
	})

	t.Run("returns nil when wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), APIKeyContextKey, "not an api key")
		result := GetAPIKeyFromContext(ctx)

		if result != nil {
			t.Errorf("Expected nil, got %v", result)
		}
	})
}

func TestGetOrgIDFromContext(t *testing.T) {
	t.Run("returns UUID when string present", func(t *testing.T) {
		testID := uuid.New()
		ctx := context.WithValue(context.Background(), OrgIDKey, testID.String())
		result := GetOrgIDFromContext(ctx)

		if result != testID {
			t.Errorf("OrgID = %v, want %v", result, testID)
		}
	})

	t.Run("returns UUID when UUID present", func(t *testing.T) {
		testID := uuid.New()
		ctx := context.WithValue(context.Background(), OrgIDKey, testID)
		result := GetOrgIDFromContext(ctx)

		if result != testID {
			t.Errorf("OrgID = %v, want %v", result, testID)
		}
	})

	t.Run("returns Nil when not present", func(t *testing.T) {
		ctx := context.Background()
		result := GetOrgIDFromContext(ctx)

		if result != uuid.Nil {
			t.Errorf("Expected uuid.Nil, got %v", result)
		}
	})
}

func TestExtractAPIKey(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		xAPIKeyHeader string
		expected      string
	}{
		{
			name:       "Bearer token",
			authHeader: "Bearer test_key_123",
			expected:   "test_key_123",
		},
		{
			name:       "ApiKey header",
			authHeader: "ApiKey test_key_456",
			expected:   "test_key_456",
		},
		{
			name:          "X-API-Key header",
			xAPIKeyHeader: "test_key_789",
			expected:      "test_key_789",
		},
		{
			name:       "Bearer takes precedence over X-API-Key",
			authHeader: "Bearer bearer_key",
			xAPIKeyHeader: "xapi_key",
			expected:   "bearer_key",
		},
		{
			name:       "unsupported auth scheme",
			authHeader: "Basic dXNlcjpwYXNz",
			expected:   "",
		},
		{
			name:     "no auth headers",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.xAPIKeyHeader != "" {
				req.Header.Set("X-API-Key", tt.xAPIKeyHeader)
			}

			result := extractAPIKey(req)
			if result != tt.expected {
				t.Errorf("extractAPIKey() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestOptionalAPIKeyAuth(t *testing.T) {
	testOrgID := uuid.New()
	testKey := &models.APIKey{
		ID:        uuid.New(),
		OrgID:     testOrgID,
		Name:      "Test Key",
		Scopes:    []string{"keys:read"},
		CreatedAt: time.Now(),
	}

	t.Run("sets context with valid key", func(t *testing.T) {
		mockService := &mockAPIKeyService{
			validateFunc: func(ctx context.Context, rawKey string) (*models.APIKey, error) {
				return testKey, nil
			},
		}

		middleware := OptionalAPIKeyAuth(mockService)
		var contextSet bool
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contextSet = GetAPIKeyFromContext(r.Context()) != nil
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer valid_key")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
		}
		if !contextSet {
			t.Error("Expected context to be set")
		}
	})

	t.Run("continues without context when no key", func(t *testing.T) {
		mockService := &mockAPIKeyService{}
		middleware := OptionalAPIKeyAuth(mockService)

		var handlerCalled bool
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			if GetAPIKeyFromContext(r.Context()) != nil {
				t.Error("Expected no API key in context")
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if !handlerCalled {
			t.Error("Handler was not called")
		}
	})

	t.Run("continues without context when invalid key", func(t *testing.T) {
		mockService := &mockAPIKeyService{
			validateFunc: func(ctx context.Context, rawKey string) (*models.APIKey, error) {
				return nil, &mockError{message: "invalid"}
			},
		}

		middleware := OptionalAPIKeyAuth(mockService)
		var handlerCalled bool
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid_key")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if !handlerCalled {
			t.Error("Handler was not called")
		}
	})
}

// mockError is a simple error implementation for testing.
type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}

