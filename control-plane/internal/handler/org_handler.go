// Package handler provides HTTP handlers for the API.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/middleware"
	"github.com/Bidon15/popsigner/control-plane/internal/models"
	apierrors "github.com/Bidon15/popsigner/control-plane/internal/pkg/errors"
	"github.com/Bidon15/popsigner/control-plane/internal/pkg/response"
	"github.com/Bidon15/popsigner/control-plane/internal/service"
)

// OrgHandler handles organization-related HTTP requests.
type OrgHandler struct {
	orgService  service.OrgService
	authService service.AuthService
	validate    *validator.Validate
}

// NewOrgHandler creates a new org handler.
func NewOrgHandler(orgService service.OrgService, authService service.AuthService) *OrgHandler {
	return &OrgHandler{
		orgService:  orgService,
		authService: authService,
		validate:    validator.New(),
	}
}

// Routes returns the organization router with all routes registered.
func (h *OrgHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Organization routes
	r.Post("/", h.CreateOrg)
	r.Get("/", h.ListOrgs)
	r.Get("/{orgId}", h.GetOrg)
	r.Patch("/{orgId}", h.UpdateOrg)
	r.Delete("/{orgId}", h.DeleteOrg)
	r.Get("/{orgId}/limits", h.GetLimits)

	// Member routes
	r.Get("/{orgId}/members", h.ListMembers)
	r.Post("/{orgId}/members", h.InviteMember)
	r.Delete("/{orgId}/members/{userId}", h.RemoveMember)
	r.Patch("/{orgId}/members/{userId}", h.UpdateMemberRole)

	// Invitation routes
	r.Get("/{orgId}/invitations", h.ListInvitations)
	r.Delete("/{orgId}/invitations/{invitationId}", h.CancelInvitation)

	// Namespace routes
	r.Get("/{orgId}/namespaces", h.ListNamespaces)
	r.Post("/{orgId}/namespaces", h.CreateNamespace)
	r.Get("/{orgId}/namespaces/{namespaceId}", h.GetNamespace)
	r.Delete("/{orgId}/namespaces/{namespaceId}", h.DeleteNamespace)

	return r
}

// InvitationRoutes returns routes for invitation acceptance (outside org context).
func (h *OrgHandler) InvitationRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/accept", h.AcceptInvitation)
	return r
}

// CreateOrgRequest represents the request body for creating an organization.
type CreateOrgRequest struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}

// CreateOrg handles organization creation.
// POST /v1/organizations
func (h *OrgHandler) CreateOrg(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	org, err := h.orgService.Create(r.Context(), req.Name, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, org)
}

// ListOrgs handles listing organizations for the current user.
// GET /v1/organizations
func (h *OrgHandler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgs, err := h.orgService.ListUserOrgs(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, orgs)
}

// GetOrg handles retrieving a single organization.
// GET /v1/organizations/{orgId}
func (h *OrgHandler) GetOrg(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Check user has access
	if err := h.orgService.CheckAccess(r.Context(), orgID, userID, models.RoleViewer); err != nil {
		response.Error(w, err)
		return
	}

	org, err := h.orgService.Get(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, org)
}

// UpdateOrgRequest represents the request body for updating an organization.
type UpdateOrgRequest struct {
	Name string `json:"name" validate:"omitempty,min=2,max=100"`
}

// UpdateOrg handles updating an organization.
// PATCH /v1/organizations/{orgId}
func (h *OrgHandler) UpdateOrg(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req UpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	org, err := h.orgService.Update(r.Context(), orgID, req.Name, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, org)
}

