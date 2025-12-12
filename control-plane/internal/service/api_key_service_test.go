package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
)

// mockAPIKeyRepo is a mock implementation of APIKeyRepository for testing.
type mockAPIKeyRepo struct {
	keys     map[uuid.UUID]*models.APIKey
	byPrefix map[string]*models.APIKey
	byHash   map[string]*models.APIKey
}

func newMockAPIKeyRepo() *mockAPIKeyRepo {
	return &mockAPIKeyRepo{
		keys:     make(map[uuid.UUID]*models.APIKey),
		byPrefix: make(map[string]*models.APIKey),
		byHash:   make(map[string]*models.APIKey),
	}
}

func (m *mockAPIKeyRepo) Create(ctx context.Context, key *models.APIKey) error {
	key.ID = uuid.New()
	key.CreatedAt = time.Now()
	m.keys[key.ID] = key
	m.byPrefix[key.KeyPrefix] = key
	m.byHash[key.KeyHash] = key
	return nil
}

func (m *mockAPIKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	return m.keys[id], nil
}

func (m *mockAPIKeyRepo) GetByPrefix(ctx context.Context, prefix string) (*models.APIKey, error) {
	return m.byPrefix[prefix], nil
}

func (m *mockAPIKeyRepo) GetByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	return m.byHash[hash], nil
}

func (m *mockAPIKeyRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error) {
	var result []*models.APIKey
	for _, key := range m.keys {
		if key.OrgID == orgID {
			result = append(result, key)
		}
	}
	return result, nil
}

func (m *mockAPIKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	if key, ok := m.keys[id]; ok {
		now := time.Now()
		key.LastUsedAt = &now
	}
	return nil
}

func (m *mockAPIKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	if key, ok := m.keys[id]; ok {
		now := time.Now()
		key.RevokedAt = &now
	}
	return nil
}

func (m *mockAPIKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if key, ok := m.keys[id]; ok {
		delete(m.byPrefix, key.KeyPrefix)
		delete(m.byHash, key.KeyHash)
		delete(m.keys, id)
	}
	return nil
}

