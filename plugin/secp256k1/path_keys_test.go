package secp256k1

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathKeyCreate(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	t.Run("creates key successfully", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/test-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "test-key",
				"exportable": false,
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.IsError(), "response should not be error: %v", resp.Error())

		// Verify response data
		assert.Equal(t, "test-key", resp.Data["name"])
		assert.NotEmpty(t, resp.Data["public_key"])
		assert.NotEmpty(t, resp.Data["address"])
		assert.Equal(t, false, resp.Data["exportable"])
		assert.NotEmpty(t, resp.Data["created_at"])

		// Verify public key is valid hex
		pubKeyHex := resp.Data["public_key"].(string)
		pubKeyBytes, err := hex.DecodeString(pubKeyHex)
		require.NoError(t, err)
		assert.Len(t, pubKeyBytes, 33, "compressed public key should be 33 bytes")

		// Verify address is valid hex (20 bytes = 40 hex chars)
		addrHex := resp.Data["address"].(string)
		addrBytes, err := hex.DecodeString(addrHex)
		require.NoError(t, err)
		assert.Len(t, addrBytes, 20, "Cosmos address should be 20 bytes")
	})

	t.Run("creates exportable key", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/exportable-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "exportable-key",
				"exportable": true,
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.IsError())
		assert.Equal(t, true, resp.Data["exportable"])
	})

	t.Run("rejects duplicate key name", func(t *testing.T) {
		// First create
		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/duplicate-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "duplicate-key",
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.IsError())

		// Second create with same name
		resp, err = b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "key already exists")
	})

	t.Run("creates key via update operation", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/update-created-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "update-created-key",
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.IsError())
		assert.Equal(t, "update-created-key", resp.Data["name"])
	})

}

func TestPathKeyRead(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Create a test key first
	createReq := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "keys/read-test-key",
		Storage:   storage,
		Data: map[string]interface{}{
			"name":       "read-test-key",
			"exportable": true,
		},
	}
	createResp, err := b.HandleRequest(ctx, createReq)
	require.NoError(t, err)
	require.NotNil(t, createResp)
	require.False(t, createResp.IsError())

	t.Run("reads key successfully", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "keys/read-test-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "read-test-key",
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.IsError())

		// Verify response matches created key
		assert.Equal(t, "read-test-key", resp.Data["name"])
		assert.Equal(t, createResp.Data["public_key"], resp.Data["public_key"])
		assert.Equal(t, createResp.Data["address"], resp.Data["address"])
		assert.Equal(t, true, resp.Data["exportable"])
		assert.Equal(t, false, resp.Data["imported"])
		assert.NotEmpty(t, resp.Data["created_at"])
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "keys/non-existent-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "non-existent-key",
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "key not found")
	})

	t.Run("reads from cache on second read", func(t *testing.T) {
		// First read populates cache
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "keys/read-test-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "read-test-key",
			},
		}

		resp1, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp1)

		// Second read should use cache
		resp2, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp2)

		// Responses should be identical
		assert.Equal(t, resp1.Data["public_key"], resp2.Data["public_key"])
	})
}

func TestPathKeyDelete(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	t.Run("deletes key successfully", func(t *testing.T) {
		// Create a key
		createReq := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/delete-test-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "delete-test-key",
			},
		}
		_, err := b.HandleRequest(ctx, createReq)
		require.NoError(t, err)

		// Delete the key
		deleteReq := &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      "keys/delete-test-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "delete-test-key",
			},
		}

		resp, err := b.HandleRequest(ctx, deleteReq)
		require.NoError(t, err)
		assert.Nil(t, resp) // Delete returns nil response on success

		// Verify key is gone
		readReq := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "keys/delete-test-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "delete-test-key",
			},
		}

		readResp, err := b.HandleRequest(ctx, readReq)
		require.NoError(t, err)
		require.NotNil(t, readResp)
		assert.True(t, readResp.IsError())
		assert.Contains(t, readResp.Error().Error(), "key not found")
	})

	t.Run("deletes key from cache", func(t *testing.T) {
		// Create and read a key (to populate cache)
		createReq := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/cache-delete-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "cache-delete-key",
			},
		}
		_, err := b.HandleRequest(ctx, createReq)
		require.NoError(t, err)

		// Verify it's in cache
		cached := b.getKeyFromCache("cache-delete-key")
		assert.NotNil(t, cached, "key should be in cache after create")

		// Delete the key
		deleteReq := &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      "keys/cache-delete-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "cache-delete-key",
			},
		}
		_, err = b.HandleRequest(ctx, deleteReq)
		require.NoError(t, err)

		// Verify it's not in cache
		cached = b.getKeyFromCache("cache-delete-key")
		assert.Nil(t, cached, "key should not be in cache after delete")
	})

	t.Run("delete non-existent key succeeds silently", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.DeleteOperation,
			Path:      "keys/non-existent-delete-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "non-existent-delete-key",
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		assert.Nil(t, resp) // Delete should succeed silently
	})
}

