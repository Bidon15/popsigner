package orchestrator

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/service"
)

// DefaultAPIKeyManager implements APIKeyManager using the existing APIKeyService.
// It creates deployment-specific API keys with full signing permissions.
type DefaultAPIKeyManager struct {
	apiKeySvc service.APIKeyService
}

// NewDefaultAPIKeyManager creates a new DefaultAPIKeyManager.
func NewDefaultAPIKeyManager(apiKeySvc service.APIKeyService) *DefaultAPIKeyManager {
	return &DefaultAPIKeyManager{
		apiKeySvc: apiKeySvc,
	}
}

// GetOrCreateForDeployment creates an API key for deployment use.
// Each deployment gets a fresh API key with sign and keys:read permissions.
func (m *DefaultAPIKeyManager) GetOrCreateForDeployment(ctx context.Context, orgID uuid.UUID) (string, error) {
	// Check if there's an existing deployment key for this org
	keys, err := m.apiKeySvc.List(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("list API keys: %w", err)
	}

	// Look for an existing deployment key
	for _, key := range keys {
		if key.Name == "Deployment Orchestrator" && key.RevokedAt == nil {
			// Found an existing deployment key, but we can't retrieve the raw key
			// We need to create a new one since raw keys are only shown at creation
			// For now, continue to create a new one
			break
		}
	}

	// Create a new deployment API key
	req := service.CreateAPIKeyRequest{
		Name:   "Deployment Orchestrator",
		Scopes: []string{"keys:sign", "keys:read"},
	}

	apiKey, rawKey, err := m.apiKeySvc.Create(ctx, orgID, req)
	if err != nil {
		return "", fmt.Errorf("create API key: %w", err)
	}

	_ = apiKey // We don't need the key metadata, just the raw key
	return rawKey, nil
}

