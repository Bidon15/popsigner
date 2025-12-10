package secp256k1

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathExport(t *testing.T) {
	b, _ := getTestBackend(t)

	paths := pathExport(b)
	require.Len(t, paths, 1)

	path := paths[0]
	assert.Contains(t, path.Pattern, "export/")
	assert.NotNil(t, path.Fields["name"])
	assert.NotNil(t, path.Operations[logical.ReadOperation])
}

func TestPathKeyExport(t *testing.T) {
	ctx := context.Background()

	t.Run("exports exportable key successfully", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Create an exportable key via import
		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		importReq := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/exportablekey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "exportablekey",
				"ciphertext": ciphertext,
				"exportable": true,
			},
		}

		resp, err := b.HandleRequest(ctx, importReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Now export the key
		exportReq := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "export/exportablekey",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "exportablekey",
			},
		}

		resp, err = b.HandleRequest(ctx, exportReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError(), "unexpected error: %v", resp.Error())

		// Verify response
		assert.Equal(t, "exportablekey", resp.Data["name"])
		assert.NotEmpty(t, resp.Data["public_key"])
		assert.NotEmpty(t, resp.Data["address"])
		assert.NotEmpty(t, resp.Data["created_at"])
		assert.Equal(t, true, resp.Data["imported"])

		// Verify the exported key matches the original
		keys, ok := resp.Data["keys"].(map[string]string)
		require.True(t, ok)
		exportedKeyB64, ok := keys["1"]
		require.True(t, ok)

		exportedKey, err := base64.StdEncoding.DecodeString(exportedKeyB64)
		require.NoError(t, err)
		assert.Equal(t, privKeyBytes, exportedKey)
	})

	t.Run("exports generated exportable key", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Create an exportable key via the keys endpoint
		createReq := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "keys/gen-export-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "gen-export-key",
				"exportable": true,
			},
		}

		resp, err := b.HandleRequest(ctx, createReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Now export the key
		exportReq := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "export/gen-export-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "gen-export-key",
			},
		}

		resp, err = b.HandleRequest(ctx, exportReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Verify keys field is present
		keys, ok := resp.Data["keys"].(map[string]string)
		require.True(t, ok)
		_, ok = keys["1"]
		require.True(t, ok)
	})

	t.Run("rejects export of non-exportable key", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Create a non-exportable key via import
		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		importReq := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/nonexportkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "nonexportkey",
				"ciphertext": ciphertext,
				"exportable": false, // Not exportable
			},
		}

		resp, err := b.HandleRequest(ctx, importReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Try to export
		exportReq := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "export/nonexportkey",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "nonexportkey",
			},
		}

		resp, err = b.HandleRequest(ctx, exportReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "key is not exportable")
	})

	t.Run("rejects export of non-existent key", func(t *testing.T) {
		b, storage := getTestBackend(t)

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "export/nonexistent",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "nonexistent",
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "key not found")
	})

	t.Run("rejects non-existent key with short name", func(t *testing.T) {
		b, storage := getTestBackend(t)

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "export/x",  // Valid path with minimal name
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "x",
			},
		}

		resp, err := b.HandleRequest(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "key not found")
	})

	t.Run("exported key can be re-imported", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Create and import an exportable key
		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		importReq := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/originalkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "originalkey",
				"ciphertext": ciphertext,
				"exportable": true,
			},
		}

		resp, err := b.HandleRequest(ctx, importReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		originalPubKey := resp.Data["public_key"].(string)

		// Export the key
		exportReq := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "export/originalkey",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "originalkey",
			},
		}

		resp, err = b.HandleRequest(ctx, exportReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		keys := resp.Data["keys"].(map[string]string)
		exportedCiphertext := keys["1"]

		// Re-import with a different name
		reimportReq := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/reimportedkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "reimportedkey",
				"ciphertext": exportedCiphertext,
				"exportable": false,
			},
		}

		resp, err = b.HandleRequest(ctx, reimportReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Verify the re-imported key has the same public key
		assert.Equal(t, originalPubKey, resp.Data["public_key"])
	})

	t.Run("export includes correct metadata", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Create an exportable key
		privKey, pubKey, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		importReq := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/metadatakey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "metadatakey",
				"ciphertext": ciphertext,
				"exportable": true,
			},
		}

		resp, err := b.HandleRequest(ctx, importReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Export and check metadata
		exportReq := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "export/metadatakey",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "metadatakey",
			},
		}

		resp, err = b.HandleRequest(ctx, exportReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Verify all expected fields
		expectedPubKey := hex.EncodeToString(SerializePublicKey(pubKey))
		expectedAddress := hex.EncodeToString(deriveCosmosAddress(SerializePublicKey(pubKey)))

		assert.Equal(t, "metadatakey", resp.Data["name"])
		assert.Equal(t, expectedPubKey, resp.Data["public_key"])
		assert.Equal(t, expectedAddress, resp.Data["address"])
		assert.Equal(t, true, resp.Data["imported"])

		// created_at should be a valid RFC3339 time
		createdAt := resp.Data["created_at"].(string)
		_, err = time.Parse(time.RFC3339, createdAt)
		assert.NoError(t, err)
	})
}

