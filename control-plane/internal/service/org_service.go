// Package service provides business logic implementations.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	apierrors "github.com/Bidon15/popsigner/control-plane/internal/pkg/errors"
	"github.com/Bidon15/popsigner/control-plane/internal/repository"
)

// OrgService defines the interface for organization operations.
type OrgService interface {
	// Organization CRUD
	Create(ctx context.Context, name string, ownerID uuid.UUID) (*models.Organization, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*models.Organization, error)
	Update(ctx context.Context, id uuid.UUID, name string, actorID uuid.UUID) (*models.Organization, error)
	Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error
	ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error)

	// Members
	InviteMember(ctx context.Context, orgID uuid.UUID, email string, role models.Role, inviterID uuid.UUID) (*models.Invitation, error)
	AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*models.Organization, error)
	RemoveMember(ctx context.Context, orgID, userID, actorID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role models.Role, actorID uuid.UUID) error
	ListMembers(ctx context.Context, orgID uuid.UUID, actorID uuid.UUID) ([]*models.OrgMember, error)
	ListPendingInvitations(ctx context.Context, orgID uuid.UUID, actorID uuid.UUID) ([]*models.Invitation, error)
	CancelInvitation(ctx context.Context, orgID, invitationID uuid.UUID, actorID uuid.UUID) error

	// Access control
	CheckAccess(ctx context.Context, orgID, userID uuid.UUID, requiredRole models.Role) error
	GetMemberRole(ctx context.Context, orgID, userID uuid.UUID) (models.Role, error)
	GetLimits(ctx context.Context, orgID uuid.UUID) (*models.PlanLimits, error)

	// Namespaces
	CreateNamespace(ctx context.Context, orgID uuid.UUID, name, description string, actorID uuid.UUID) (*models.Namespace, error)
	GetNamespace(ctx context.Context, orgID, nsID uuid.UUID, actorID uuid.UUID) (*models.Namespace, error)
	ListNamespaces(ctx context.Context, orgID uuid.UUID, actorID uuid.UUID) ([]*models.Namespace, error)
	DeleteNamespace(ctx context.Context, orgID, nsID uuid.UUID, actorID uuid.UUID) error
}

// OrgServiceConfig holds configuration for the org service.
type OrgServiceConfig struct {
	InvitationExpiry time.Duration
}

// DefaultOrgServiceConfig returns sensible default configuration.
func DefaultOrgServiceConfig() OrgServiceConfig {
	return OrgServiceConfig{
		InvitationExpiry: 7 * 24 * time.Hour, // 7 days
	}
}

type orgService struct {
	orgRepo  repository.OrgRepository
	userRepo repository.UserRepository
	config   OrgServiceConfig
}

// NewOrgService creates a new organization service.
func NewOrgService(
	orgRepo repository.OrgRepository,
	userRepo repository.UserRepository,
	config OrgServiceConfig,
) OrgService {
	return &orgService{
		orgRepo:  orgRepo,
		userRepo: userRepo,
		config:   config,
	}
}

// Create creates a new organization with the specified owner.
func (s *orgService) Create(ctx context.Context, name string, ownerID uuid.UUID) (*models.Organization, error) {
	if name == "" {
		return nil, apierrors.NewValidationError("name", "Organization name is required")
	}

	org := &models.Organization{Name: name}
	if err := s.orgRepo.Create(ctx, org, ownerID); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}
	return org, nil
}

// Get retrieves an organization by ID.
func (s *orgService) Get(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, apierrors.NewNotFoundError("Organization")
	}
	return org, nil
}

// GetBySlug retrieves an organization by slug.
func (s *orgService) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	org, err := s.orgRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, apierrors.NewNotFoundError("Organization")
	}
	return org, nil
}

// Update updates an organization's name.
func (s *orgService) Update(ctx context.Context, id uuid.UUID, name string, actorID uuid.UUID) (*models.Organization, error) {
	// Check actor has admin or owner access
	if err := s.CheckAccess(ctx, id, actorID, models.RoleAdmin); err != nil {
		return nil, err
	}

	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, apierrors.NewNotFoundError("Organization")
	}

	if name != "" {
		org.Name = name
	}

	if err := s.orgRepo.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return org, nil
}

