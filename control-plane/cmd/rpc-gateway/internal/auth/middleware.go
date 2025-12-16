package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Bidon15/popsigner/control-plane/internal/middleware"
	"github.com/Bidon15/popsigner/control-plane/internal/models"
	"github.com/Bidon15/popsigner/control-plane/internal/repository"
)

// APIKeyValidator is an interface for validating API keys.
// This avoids importing the service package directly which would cause an import cycle.
type APIKeyValidator interface {
	Validate(ctx context.Context, rawKey string) (*models.APIKey, error)
}

// Context keys for authentication.
// Note: We use the same context key type as the middleware package for compatibility.
type contextKey string

const (
	// AuthMethodKey is the context key for authentication method.
	AuthMethodKey contextKey = "auth_method"
	// CertFingerprintKey is the context key for certificate fingerprint (mTLS).
	CertFingerprintKey contextKey = "cert_fingerprint"
)

// Note: We use middleware.OrgIDKey and middleware.APIKeyIDKey for org_id and api_key_id
// to maintain compatibility with the jsonrpc handlers.

// DualAuthMiddleware provides authentication via API key or mTLS.
type DualAuthMiddleware struct {
	apiKeyValidator APIKeyValidator
	mtlsAuth        *MTLSAuthenticator
	logger          *slog.Logger
}

// NewDualAuthMiddleware creates a new dual auth middleware.
func NewDualAuthMiddleware(
	apiKeyValidator APIKeyValidator,
	certRepo repository.CertificateRepository,
	logger *slog.Logger,
) *DualAuthMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &DualAuthMiddleware{
		apiKeyValidator: apiKeyValidator,
		mtlsAuth:        NewMTLSAuthenticator(certRepo),
		logger:          logger,
	}
}

// Handler returns the middleware handler.
func (m *DualAuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var result *AuthResult
		var err error

		// Try API Key first (OP Stack)
		apiKey := extractAPIKey(r)
		if apiKey != "" {
			result, err = m.authenticateAPIKey(ctx, apiKey)
			if err != nil {
				m.logger.Warn("API key auth failed",
					slog.String("method", "api_key"),
					slog.String("error", err.Error()),
				)
			}
		}

		// If API key didn't work, try mTLS (Arbitrum Nitro)
		if result == nil && r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			result, err = m.mtlsAuth.Authenticate(ctx, r)
			if err != nil {
				m.logger.Warn("mTLS auth failed",
					slog.String("method", "mtls"),
					slog.String("error", err.Error()),
				)
			}
		}

		// Neither method succeeded
		if result == nil {
			jsonRPCError(w, -32001, "Unauthorized: valid API key or client certificate required")
			return
		}

		// Add auth info to context
		// Use middleware.OrgIDKey for compatibility with jsonrpc handlers
		ctx = context.WithValue(ctx, middleware.OrgIDKey, result.OrgID)
		ctx = context.WithValue(ctx, AuthMethodKey, result.Method)

		// Log successful auth
		m.logger.Debug("Request authenticated",
			slog.String("org_id", result.OrgID),
			slog.String("method", result.Method),
			slog.String("identifier", result.Identifier),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticateAPIKey validates an API key and returns auth result.
func (m *DualAuthMiddleware) authenticateAPIKey(ctx context.Context, apiKey string) (*AuthResult, error) {
	// Validate API key
	key, err := m.apiKeyValidator.Validate(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	// Check for EVM signing scope
	if !key.HasScope("keys:sign:evm") && !key.HasScope("keys:sign") {
		return nil, fmt.Errorf("API key missing required scope: keys:sign:evm or keys:sign")
	}

	// Truncate key for safe logging
	identifier := apiKey
	if len(identifier) > 12 {
		identifier = identifier[:12] + "..."
	}

	return &AuthResult{
		OrgID:      key.OrgID.String(),
		Method:     "api_key",
		Identifier: identifier,
	}, nil
}

// extractAPIKey extracts the API key from the request.
// Supports Authorization header with "Bearer" scheme and X-API-Key header.
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

// GetOrgID extracts the organization ID from the request context.
func GetOrgID(ctx context.Context) string {
	orgID, _ := ctx.Value(middleware.OrgIDKey).(string)
	return orgID
}

// GetAuthMethod extracts the authentication method from the request context.
func GetAuthMethod(ctx context.Context) string {
	method, _ := ctx.Value(AuthMethodKey).(string)
	return method
}

// jsonRPCError writes a JSON-RPC error response.
func jsonRPCError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, `{"jsonrpc":"2.0","error":{"code":%d,"message":"%s"},"id":null}`, code, message)
}

// MTLSOnlyMiddleware creates a middleware that only accepts mTLS authentication.
// Use this for endpoints that must use client certificates (e.g., Arbitrum Nitro).
func MTLSOnlyMiddleware(certRepo repository.CertificateRepository, logger *slog.Logger) func(http.Handler) http.Handler {
	auth := NewMTLSAuthenticator(certRepo)
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result, err := auth.Authenticate(r.Context(), r)
			if err != nil {
				logger.Warn("mTLS auth failed",
					slog.String("error", err.Error()),
				)
				jsonRPCError(w, -32001, "Unauthorized: valid client certificate required")
				return
			}

			// Use middleware.OrgIDKey for compatibility with jsonrpc handlers
			ctx := context.WithValue(r.Context(), middleware.OrgIDKey, result.OrgID)
			ctx = context.WithValue(ctx, AuthMethodKey, result.Method)

			logger.Debug("Request authenticated via mTLS",
				slog.String("org_id", result.OrgID),
				slog.String("identifier", result.Identifier),
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyOnlyMiddleware creates a middleware that only accepts API key authentication.
// Use this for endpoints that must use API keys (e.g., OP Stack).
func APIKeyOnlyMiddleware(apiKeyValidator APIKeyValidator, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := extractAPIKey(r)
			if apiKey == "" {
				jsonRPCError(w, -32001, "Unauthorized: API key required")
				return
			}

			key, err := apiKeyValidator.Validate(r.Context(), apiKey)
			if err != nil {
				logger.Warn("API key auth failed",
					slog.String("error", err.Error()),
				)
				jsonRPCError(w, -32001, "Unauthorized: invalid API key")
				return
			}

			// Check for EVM signing scope
			if !key.HasScope("keys:sign:evm") && !key.HasScope("keys:sign") {
				jsonRPCError(w, -32003, "Forbidden: API key missing required scope")
				return
			}

			identifier := apiKey
			if len(identifier) > 12 {
				identifier = identifier[:12] + "..."
			}

			// Use middleware.OrgIDKey and middleware.APIKeyIDKey for compatibility
			ctx := context.WithValue(r.Context(), middleware.OrgIDKey, key.OrgID.String())
			ctx = context.WithValue(ctx, AuthMethodKey, "api_key")
			ctx = context.WithValue(ctx, middleware.APIKeyIDKey, key.ID.String())

			logger.Debug("Request authenticated via API key",
				slog.String("org_id", key.OrgID.String()),
				slog.String("identifier", identifier),
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

