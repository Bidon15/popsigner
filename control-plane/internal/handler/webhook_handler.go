// Package handler provides HTTP handlers for the control plane API.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/middleware"
	"github.com/Bidon15/popsigner/control-plane/internal/models"
	apierrors "github.com/Bidon15/popsigner/control-plane/internal/pkg/errors"
	"github.com/Bidon15/popsigner/control-plane/internal/pkg/response"
	"github.com/Bidon15/popsigner/control-plane/internal/service"
)

// WebhookHandler handles webhook-related HTTP requests.
type WebhookHandler struct {
	webhookService service.WebhookService
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(webhookService service.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

// Routes returns a chi router with webhook routes.
func (h *WebhookHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Webhook CRUD
	r.With(middleware.RequireScope("webhooks:read")).Get("/", h.List)
	r.With(middleware.RequireScope("webhooks:write")).Post("/", h.Create)
	r.With(middleware.RequireScope("webhooks:read")).Get("/{id}", h.Get)
	r.With(middleware.RequireScope("webhooks:write")).Patch("/{id}", h.Update)
	r.With(middleware.RequireScope("webhooks:write")).Delete("/{id}", h.Delete)

	// Secret management
	r.With(middleware.RequireScope("webhooks:write")).Post("/{id}/rotate-secret", h.RotateSecret)

	// Deliveries
	r.With(middleware.RequireScope("webhooks:read")).Get("/{id}/deliveries", h.ListDeliveries)
	r.With(middleware.RequireScope("webhooks:write")).Post("/{id}/deliveries/{deliveryId}/retry", h.RetryDelivery)

	return r
}

// CreateWebhookHTTPRequest is the HTTP request body for creating a webhook.
type CreateWebhookHTTPRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// UpdateWebhookHTTPRequest is the HTTP request body for updating a webhook.
type UpdateWebhookHTTPRequest struct {
	URL     *string  `json:"url,omitempty"`
	Events  []string `json:"events,omitempty"`
	Enabled *bool    `json:"enabled,omitempty"`
}

// WebhookResponse is the API response format for webhooks.
type WebhookResponse struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	URL             string    `json:"url"`
	Secret          string    `json:"secret,omitempty"` // Only included on create
	Events          []string  `json:"events"`
	Enabled         bool      `json:"enabled"`
	LastTriggeredAt *string   `json:"last_triggered_at,omitempty"`
	FailureCount    int       `json:"failure_count"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

// WebhookDeliveryResponse is the API response format for webhook deliveries.
type WebhookDeliveryResponse struct {
	ID           uuid.UUID `json:"id"`
	WebhookID    uuid.UUID `json:"webhook_id"`
	Event        string    `json:"event"`
	StatusCode   int       `json:"status_code,omitempty"`
	ResponseBody string    `json:"response_body,omitempty"`
	DurationMs   int64     `json:"duration_ms,omitempty"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	AttemptedAt  string    `json:"attempted_at"`
}

// toWebhookResponse converts a Webhook model to a response.
func toWebhookResponse(webhook *models.Webhook, includeSecret bool) *WebhookResponse {
	resp := &WebhookResponse{
		ID:           webhook.ID,
		OrgID:        webhook.OrgID,
		URL:          webhook.URL,
		Events:       make([]string, len(webhook.Events)),
		Enabled:      webhook.Enabled,
		FailureCount: webhook.FailureCount,
		CreatedAt:    webhook.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    webhook.UpdatedAt.Format(time.RFC3339),
	}

	// Include secret only when requested (e.g., on create)
	if includeSecret {
		resp.Secret = webhook.Secret
	}

	// Convert events
	for i, e := range webhook.Events {
		resp.Events[i] = string(e)
	}

	// Format last triggered time
	if webhook.LastTriggeredAt != nil {
		t := webhook.LastTriggeredAt.Format(time.RFC3339)
		resp.LastTriggeredAt = &t
	}

	return resp
}

// toDeliveryResponse converts a WebhookDelivery model to a response.
func toDeliveryResponse(d *models.WebhookDelivery) *WebhookDeliveryResponse {
	return &WebhookDeliveryResponse{
		ID:           d.ID,
		WebhookID:    d.WebhookID,
		Event:        string(d.Event),
		StatusCode:   d.StatusCode,
		ResponseBody: d.ResponseBody,
		DurationMs:   int64(d.Duration / time.Millisecond),
		Success:      d.Success,
		Error:        d.Error,
		AttemptedAt:  d.AttemptedAt.Format(time.RFC3339),
	}
}

// Create handles POST /v1/webhooks
// @Summary Create a webhook
// @Description Create a new webhook for the organization
// @Tags webhooks
// @Accept json
// @Produce json
// @Param webhook body CreateWebhookHTTPRequest true "Webhook details"
// @Success 201 {object} response.Response{data=WebhookResponse}
// @Failure 400 {object} response.Response{error=apierrors.APIError}
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Router /v1/webhooks [post]
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	var req CreateWebhookHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid request body"))
		return
	}

	// Validate required fields
	if req.URL == "" {
		response.Error(w, apierrors.NewValidationError("url", "URL is required"))
		return
	}

	if len(req.Events) == 0 {
		response.Error(w, apierrors.NewValidationError("events", "At least one event is required"))
		return
	}

	// Convert string events to WebhookEvent type
	events := make([]models.WebhookEvent, len(req.Events))
	for i, e := range req.Events {
		events[i] = models.WebhookEvent(e)
	}

	webhook, err := h.webhookService.Create(r.Context(), orgID, service.CreateWebhookRequest{
		URL:    req.URL,
		Events: events,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	// Include secret in response on creation
	response.Created(w, toWebhookResponse(webhook, true))
}

// List handles GET /v1/webhooks
// @Summary List webhooks
// @Description List all webhooks for the organization
// @Tags webhooks
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]WebhookResponse}
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Router /v1/webhooks [get]
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	webhooks, err := h.webhookService.List(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Convert to response format (without secrets)
	webhookResponses := make([]*WebhookResponse, len(webhooks))
	for i, webhook := range webhooks {
		webhookResponses[i] = toWebhookResponse(webhook, false)
	}

	response.OK(w, webhookResponses)
}

