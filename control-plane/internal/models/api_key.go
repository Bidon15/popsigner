// Package models contains data models for the control plane.
package models

import (
	"time"

	"github.com/google/uuid"
)

// APIKey represents an API key for programmatic access.
type APIKey struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	OrgID      uuid.UUID  `json:"org_id" db:"org_id"`
	UserID     *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	Name       string     `json:"name" db:"name"`
	KeyPrefix  string     `json:"key_prefix" db:"key_prefix"` // bbr_live_xxxx (for display)
	KeyHash    string     `json:"-" db:"key_hash"`            // Argon2 hash
	Scopes     []string   `json:"scopes" db:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// APIKeyScopes defines the available API key scopes.
var APIKeyScopes = map[string]bool{
	"keys:read":      true,
	"keys:write":     true,
	"keys:sign":      true,
	"audit:read":     true,
	"billing:read":   true,
	"billing:write":  true,
	"webhooks:read":  true,
	"webhooks:write": true,
	"*":              true, // Wildcard scope for full access
}

// AllScopes returns all available scope names.
func AllScopes() []string {
	return []string{
		"keys:read",
		"keys:write",
		"keys:sign",
		"audit:read",
		"billing:read",
		"billing:write",
		"webhooks:read",
		"webhooks:write",
	}
}

// IsValidScope checks if a scope is valid.
func IsValidScope(scope string) bool {
	return APIKeyScopes[scope]
}

// ValidateScopes checks if all scopes in a list are valid.
func ValidateScopes(scopes []string) bool {
	for _, s := range scopes {
		if !IsValidScope(s) {
			return false
		}
	}
	return true
}

// IsValid checks if the API key is currently valid (not revoked or expired).
func (k *APIKey) IsValid() bool {
	if k.RevokedAt != nil {
		return false
	}
	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

// HasScope checks if the API key has a specific scope.
// Supports wildcard scope "*" which grants all permissions.
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// HasAnyScope checks if the API key has any of the specified scopes.
func (k *APIKey) HasAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if k.HasScope(scope) {
			return true
		}
	}
	return false
}

// APIKeyResponse is the response format for API key operations.
// It includes the full key only on creation.
type APIKeyResponse struct {
	ID         uuid.UUID  `json:"id"`
	OrgID      uuid.UUID  `json:"org_id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Key        string     `json:"key,omitempty"` // Only set on creation
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ToResponse converts an APIKey to an APIKeyResponse.
func (k *APIKey) ToResponse() *APIKeyResponse {
	return &APIKeyResponse{
		ID:         k.ID,
		OrgID:      k.OrgID,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		Scopes:     k.Scopes,
		LastUsedAt: k.LastUsedAt,
		ExpiresAt:  k.ExpiresAt,
		RevokedAt:  k.RevokedAt,
		CreatedAt:  k.CreatedAt,
	}
}

// CreateAPIKeyResponse is the response for API key creation.
// Contains the full key which will not be shown again.
type CreateAPIKeyResponse struct {
	APIKey  *APIKeyResponse `json:"api_key"`
	Key     string          `json:"key"`
	Warning string          `json:"warning"`
}
