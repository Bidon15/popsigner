# Implementation: Organizations & Namespaces

## Agent: 09A - Multi-Tenancy

> **Phase 5.2** - Can run in parallel with 09B, 09C after Phase 5.1 completes.

---

## 1. Overview

Implement organization management, namespaces, team members, and RBAC.

---

## 2. Scope

| Feature | Included |
|---------|----------|
| Organization CRUD | ✅ |
| Namespace CRUD | ✅ |
| Team member management | ✅ |
| RBAC (Owner/Admin/Operator/Viewer) | ✅ |
| Resource quotas by plan | ✅ |
| Invitations | ✅ |

---

## 3. Models

**File:** `internal/models/organization.go`

```go
package models

import (
    "time"

    "github.com/google/uuid"
)

type Organization struct {
    ID                   uuid.UUID `json:"id" db:"id"`
    Name                 string    `json:"name" db:"name"`
    Slug                 string    `json:"slug" db:"slug"`
    Plan                 string    `json:"plan" db:"plan"`
    StripeCustomerID     *string   `json:"-" db:"stripe_customer_id"`
    StripeSubscriptionID *string   `json:"-" db:"stripe_subscription_id"`
    CreatedAt            time.Time `json:"created_at" db:"created_at"`
    UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

type OrgMember struct {
    OrgID     uuid.UUID  `json:"org_id" db:"org_id"`
    UserID    uuid.UUID  `json:"user_id" db:"user_id"`
    Role      string     `json:"role" db:"role"`
    InvitedBy *uuid.UUID `json:"invited_by,omitempty" db:"invited_by"`
    JoinedAt  time.Time  `json:"joined_at" db:"joined_at"`
    
    // Joined fields
    User *User `json:"user,omitempty"`
}

type Namespace struct {
    ID          uuid.UUID `json:"id" db:"id"`
    OrgID       uuid.UUID `json:"org_id" db:"org_id"`
    Name        string    `json:"name" db:"name"`
    Description string    `json:"description,omitempty" db:"description"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Roles
const (
    RoleOwner    = "owner"
    RoleAdmin    = "admin"
    RoleOperator = "operator"
    RoleViewer   = "viewer"
)

// Plans
const (
    PlanFree       = "free"
    PlanPro        = "pro"
    PlanEnterprise = "enterprise"
)

type PlanLimits struct {
    Keys           int
    SignaturesPerMonth int64
    Namespaces     int
    TeamMembers    int
    AuditRetentionDays int
}

var PlanLimitsMap = map[string]PlanLimits{
    PlanFree: {
        Keys:               3,
        SignaturesPerMonth: 10000,
        Namespaces:         1,
        TeamMembers:        1,
        AuditRetentionDays: 7,
    },
    PlanPro: {
        Keys:               25,
        SignaturesPerMonth: 500000,
        Namespaces:         5,
        TeamMembers:        10,
        AuditRetentionDays: 90,
    },
    PlanEnterprise: {
        Keys:               -1, // unlimited
        SignaturesPerMonth: -1,
        Namespaces:         -1,
        TeamMembers:        -1,
        AuditRetentionDays: 365,
    },
}
```

---

## 4. Repository

**File:** `internal/repository/org_repo.go`

```go
package repository

import (
    "context"
    "database/sql"
    "strings"

    "github.com/google/uuid"
    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

type OrgRepository interface {
    Create(ctx context.Context, org *models.Organization, ownerID uuid.UUID) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)
    GetBySlug(ctx context.Context, slug string) (*models.Organization, error)
    Update(ctx context.Context, org *models.Organization) error
    Delete(ctx context.Context, id uuid.UUID) error
    
    // Members
    AddMember(ctx context.Context, orgID, userID uuid.UUID, role string, invitedBy *uuid.UUID) error
    RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error
    UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
    GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrgMember, error)
    ListMembers(ctx context.Context, orgID uuid.UUID) ([]*models.OrgMember, error)
    ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error)
    
    // Namespaces
    CreateNamespace(ctx context.Context, ns *models.Namespace) error
    GetNamespace(ctx context.Context, id uuid.UUID) (*models.Namespace, error)
    GetNamespaceByName(ctx context.Context, orgID uuid.UUID, name string) (*models.Namespace, error)
    ListNamespaces(ctx context.Context, orgID uuid.UUID) ([]*models.Namespace, error)
    DeleteNamespace(ctx context.Context, id uuid.UUID) error
    CountNamespaces(ctx context.Context, orgID uuid.UUID) (int, error)
}

type orgRepo struct {
    db *sql.DB
}

func NewOrgRepository(db *sql.DB) OrgRepository {
    return &orgRepo{db: db}
}

