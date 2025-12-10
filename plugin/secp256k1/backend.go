package secp256k1

import (
	"context"

	"github.com/openbao/openbao/sdk/v2/logical"
)

// TODO(03A): Implement backend

// Factory creates a new backend.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	panic("TODO(03A): implement Factory")
}
