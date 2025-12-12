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

// AuditHandler handles audit log HTTP requests.
type AuditHandler struct {
	auditService service.AuditService
}

// NewAuditHandler creates a new audit handler.
func NewAuditHandler(auditService service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// Routes returns a chi router with audit routes.
func (h *AuditHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// All audit endpoints require audit:read scope
	r.With(middleware.RequireScope("audit:read")).Get("/logs", h.ListLogs)
	r.With(middleware.RequireScope("audit:read")).Get("/logs/{id}", h.GetLog)

	return r
}

// AuditLogResponse is the API response format for audit logs.
type AuditLogResponse struct {
	ID           uuid.UUID              `json:"id"`
	OrgID        uuid.UUID              `json:"org_id"`
	Event        models.AuditEvent      `json:"event"`
	ActorID      *uuid.UUID             `json:"actor_id,omitempty"`
	ActorType    models.ActorType       `json:"actor_type"`
	ResourceType *models.ResourceType   `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID             `json:"resource_id,omitempty"`
	IPAddress    *string                `json:"ip_address,omitempty"`
	UserAgent    *string                `json:"user_agent,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    string                 `json:"created_at"`
}

// toAuditLogResponse converts an AuditLog model to a response.
func toAuditLogResponse(log *models.AuditLog) *AuditLogResponse {
	resp := &AuditLogResponse{
		ID:           log.ID,
		OrgID:        log.OrgID,
		Event:        log.Event,
		ActorID:      log.ActorID,
		ActorType:    log.ActorType,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		UserAgent:    log.UserAgent,
		CreatedAt:    log.CreatedAt.Format(time.RFC3339),
	}

	// Convert IP address to string
	if log.IPAddress != nil {
		ip := log.IPAddress.String()
		resp.IPAddress = &ip
	}

	// Parse metadata if present
	if len(log.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(log.Metadata, &metadata); err == nil {
			resp.Metadata = metadata
		}
	}

	return resp
}

// ListLogs handles GET /v1/audit/logs
// @Summary List audit logs
// @Description Retrieve audit logs for the organization with optional filters
// @Tags audit
// @Accept json
// @Produce json
// @Param event query string false "Filter by event type"
// @Param resource_type query string false "Filter by resource type"
// @Param resource_id query string false "Filter by resource ID (UUID)"
// @Param actor_id query string false "Filter by actor ID (UUID)"
// @Param start_time query string false "Filter by start time (RFC3339)"
// @Param end_time query string false "Filter by end time (RFC3339)"
// @Param limit query int false "Number of results (max 100)"
// @Param cursor query string false "Pagination cursor"
// @Success 200 {object} response.Response{data=[]AuditLogResponse}
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Failure 500 {object} response.Response{error=apierrors.APIError}
// @Router /v1/audit/logs [get]
func (h *AuditHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	// Build filter from query parameters
	filter := service.AuditFilter{
		Cursor: r.URL.Query().Get("cursor"),
	}

	// Parse event type
	if eventStr := r.URL.Query().Get("event"); eventStr != "" {
		event := models.AuditEvent(eventStr)
		filter.Event = &event
	}

	// Parse resource type
	if rtStr := r.URL.Query().Get("resource_type"); rtStr != "" {
		rt := models.ResourceType(rtStr)
		filter.ResourceType = &rt
	}

	// Parse resource ID
	if ridStr := r.URL.Query().Get("resource_id"); ridStr != "" {
		if rid, err := uuid.Parse(ridStr); err == nil {
			filter.ResourceID = &rid
		}
	}

	// Parse actor ID
	if aidStr := r.URL.Query().Get("actor_id"); aidStr != "" {
		if aid, err := uuid.Parse(aidStr); err == nil {
			filter.ActorID = &aid
		}
	}

	// Parse time filters
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &t
		}
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	// Query logs
	logs, nextCursor, err := h.auditService.Query(r.Context(), orgID, filter)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Convert to response format
	logResponses := make([]*AuditLogResponse, len(logs))
	for i, log := range logs {
		logResponses[i] = toAuditLogResponse(log)
	}

	// Build response with pagination metadata
	meta := &response.Meta{}
	if nextCursor != "" {
		meta.NextCursor = nextCursor
	}

	response.JSONWithMeta(w, http.StatusOK, logResponses, meta)
}

// GetLog handles GET /v1/audit/logs/{id}
// @Summary Get a specific audit log
// @Description Retrieve a single audit log entry by ID
// @Tags audit
// @Accept json
// @Produce json
// @Param id path string true "Audit log ID (UUID)"
// @Success 200 {object} response.Response{data=AuditLogResponse}
// @Failure 401 {object} response.Response{error=apierrors.APIError}
// @Failure 404 {object} response.Response{error=apierrors.APIError}
// @Failure 500 {object} response.Response{error=apierrors.APIError}
// @Router /v1/audit/logs/{id} [get]
func (h *AuditHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	logID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid log ID"))
		return
	}

	log, err := h.auditService.GetByID(r.Context(), logID)
	if err != nil {
		response.Error(w, err)
		return
	}
	if log == nil {
		response.Error(w, apierrors.NewNotFoundError("Audit log"))
		return
	}

	// Verify the log belongs to the org
	if log.OrgID != orgID {
		response.Error(w, apierrors.NewNotFoundError("Audit log"))
		return
	}

	response.OK(w, toAuditLogResponse(log))
}
