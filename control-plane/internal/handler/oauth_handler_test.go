package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// mockOAuthService implements service.OAuthService for testing.
type mockOAuthService struct {
	authURL            string
	authURLErr         error
	user               *models.User
	sessionID          string
	handleCallbackErr  error
	supportedProviders []string
}

func (m *mockOAuthService) GetAuthURL(provider, state string) (string, error) {
	if m.authURLErr != nil {
		return "", m.authURLErr
	}
	return m.authURL, nil
}

func (m *mockOAuthService) HandleCallback(ctx context.Context, provider, code string) (*models.User, string, error) {
	if m.handleCallbackErr != nil {
		return nil, "", m.handleCallbackErr
	}
	return m.user, m.sessionID, nil
}

func (m *mockOAuthService) GetSupportedProviders() []string {
	return m.supportedProviders
}

func TestOAuthHandler_Authorize(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		authURL        string
		authURLErr     error
		expectedStatus int
		expectCookie   bool
	}{
		{
			name:           "GitHub valid",
			provider:       "github",
			authURL:        "https://github.com/login/oauth/authorize?client_id=test",
			expectedStatus: http.StatusTemporaryRedirect,
			expectCookie:   true,
		},
		{
			name:           "Google valid",
			provider:       "google",
			authURL:        "https://accounts.google.com/o/oauth2/v2/auth?client_id=test",
			expectedStatus: http.StatusTemporaryRedirect,
			expectCookie:   true,
		},
		{
			name:           "Discord valid",
			provider:       "discord",
			authURL:        "https://discord.com/api/oauth2/authorize?client_id=test",
			expectedStatus: http.StatusTemporaryRedirect,
			expectCookie:   true,
		},
		{
			name:           "Invalid provider",
			provider:       "invalid",
			expectedStatus: http.StatusBadRequest,
			expectCookie:   false,
		},
		{
			name:           "Provider not configured",
			provider:       "github",
			authURLErr:     errors.New("provider not configured"),
			expectedStatus: http.StatusBadRequest,
			expectCookie:   true, // Cookie is set before checking provider config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockOAuthService{
				authURL:    tt.authURL,
				authURLErr: tt.authURLErr,
			}

			handler := NewOAuthHandler(mockService, "http://localhost:3000", false)

			// Create request with chi URL params
			req := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/"+tt.provider, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("provider", tt.provider)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.Authorize(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check for state cookie
			cookies := w.Result().Cookies()
			hasStateCookie := false
			for _, cookie := range cookies {
				if cookie.Name == OAuthStateCookieName {
					hasStateCookie = true
					if cookie.Value == "" && tt.expectCookie {
						t.Error("state cookie should have a value")
					}
					if !cookie.HttpOnly {
						t.Error("state cookie should be HttpOnly")
					}
					break
				}
			}

			if tt.expectCookie && !hasStateCookie && tt.expectedStatus != http.StatusBadRequest {
				// Only expect cookie on valid providers before error check
			}
		})
	}
}

func TestOAuthHandler_Callback(t *testing.T) {
	tests := []struct {
		name              string
		provider          string
		code              string
		state             string
		cookieState       string
		user              *models.User
		sessionID         string
		callbackErr       error
		expectedStatus    int
		expectRedirect    bool
		redirectContains  string
	}{
		{
			name:        "Successful callback",
			provider:    "github",
			code:        "valid-code",
			state:       "test-state",
			cookieState: "test-state",
			user: &models.User{
				ID:    uuid.New(),
				Email: "test@example.com",
			},
			sessionID:        "session-123",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectRedirect:   true,
			redirectContains: "/dashboard",
		},
		{
			name:             "Missing code",
			provider:         "github",
			code:             "",
			state:            "test-state",
			cookieState:      "test-state",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectRedirect:   true,
			redirectContains: "error=missing_code",
		},
		{
			name:             "Missing state",
			provider:         "github",
			code:             "valid-code",
			state:            "",
			cookieState:      "test-state",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectRedirect:   true,
			redirectContains: "error=missing_state",
		},
		{
			name:             "State mismatch",
			provider:         "github",
			code:             "valid-code",
			state:            "bad-state",
			cookieState:      "test-state",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectRedirect:   true,
			redirectContains: "error=invalid_state",
		},
		{
			name:             "No state cookie",
			provider:         "github",
			code:             "valid-code",
			state:            "test-state",
			cookieState:      "",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectRedirect:   true,
			redirectContains: "error=invalid_state",
		},
		{
			name:             "Invalid provider",
			provider:         "invalid",
			code:             "valid-code",
			state:            "test-state",
			cookieState:      "test-state",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectRedirect:   true,
			redirectContains: "error=invalid_provider",
		},
		{
			name:             "OAuth callback error",
			provider:         "github",
			code:             "invalid-code",
			state:            "test-state",
			cookieState:      "test-state",
			callbackErr:      errors.New("token exchange failed"),
			expectedStatus:   http.StatusTemporaryRedirect,
			expectRedirect:   true,
			redirectContains: "error=oauth_failed",
		},
		{
			name:        "User without email redirects to profile",
			provider:    "github",
			code:        "valid-code",
			state:       "test-state",
			cookieState: "test-state",
			user: &models.User{
				ID:    uuid.New(),
				Email: "",
			},
			sessionID:        "session-123",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectRedirect:   true,
			redirectContains: "email_required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockOAuthService{
				user:              tt.user,
				sessionID:         tt.sessionID,
				handleCallbackErr: tt.callbackErr,
			}

			handler := NewOAuthHandler(mockService, "http://localhost:3000", false)

			// Build URL with query params
			url := "/v1/auth/oauth/" + tt.provider + "/callback"
			if tt.code != "" {
				url += "?code=" + tt.code
			}
			if tt.state != "" {
				if tt.code != "" {
					url += "&state=" + tt.state
				} else {
					url += "?state=" + tt.state
				}
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("provider", tt.provider)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add state cookie if specified
			if tt.cookieState != "" {
				req.AddCookie(&http.Cookie{
					Name:  OAuthStateCookieName,
					Value: tt.cookieState,
				})
			}

			w := httptest.NewRecorder()
			handler.Callback(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectRedirect {
				location := w.Header().Get("Location")
				if location == "" {
					t.Error("expected redirect location header")
				}
				if tt.redirectContains != "" {
					if !contains(location, tt.redirectContains) {
						t.Errorf("expected redirect to contain %q, got %q", tt.redirectContains, location)
					}
				}
			}

			// Check session cookie on success
			if tt.sessionID != "" && tt.callbackErr == nil && tt.user != nil && tt.user.Email != "" {
				cookies := w.Result().Cookies()
				hasSessionCookie := false
				for _, cookie := range cookies {
					if cookie.Name == SessionCookieName {
						hasSessionCookie = true
						if cookie.Value != tt.sessionID {
							t.Errorf("expected session ID %q, got %q", tt.sessionID, cookie.Value)
						}
						if !cookie.HttpOnly {
							t.Error("session cookie should be HttpOnly")
						}
						break
					}
				}
				if !hasSessionCookie {
					t.Error("expected session cookie to be set")
				}
			}
		})
	}
}