func (r *orgRepo) Create(ctx context.Context, org *models.Organization, ownerID uuid.UUID) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Generate slug from name
    org.ID = uuid.New()
    org.Slug = generateSlug(org.Name)
    org.Plan = models.PlanFree

    // Create org
    query := `
        INSERT INTO organizations (id, name, slug, plan)
        VALUES ($1, $2, $3, $4)
        RETURNING created_at, updated_at`
    
    err = tx.QueryRowContext(ctx, query, org.ID, org.Name, org.Slug, org.Plan).
        Scan(&org.CreatedAt, &org.UpdatedAt)
    if err != nil {
        return err
    }

    // Add owner
    memberQuery := `
        INSERT INTO org_members (org_id, user_id, role)
        VALUES ($1, $2, $3)`
    
    if _, err := tx.ExecContext(ctx, memberQuery, org.ID, ownerID, models.RoleOwner); err != nil {
        return err
    }

    // Create default namespace
    nsQuery := `
        INSERT INTO namespaces (id, org_id, name, description)
        VALUES ($1, $2, $3, $4)`
    
    if _, err := tx.ExecContext(ctx, nsQuery, uuid.New(), org.ID, "production", "Production environment"); err != nil {
        return err
    }

    return tx.Commit()
}

func (r *orgRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
    query := `
        SELECT id, name, slug, plan, stripe_customer_id, stripe_subscription_id,
               created_at, updated_at
        FROM organizations WHERE id = $1`
    
    var org models.Organization
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &org.ID, &org.Name, &org.Slug, &org.Plan,
        &org.StripeCustomerID, &org.StripeSubscriptionID,
        &org.CreatedAt, &org.UpdatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &org, err
}

func (r *orgRepo) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrgMember, error) {
    query := `
        SELECT org_id, user_id, role, invited_by, joined_at
        FROM org_members WHERE org_id = $1 AND user_id = $2`
    
    var m models.OrgMember
    err := r.db.QueryRowContext(ctx, query, orgID, userID).Scan(
        &m.OrgID, &m.UserID, &m.Role, &m.InvitedBy, &m.JoinedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &m, err
}

func (r *orgRepo) ListMembers(ctx context.Context, orgID uuid.UUID) ([]*models.OrgMember, error) {
    query := `
        SELECT m.org_id, m.user_id, m.role, m.invited_by, m.joined_at,
               u.id, u.email, u.name, u.avatar_url
        FROM org_members m
        JOIN users u ON m.user_id = u.id
        WHERE m.org_id = $1
        ORDER BY m.joined_at`
    
    rows, err := r.db.QueryContext(ctx, query, orgID)
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

func (r *orgRepo) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error) {
    query := `
        SELECT o.id, o.name, o.slug, o.plan, o.created_at, o.updated_at
        FROM organizations o
        JOIN org_members m ON o.id = m.org_id
        WHERE m.user_id = $1
        ORDER BY o.created_at`
    
    rows, err := r.db.QueryContext(ctx, query, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var orgs []*models.Organization
    for rows.Next() {
        var o models.Organization
        if err := rows.Scan(
            &o.ID, &o.Name, &o.Slug, &o.Plan, &o.CreatedAt, &o.UpdatedAt,
        ); err != nil {
            return nil, err
        }
        orgs = append(orgs, &o)
    }
    return orgs, rows.Err()
}

// ... implement remaining methods

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
```

---

## 5. Service

**File:** `internal/service/org_service.go`

```go
package service

import (
    "context"
    "fmt"

    "github.com/google/uuid"

    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
    "github.com/Bidon15/banhbaoring/control-plane/internal/repository"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type OrgService interface {
    Create(ctx context.Context, name string, ownerID uuid.UUID) (*models.Organization, error)
    Get(ctx context.Context, id uuid.UUID) (*models.Organization, error)
    Update(ctx context.Context, id uuid.UUID, name string) (*models.Organization, error)
    Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error
    ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error)
    
    // Members
    InviteMember(ctx context.Context, orgID uuid.UUID, email, role string, inviterID uuid.UUID) error
    RemoveMember(ctx context.Context, orgID, userID, actorID uuid.UUID) error
    UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string, actorID uuid.UUID) error
    ListMembers(ctx context.Context, orgID uuid.UUID) ([]*models.OrgMember, error)
    
    // Access control
    CheckAccess(ctx context.Context, orgID, userID uuid.UUID, requiredRole string) error
    GetLimits(ctx context.Context, orgID uuid.UUID) (*models.PlanLimits, error)
    
    // Namespaces
    CreateNamespace(ctx context.Context, orgID uuid.UUID, name, description string) (*models.Namespace, error)
    ListNamespaces(ctx context.Context, orgID uuid.UUID) ([]*models.Namespace, error)
    DeleteNamespace(ctx context.Context, orgID, nsID uuid.UUID) error
}

type orgService struct {
    orgRepo  repository.OrgRepository
    userRepo repository.UserRepository
}

func NewOrgService(orgRepo repository.OrgRepository, userRepo repository.UserRepository) OrgService {
    return &orgService{
        orgRepo:  orgRepo,
        userRepo: userRepo,
    }
}

func (s *orgService) Create(ctx context.Context, name string, ownerID uuid.UUID) (*models.Organization, error) {
    org := &models.Organization{Name: name}
    if err := s.orgRepo.Create(ctx, org, ownerID); err != nil {
        return nil, err
    }
    return org, nil
}

func (s *orgService) CheckAccess(ctx context.Context, orgID, userID uuid.UUID, requiredRole string) error {
    member, err := s.orgRepo.GetMember(ctx, orgID, userID)
    if err != nil {
        return err
    }
    if member == nil {
        return apierrors.ErrForbidden
    }

    // Role hierarchy: owner > admin > operator > viewer
    roleLevel := map[string]int{
        models.RoleOwner:    4,
        models.RoleAdmin:    3,
        models.RoleOperator: 2,
        models.RoleViewer:   1,
    }

    if roleLevel[member.Role] < roleLevel[requiredRole] {
        return apierrors.ErrForbidden
    }

    return nil
}

func (s *orgService) GetLimits(ctx context.Context, orgID uuid.UUID) (*models.PlanLimits, error) {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return nil, err
    }
    if org == nil {
        return nil, apierrors.NewNotFoundError("Organization")
    }

    limits := models.PlanLimitsMap[org.Plan]
    return &limits, nil
}