func TestAPIKeyService_Create(t *testing.T) {
	repo := newMockAPIKeyRepo()
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("creates key with valid request", func(t *testing.T) {
		req := CreateAPIKeyRequest{
			Name:   "Test Key",
			Scopes: []string{"keys:read", "keys:write"},
		}

		key, rawKey, err := svc.Create(ctx, orgID, req)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Check key model
		if key == nil {
			t.Fatal("Create() returned nil key")
		}
		if key.Name != req.Name {
			t.Errorf("Name = %v, want %v", key.Name, req.Name)
		}
		if key.OrgID != orgID {
			t.Errorf("OrgID = %v, want %v", key.OrgID, orgID)
		}
		if len(key.Scopes) != len(req.Scopes) {
			t.Errorf("Scopes length = %v, want %v", len(key.Scopes), len(req.Scopes))
		}

		// Check raw key format
		if !strings.HasPrefix(rawKey, "bbr_live_") {
			t.Errorf("rawKey = %v, want prefix 'bbr_live_'", rawKey)
		}

		// Check prefix is stored
		if !strings.HasPrefix(key.KeyPrefix, "bbr_live_") {
			t.Errorf("KeyPrefix = %v, want prefix 'bbr_live_'", key.KeyPrefix)
		}
		if len(key.KeyPrefix) != len("bbr_live_")+8 {
			t.Errorf("KeyPrefix length = %v, want %v", len(key.KeyPrefix), len("bbr_live_")+8)
		}

		// Check hash is stored (not empty)
		if key.KeyHash == "" {
			t.Error("KeyHash is empty")
		}
	})

	t.Run("creates key with test environment", func(t *testing.T) {
		req := CreateAPIKeyRequest{
			Name:        "Test Key",
			Scopes:      []string{"keys:read"},
			Environment: "test",
		}

		key, rawKey, err := svc.Create(ctx, orgID, req)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if !strings.HasPrefix(rawKey, "bbr_test_") {
			t.Errorf("rawKey = %v, want prefix 'bbr_test_'", rawKey)
		}
		if !strings.HasPrefix(key.KeyPrefix, "bbr_test_") {
			t.Errorf("KeyPrefix = %v, want prefix 'bbr_test_'", key.KeyPrefix)
		}
	})

	t.Run("creates key with expiration", func(t *testing.T) {
		days := 30
		req := CreateAPIKeyRequest{
			Name:          "Expiring Key",
			Scopes:        []string{"keys:read"},
			ExpiresInDays: &days,
		}

		key, _, err := svc.Create(ctx, orgID, req)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if key.ExpiresAt == nil {
			t.Error("ExpiresAt is nil, expected expiration")
		}

		// Check expiration is approximately 30 days from now
		expectedExpiry := time.Now().AddDate(0, 0, 30)
		diff := key.ExpiresAt.Sub(expectedExpiry)
		if diff > time.Minute || diff < -time.Minute {
			t.Errorf("ExpiresAt = %v, want approximately %v", key.ExpiresAt, expectedExpiry)
		}
	})

	t.Run("rejects invalid scope", func(t *testing.T) {
		req := CreateAPIKeyRequest{
			Name:   "Test Key",
			Scopes: []string{"invalid:scope"},
		}

		_, _, err := svc.Create(ctx, orgID, req)
		if err == nil {
			t.Error("Create() expected error for invalid scope")
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		req := CreateAPIKeyRequest{
			Name:   "",
			Scopes: []string{"keys:read"},
		}

		_, _, err := svc.Create(ctx, orgID, req)
		if err == nil {
			t.Error("Create() expected error for empty name")
		}
	})

	t.Run("rejects invalid environment", func(t *testing.T) {
		req := CreateAPIKeyRequest{
			Name:        "Test Key",
			Scopes:      []string{"keys:read"},
			Environment: "production", // should be "live" or "test"
		}

		_, _, err := svc.Create(ctx, orgID, req)
		if err == nil {
			t.Error("Create() expected error for invalid environment")
		}
	})
}

func TestAPIKeyService_Validate(t *testing.T) {
	repo := newMockAPIKeyRepo()
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	orgID := uuid.New()

	// Create a key to validate
	req := CreateAPIKeyRequest{
		Name:   "Test Key",
		Scopes: []string{"keys:read", "keys:write"},
	}
	createdKey, rawKey, err := svc.Create(ctx, orgID, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("validates correct key", func(t *testing.T) {
		key, err := svc.Validate(ctx, rawKey)
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}

		if key.ID != createdKey.ID {
			t.Errorf("ID = %v, want %v", key.ID, createdKey.ID)
		}
	})

	t.Run("rejects invalid key format", func(t *testing.T) {
		_, err := svc.Validate(ctx, "invalid_key")
		if err == nil {
			t.Error("Validate() expected error for invalid format")
		}
	})

	t.Run("rejects wrong prefix", func(t *testing.T) {
		_, err := svc.Validate(ctx, "xxx_live_abcd1234567890")
		if err == nil {
			t.Error("Validate() expected error for wrong prefix")
		}
	})

	t.Run("rejects nonexistent key", func(t *testing.T) {
		_, err := svc.Validate(ctx, "bbr_live_nonexistent12345678")
		if err == nil {
			t.Error("Validate() expected error for nonexistent key")
		}
	})

	t.Run("rejects revoked key", func(t *testing.T) {
		// Create and revoke a key
		revokeReq := CreateAPIKeyRequest{
			Name:   "Revoked Key",
			Scopes: []string{"keys:read"},
		}
		revokedKey, revokedRawKey, _ := svc.Create(ctx, orgID, revokeReq)
		_ = svc.Revoke(ctx, orgID, revokedKey.ID)

		_, err := svc.Validate(ctx, revokedRawKey)
		if err == nil {
			t.Error("Validate() expected error for revoked key")
		}
	})
}

func TestAPIKeyService_List(t *testing.T) {
	repo := newMockAPIKeyRepo()
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	orgID := uuid.New()
	otherOrgID := uuid.New()

	// Create keys for different orgs
	for i := 0; i < 3; i++ {
		req := CreateAPIKeyRequest{
			Name:   "Test Key",
			Scopes: []string{"keys:read"},
		}
		_, _, _ = svc.Create(ctx, orgID, req)
	}

	// Create key for other org
	req := CreateAPIKeyRequest{
		Name:   "Other Org Key",
		Scopes: []string{"keys:read"},
	}
	_, _, _ = svc.Create(ctx, otherOrgID, req)

	t.Run("lists only org's keys", func(t *testing.T) {
		keys, err := svc.List(ctx, orgID)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(keys) != 3 {
			t.Errorf("List() returned %d keys, want 3", len(keys))
		}

		for _, key := range keys {
			if key.OrgID != orgID {
				t.Errorf("Key OrgID = %v, want %v", key.OrgID, orgID)
			}
		}
	})

	t.Run("returns empty list for org with no keys", func(t *testing.T) {
		emptyOrgID := uuid.New()
		keys, err := svc.List(ctx, emptyOrgID)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(keys) != 0 {
			t.Errorf("List() returned %d keys, want 0", len(keys))
		}
	})
}

func TestAPIKeyService_Revoke(t *testing.T) {
	repo := newMockAPIKeyRepo()
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	orgID := uuid.New()

	// Create a key
	req := CreateAPIKeyRequest{
		Name:   "Test Key",
		Scopes: []string{"keys:read"},
	}
	key, _, err := svc.Create(ctx, orgID, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("revokes existing key", func(t *testing.T) {
		err := svc.Revoke(ctx, orgID, key.ID)
		if err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}

		// Check key is revoked
		revokedKey, _ := svc.Get(ctx, orgID, key.ID)
		if revokedKey.RevokedAt == nil {
			t.Error("Key RevokedAt is nil after revocation")
		}
	})

	t.Run("rejects revoke for nonexistent key", func(t *testing.T) {
		err := svc.Revoke(ctx, orgID, uuid.New())
		if err == nil {
			t.Error("Revoke() expected error for nonexistent key")
		}
	})

	t.Run("rejects revoke for other org's key", func(t *testing.T) {
		// Create key for another org
		otherOrgID := uuid.New()
		otherKey, _, _ := svc.Create(ctx, otherOrgID, req)

		err := svc.Revoke(ctx, orgID, otherKey.ID)
		if err == nil {
			t.Error("Revoke() expected error for other org's key")
		}
	})
}

func TestAPIKeyService_Delete(t *testing.T) {
	repo := newMockAPIKeyRepo()
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	orgID := uuid.New()

	// Create a key
	req := CreateAPIKeyRequest{
		Name:   "Test Key",
		Scopes: []string{"keys:read"},
	}
	key, _, err := svc.Create(ctx, orgID, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("deletes existing key", func(t *testing.T) {
		err := svc.Delete(ctx, orgID, key.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Check key is deleted
		deletedKey, _ := svc.Get(ctx, orgID, key.ID)
		if deletedKey != nil {
			t.Error("Key still exists after deletion")
		}
	})

	t.Run("rejects delete for nonexistent key", func(t *testing.T) {
		err := svc.Delete(ctx, orgID, uuid.New())
		if err == nil {
			t.Error("Delete() expected error for nonexistent key")
		}
	})
}

func TestBase62Encode(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int // expected output length
	}{
		{
			name:     "24 bytes produces expected length",
			input:    make([]byte, 24),
			expected: 24,
		},
		{
			name:     "empty input produces empty output",
			input:    []byte{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := base62Encode(tt.input)
			if len(result) != tt.expected {
				t.Errorf("base62Encode() length = %v, want %v", len(result), tt.expected)
			}

			// Check all characters are in base62 alphabet
			for _, c := range result {
				if !strings.ContainsRune(base62Alphabet, c) {
					t.Errorf("base62Encode() contains invalid character: %c", c)
				}
			}
		})
	}
}

func TestAPIKeyService_HashAndVerify(t *testing.T) {
	svc := &apiKeyService{}

	t.Run("hash and verify roundtrip", func(t *testing.T) {
		rawKey := "bbr_live_testkey12345678901234567890"

		hash, err := svc.hashKey(rawKey)
		if err != nil {
			t.Fatalf("hashKey() error = %v", err)
		}

		if !svc.verifyKey(rawKey, hash) {
			t.Error("verifyKey() returned false for correct key")
		}
	})

	t.Run("verify rejects wrong key", func(t *testing.T) {
		rawKey := "bbr_live_testkey12345678901234567890"

		hash, err := svc.hashKey(rawKey)
		if err != nil {
			t.Fatalf("hashKey() error = %v", err)
		}

		wrongKey := "bbr_live_wrongkey12345678901234567890"
		if svc.verifyKey(wrongKey, hash) {
			t.Error("verifyKey() returned true for wrong key")
		}
	})

	t.Run("different hashes for same key (random salt)", func(t *testing.T) {
		rawKey := "bbr_live_testkey12345678901234567890"

		hash1, _ := svc.hashKey(rawKey)
		hash2, _ := svc.hashKey(rawKey)

		if hash1 == hash2 {
			t.Error("hashKey() produced same hash twice (salt should be random)")
		}

		// Both should still verify
		if !svc.verifyKey(rawKey, hash1) {
			t.Error("verifyKey() failed for hash1")
		}
		if !svc.verifyKey(rawKey, hash2) {
			t.Error("verifyKey() failed for hash2")
		}
	})

	t.Run("verify rejects malformed hash", func(t *testing.T) {
		if svc.verifyKey("any_key", "invalid_hash") {
			t.Error("verifyKey() should reject malformed hash")
		}
	})
}