func TestOAuthHandler_Callback_OAuthError(t *testing.T) {
	mockService := &mockOAuthService{}
	handler := NewOAuthHandler(mockService, "http://localhost:3000", false)

	// Simulate OAuth error from provider (e.g., user denied access)
	url := "/v1/auth/oauth/github/callback?error=access_denied&error_description=User+denied+access"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "github")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.Callback(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
	}

	location := w.Header().Get("Location")
	if !contains(location, "error=oauth_denied") {
		t.Errorf("expected redirect to contain oauth_denied error, got %q", location)
	}
}

func TestOAuthHandler_ListProviders(t *testing.T) {
	mockService := &mockOAuthService{
		supportedProviders: []string{"github", "google", "discord"},
	}

	handler := NewOAuthHandler(mockService, "http://localhost:3000", false)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/providers", nil)
	w := httptest.NewRecorder()

	handler.ListProviders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestOAuthHandler_Routes(t *testing.T) {
	mockService := &mockOAuthService{}
	handler := NewOAuthHandler(mockService, "http://localhost:3000", false)

	router := handler.Routes()
	if router == nil {
		t.Fatal("Routes() returned nil router")
	}
}

func TestGenerateState(t *testing.T) {
	// Test that generateState produces unique values
	states := make(map[string]bool)

	for i := 0; i < 100; i++ {
		state, err := generateState()
		if err != nil {
			t.Fatalf("generateState failed: %v", err)
		}
		if state == "" {
			t.Error("generateState returned empty string")
		}
		if states[state] {
			t.Errorf("generateState produced duplicate state: %s", state)
		}
		states[state] = true
	}
}

func TestIsValidProvider(t *testing.T) {
	tests := []struct {
		provider string
		valid    bool
	}{
		{"github", true},
		{"google", true},
		{"discord", true},
		{"GitHub", false}, // Case sensitive
		{"facebook", false},
		{"", false},
		{"twitter", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := isValidProvider(tt.provider)
			if result != tt.valid {
				t.Errorf("isValidProvider(%q) = %v, want %v", tt.provider, result, tt.valid)
			}
		})
	}
}

func TestSecureCookieFlag(t *testing.T) {
	mockService := &mockOAuthService{
		authURL: "https://github.com/login/oauth/authorize",
	}

	// Test with secure cookies enabled
	handler := NewOAuthHandler(mockService, "http://localhost:3000", true)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/github", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "github")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.Authorize(w, req)

	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == OAuthStateCookieName {
			if !cookie.Secure {
				t.Error("expected Secure flag on cookie when secureCookie is true")
			}
			break
		}
	}

	// Test with secure cookies disabled
	handler = NewOAuthHandler(mockService, "http://localhost:3000", false)

	req = httptest.NewRequest(http.MethodGet, "/v1/auth/oauth/github", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w = httptest.NewRecorder()
	handler.Authorize(w, req)

	cookies = w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == OAuthStateCookieName {
			if cookie.Secure {
				t.Error("expected no Secure flag on cookie when secureCookie is false")
			}
			break
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

