// Package handler provides HTTP request handlers for the API.
package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/go-chi/chi/v5"

	apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
	"github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
	"github.com/Bidon15/banhbaoring/control-plane/internal/service"
)

const (
	// OAuthStateCookieName is the name of the cookie used for CSRF state.
	OAuthStateCookieName = "oauth_state"

	// SessionCookieName is the name of the session cookie.
	SessionCookieName = "session"

	// StateExpirySeconds is how long the OAuth state cookie is valid.
	StateExpirySeconds = 300 // 5 minutes

	// SessionExpirySeconds is the default session cookie lifetime (7 days).
	SessionExpirySeconds = 7 * 24 * 60 * 60
)

// OAuthHandler handles OAuth authentication requests.
type OAuthHandler struct {
	oauthService service.OAuthService
	dashboardURL string
	secureCookie bool
}

// NewOAuthHandler creates a new OAuthHandler.
func NewOAuthHandler(oauthService service.OAuthService, dashboardURL string, secureCookie bool) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		dashboardURL: dashboardURL,
		secureCookie: secureCookie,
	}
}

// Routes returns the OAuth router.
func (h *OAuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// OAuth flow endpoints
	r.Get("/{provider}", h.Authorize)
	r.Get("/{provider}/callback", h.Callback)

	// List available providers
	r.Get("/providers", h.ListProviders)

	return r
}

// Authorize initiates the OAuth flow for a provider.
// @Summary Start OAuth flow
// @Description Redirects to the OAuth provider's authorization page
// @Tags auth
// @Param provider path string true "OAuth provider (github, google, discord)"
// @Success 307 {string} string "Redirect to provider"
// @Failure 400 {object} response.Response "Bad request"
// @Router /v1/auth/oauth/{provider} [get]
func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Validate provider
	if !isValidProvider(provider) {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid OAuth provider"))
		return
	}

	// Generate cryptographically secure state for CSRF protection
	state, err := generateState()
	if err != nil {
		response.Error(w, apierrors.ErrInternal)
		return
	}

	// Store state in a secure HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   StateExpirySeconds,
	})

	// Get the OAuth authorization URL
	authURL, err := h.oauthService.GetAuthURL(provider, state)
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Provider not configured"))
		return
	}

	// Redirect to the provider's authorization page
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// Callback handles the OAuth callback from the provider.
// @Summary OAuth callback
// @Description Handles the OAuth provider's callback after user authorization
// @Tags auth
// @Param provider path string true "OAuth provider (github, google, discord)"
// @Param code query string true "Authorization code from provider"
// @Param state query string true "State parameter for CSRF validation"
// @Success 307 {string} string "Redirect to dashboard"
// @Failure 400 {object} response.Response "Bad request"
// @Router /v1/auth/oauth/{provider}/callback [get]
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Check for OAuth errors from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		h.redirectWithError(w, r, "oauth_denied", errDesc)
		return
	}

	// Validate provider
	if !isValidProvider(provider) {
		h.redirectWithError(w, r, "invalid_provider", "")
		return
	}

	// Validate required parameters
	if code == "" {
		h.redirectWithError(w, r, "missing_code", "")
		return
	}

	if state == "" {
		h.redirectWithError(w, r, "missing_state", "")
		return
	}

	// Verify CSRF state matches
	cookie, err := r.Cookie(OAuthStateCookieName)
	if err != nil || cookie.Value == "" {
		h.redirectWithError(w, r, "invalid_state", "State cookie not found")
		return
	}

	if cookie.Value != state {
		h.redirectWithError(w, r, "invalid_state", "State mismatch")
		return
	}

	// Clear the state cookie now that we've validated it
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		MaxAge:   -1, // Delete the cookie
	})

	// Process the OAuth callback
	user, sessionID, err := h.oauthService.HandleCallback(r.Context(), provider, code)
	if err != nil {
		h.redirectWithError(w, r, "oauth_failed", err.Error())
		return
	}

	// Set the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   SessionExpirySeconds,
	})

	// Redirect to the dashboard with success
	redirectURL := h.dashboardURL + "/dashboard"
	if user.Email == "" {
		// If we don't have an email, prompt user to add one
		redirectURL = h.dashboardURL + "/settings/profile?notice=email_required"
	}

	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// ListProviders returns a list of configured OAuth providers.
// @Summary List OAuth providers
// @Description Returns a list of available OAuth providers
// @Tags auth
// @Produce json
// @Success 200 {object} response.Response{data=[]string}
// @Router /v1/auth/oauth/providers [get]
func (h *OAuthHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.oauthService.GetSupportedProviders()
	response.OK(w, map[string]interface{}{
		"providers": providers,
	})
}

// redirectWithError redirects to the login page with an error message.
func (h *OAuthHandler) redirectWithError(w http.ResponseWriter, r *http.Request, errorCode, errorMessage string) {
	redirectURL := h.dashboardURL + "/login?error=" + errorCode
	if errorMessage != "" {
		// Note: In production, consider logging the error message instead of
		// exposing it to users to avoid information leakage
	}
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// generateState generates a cryptographically secure random state string.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// isValidProvider checks if the provider is one of the supported OAuth providers.
func isValidProvider(provider string) bool {
	switch provider {
	case "github", "google", "discord":
		return true
	default:
		return false
	}
}

