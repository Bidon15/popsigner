package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAPIKey_IsValid(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name     string
		key      *APIKey
		expected bool
	}{
		{
			name: "valid key with no expiration",
			key: &APIKey{
				ID:        uuid.New(),
				OrgID:     uuid.New(),
				Name:      "Test Key",
				Scopes:    []string{"keys:read"},
				CreatedAt: now,
			},
			expected: true,
		},
		{
			name: "valid key with future expiration",
			key: &APIKey{
				ID:        uuid.New(),
				OrgID:     uuid.New(),
				Name:      "Test Key",
				Scopes:    []string{"keys:read"},
				ExpiresAt: &future,
				CreatedAt: now,
			},
			expected: true,
		},
		{
			name: "expired key",
			key: &APIKey{
				ID:        uuid.New(),
				OrgID:     uuid.New(),
				Name:      "Test Key",
				Scopes:    []string{"keys:read"},
				ExpiresAt: &past,
				CreatedAt: now,
			},
			expected: false,
		},
		{
			name: "revoked key",
			key: &APIKey{
				ID:        uuid.New(),
				OrgID:     uuid.New(),
				Name:      "Test Key",
				Scopes:    []string{"keys:read"},
				RevokedAt: &past,
				CreatedAt: now,
			},
			expected: false,
		},
		{
			name: "revoked and expired key",
			key: &APIKey{
				ID:        uuid.New(),
				OrgID:     uuid.New(),
				Name:      "Test Key",
				Scopes:    []string{"keys:read"},
				ExpiresAt: &past,
				RevokedAt: &past,
				CreatedAt: now,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.key.IsValid()
			if got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAPIKey_HasScope(t *testing.T) {
	tests := []struct {
		name       string
		keyScopes  []string
		checkScope string
		expected   bool
	}{
		{
			name:       "has exact scope",
			keyScopes:  []string{"keys:read", "keys:write"},
			checkScope: "keys:read",
			expected:   true,
		},
		{
			name:       "does not have scope",
			keyScopes:  []string{"keys:read"},
			checkScope: "keys:write",
			expected:   false,
		},
		{
			name:       "wildcard scope grants all",
			keyScopes:  []string{"*"},
			checkScope: "keys:read",
			expected:   true,
		},
		{
			name:       "wildcard scope grants billing",
			keyScopes:  []string{"*"},
			checkScope: "billing:write",
			expected:   true,
		},
		{
			name:       "empty scopes",
			keyScopes:  []string{},
			checkScope: "keys:read",
			expected:   false,
		},
		{
			name:       "nil scopes",
			keyScopes:  nil,
			checkScope: "keys:read",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{
				ID:     uuid.New(),
				OrgID:  uuid.New(),
				Name:   "Test Key",
				Scopes: tt.keyScopes,
			}
			got := key.HasScope(tt.checkScope)
			if got != tt.expected {
				t.Errorf("HasScope(%q) = %v, want %v", tt.checkScope, got, tt.expected)
			}
		})
	}
}

func TestAPIKey_HasAnyScope(t *testing.T) {
	tests := []struct {
		name        string
		keyScopes   []string
		checkScopes []string
		expected    bool
	}{
		{
			name:        "has one of the scopes",
			keyScopes:   []string{"keys:read", "keys:write"},
			checkScopes: []string{"keys:write", "audit:read"},
			expected:    true,
		},
		{
			name:        "has none of the scopes",
			keyScopes:   []string{"keys:read"},
			checkScopes: []string{"keys:write", "audit:read"},
			expected:    false,
		},
		{
			name:        "wildcard grants any scope",
			keyScopes:   []string{"*"},
			checkScopes: []string{"keys:write", "audit:read"},
			expected:    true,
		},
		{
			name:        "empty check scopes",
			keyScopes:   []string{"keys:read"},
			checkScopes: []string{},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{
				ID:     uuid.New(),
				OrgID:  uuid.New(),
				Name:   "Test Key",
				Scopes: tt.keyScopes,
			}
			got := key.HasAnyScope(tt.checkScopes...)
			if got != tt.expected {
				t.Errorf("HasAnyScope(%v) = %v, want %v", tt.checkScopes, got, tt.expected)
			}
		})
	}
}

func TestIsValidScope(t *testing.T) {
	tests := []struct {
		scope    string
		expected bool
	}{
		{"keys:read", true},
		{"keys:write", true},
		{"keys:sign", true},
		{"audit:read", true},
		{"billing:read", true},
		{"billing:write", true},
		{"webhooks:read", true},
		{"webhooks:write", true},
		{"*", true},
		{"invalid:scope", false},
		{"", false},
		{"keys", false},
		{"KEYS:READ", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			got := IsValidScope(tt.scope)
			if got != tt.expected {
				t.Errorf("IsValidScope(%q) = %v, want %v", tt.scope, got, tt.expected)
			}
		})
	}
}

func TestValidateScopes(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		expected bool
	}{
		{
			name:     "all valid scopes",
			scopes:   []string{"keys:read", "keys:write"},
			expected: true,
		},
		{
			name:     "one invalid scope",
			scopes:   []string{"keys:read", "invalid:scope"},
			expected: false,
		},
		{
			name:     "empty scopes",
			scopes:   []string{},
			expected: true,
		},
		{
			name:     "wildcard scope",
			scopes:   []string{"*"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateScopes(tt.scopes)
			if got != tt.expected {
				t.Errorf("ValidateScopes(%v) = %v, want %v", tt.scopes, got, tt.expected)
			}
		})
	}
}

func TestAPIKey_ToResponse(t *testing.T) {
	now := time.Now()
	expires := now.Add(24 * time.Hour)
	lastUsed := now.Add(-time.Hour)

	key := &APIKey{
		ID:         uuid.New(),
		OrgID:      uuid.New(),
		Name:       "Test Key",
		KeyPrefix:  "bbr_live_abcd1234",
		KeyHash:    "secret_hash_should_not_appear",
		Scopes:     []string{"keys:read", "keys:write"},
		LastUsedAt: &lastUsed,
		ExpiresAt:  &expires,
		CreatedAt:  now,
	}

	resp := key.ToResponse()

	if resp.ID != key.ID {
		t.Errorf("ID = %v, want %v", resp.ID, key.ID)
	}
	if resp.OrgID != key.OrgID {
		t.Errorf("OrgID = %v, want %v", resp.OrgID, key.OrgID)
	}
	if resp.Name != key.Name {
		t.Errorf("Name = %v, want %v", resp.Name, key.Name)
	}
	if resp.KeyPrefix != key.KeyPrefix {
		t.Errorf("KeyPrefix = %v, want %v", resp.KeyPrefix, key.KeyPrefix)
	}
	if resp.Key != "" {
		t.Error("Key should be empty in response")
	}
	if len(resp.Scopes) != len(key.Scopes) {
		t.Errorf("Scopes length = %v, want %v", len(resp.Scopes), len(key.Scopes))
	}
}

func TestAllScopes(t *testing.T) {
	scopes := AllScopes()

	expectedScopes := []string{
		"keys:read",
		"keys:write",
		"keys:sign",
		"audit:read",
		"billing:read",
		"billing:write",
		"webhooks:read",
		"webhooks:write",
	}

	if len(scopes) != len(expectedScopes) {
		t.Errorf("AllScopes() returned %d scopes, want %d", len(scopes), len(expectedScopes))
	}

	for _, expected := range expectedScopes {
		found := false
		for _, got := range scopes {
			if got == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllScopes() missing scope %q", expected)
		}
	}
}

