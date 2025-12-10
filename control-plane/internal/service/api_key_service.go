// Package service provides business logic implementations.
package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
	apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
	"github.com/Bidon15/banhbaoring/control-plane/internal/repository"
)

// Argon2 parameters for key hashing.
// These are tuned for security while keeping validation fast.
const (
	argon2Time    = 1        // Number of iterations
	argon2Memory  = 64 * 1024 // 64 MB memory
	argon2Threads = 4        // Number of parallel threads
	argon2KeyLen  = 32       // Output key length
	argon2SaltLen = 16       // Salt length
)

// API key format constants.
const (
	keyPrefixBanhbao = "bbr"   // banhbaoring prefix
	keyEnvLive       = "live"  // Production environment
	keyEnvTest       = "test"  // Test environment
	keySecretLen     = 24      // Random bytes for secret (produces ~32 base62 chars)
	keyPrefixDisplay = 8       // Characters of secret to show in prefix
)

// APIKeyService defines the interface for API key operations.
type APIKeyService interface {
	Create(ctx context.Context, orgID uuid.UUID, req CreateAPIKeyRequest) (*models.APIKey, string, error)
	Validate(ctx context.Context, rawKey string) (*models.APIKey, error)
	List(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error)
	Get(ctx context.Context, orgID, keyID uuid.UUID) (*models.APIKey, error)
	Revoke(ctx context.Context, orgID, keyID uuid.UUID) error
	Delete(ctx context.Context, orgID, keyID uuid.UUID) error
}

// CreateAPIKeyRequest is the request for creating a new API key.
type CreateAPIKeyRequest struct {
	Name         string   `json:"name" validate:"required,min=1,max=255"`
	Scopes       []string `json:"scopes" validate:"required,min=1"`
	ExpiresInDays *int    `json:"expires_in_days,omitempty"` // Days until expiry, nil = no expiry
	Environment  string   `json:"environment,omitempty"`     // "live" or "test", defaults to "live"
}

type apiKeyService struct {
	keyRepo repository.APIKeyRepository
}

// NewAPIKeyService creates a new API key service.
func NewAPIKeyService(keyRepo repository.APIKeyRepository) APIKeyService {
	return &apiKeyService{keyRepo: keyRepo}
}

// Create generates a new API key for an organization.
// Returns the key model and the raw key. The raw key is only shown once.
func (s *apiKeyService) Create(ctx context.Context, orgID uuid.UUID, req CreateAPIKeyRequest) (*models.APIKey, string, error) {
	// Validate scopes
	for _, scope := range req.Scopes {
		if !models.IsValidScope(scope) {
			return nil, "", apierrors.NewValidationError("scopes", fmt.Sprintf("invalid scope: %s", scope))
		}
	}

	// Validate name
	if strings.TrimSpace(req.Name) == "" {
		return nil, "", apierrors.NewValidationError("name", "name is required")
	}
	if len(req.Name) > 255 {
		return nil, "", apierrors.NewValidationError("name", "name must be 255 characters or less")
	}

	// Determine environment
	env := keyEnvLive
	if req.Environment != "" {
		if req.Environment != keyEnvLive && req.Environment != keyEnvTest {
			return nil, "", apierrors.NewValidationError("environment", "must be 'live' or 'test'")
		}
		env = req.Environment
	}

	// Generate raw key
	rawKey, prefix, err := s.generateKey(env)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Hash the key with Argon2
	hash, err := s.hashKey(rawKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash key: %w", err)
	}

	key := &models.APIKey{
		OrgID:     orgID,
		Name:      req.Name,
		KeyPrefix: prefix,
		KeyHash:   hash,
		Scopes:    req.Scopes,
	}

	// Set expiration if specified
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		exp := time.Now().AddDate(0, 0, *req.ExpiresInDays)
		key.ExpiresAt = &exp
	}

	if err := s.keyRepo.Create(ctx, key); err != nil {
		return nil, "", fmt.Errorf("failed to create key: %w", err)
	}

	// Return full key only once - it won't be retrievable later
	return key, rawKey, nil
}