// Delete removes an organization. Only the owner can delete.
func (s *orgService) Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	// Only owner can delete
	if err := s.CheckAccess(ctx, id, actorID, models.RoleOwner); err != nil {
		return err
	}

	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return apierrors.NewNotFoundError("Organization")
	}

	if err := s.orgRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return nil
}

// ListUserOrgs lists all organizations a user is a member of.
func (s *orgService) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error) {
	orgs, err := s.orgRepo.ListUserOrgs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	return orgs, nil
}

// InviteMember sends an invitation to join the organization.
func (s *orgService) InviteMember(ctx context.Context, orgID uuid.UUID, email string, role models.Role, inviterID uuid.UUID) (*models.Invitation, error) {
	// Check inviter has admin access
	if err := s.CheckAccess(ctx, orgID, inviterID, models.RoleAdmin); err != nil {
		return nil, err
	}

	// Validate role
	if !models.ValidRole(role) {
		return nil, apierrors.NewValidationError("role", "Invalid role")
	}

	// Cannot invite as owner
	if role == models.RoleOwner {
		return nil, apierrors.NewValidationError("role", "Cannot invite as owner")
	}

	// Check plan limits for team members
	limits, err := s.GetLimits(ctx, orgID)
	if err != nil {
		return nil, err
	}

	memberCount, err := s.orgRepo.CountMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to count members: %w", err)
	}

	// Also count pending invitations
	pending, err := s.orgRepo.ListPendingInvitations(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}

	totalMembers := memberCount + len(pending)
	if limits.TeamMembers > 0 && totalMembers >= limits.TeamMembers {
		return nil, apierrors.ErrQuotaExceeded.WithMessage(
			fmt.Sprintf("Team member limit reached (%d members). Please upgrade your plan.", limits.TeamMembers),
		)
	}

	// Check if user is already a member
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check user: %w", err)
	}
	if user != nil {
		member, err := s.orgRepo.GetMember(ctx, orgID, user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check membership: %w", err)
		}
		if member != nil {
			return nil, apierrors.NewConflictError("User is already a member")
		}
	}

	// Check for existing invitation
	existing, err := s.orgRepo.GetInvitationByEmail(ctx, orgID, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invitation: %w", err)
	}
	if existing != nil {
		return nil, apierrors.NewConflictError("Invitation already sent to this email")
	}

	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}

	inv := &models.Invitation{
		OrgID:     orgID,
		Email:     email,
		Role:      role,
		Token:     token,
		InvitedBy: inviterID,
		ExpiresAt: time.Now().Add(s.config.InvitationExpiry),
	}

	if err := s.orgRepo.CreateInvitation(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	// TODO: Send invitation email

	return inv, nil
}

// AcceptInvitation accepts an invitation and adds the user to the organization.
func (s *orgService) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*models.Organization, error) {
	// Get the invitation to verify it exists and get org info
	inv, err := s.orgRepo.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}
	if inv == nil {
		return nil, apierrors.NewNotFoundError("Invitation")
	}

	// Check if expired
	if inv.ExpiresAt.Before(time.Now()) {
		return nil, apierrors.ErrBadRequest.WithMessage("Invitation has expired")
	}

	// Check if already accepted
	if inv.AcceptedAt != nil {
		return nil, apierrors.ErrBadRequest.WithMessage("Invitation has already been accepted")
	}

	// Verify user email matches invitation (if we have the user)
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, apierrors.NewNotFoundError("User")
	}
	if user.Email != inv.Email {
		return nil, apierrors.ErrForbidden.WithMessage("This invitation is for a different email address")
	}

	// Accept invitation
	if err := s.orgRepo.AcceptInvitation(ctx, token, userID); err != nil {
		return nil, fmt.Errorf("failed to accept invitation: %w", err)
	}

	// Return the organization
	return inv.Organization, nil
}

