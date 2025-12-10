// Package handler provides HTTP handlers for the API.
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
	"github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
	"github.com/Bidon15/banhbaoring/control-plane/internal/service"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService   service.AuthService
	validate      *validator.Validate
	sessionExpiry time.Duration
	secureCookie  bool
}

// AuthHandlerConfig holds configuration for the auth handler.
type AuthHandlerConfig struct {
	SessionExpiry time.Duration
	SecureCookie  bool // Set to true in production (HTTPS)
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(authService service.AuthService, cfg AuthHandlerConfig) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		validate:      validator.New(),
		sessionExpiry: cfg.SessionExpiry,
		secureCookie:  cfg.SecureCookie,
	}
}

// Routes returns the auth router with all routes registered.
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	r.Get("/me", h.Me)
	r.Put("/me", h.UpdateProfile)
	r.Post("/password/change", h.ChangePassword)
	r.Post("/password/forgot", h.ForgotPassword)
	r.Post("/password/reset", h.ResetPassword)
	r.Post("/email/verify", h.VerifyEmail)

	return r
}

// Register handles user registration.
// POST /v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	user, err := h.authService.Register(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, user)
}

// Login handles user login.
// POST /v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	user, sessionID, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Set session cookie
	h.setSessionCookie(w, sessionID)

	response.OK(w, map[string]interface{}{
		"user":       user,
		"session_id": sessionID,
	})
}

// Logout handles user logout.
// POST /v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID := h.getSessionID(r)
	if sessionID == "" {
		response.NoContent(w)
		return
	}

	_ = h.authService.Logout(r.Context(), sessionID)

	// Clear cookie
	h.clearSessionCookie(w)

	response.NoContent(w)
}

// Me returns the current authenticated user.
// GET /v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	sessionID := h.getSessionID(r)
	if sessionID == "" {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	user, err := h.authService.ValidateSession(r.Context(), sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, user)
}

// UpdateProfile updates the current user's profile.
// PUT /v1/auth/me
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	sessionID := h.getSessionID(r)
	if sessionID == "" {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	currentUser, err := h.authService.ValidateSession(r.Context(), sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req service.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	user, err := h.authService.UpdateProfile(r.Context(), currentUser.ID, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, user)
}

// ChangePassword changes the current user's password.
// POST /v1/auth/password/change
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	sessionID := h.getSessionID(r)
	if sessionID == "" {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	currentUser, err := h.authService.ValidateSession(r.Context(), sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req struct {
		OldPassword string `json:"old_password" validate:"required"`
		NewPassword string `json:"new_password" validate:"required,min=8"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	if err := h.authService.ChangePassword(r.Context(), currentUser.ID, req.OldPassword, req.NewPassword); err != nil {
		response.Error(w, err)
		return
	}

	// Clear session cookie since all sessions are invalidated
	h.clearSessionCookie(w)

	response.OK(w, map[string]string{
		"message": "Password changed successfully. Please log in again.",
	})
}

// ForgotPassword initiates a password reset.
// POST /v1/auth/password/forgot
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	// Always return success to prevent email enumeration
	_, _ = h.authService.RequestPasswordReset(r.Context(), req.Email)

	response.OK(w, map[string]string{
		"message": "If an account with that email exists, a password reset link has been sent.",
	})
}

// ResetPassword completes a password reset.
// POST /v1/auth/password/reset
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token" validate:"required"`
		NewPassword string `json:"new_password" validate:"required,min=8"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	if err := h.authService.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, map[string]string{
		"message": "Password reset successfully. Please log in with your new password.",
	})
}

// VerifyEmail verifies a user's email address.
// POST /v1/auth/email/verify
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	if err := h.authService.VerifyEmail(r.Context(), req.Token); err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, map[string]string{
		"message": "Email verified successfully.",
	})
}

// Helper methods

// getSessionID extracts the session ID from the request (cookie or header).
func (h *AuthHandler) getSessionID(r *http.Request) string {
	// Try cookie first
	if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Fall back to Authorization header (for API clients)
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}

	return ""
}

// setSessionCookie sets the session cookie.
func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionExpiry.Seconds()),
	})
}

// clearSessionCookie clears the session cookie.
func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// formatValidationErrors converts validator errors to a map.
func formatValidationErrors(err error) map[string]string {
	errors := make(map[string]string)
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			field := fieldError.Field()
			switch fieldError.Tag() {
			case "required":
				errors[field] = field + " is required"
			case "email":
				errors[field] = field + " must be a valid email address"
			case "min":
				errors[field] = field + " must be at least " + fieldError.Param() + " characters"
			case "url":
				errors[field] = field + " must be a valid URL"
			default:
				errors[field] = field + " is invalid"
			}
		}
	}
	return errors
}

// GetUserFromContext retrieves the user ID from context (for use in authenticated routes).
func GetUserFromContext(r *http.Request) (uuid.UUID, bool) {
	userIDStr, ok := r.Context().Value(userIDContextKey).(string)
	if !ok {
		return uuid.UUID{}, false
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.UUID{}, false
	}
	return userID, true
}

// Context key type to avoid collisions
type contextKey string

const userIDContextKey contextKey = "user_id"