func TestPathKeysList(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	t.Run("lists empty keys", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.ListOperation,
			Path:      "keys/",
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.IsError())

		// Keys might be nil or empty slice when no keys exist
		if resp.Data["keys"] != nil {
			keys := resp.Data["keys"].([]string)
			assert.Empty(t, keys)
		}
	})

	t.Run("lists multiple keys", func(t *testing.T) {
		// Create multiple keys
		keyNames := []string{"list-key-1", "list-key-2", "list-key-3"}
		for _, name := range keyNames {
			createReq := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "keys/" + name,
				Storage:   storage,
				Data: map[string]interface{}{
					"name": name,
				},
			}
			_, err := b.HandleRequest(ctx, createReq)
			require.NoError(t, err)
		}

		// List keys
		req := &logical.Request{
			Operation: logical.ListOperation,
			Path:      "keys/",
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.IsError())

		keys := resp.Data["keys"].([]string)
		for _, name := range keyNames {
			assert.Contains(t, keys, name)
		}
	})
}

func TestPathKeyExistenceCheck(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	t.Run("returns false for non-existent key", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/existence-check-key",
			Storage:   storage,
		}

		fd := &framework.FieldData{
			Raw: map[string]interface{}{
				"name": "existence-check-key",
			},
			Schema: map[string]*framework.FieldSchema{
				"name": {Type: framework.TypeString},
			},
		}

		exists, err := b.pathKeyExistenceCheck(ctx, req, fd)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns true for existing key", func(t *testing.T) {
		// Create a key
		createReq := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/exists-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "exists-key",
			},
		}
		_, err := b.HandleRequest(ctx, createReq)
		require.NoError(t, err)

		req := &logical.Request{
			Storage: storage,
		}

		fd := &framework.FieldData{
			Raw: map[string]interface{}{
				"name": "exists-key",
			},
			Schema: map[string]*framework.FieldSchema{
				"name": {Type: framework.TypeString},
			},
		}

		exists, err := b.pathKeyExistenceCheck(ctx, req, fd)
		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestGetKey(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	t.Run("returns nil for non-existent key", func(t *testing.T) {
		entry, err := b.getKey(ctx, storage, "non-existent")
		require.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("returns key from storage", func(t *testing.T) {
		// Create a key
		createReq := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/getkey-test",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "getkey-test",
				"exportable": true,
			},
		}
		_, err := b.HandleRequest(ctx, createReq)
		require.NoError(t, err)

		// Clear cache to force storage read
		b.deleteKeyFromCache("getkey-test")

		entry, err := b.getKey(ctx, storage, "getkey-test")
		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.True(t, entry.Exportable)
		assert.False(t, entry.Imported)
		assert.Len(t, entry.PrivateKey, 32)
		assert.Len(t, entry.PublicKey, 33)
	})

	t.Run("returns key from cache", func(t *testing.T) {
		// Create a key (which populates cache)
		createReq := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/cached-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "cached-key",
			},
		}
		_, err := b.HandleRequest(ctx, createReq)
		require.NoError(t, err)

		// Verify it's in cache
		cachedEntry := b.getKeyFromCache("cached-key")
		require.NotNil(t, cachedEntry)

		// Get key (should use cache)
		entry, err := b.getKey(ctx, storage, "cached-key")
		require.NoError(t, err)
		require.NotNil(t, entry)

		// Should be the same pointer from cache
		assert.Equal(t, cachedEntry, entry)
	})
}

func TestKeyEntry(t *testing.T) {
	t.Run("stores all fields correctly", func(t *testing.T) {
		now := time.Now().UTC()
		entry := &keyEntry{
			PrivateKey: make([]byte, 32),
			PublicKey:  make([]byte, 33),
			Exportable: true,
			CreatedAt:  now,
			Imported:   true,
		}

		assert.Equal(t, 32, len(entry.PrivateKey))
		assert.Equal(t, 33, len(entry.PublicKey))
		assert.True(t, entry.Exportable)
		assert.Equal(t, now, entry.CreatedAt)
		assert.True(t, entry.Imported)
	})
}