// RemoveMember removes a member from the organization.
func (s *orgService) RemoveMember(ctx context.Context, orgID, userID, actorID uuid.UUID) error {
	// Check actor has admin access
	if err := s.CheckAccess(ctx, orgID, actorID, models.RoleAdmin); err != nil {
		return err
	}

	// Get the member to remove
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return apierrors.NewNotFoundError("Member")
	}

	// Cannot remove owner
	if member.Role == models.RoleOwner {
		return apierrors.ErrForbidden.WithMessage("Cannot remove the organization owner")
	}

	// Get actor's role
	actorRole, err := s.GetMemberRole(ctx, orgID, actorID)
	if err != nil {
		return err
	}

	// Only owner can remove admins
	if member.Role == models.RoleAdmin && actorRole != models.RoleOwner {
		return apierrors.ErrForbidden.WithMessage("Only the owner can remove admins")
	}

	if err := s.orgRepo.RemoveMember(ctx, orgID, userID); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	return nil
}

// UpdateMemberRole updates a member's role.
func (s *orgService) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role models.Role, actorID uuid.UUID) error {
	// Check actor has admin access
	if err := s.CheckAccess(ctx, orgID, actorID, models.RoleAdmin); err != nil {
		return err
	}

	// Validate role
	if !models.ValidRole(role) {
		return apierrors.NewValidationError("role", "Invalid role")
	}

	// Cannot change to owner
	if role == models.RoleOwner {
		return apierrors.ErrForbidden.WithMessage("Cannot change role to owner")
	}

	// Get the member
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return apierrors.NewNotFoundError("Member")
	}

	// Cannot change owner's role
	if member.Role == models.RoleOwner {
		return apierrors.ErrForbidden.WithMessage("Cannot change the owner's role")
	}

	// Get actor's role
	actorRole, err := s.GetMemberRole(ctx, orgID, actorID)
	if err != nil {
		return err
	}

	// Only owner can promote to admin
	if role == models.RoleAdmin && actorRole != models.RoleOwner {
		return apierrors.ErrForbidden.WithMessage("Only the owner can promote to admin")
	}

	// Only owner can demote admins
	if member.Role == models.RoleAdmin && actorRole != models.RoleOwner {
		return apierrors.ErrForbidden.WithMessage("Only the owner can change admin roles")
	}

	if err := s.orgRepo.UpdateMemberRole(ctx, orgID, userID, role); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	return nil
}

// ListMembers lists all members of an organization.
func (s *orgService) ListMembers(ctx context.Context, orgID uuid.UUID, actorID uuid.UUID) ([]*models.OrgMember, error) {
	// Check actor has viewer access
	if err := s.CheckAccess(ctx, orgID, actorID, models.RoleViewer); err != nil {
		return nil, err
	}

	members, err := s.orgRepo.ListMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}
	return members, nil
}

// ListPendingInvitations lists all pending invitations for an organization.
func (s *orgService) ListPendingInvitations(ctx context.Context, orgID uuid.UUID, actorID uuid.UUID) ([]*models.Invitation, error) {
	// Check actor has admin access
	if err := s.CheckAccess(ctx, orgID, actorID, models.RoleAdmin); err != nil {
		return nil, err
	}

	invitations, err := s.orgRepo.ListPendingInvitations(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}
	return invitations, nil
}

// CancelInvitation cancels a pending invitation.
func (s *orgService) CancelInvitation(ctx context.Context, orgID, invitationID uuid.UUID, actorID uuid.UUID) error {
	// Check actor has admin access
	if err := s.CheckAccess(ctx, orgID, actorID, models.RoleAdmin); err != nil {
		return err
	}

	if err := s.orgRepo.DeleteInvitation(ctx, invitationID); err != nil {
		return fmt.Errorf("failed to delete invitation: %w", err)
	}

	return nil
}

// CheckAccess verifies that a user has at least the required role in an organization.
func (s *orgService) CheckAccess(ctx context.Context, orgID, userID uuid.UUID, requiredRole models.Role) error {
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return apierrors.ErrForbidden
	}

	// Role hierarchy: owner > admin > operator > viewer
	if models.RoleLevel(member.Role) < models.RoleLevel(requiredRole) {
		return apierrors.ErrForbidden
	}

	return nil
}

