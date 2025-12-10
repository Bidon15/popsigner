package secp256k1

import (
	"context"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/require"
)

// getTestBackend creates a backend for testing with an in-memory storage.
// Returns both the backend and the storage for use in tests.
func getTestBackend(t *testing.T) (*backend, logical.Storage) {
	t.Helper()

	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}

	b, err := Factory(context.Background(), config)
	require.NoError(t, err)
	require.NotNil(t, b)

	return b.(*backend), config.StorageView
}

