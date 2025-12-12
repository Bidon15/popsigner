// Package repository provides database access layer implementations.
package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
)

// OrgRepository defines methods for organization data access.
type OrgRepository interface {
	// Organization CRUD
	Create(ctx context.Context, org *models.Organization, ownerID uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*models.Organization, error)
	Update(ctx context.Context, org *models.Organization) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Members
	AddMember(ctx context.Context, orgID, userID uuid.UUID, role models.Role, invitedBy *uuid.UUID) error
	RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role models.Role) error
	GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrgMember, error)
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]*models.OrgMember, error)
	ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error)
	CountMembers(ctx context.Context, orgID uuid.UUID) (int, error)

	// Namespaces
	CreateNamespace(ctx context.Context, ns *models.Namespace) error
	GetNamespace(ctx context.Context, id uuid.UUID) (*models.Namespace, error)
	GetNamespaceByName(ctx context.Context, orgID uuid.UUID, name string) (*models.Namespace, error)
	ListNamespaces(ctx context.Context, orgID uuid.UUID) ([]*models.Namespace, error)
	DeleteNamespace(ctx context.Context, id uuid.UUID) error
	CountNamespaces(ctx context.Context, orgID uuid.UUID) (int, error)

	// Invitations
	CreateInvitation(ctx context.Context, inv *models.Invitation) error
	GetInvitationByToken(ctx context.Context, token string) (*models.Invitation, error)
	GetInvitationByEmail(ctx context.Context, orgID uuid.UUID, email string) (*models.Invitation, error)
	ListPendingInvitations(ctx context.Context, orgID uuid.UUID) ([]*models.Invitation, error)
	AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error
	DeleteInvitation(ctx context.Context, id uuid.UUID) error

	// Stripe billing
	GetByStripeCustomer(ctx context.Context, customerID string) (*models.Organization, error)
	UpdateStripeCustomer(ctx context.Context, orgID uuid.UUID, customerID string) error
	UpdateStripeSubscription(ctx context.Context, orgID uuid.UUID, subscriptionID string) error
	ClearStripeSubscription(ctx context.Context, orgID uuid.UUID) error
	UpdatePlan(ctx context.Context, orgID uuid.UUID, plan models.Plan) error
}

type orgRepo struct {
	pool *pgxpool.Pool
}

// NewOrgRepository creates a new OrgRepository instance.
func NewOrgRepository(pool *pgxpool.Pool) OrgRepository {
	return &orgRepo{pool: pool}
}