// Validate validates an API key and returns the associated key model.
func (s *apiKeyService) Validate(ctx context.Context, rawKey string) (*models.APIKey, error) {
	// Parse the key format: bbr_<env>_<secret>
	parts := strings.Split(rawKey, "_")
	if len(parts) != 3 {
		return nil, apierrors.ErrUnauthorized
	}

	if parts[0] != keyPrefixBanhbao {
		return nil, apierrors.ErrUnauthorized
	}

	if parts[1] != keyEnvLive && parts[1] != keyEnvTest {
		return nil, apierrors.ErrUnauthorized
	}

	secret := parts[2]
	if len(secret) < keyPrefixDisplay {
		return nil, apierrors.ErrUnauthorized
	}

	// Build prefix for lookup: bbr_<env>_<first 8 chars>
	prefix := fmt.Sprintf("%s_%s_%s", parts[0], parts[1], secret[:keyPrefixDisplay])

	// Lookup by prefix
	key, err := s.keyRepo.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup key: %w", err)
	}
	if key == nil {
		return nil, apierrors.ErrUnauthorized
	}

	// Verify the hash using constant-time comparison
	if !s.verifyKey(rawKey, key.KeyHash) {
		return nil, apierrors.ErrUnauthorized
	}

	// Check validity (not revoked, not expired)
	if !key.IsValid() {
		return nil, apierrors.ErrUnauthorized
	}

	// Update last used asynchronously (don't block the request)
	go func() {
		_ = s.keyRepo.UpdateLastUsed(context.Background(), key.ID)
	}()

	return key, nil
}

// List returns all API keys for an organization.
func (s *apiKeyService) List(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error) {
	keys, err := s.keyRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	return keys, nil
}

// Get returns a specific API key if it belongs to the organization.
func (s *apiKeyService) Get(ctx context.Context, orgID, keyID uuid.UUID) (*models.APIKey, error) {
	key, err := s.keyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}
	if key == nil || key.OrgID != orgID {
		return nil, apierrors.NewNotFoundError("API key")
	}
	return key, nil
}

// Revoke marks an API key as revoked.
func (s *apiKeyService) Revoke(ctx context.Context, orgID, keyID uuid.UUID) error {
	key, err := s.keyRepo.GetByID(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}
	if key == nil || key.OrgID != orgID {
		return apierrors.NewNotFoundError("API key")
	}
	if key.RevokedAt != nil {
		return apierrors.NewConflictError("API key is already revoked")
	}
	if err := s.keyRepo.Revoke(ctx, keyID); err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}
	return nil
}

// Delete permanently removes an API key.
func (s *apiKeyService) Delete(ctx context.Context, orgID, keyID uuid.UUID) error {
	key, err := s.keyRepo.GetByID(ctx, keyID)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}
	if key == nil || key.OrgID != orgID {
		return apierrors.NewNotFoundError("API key")
	}
	if err := s.keyRepo.Delete(ctx, keyID); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	return nil
}

// generateKey generates a new API key in the format bbr_<env>_<secret>.
// Returns the full raw key and the display prefix.
func (s *apiKeyService) generateKey(env string) (rawKey, prefix string, err error) {
	// Generate random bytes for the secret
	secretBytes := make([]byte, keySecretLen)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode to base62 for URL-safe string
	secret := base62Encode(secretBytes)

	// Build the full key: bbr_<env>_<secret>
	rawKey = fmt.Sprintf("%s_%s_%s", keyPrefixBanhbao, env, secret)

	// Build the display prefix: bbr_<env>_<first 8 chars>
	displaySecret := secret
	if len(displaySecret) > keyPrefixDisplay {
		displaySecret = displaySecret[:keyPrefixDisplay]
	}
	prefix = fmt.Sprintf("%s_%s_%s", keyPrefixBanhbao, env, displaySecret)

	return rawKey, prefix, nil
}

// hashKey hashes an API key using Argon2id.
// The hash includes a random salt for each key.
func (s *apiKeyService) hashKey(rawKey string) (string, error) {
	// Generate random salt
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash with Argon2id
	hash := argon2.IDKey([]byte(rawKey), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Encode salt and hash together for storage
	// Format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// verifyKey verifies an API key against a stored hash using constant-time comparison.
func (s *apiKeyService) verifyKey(rawKey, encodedHash string) bool {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" {
		return false
	}

	// Parse parameters
	var memory, time, threads uint32
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false
	}

	// Decode salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	// Decode expected hash
	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Compute hash of provided key with same parameters
	computedHash := argon2.IDKey([]byte(rawKey), salt, time, memory, uint8(threads), uint32(len(expectedHash)))

	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(computedHash, expectedHash) == 1
}

// base62Alphabet for URL-safe encoding.
const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// base62Encode encodes bytes to a base62 string.
// This produces a URL-safe string without special characters.
func base62Encode(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// Simple encoding: map each byte to base62
	result := make([]byte, 0, len(data)*4/3+1)
	for _, b := range data {
		result = append(result, base62Alphabet[int(b)%62])
	}

	return string(result)
}

