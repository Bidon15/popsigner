// Package models defines the data models for the Control Plane API.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Plan represents a subscription plan.
type Plan string

const (
	PlanFree       Plan = "free"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

// Organization represents a tenant in the system.
type Organization struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	Name                 string    `json:"name" db:"name"`
	Slug                 string    `json:"slug" db:"slug"`
	Plan                 Plan      `json:"plan" db:"plan"`
	StripeCustomerID     *string   `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	StripeSubscriptionID *string   `json:"stripe_subscription_id,omitempty" db:"stripe_subscription_id"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

// OrgMember represents a user's membership in an organization.
type OrgMember struct {
	OrgID     uuid.UUID  `json:"org_id" db:"org_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Role      Role       `json:"role" db:"role"`
	InvitedBy *uuid.UUID `json:"invited_by,omitempty" db:"invited_by"`
	JoinedAt  time.Time  `json:"joined_at" db:"joined_at"`

	// Joined fields (populated by queries)
	User *User `json:"user,omitempty"`
}

// Role represents a user's role within an organization.
type Role string

const (
	RoleOwner    Role = "owner"
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

// RoleLevel returns the numeric level for a role (higher = more permissions).
func RoleLevel(role Role) int {
	levels := map[Role]int{
		RoleOwner:    4,
		RoleAdmin:    3,
		RoleOperator: 2,
		RoleViewer:   1,
	}
	if level, ok := levels[role]; ok {
		return level
	}
	return 0
}

// ValidRole checks if a role string is valid.
func ValidRole(role Role) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleOperator, RoleViewer:
		return true
	}
	return false
}

// Namespace represents an environment within an organization.
type Namespace struct {
	ID          uuid.UUID `json:"id" db:"id"`
	OrgID       uuid.UUID `json:"org_id" db:"org_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Invitation represents a pending invitation to join an organization.
type Invitation struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	OrgID      uuid.UUID  `json:"org_id" db:"org_id"`
	Email      string     `json:"email" db:"email"`
	Role       Role       `json:"role" db:"role"`
	Token      string     `json:"-" db:"token"`
	InvitedBy  uuid.UUID  `json:"invited_by" db:"invited_by"`
	ExpiresAt  time.Time  `json:"expires_at" db:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty" db:"accepted_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`

	// Joined fields
	Organization *Organization `json:"organization,omitempty"`
}

// PlanLimits defines resource limits for a subscription plan.
type PlanLimits struct {
	Keys               int   `json:"keys"`
	SignaturesPerMonth int64 `json:"signatures_per_month"`
	Namespaces         int   `json:"namespaces"`
	TeamMembers        int   `json:"team_members"`
	AuditRetentionDays int   `json:"audit_retention_days"`
}

// PlanLimitsMap contains the limits for each plan.
// A value of -1 means unlimited.
var PlanLimitsMap = map[Plan]PlanLimits{
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

// GetPlanLimits returns the limits for a given plan.
func GetPlanLimits(plan Plan) PlanLimits {
	if limits, ok := PlanLimitsMap[plan]; ok {
		return limits
	}
	// Default to free plan limits
	return PlanLimitsMap[PlanFree]
}

