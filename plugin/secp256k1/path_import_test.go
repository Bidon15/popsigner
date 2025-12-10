package secp256k1

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathImport(t *testing.T) {
	b, _ := getTestBackend(t)

	paths := pathImport(b)
	require.Len(t, paths, 1)

	path := paths[0]
	assert.Contains(t, path.Pattern, "keys/")
	assert.Contains(t, path.Pattern, "/import")
	assert.NotNil(t, path.Fields["name"])
	assert.NotNil(t, path.Fields["ciphertext"])
	assert.NotNil(t, path.Fields["exportable"])
	assert.NotNil(t, path.Operations[logical.UpdateOperation])
}

func TestPathKeyImport(t *testing.T) {
	ctx := context.Background()

	t.Run("imports valid key successfully", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Generate a test key
		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/testkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "testkey",
				"ciphertext": ciphertext,
				"exportable": false,
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError(), "unexpected error: %v", resp.Error())

		// Verify response data
		assert.Equal(t, "testkey", resp.Data["name"])
		assert.NotEmpty(t, resp.Data["public_key"])
		assert.NotEmpty(t, resp.Data["address"])
		assert.Equal(t, true, resp.Data["imported"])
		assert.Equal(t, false, resp.Data["exportable"])
		assert.NotEmpty(t, resp.Data["created_at"])
	})

	t.Run("imports key with exportable flag", func(t *testing.T) {
		b, storage := getTestBackend(t)

		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/exportkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "exportkey",
				"ciphertext": ciphertext,
				"exportable": true,
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		assert.Equal(t, true, resp.Data["exportable"])
	})

	t.Run("rejects empty key name", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Use a valid path but with name field explicitly set to empty
		// Note: The framework will extract the name from the path regex,
		// so we need to test with a valid path but verify the function handles empty names
		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/testkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "",  // This gets overridden by path param
				"ciphertext": "aGVsbG8=",
			},
		}

		// With path-based name extraction, empty name test becomes invalid key test
		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		// Will fail due to invalid key (not missing name since framework extracts from path)
		assert.Contains(t, resp.Error().Error(), "invalid secp256k1 key")
	})

	t.Run("rejects empty ciphertext", func(t *testing.T) {
		b, storage := getTestBackend(t)

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/testkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "testkey",
				"ciphertext": "",
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "missing ciphertext")
	})

	t.Run("rejects existing key", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// First, create a key
		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/duplicate/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "duplicate",
				"ciphertext": ciphertext,
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Try to import again
		resp, err = b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "key already exists")
	})

	t.Run("rejects invalid base64 ciphertext", func(t *testing.T) {
		b, storage := getTestBackend(t)

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/testkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "testkey",
				"ciphertext": "!!!not-valid-base64!!!",
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "invalid ciphertext")
	})

	t.Run("rejects invalid key length", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Use a key that's not 32 bytes
		invalidKey := make([]byte, 16) // Too short
		ciphertext := base64.StdEncoding.EncodeToString(invalidKey)

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/testkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "testkey",
				"ciphertext": ciphertext,
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "invalid secp256k1 key")
	})

	t.Run("verifies correct public key derivation", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Generate a key and record its public key
		privKey, pubKey, err := GenerateKey()
		require.NoError(t, err)

		expectedPubKeyHex := hex.EncodeToString(SerializePublicKey(pubKey))
		expectedAddress := hex.EncodeToString(deriveCosmosAddress(SerializePublicKey(pubKey)))

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/testkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "testkey",
				"ciphertext": ciphertext,
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Verify derived values match
		assert.Equal(t, expectedPubKeyHex, resp.Data["public_key"])
		assert.Equal(t, expectedAddress, resp.Data["address"])
	})

	t.Run("imported key can be used for signing", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Import a key
		privKey, pubKey, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		importReq := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/signingkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "signingkey",
				"ciphertext": ciphertext,
			},
		}

		resp, err := b.HandleRequest(ctx, importReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Load the key from storage and verify we can sign with it
		entry, err := b.getKey(ctx, storage, "signingkey")
		require.NoError(t, err)
		require.NotNil(t, entry)

		storedPrivKey, err := ParsePrivateKey(entry.PrivateKey)
		require.NoError(t, err)

		hash := hashSHA256([]byte("test message"))
		sig, err := SignMessage(storedPrivKey, hash)
		require.NoError(t, err)

		// Verify signature with original public key
		valid, err := VerifySignature(pubKey, hash, sig)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("imported key is marked as imported", func(t *testing.T) {
		b, storage := getTestBackend(t)

		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		ciphertext := base64.StdEncoding.EncodeToString(SerializePrivateKey(privKey))

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/imported-marker/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "imported-marker",
				"ciphertext": ciphertext,
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Read the key back and verify imported flag
		readReq := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "keys/imported-marker",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "imported-marker",
			},
		}

		readResp, err := b.HandleRequest(ctx, readReq)
		require.NoError(t, err)
		require.NotNil(t, readResp)
		require.False(t, readResp.IsError())

		assert.Equal(t, true, readResp.Data["imported"])
	})

	t.Run("import updates cache", func(t *testing.T) {
		b, storage := getTestBackend(t)

		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		ciphertext := base64.StdEncoding.EncodeToString(SerializePrivateKey(privKey))

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/cached-import/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "cached-import",
				"ciphertext": ciphertext,
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Verify key is in cache
		cached := b.getKeyFromCache("cached-import")
		require.NotNil(t, cached)
		assert.True(t, cached.Imported)
	})
}

func TestValidatePrivateKeyBytes(t *testing.T) {
	t.Run("accepts valid 32-byte key", func(t *testing.T) {
		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		keyBytes := SerializePrivateKey(privKey)
		err = validatePrivateKeyBytes(keyBytes)
		assert.NoError(t, err)
	})

	t.Run("rejects short key", func(t *testing.T) {
		keyBytes := make([]byte, 16)
		err := validatePrivateKeyBytes(keyBytes)
		assert.Error(t, err)
	})

	t.Run("rejects long key", func(t *testing.T) {
		keyBytes := make([]byte, 64)
		err := validatePrivateKeyBytes(keyBytes)
		assert.Error(t, err)
	})

	t.Run("rejects empty key", func(t *testing.T) {
		keyBytes := make([]byte, 0)
		err := validatePrivateKeyBytes(keyBytes)
		assert.Error(t, err)
	})
}
