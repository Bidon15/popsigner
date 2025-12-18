package orchestrator

import (
	"context"

	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	"github.com/Bidon15/popsigner/control-plane/internal/service"
)

// KeyServiceResolver wraps a KeyService to implement the KeyResolver interface.
type KeyServiceResolver struct {
	keySvc service.KeyService
}

// NewKeyServiceResolver creates a new KeyServiceResolver.
func NewKeyServiceResolver(keySvc service.KeyService) *KeyServiceResolver {
	return &KeyServiceResolver{
		keySvc: keySvc,
	}
}

// Get returns a key by its org and key ID.
func (r *KeyServiceResolver) Get(ctx context.Context, orgID, keyID uuid.UUID) (*models.Key, error) {
	return r.keySvc.Get(ctx, orgID, keyID)
}

