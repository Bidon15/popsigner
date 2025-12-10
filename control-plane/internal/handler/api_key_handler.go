// Package handler provides HTTP handlers for the control plane API.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Bidon15/banhbaoring/control-plane/internal/middleware"
	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
	apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
	"github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
	"github.com/Bidon15/banhbaoring/control-plane/internal/service"
)

// APIKeyHandler handles API key related HTTP requests.
type APIKeyHandler struct {
	apiKeyService service.APIKeyService
}

// NewAPIKeyHandler creates a new API key handler.
func NewAPIKeyHandler(apiKeyService service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// Routes returns a chi router with API key routes.
// All routes require authentication via JWT or existing API key with appropriate scopes.
func (h *APIKeyHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// List all API keys for the organization
	r.Get("/", h.List)

	// Create a new API key
	r.Post("/", h.Create)

	// Get a specific API key
	r.Get("/{id}", h.Get)

	// Delete an API key
	r.Delete("/{id}", h.Delete)

	// Revoke an API key (soft delete - key remains in DB but becomes invalid)
	r.Post("/{id}/revoke", h.Revoke)

	return r
}

// CreateAPIKeyRequest represents the request body for creating an API key.
type CreateAPIKeyRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays *int     `json:"expires_in_days,omitempty"`
	Environment   string   `json:"environment,omitempty"`
}

// Create handles POST /v1/api-keys
// Creates a new API key and returns the full key (shown only once).
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid request body"))
		return
	}

	// Validate required fields
	if req.Name == "" {
		response.Error(w, apierrors.NewValidationError("name", "name is required"))
		return
	}

	if len(req.Scopes) == 0 {
		response.Error(w, apierrors.NewValidationError("scopes", "at least one scope is required"))
		return
	}

	// Validate scopes
	if !models.ValidateScopes(req.Scopes) {
		response.Error(w, apierrors.NewValidationError("scopes", "one or more scopes are invalid"))
		return
	}

	// Create the API key
	key, rawKey, err := h.apiKeyService.Create(r.Context(), orgID, service.CreateAPIKeyRequest{
		Name:          req.Name,
		Scopes:        req.Scopes,
		ExpiresInDays: req.ExpiresInDays,
		Environment:   req.Environment,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	// Return the key with warning
	response.Created(w, models.CreateAPIKeyResponse{
		APIKey:  key.ToResponse(),
		Key:     rawKey,
		Warning: "This key will not be shown again. Store it securely.",
	})
}

// List handles GET /v1/api-keys
// Returns all API keys for the organization (without the actual key values).
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	keys, err := h.apiKeyService.List(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Convert to response format
	keyResponses := make([]*models.APIKeyResponse, len(keys))
	for i, key := range keys {
		keyResponses[i] = key.ToResponse()
	}

	response.OK(w, keyResponses)
}

// Get handles GET /v1/api-keys/{id}
// Returns a specific API key's metadata.
func (h *APIKeyHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	keyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid API key ID"))
		return
	}

	key, err := h.apiKeyService.Get(r.Context(), orgID, keyID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, key.ToResponse())
}

// Delete handles DELETE /v1/api-keys/{id}
// Permanently deletes an API key.
func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	keyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid API key ID"))
		return
	}

	if err := h.apiKeyService.Delete(r.Context(), orgID, keyID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// Revoke handles POST /v1/api-keys/{id}/revoke
// Revokes an API key (soft delete - can't be undone).
func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	keyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid API key ID"))
		return
	}

	if err := h.apiKeyService.Revoke(r.Context(), orgID, keyID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