func TestPathKeysDefinition(t *testing.T) {
	b, _ := getTestBackend(t)

	paths := pathKeys(b)
	require.Len(t, paths, 2)

	t.Run("key CRUD path", func(t *testing.T) {
		crudPath := paths[0]
		assert.Contains(t, crudPath.Pattern, "keys/")
		assert.NotNil(t, crudPath.Operations[logical.CreateOperation])
		assert.NotNil(t, crudPath.Operations[logical.UpdateOperation])
		assert.NotNil(t, crudPath.Operations[logical.ReadOperation])
		assert.NotNil(t, crudPath.Operations[logical.DeleteOperation])
		assert.NotNil(t, crudPath.Fields["name"])
		assert.NotNil(t, crudPath.Fields["exportable"])
	})

	t.Run("key list path", func(t *testing.T) {
		listPath := paths[1]
		assert.Equal(t, "keys/?$", listPath.Pattern)
		assert.NotNil(t, listPath.Operations[logical.ListOperation])
	})
}

func TestKeyCreationTimestamp(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Truncate to second precision since RFC3339 only has second precision
	before := time.Now().UTC().Truncate(time.Second)

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "keys/timestamp-key",
		Storage:   storage,
		Data: map[string]interface{}{
			"name": "timestamp-key",
		},
	}

	resp, err := b.HandleRequest(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	after := time.Now().UTC().Add(time.Second) // Add a second buffer

	createdAtStr := resp.Data["created_at"].(string)
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	require.NoError(t, err)

	assert.True(t, !createdAt.Before(before), "created_at should not be before test start")
	assert.True(t, !createdAt.After(after), "created_at should not be after test end")
}

func TestAddressDerivation(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Create a key
	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "keys/addr-test-key",
		Storage:   storage,
		Data: map[string]interface{}{
			"name": "addr-test-key",
		},
	}

	resp, err := b.HandleRequest(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	pubKeyHex := resp.Data["public_key"].(string)
	addressHex := resp.Data["address"].(string)

	// Decode and verify
	pubKey, err := hex.DecodeString(pubKeyHex)
	require.NoError(t, err)

	// Derive address ourselves and compare
	expectedAddr := deriveCosmosAddress(pubKey)
	expectedAddrHex := hex.EncodeToString(expectedAddr)

	assert.Equal(t, expectedAddrHex, addressHex)
}

func TestKeyGenerationUniqueness(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Create multiple keys and verify they're all unique
	var pubKeys []string
	for i := 0; i < 5; i++ {
		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/unique-key-" + string(rune('a'+i)),
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "unique-key-" + string(rune('a'+i)),
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		pubKey := resp.Data["public_key"].(string)
		for _, existing := range pubKeys {
			assert.NotEqual(t, existing, pubKey, "public keys should be unique")
		}
		pubKeys = append(pubKeys, pubKey)
	}
}

func TestKeyReadAfterCacheClear(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Create a key
	createReq := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "keys/cache-clear-test",
		Storage:   storage,
		Data: map[string]interface{}{
			"name": "cache-clear-test",
		},
	}
	createResp, err := b.HandleRequest(ctx, createReq)
	require.NoError(t, err)
	require.NotNil(t, createResp)

	originalPubKey := createResp.Data["public_key"]

	// Clear the cache
	b.clearCache()

	// Read should still work (from storage)
	readReq := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "keys/cache-clear-test",
		Storage:   storage,
		Data: map[string]interface{}{
			"name": "cache-clear-test",
		},
	}

	readResp, err := b.HandleRequest(ctx, readReq)
	require.NoError(t, err)
	require.NotNil(t, readResp)
	assert.False(t, readResp.IsError())
	assert.Equal(t, originalPubKey, readResp.Data["public_key"])
}

func TestKeyDefaultExportable(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Create key without specifying exportable (should default to false)
	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "keys/default-exportable-key",
		Storage:   storage,
		Data: map[string]interface{}{
			"name": "default-exportable-key",
		},
	}

	resp, err := b.HandleRequest(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.IsError())
	assert.Equal(t, false, resp.Data["exportable"])
}