func TestExportedKeyIntegrity(t *testing.T) {
	ctx := context.Background()

	t.Run("exported key produces valid signatures", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Create and import an exportable key
		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		importReq := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/sigkey/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "sigkey",
				"ciphertext": ciphertext,
				"exportable": true,
			},
		}

		resp, err := b.HandleRequest(ctx, importReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Export the key
		exportReq := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "export/sigkey",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "sigkey",
			},
		}

		resp, err = b.HandleRequest(ctx, exportReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Parse the exported key
		keys := resp.Data["keys"].(map[string]string)
		exportedKeyB64 := keys["1"]
		exportedKeyBytes, err := base64.StdEncoding.DecodeString(exportedKeyB64)
		require.NoError(t, err)

		exportedPrivKey, err := ParsePrivateKey(exportedKeyBytes)
		require.NoError(t, err)

		// Sign with exported key
		hash := hashSHA256([]byte("test message for export"))
		sig, err := SignMessage(exportedPrivKey, hash)
		require.NoError(t, err)

		// Verify with original public key
		pubKeyHex := resp.Data["public_key"].(string)
		pubKeyBytes, err := hex.DecodeString(pubKeyHex)
		require.NoError(t, err)

		pubKey, err := ParsePublicKey(pubKeyBytes)
		require.NoError(t, err)

		valid, err := VerifySignature(pubKey, hash, sig)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("signature via path matches exported key signature", func(t *testing.T) {
		b, storage := getTestBackend(t)

		// Create an exportable key
		privKey, _, err := GenerateKey()
		require.NoError(t, err)

		privKeyBytes := SerializePrivateKey(privKey)
		ciphertext := base64.StdEncoding.EncodeToString(privKeyBytes)

		importReq := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/verify-sig-key/import",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":       "verify-sig-key",
				"ciphertext": ciphertext,
				"exportable": true,
			},
		}

		resp, err := b.HandleRequest(ctx, importReq)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.IsError())

		// Sign via the sign path
		message := []byte("test message")
		inputB64 := base64.StdEncoding.EncodeToString(message)

		signReq := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign/verify-sig-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name":  "verify-sig-key",
				"input": inputB64,
			},
		}

		signResp, err := b.HandleRequest(ctx, signReq)
		require.NoError(t, err)
		require.NotNil(t, signResp)
		require.False(t, signResp.IsError(), "sign error: %v", signResp.Error())

		sigB64 := signResp.Data["signature"].(string)
		sig, err := base64.StdEncoding.DecodeString(sigB64)
		require.NoError(t, err)

		// Compute the hash the same way the sign path does
		hash := hashSHA256(message)

		// Export key and verify signature matches
		exportReq := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "export/verify-sig-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"name": "verify-sig-key",
			},
		}

		exportResp, err := b.HandleRequest(ctx, exportReq)
		require.NoError(t, err)
		require.NotNil(t, exportResp)
		require.False(t, exportResp.IsError())

		// Parse exported key and verify signature
		keys := exportResp.Data["keys"].(map[string]string)
		exportedKeyB64 := keys["1"]
		exportedKeyBytes, err := base64.StdEncoding.DecodeString(exportedKeyB64)
		require.NoError(t, err)

		exportedPrivKey, err := ParsePrivateKey(exportedKeyBytes)
		require.NoError(t, err)

		// Signature should verify with exported key's public key
		valid, err := VerifySignature(exportedPrivKey.PubKey(), hash, sig)
		require.NoError(t, err)
		assert.True(t, valid, "signature from sign path should verify with exported key")
	})
}