// GetMemberRole returns the user's role in an organization.
func (s *orgService) GetMemberRole(ctx context.Context, orgID, userID uuid.UUID) (models.Role, error) {
	member, err := s.orgRepo.GetMember(ctx, orgID, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get member: %w", err)
	}
	if member == nil {
		return "", apierrors.ErrForbidden
	}
	return member.Role, nil
}

// GetLimits returns the plan limits for an organization.
func (s *orgService) GetLimits(ctx context.Context, orgID uuid.UUID) (*models.PlanLimits, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		return nil, apierrors.NewNotFoundError("Organization")
	}

	limits := models.GetPlanLimits(org.Plan)
	return &limits, nil
}

// CreateNamespace creates a new namespace in an organization.
func (s *orgService) CreateNamespace(ctx context.Context, orgID uuid.UUID, name, description string, actorID uuid.UUID) (*models.Namespace, error) {
	// Check actor has operator access
	if err := s.CheckAccess(ctx, orgID, actorID, models.RoleOperator); err != nil {
		return nil, err
	}

	// Validate name
	if name == "" {
		return nil, apierrors.NewValidationError("name", "Namespace name is required")
	}

	// Check plan limits
	limits, err := s.GetLimits(ctx, orgID)
	if err != nil {
		return nil, err
	}

	count, err := s.orgRepo.CountNamespaces(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to count namespaces: %w", err)
	}

	if limits.Namespaces > 0 && count >= limits.Namespaces {
		return nil, apierrors.ErrQuotaExceeded.WithMessage(
			fmt.Sprintf("Namespace limit reached (%d namespaces). Please upgrade your plan.", limits.Namespaces),
		)
	}

	// Check for duplicate name
	existing, err := s.orgRepo.GetNamespaceByName(ctx, orgID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check namespace: %w", err)
	}
	if existing != nil {
		return nil, apierrors.NewConflictError("Namespace with this name already exists")
	}

	var desc *string
	if description != "" {
		desc = &description
	}

	ns := &models.Namespace{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        name,
		Description: desc,
	}

	if err := s.orgRepo.CreateNamespace(ctx, ns); err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	return ns, nil
}

// GetNamespace retrieves a namespace.
func (s *orgService) GetNamespace(ctx context.Context, orgID, nsID uuid.UUID, actorID uuid.UUID) (*models.Namespace, error) {
	// Check actor has viewer access
	if err := s.CheckAccess(ctx, orgID, actorID, models.RoleViewer); err != nil {
		return nil, err
	}

	ns, err := s.orgRepo.GetNamespace(ctx, nsID)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	if ns == nil {
		return nil, apierrors.NewNotFoundError("Namespace")
	}

	// Verify namespace belongs to the org
	if ns.OrgID != orgID {
		return nil, apierrors.NewNotFoundError("Namespace")
	}

	return ns, nil
}

// ListNamespaces lists all namespaces in an organization.
func (s *orgService) ListNamespaces(ctx context.Context, orgID uuid.UUID, actorID uuid.UUID) ([]*models.Namespace, error) {
	// Check actor has viewer access
	if err := s.CheckAccess(ctx, orgID, actorID, models.RoleViewer); err != nil {
		return nil, err
	}

	namespaces, err := s.orgRepo.ListNamespaces(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	return namespaces, nil
}

// DeleteNamespace removes a namespace.
func (s *orgService) DeleteNamespace(ctx context.Context, orgID, nsID uuid.UUID, actorID uuid.UUID) error {
	// Check actor has admin access
	if err := s.CheckAccess(ctx, orgID, actorID, models.RoleAdmin); err != nil {
		return err
	}

	ns, err := s.orgRepo.GetNamespace(ctx, nsID)
	if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}
	if ns == nil {
		return apierrors.NewNotFoundError("Namespace")
	}

	// Verify namespace belongs to the org
	if ns.OrgID != orgID {
		return apierrors.NewNotFoundError("Namespace")
	}

	// Prevent deleting the last namespace
	count, err := s.orgRepo.CountNamespaces(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to count namespaces: %w", err)
	}
	if count <= 1 {
		return apierrors.ErrBadRequest.WithMessage("Cannot delete the last namespace")
	}

	if err := s.orgRepo.DeleteNamespace(ctx, nsID); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	return nil
}