func (s *orgService) CreateNamespace(ctx context.Context, orgID uuid.UUID, name, description string) (*models.Namespace, error) {
    // Check limits
    limits, err := s.GetLimits(ctx, orgID)
    if err != nil {
        return nil, err
    }

    count, err := s.orgRepo.CountNamespaces(ctx, orgID)
    if err != nil {
        return nil, err
    }

    if limits.Namespaces > 0 && count >= limits.Namespaces {
        return nil, apierrors.ErrQuotaExceeded
    }

    ns := &models.Namespace{
        ID:          uuid.New(),
        OrgID:       orgID,
        Name:        name,
        Description: description,
    }

    if err := s.orgRepo.CreateNamespace(ctx, ns); err != nil {
        return nil, err
    }

    return ns, nil
}

// ... implement remaining methods
```

---

## 6. Deliverables

| File | Description |
|------|-------------|
| `internal/models/organization.go` | Org, Namespace, Member models |
| `internal/repository/org_repo.go` | Database operations |
| `internal/service/org_service.go` | Business logic |
| `internal/handler/org_handler.go` | HTTP handlers |
| `internal/handler/namespace_handler.go` | Namespace handlers |
| `internal/handler/member_handler.go` | Member handlers |

---

## 7. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/organizations` | Create org |
| GET | `/v1/organizations` | List user orgs |
| GET | `/v1/organizations/{id}` | Get org |
| PATCH | `/v1/organizations/{id}` | Update org |
| DELETE | `/v1/organizations/{id}` | Delete org |
| GET | `/v1/organizations/{id}/members` | List members |
| POST | `/v1/organizations/{id}/members` | Invite member |
| DELETE | `/v1/organizations/{id}/members/{userId}` | Remove member |
| PATCH | `/v1/organizations/{id}/members/{userId}` | Update role |
| GET | `/v1/namespaces` | List namespaces |
| POST | `/v1/namespaces` | Create namespace |
| DELETE | `/v1/namespaces/{id}` | Delete namespace |

---

## 8. Success Criteria

- [ ] Organization CRUD works
- [ ] Namespace CRUD works
- [ ] Member management works
- [ ] RBAC role hierarchy enforced
- [ ] Plan limits enforced
- [ ] Tests pass

---

## 9. Agent Prompt

```
You are Agent 09A - Organizations & Multi-Tenancy. Implement organization management.

Read the spec: doc/implementation/IMPL_09A_ORGS_NAMESPACES.md

Deliverables:
1. Organization, Namespace, OrgMember models
2. Repository with all CRUD operations
3. Service with RBAC enforcement
4. Plan limits checking
5. HTTP handlers
6. Tests

Dependencies: Agents 07, 08A/08B/08C must complete first.

Test: go test ./internal/... -v
```

