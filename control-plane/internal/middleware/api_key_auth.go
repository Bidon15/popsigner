package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	apierrors "github.com/Bidon15/popsigner/control-plane/internal/pkg/errors"
	"github.com/Bidon15/popsigner/control-plane/internal/pkg/response"
	"github.com/Bidon15/popsigner/control-plane/internal/service"
)

// Context keys for API key authentication.
const (
	// APIKeyContextKey is the context key for the authenticated API key.
	APIKeyContextKey contextKey = "api_key"
	// ScopesContextKey is the context key for the API key scopes.
	ScopesContextKey contextKey = "scopes"
)

// APIKeyAuth returns a middleware that authenticates requests using API keys.
// It supports both "Bearer <key>" and "ApiKey <key>" authorization headers.
func APIKeyAuth(apiKeyService service.APIKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from header
			rawKey := extractAPIKey(r)
			if rawKey == "" {
				response.Error(w, apierrors.ErrUnauthorized)
				return
			}

			// Validate API key
			apiKey, err := apiKeyService.Validate(r.Context(), rawKey)
			if err != nil {
				response.Error(w, apierrors.ErrUnauthorized)
				return
			}

			// Add API key info to context
			ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
			ctx = context.WithValue(ctx, OrgIDKey, apiKey.OrgID.String())
			ctx = context.WithValue(ctx, APIKeyIDKey, apiKey.ID.String())
			ctx = context.WithValue(ctx, ScopesContextKey, apiKey.Scopes)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractAPIKey extracts the API key from the request.
// Supports Authorization header with "Bearer" or "ApiKey" scheme,
// and X-API-Key header.
func extractAPIKey(r *http.Request) string {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" {
		// Support "Bearer <key>" format
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		// Support "ApiKey <key>" format
		if strings.HasPrefix(auth, "ApiKey ") {
			return strings.TrimPrefix(auth, "ApiKey ")
		}
	}

	// Check X-API-Key header as fallback
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	return ""
}

// RequireAPIKeyScope returns a middleware that checks for a required scope.
// Must be used after APIKeyAuth middleware.
func RequireAPIKeyScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := GetAPIKeyFromContext(r.Context())
			if apiKey == nil {
				response.Error(w, apierrors.ErrUnauthorized)
				return
			}

			if !apiKey.HasScope(scope) {
				response.Error(w, apierrors.ErrForbidden.WithMessage(
					"API key does not have required scope: "+scope,
				))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAPIKeyScopes returns a middleware that checks for any of the required scopes.
// Must be used after APIKeyAuth middleware.
func RequireAPIKeyScopes(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := GetAPIKeyFromContext(r.Context())
			if apiKey == nil {
				response.Error(w, apierrors.ErrUnauthorized)
				return
			}

			if !apiKey.HasAnyScope(scopes...) {
				response.Error(w, apierrors.ErrForbidden.WithMessage(
					"API key does not have any of the required scopes",
				))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAPIKeyFromContext retrieves the authenticated API key from the context.
func GetAPIKeyFromContext(ctx context.Context) *models.APIKey {
	if v := ctx.Value(APIKeyContextKey); v != nil {
		if apiKey, ok := v.(*models.APIKey); ok {
			return apiKey
		}
	}
	return nil
}

// GetOrgIDFromContext retrieves the organization ID from the context.
// Returns the org ID as a UUID, or uuid.Nil if not present.
func GetOrgIDFromContext(ctx context.Context) uuid.UUID {
	if v := ctx.Value(OrgIDKey); v != nil {
		if orgIDStr, ok := v.(string); ok {
			if id, err := uuid.Parse(orgIDStr); err == nil {
				return id
			}
		}
		// Also support direct UUID storage
		if orgID, ok := v.(uuid.UUID); ok {
			return orgID
		}
	}
	return uuid.Nil
}

// GetAPIKeyIDFromContext retrieves the API key ID from the context.
func GetAPIKeyIDFromContext(ctx context.Context) string {
	if v := ctx.Value(APIKeyIDKey); v != nil {
		if keyID, ok := v.(string); ok {
			return keyID
		}
	}
	return ""
}

// GetScopesFromContext retrieves the scopes from the context.
func GetScopesFromContext(ctx context.Context) []string {
	if v := ctx.Value(ScopesContextKey); v != nil {
		if scopes, ok := v.([]string); ok {
			return scopes
		}
	}
	return nil
}

// OptionalAPIKeyAuth returns a middleware that attempts API key authentication
// but doesn't require it. If authentication fails, the request continues
// without authentication context.
func OptionalAPIKeyAuth(apiKeyService service.APIKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to extract API key
			rawKey := extractAPIKey(r)
			if rawKey == "" {
				// No key provided, continue without auth
				next.ServeHTTP(w, r)
				return
			}

			// Try to validate API key
			apiKey, err := apiKeyService.Validate(r.Context(), rawKey)
			if err != nil {
				// Invalid key, continue without auth
				next.ServeHTTP(w, r)
				return
			}

			// Add API key info to context
			ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
			ctx = context.WithValue(ctx, OrgIDKey, apiKey.OrgID.String())
			ctx = context.WithValue(ctx, APIKeyIDKey, apiKey.ID.String())
			ctx = context.WithValue(ctx, ScopesContextKey, apiKey.Scopes)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