// Get handles GET /v1/webhooks/{id}
// @Summary Get a webhook
// @Description Get a specific webhook by ID
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID (UUID)"
// @Success 200 {object} response.Response{data=WebhookResponse}
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Failure 404 {object} response.Response{error=apierrors.APIError}
// @Router /v1/webhooks/{id} [get]
func (h *WebhookHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid webhook ID"))
		return
	}

	webhook, err := h.webhookService.Get(r.Context(), orgID, webhookID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, toWebhookResponse(webhook, false))
}

// Update handles PATCH /v1/webhooks/{id}
// @Summary Update a webhook
// @Description Update a webhook's URL, events, or enabled status
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID (UUID)"
// @Param webhook body UpdateWebhookHTTPRequest true "Update fields"
// @Success 200 {object} response.Response{data=WebhookResponse}
// @Failure 400 {object} response.Response{error=apierrors.APIError}
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Failure 404 {object} response.Response{error=apierrors.APIError}
// @Router /v1/webhooks/{id} [patch]
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid webhook ID"))
		return
	}

	var req UpdateWebhookHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid request body"))
		return
	}

	// Convert string events to WebhookEvent type
	var events []models.WebhookEvent
	if len(req.Events) > 0 {
		events = make([]models.WebhookEvent, len(req.Events))
		for i, e := range req.Events {
			events[i] = models.WebhookEvent(e)
		}
	}

	webhook, err := h.webhookService.Update(r.Context(), orgID, webhookID, service.UpdateWebhookRequest{
		URL:     req.URL,
		Events:  events,
		Enabled: req.Enabled,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, toWebhookResponse(webhook, false))
}

// Delete handles DELETE /v1/webhooks/{id}
// @Summary Delete a webhook
// @Description Delete a webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID (UUID)"
// @Success 204 "No Content"
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Failure 404 {object} response.Response{error=apierrors.APIError}
// @Router /v1/webhooks/{id} [delete]
func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid webhook ID"))
		return
	}

	if err := h.webhookService.Delete(r.Context(), orgID, webhookID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// RotateSecret handles POST /v1/webhooks/{id}/rotate-secret
// @Summary Rotate webhook secret
// @Description Generate a new signing secret for the webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID (UUID)"
// @Success 200 {object} response.Response{data=WebhookResponse}
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Failure 404 {object} response.Response{error=apierrors.APIError}
// @Router /v1/webhooks/{id}/rotate-secret [post]
func (h *WebhookHandler) RotateSecret(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid webhook ID"))
		return
	}

	webhook, err := h.webhookService.RotateSecret(r.Context(), orgID, webhookID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Include new secret in response
	response.OK(w, toWebhookResponse(webhook, true))
}

// ListDeliveries handles GET /v1/webhooks/{id}/deliveries
// @Summary List webhook deliveries
// @Description List recent delivery attempts for a webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID (UUID)"
// @Param limit query int false "Number of results (max 100)"
// @Success 200 {object} response.Response{data=[]WebhookDeliveryResponse}
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Failure 404 {object} response.Response{error=apierrors.APIError}
// @Router /v1/webhooks/{id}/deliveries [get]
func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid webhook ID"))
		return
	}

	// Parse limit
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	deliveries, err := h.webhookService.GetDeliveries(r.Context(), orgID, webhookID, limit)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Convert to response format
	deliveryResponses := make([]*WebhookDeliveryResponse, len(deliveries))
	for i, d := range deliveries {
		deliveryResponses[i] = toDeliveryResponse(d)
	}

	response.OK(w, deliveryResponses)
}

// RetryDelivery handles POST /v1/webhooks/{id}/deliveries/{deliveryId}/retry
// @Summary Retry a webhook delivery
// @Description Retry a failed webhook delivery
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID (UUID)"
// @Param deliveryId path string true "Delivery ID (UUID)"
// @Success 202 {object} response.Response{data=map[string]string}
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Failure 404 {object} response.Response{error=apierrors.APIError}
// @Router /v1/webhooks/{id}/deliveries/{deliveryId}/retry [post]
func (h *WebhookHandler) RetryDelivery(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	webhookID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid webhook ID"))
		return
	}

	deliveryID, err := uuid.Parse(chi.URLParam(r, "deliveryId"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid delivery ID"))
		return
	}

	if err := h.webhookService.RetryDelivery(r.Context(), orgID, webhookID, deliveryID); err != nil {
		response.Error(w, err)
		return
	}

	response.Accepted(w, map[string]string{"status": "retry_queued"})
}