// Create creates a new organization with the given owner.
// It creates a default "production" namespace and assigns the owner.
func (r *orgRepo) Create(ctx context.Context, org *models.Organization, ownerID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Generate ID and slug
	org.ID = uuid.New()
	org.Slug = generateSlug(org.Name)
	org.Plan = models.PlanFree
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now

	// Create organization
	query := `
		INSERT INTO organizations (id, name, slug, plan, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = tx.Exec(ctx, query, org.ID, org.Name, org.Slug, org.Plan, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return err
	}

	// Add owner as member
	memberQuery := `
		INSERT INTO org_members (org_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)`

	_, err = tx.Exec(ctx, memberQuery, org.ID, ownerID, models.RoleOwner, now)
	if err != nil {
		return err
	}

	// Create default namespace
	nsID := uuid.New()
	nsQuery := `
		INSERT INTO namespaces (id, org_id, name, description, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err = tx.Exec(ctx, nsQuery, nsID, org.ID, "production", "Production environment", now)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetByID retrieves an organization by its ID.
func (r *orgRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	query := `
		SELECT id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
		       created_at, updated_at
		FROM organizations WHERE id = $1`

	var org models.Organization
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.CreatedAt, &org.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// GetBySlug retrieves an organization by its slug.
func (r *orgRepo) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	query := `
		SELECT id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
		       created_at, updated_at
		FROM organizations WHERE slug = $1`

	var org models.Organization
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.CreatedAt, &org.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// Update updates an organization's name and slug.
func (r *orgRepo) Update(ctx context.Context, org *models.Organization) error {
	query := `
		UPDATE organizations 
		SET name = $2, slug = $3, updated_at = NOW()
		WHERE id = $1`

	org.Slug = generateSlug(org.Name)
	_, err := r.pool.Exec(ctx, query, org.ID, org.Name, org.Slug)
	return err
}

// Delete removes an organization (cascades to members, namespaces, etc).
func (r *orgRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM organizations WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// AddMember adds a user as a member of an organization.
func (r *orgRepo) AddMember(ctx context.Context, orgID, userID uuid.UUID, role models.Role, invitedBy *uuid.UUID) error {
	query := `
		INSERT INTO org_members (org_id, user_id, role, invited_by, joined_at)
		VALUES ($1, $2, $3, $4, NOW())`

	_, err := r.pool.Exec(ctx, query, orgID, userID, role, invitedBy)
	return err
}

// RemoveMember removes a user from an organization.
func (r *orgRepo) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	query := `DELETE FROM org_members WHERE org_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, orgID, userID)
	return err
}

// UpdateMemberRole updates a member's role in an organization.
func (r *orgRepo) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role models.Role) error {
	query := `UPDATE org_members SET role = $3 WHERE org_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, orgID, userID, role)
	return err
}

// GetMember retrieves a specific organization membership.
func (r *orgRepo) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrgMember, error) {
	query := `
		SELECT org_id, user_id, role, invited_by, joined_at
		FROM org_members WHERE org_id = $1 AND user_id = $2`

	var m models.OrgMember
	err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(
		&m.OrgID, &m.UserID, &m.Role, &m.InvitedBy, &m.JoinedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ListMembers lists all members of an organization with user details.
func (r *orgRepo) ListMembers(ctx context.Context, orgID uuid.UUID) ([]*models.OrgMember, error) {
	query := `
		SELECT m.org_id, m.user_id, m.role, m.invited_by, m.joined_at,
		       u.id, u.email, u.name, u.avatar_url
		FROM org_members m
		JOIN users u ON m.user_id = u.id
		WHERE m.org_id = $1
		ORDER BY m.joined_at`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*models.OrgMember
	for rows.Next() {
		var m models.OrgMember
		var u models.User
		if err := rows.Scan(
			&m.OrgID, &m.UserID, &m.Role, &m.InvitedBy, &m.JoinedAt,
			&u.ID, &u.Email, &u.Name, &u.AvatarURL,
		); err != nil {
			return nil, err
		}
		m.User = &u
		members = append(members, &m)
	}
	return members, rows.Err()
}

// ListUserOrgs lists all organizations a user is a member of.
func (r *orgRepo) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error) {
	query := `
		SELECT o.id, o.name, o.slug, o.plan, o.stripe_customer_id, 
		       o.stripe_subscription_id, o.created_at, o.updated_at
		FROM organizations o
		JOIN org_members m ON o.id = m.org_id
		WHERE m.user_id = $1
		ORDER BY o.created_at`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*models.Organization
	for rows.Next() {
		var o models.Organization
		if err := rows.Scan(
			&o.ID, &o.Name, &o.Slug, &o.Plan,
			&o.StripeCustomerID, &o.StripeSubscriptionID,
			&o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orgs = append(orgs, &o)
	}
	return orgs, rows.Err()
}

// CountMembers returns the number of members in an organization.
func (r *orgRepo) CountMembers(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM org_members WHERE org_id = $1`, orgID).Scan(&count)
	return count, err
}

// CreateNamespace creates a new namespace.
func (r *orgRepo) CreateNamespace(ctx context.Context, ns *models.Namespace) error {
	if ns.ID == uuid.Nil {
		ns.ID = uuid.New()
	}
	ns.CreatedAt = time.Now()

	query := `
		INSERT INTO namespaces (id, org_id, name, description, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.pool.Exec(ctx, query, ns.ID, ns.OrgID, ns.Name, ns.Description, ns.CreatedAt)
	return err
}

// GetNamespace retrieves a namespace by its ID.
func (r *orgRepo) GetNamespace(ctx context.Context, id uuid.UUID) (*models.Namespace, error) {
	query := `SELECT id, org_id, name, description, created_at FROM namespaces WHERE id = $1`

	var ns models.Namespace
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ns.ID, &ns.OrgID, &ns.Name, &ns.Description, &ns.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ns, nil
}

// GetNamespaceByName retrieves a namespace by org ID and name.
func (r *orgRepo) GetNamespaceByName(ctx context.Context, orgID uuid.UUID, name string) (*models.Namespace, error) {
	query := `SELECT id, org_id, name, description, created_at FROM namespaces WHERE org_id = $1 AND name = $2`

	var ns models.Namespace
	err := r.pool.QueryRow(ctx, query, orgID, name).Scan(
		&ns.ID, &ns.OrgID, &ns.Name, &ns.Description, &ns.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ns, nil
}

// ListNamespaces lists all namespaces in an organization.
func (r *orgRepo) ListNamespaces(ctx context.Context, orgID uuid.UUID) ([]*models.Namespace, error) {
	query := `SELECT id, org_id, name, description, created_at FROM namespaces WHERE org_id = $1 ORDER BY name`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var namespaces []*models.Namespace
	for rows.Next() {
		var ns models.Namespace
		if err := rows.Scan(&ns.ID, &ns.OrgID, &ns.Name, &ns.Description, &ns.CreatedAt); err != nil {
			return nil, err
		}
		namespaces = append(namespaces, &ns)
	}
	return namespaces, rows.Err()
}

// DeleteNamespace removes a namespace.
func (r *orgRepo) DeleteNamespace(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM namespaces WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CountNamespaces returns the number of namespaces in an organization.
func (r *orgRepo) CountNamespaces(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM namespaces WHERE org_id = $1`, orgID).Scan(&count)
	return count, err
}

// CreateInvitation creates a new invitation.
func (r *orgRepo) CreateInvitation(ctx context.Context, inv *models.Invitation) error {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	if inv.Token == "" {
		inv.Token = generateInvitationToken()
	}
	inv.CreatedAt = time.Now()

	query := `
		INSERT INTO invitations (id, org_id, email, role, token, invited_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		inv.ID, inv.OrgID, inv.Email, inv.Role,
		inv.Token, inv.InvitedBy, inv.ExpiresAt, inv.CreatedAt,
	)
	return err
}

// GetInvitationByToken retrieves an invitation by its token.
func (r *orgRepo) GetInvitationByToken(ctx context.Context, token string) (*models.Invitation, error) {
	query := `
		SELECT i.id, i.org_id, i.email, i.role, i.token, i.invited_by, 
		       i.expires_at, i.accepted_at, i.created_at,
		       o.id, o.name, o.slug, o.plan
		FROM invitations i
		JOIN organizations o ON i.org_id = o.id
		WHERE i.token = $1`

	var inv models.Invitation
	var org models.Organization
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&inv.ID, &inv.OrgID, &inv.Email, &inv.Role,
		&inv.Token, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt,
		&org.ID, &org.Name, &org.Slug, &org.Plan,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	inv.Organization = &org
	return &inv, nil
}

// GetInvitationByEmail retrieves a pending invitation for an email in an org.
func (r *orgRepo) GetInvitationByEmail(ctx context.Context, orgID uuid.UUID, email string) (*models.Invitation, error) {
	query := `
		SELECT id, org_id, email, role, token, invited_by, expires_at, accepted_at, created_at
		FROM invitations 
		WHERE org_id = $1 AND email = $2 AND accepted_at IS NULL AND expires_at > NOW()`

	var inv models.Invitation
	err := r.pool.QueryRow(ctx, query, orgID, email).Scan(
		&inv.ID, &inv.OrgID, &inv.Email, &inv.Role,
		&inv.Token, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// ListPendingInvitations lists all pending invitations for an organization.
func (r *orgRepo) ListPendingInvitations(ctx context.Context, orgID uuid.UUID) ([]*models.Invitation, error) {
	query := `
		SELECT id, org_id, email, role, token, invited_by, expires_at, accepted_at, created_at
		FROM invitations 
		WHERE org_id = $1 AND accepted_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []*models.Invitation
	for rows.Next() {
		var inv models.Invitation
		if err := rows.Scan(
			&inv.ID, &inv.OrgID, &inv.Email, &inv.Role,
			&inv.Token, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		invitations = append(invitations, &inv)
	}
	return invitations, rows.Err()
}

// AcceptInvitation marks an invitation as accepted and adds the user as a member.
func (r *orgRepo) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get invitation
	var inv models.Invitation
	query := `
		SELECT id, org_id, email, role, invited_by, expires_at, accepted_at
		FROM invitations WHERE token = $1`

	err = tx.QueryRow(ctx, query, token).Scan(
		&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("invitation not found")
	}
	if err != nil {
		return err
	}

	// Check if already accepted
	if inv.AcceptedAt != nil {
		return errors.New("invitation already accepted")
	}

	// Check if expired
	if inv.ExpiresAt.Before(time.Now()) {
		return errors.New("invitation expired")
	}

	// Mark as accepted
	now := time.Now()
	updateQuery := `UPDATE invitations SET accepted_at = $2 WHERE id = $1`
	_, err = tx.Exec(ctx, updateQuery, inv.ID, now)
	if err != nil {
		return err
	}

	// Add member
	memberQuery := `
		INSERT INTO org_members (org_id, user_id, role, invited_by, joined_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (org_id, user_id) DO NOTHING`

	_, err = tx.Exec(ctx, memberQuery, inv.OrgID, userID, inv.Role, inv.InvitedBy, now)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteInvitation removes an invitation.
func (r *orgRepo) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM invitations WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// generateSlug creates a URL-safe slug from a name.
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric except hyphens
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// generateInvitationToken creates a secure random token.
func generateInvitationToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GetByStripeCustomer retrieves an organization by its Stripe customer ID.
func (r *orgRepo) GetByStripeCustomer(ctx context.Context, customerID string) (*models.Organization, error) {
	query := `
		SELECT id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
		       created_at, updated_at
		FROM organizations WHERE stripe_customer_id = $1`

	var org models.Organization
	err := r.pool.QueryRow(ctx, query, customerID).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Plan,
		&org.StripeCustomerID, &org.StripeSubscriptionID,
		&org.CreatedAt, &org.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// UpdateStripeCustomer updates the Stripe customer ID for an organization.
func (r *orgRepo) UpdateStripeCustomer(ctx context.Context, orgID uuid.UUID, customerID string) error {
	query := `UPDATE organizations SET stripe_customer_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, orgID, customerID)
	return err
}

// UpdateStripeSubscription updates the Stripe subscription ID for an organization.
func (r *orgRepo) UpdateStripeSubscription(ctx context.Context, orgID uuid.UUID, subscriptionID string) error {
	query := `UPDATE organizations SET stripe_subscription_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, orgID, subscriptionID)
	return err
}

// ClearStripeSubscription removes the Stripe subscription ID from an organization.
func (r *orgRepo) ClearStripeSubscription(ctx context.Context, orgID uuid.UUID) error {
	query := `UPDATE organizations SET stripe_subscription_id = NULL, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, orgID)
	return err
}

// UpdatePlan updates the subscription plan for an organization.
func (r *orgRepo) UpdatePlan(ctx context.Context, orgID uuid.UUID, plan models.Plan) error {
	query := `UPDATE organizations SET plan = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, orgID, plan)
	return err
}

// Compile-time check to ensure orgRepo implements OrgRepository.
var _ OrgRepository = (*orgRepo)(nil)