// DeleteOrg handles deleting an organization.
// DELETE /v1/organizations/{orgId}
func (h *OrgHandler) DeleteOrg(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	if err := h.orgService.Delete(r.Context(), orgID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// GetLimits handles retrieving plan limits for an organization.
// GET /v1/organizations/{orgId}/limits
func (h *OrgHandler) GetLimits(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Check user has access
	if err := h.orgService.CheckAccess(r.Context(), orgID, userID, models.RoleViewer); err != nil {
		response.Error(w, err)
		return
	}

	limits, err := h.orgService.GetLimits(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, limits)
}

// ListMembers handles listing organization members.
// GET /v1/organizations/{orgId}/members
func (h *OrgHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	members, err := h.orgService.ListMembers(r.Context(), orgID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, members)
}

// InviteMemberRequest represents the request body for inviting a member.
type InviteMemberRequest struct {
	Email string      `json:"email" validate:"required,email"`
	Role  models.Role `json:"role" validate:"required,oneof=admin operator viewer"`
}

// InviteMember handles inviting a new member to an organization.
// POST /v1/organizations/{orgId}/members
func (h *OrgHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req InviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	invitation, err := h.orgService.InviteMember(r.Context(), orgID, req.Email, req.Role, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, invitation)
}

// RemoveMember handles removing a member from an organization.
// DELETE /v1/organizations/{orgId}/members/{userId}
func (h *OrgHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	actorID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	memberIDStr := chi.URLParam(r, "userId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		response.Error(w, apierrors.NewValidationError("userId", "Invalid user ID"))
		return
	}

	if err := h.orgService.RemoveMember(r.Context(), orgID, memberID, actorID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// UpdateMemberRoleRequest represents the request body for updating a member's role.
type UpdateMemberRoleRequest struct {
	Role models.Role `json:"role" validate:"required,oneof=admin operator viewer"`
}

// UpdateMemberRole handles updating a member's role.
// PATCH /v1/organizations/{orgId}/members/{userId}
func (h *OrgHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	actorID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	memberIDStr := chi.URLParam(r, "userId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		response.Error(w, apierrors.NewValidationError("userId", "Invalid user ID"))
		return
	}

	var req UpdateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	if err := h.orgService.UpdateMemberRole(r.Context(), orgID, memberID, req.Role, actorID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ListInvitations handles listing pending invitations.
// GET /v1/organizations/{orgId}/invitations
func (h *OrgHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	invitations, err := h.orgService.ListPendingInvitations(r.Context(), orgID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, invitations)
}

// CancelInvitation handles canceling a pending invitation.
// DELETE /v1/organizations/{orgId}/invitations/{invitationId}
func (h *OrgHandler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	invitationIDStr := chi.URLParam(r, "invitationId")
	invitationID, err := uuid.Parse(invitationIDStr)
	if err != nil {
		response.Error(w, apierrors.NewValidationError("invitationId", "Invalid invitation ID"))
		return
	}

	if err := h.orgService.CancelInvitation(r.Context(), orgID, invitationID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// AcceptInvitationRequest represents the request body for accepting an invitation.
type AcceptInvitationRequest struct {
	Token string `json:"token" validate:"required"`
}

// AcceptInvitation handles accepting an invitation.
// POST /v1/invitations/accept
func (h *OrgHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req AcceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	org, err := h.orgService.AcceptInvitation(r.Context(), req.Token, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, org)
}

// CreateNamespaceRequest represents the request body for creating a namespace.
type CreateNamespaceRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description,omitempty" validate:"max=500"`
}

// ListNamespaces handles listing namespaces in an organization.
// GET /v1/organizations/{orgId}/namespaces
func (h *OrgHandler) ListNamespaces(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	namespaces, err := h.orgService.ListNamespaces(r.Context(), orgID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, namespaces)
}

// CreateNamespace handles creating a new namespace.
// POST /v1/organizations/{orgId}/namespaces
func (h *OrgHandler) CreateNamespace(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req CreateNamespaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid JSON body"))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		validationErrors := formatValidationErrors(err)
		response.Error(w, apierrors.NewValidationErrors(validationErrors))
		return
	}

	ns, err := h.orgService.CreateNamespace(r.Context(), orgID, req.Name, req.Description, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, ns)
}

// GetNamespace handles retrieving a single namespace.
// GET /v1/organizations/{orgId}/namespaces/{namespaceId}
func (h *OrgHandler) GetNamespace(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	namespaceIDStr := chi.URLParam(r, "namespaceId")
	namespaceID, err := uuid.Parse(namespaceIDStr)
	if err != nil {
		response.Error(w, apierrors.NewValidationError("namespaceId", "Invalid namespace ID"))
		return
	}

	ns, err := h.orgService.GetNamespace(r.Context(), orgID, namespaceID, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, ns)
}

// DeleteNamespace handles deleting a namespace.
// DELETE /v1/organizations/{orgId}/namespaces/{namespaceId}
func (h *OrgHandler) DeleteNamespace(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	orgID, err := h.parseOrgID(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	namespaceIDStr := chi.URLParam(r, "namespaceId")
	namespaceID, err := uuid.Parse(namespaceIDStr)
	if err != nil {
		response.Error(w, apierrors.NewValidationError("namespaceId", "Invalid namespace ID"))
		return
	}

	if err := h.orgService.DeleteNamespace(r.Context(), orgID, namespaceID, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// Helper methods

// getUserID extracts the user ID from the request context.
func (h *OrgHandler) getUserID(r *http.Request) (uuid.UUID, error) {
	// Try to get from middleware context
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID != uuid.Nil {
		return userID, nil
	}

	// Try session-based auth
	sessionID := h.getSessionID(r)
	if sessionID == "" {
		return uuid.Nil, apierrors.ErrUnauthorized
	}

	user, err := h.authService.ValidateSession(r.Context(), sessionID)
	if err != nil {
		return uuid.Nil, err
	}

	return user.ID, nil
}

// getSessionID extracts the session ID from the request.
func (h *OrgHandler) getSessionID(r *http.Request) string {
	// Try cookie first
	if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Fall back to Authorization header
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}

	return ""
}

// parseOrgID extracts and validates the organization ID from the URL.
func (h *OrgHandler) parseOrgID(r *http.Request) (uuid.UUID, error) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return uuid.Nil, apierrors.NewValidationError("orgId", "Invalid organization ID")
	}
	return orgID, nil
}

