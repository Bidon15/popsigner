package secp256k1

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathSignEVM(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Create a test key
	_, err := b.HandleRequest(ctx, &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "keys/test-evm-key",
		Storage:   storage,
		Data: map[string]interface{}{
			"exportable": true,
		},
	})
	require.NoError(t, err)

	t.Run("signs with EIP-155 chainID=10", func(t *testing.T) {
		// Create a test hash (32 bytes)
		hash := hashKeccak256([]byte("test transaction data"))
		hashB64 := base64.StdEncoding.EncodeToString(hash)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     hashB64,
				"chain_id": 10, // OP Mainnet
			},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError(), "response error: %v", resp.Error())

		// Verify response fields
		assert.NotEmpty(t, resp.Data["v"])
		assert.NotEmpty(t, resp.Data["r"])
		assert.NotEmpty(t, resp.Data["s"])
		assert.NotEmpty(t, resp.Data["eth_address"])

		// Verify v is EIP-155 format (chainId * 2 + 35 + recovery)
		// For chainId=10: v should be 55 or 56
		vInt := resp.Data["v_int"].(int64)
		assert.True(t, vInt == 55 || vInt == 56, "v should be 55 or 56 for chain_id=10, got %d", vInt)

		// Verify r and s are 64 hex chars (32 bytes)
		rHex := resp.Data["r"].(string)
		sHex := resp.Data["s"].(string)
		assert.Len(t, rHex, 64)
		assert.Len(t, sHex, 64)

		// Verify eth_address format
		ethAddr := resp.Data["eth_address"].(string)
		assert.Len(t, ethAddr, 42)
		assert.Equal(t, "0x", ethAddr[:2])
	})

	t.Run("signs with EIP-155 chainID=1", func(t *testing.T) {
		hash := hashKeccak256([]byte("mainnet transaction"))
		hashB64 := base64.StdEncoding.EncodeToString(hash)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     hashB64,
				"chain_id": 1, // Ethereum mainnet
			},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError())

		// For chainId=1: v should be 37 or 38
		vInt := resp.Data["v_int"].(int64)
		assert.True(t, vInt == 37 || vInt == 38, "v should be 37 or 38 for chain_id=1, got %d", vInt)
	})

	t.Run("signs with legacy format", func(t *testing.T) {
		hash := hashKeccak256([]byte("legacy transaction"))
		hashB64 := base64.StdEncoding.EncodeToString(hash)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash": hashB64,
				// No chain_id = legacy signing
			},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError())

		// Legacy v should be 27 or 28
		vInt := resp.Data["v_int"].(int64)
		assert.True(t, vInt == 27 || vInt == 28, "legacy v should be 27 or 28, got %d", vInt)
	})

	t.Run("signs with chain_id=0 uses legacy", func(t *testing.T) {
		hash := hashKeccak256([]byte("zero chain id"))
		hashB64 := base64.StdEncoding.EncodeToString(hash)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     hashB64,
				"chain_id": 0,
			},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError())

		// Legacy v should be 27 or 28
		vInt := resp.Data["v_int"].(int64)
		assert.True(t, vInt == 27 || vInt == 28, "chain_id=0 should use legacy v=27/28, got %d", vInt)
	})

	t.Run("rejects invalid hash length", func(t *testing.T) {
		shortHash := base64.StdEncoding.EncodeToString([]byte("too short"))

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     shortHash,
				"chain_id": 1,
			},
		})
		require.NoError(t, err)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "32 bytes")
	})

	t.Run("rejects non-existent key", func(t *testing.T) {
		hash := hashKeccak256([]byte("test"))
		hashB64 := base64.StdEncoding.EncodeToString(hash)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/non-existent-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     hashB64,
				"chain_id": 1,
			},
		})
		require.NoError(t, err)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "not found")
	})

	t.Run("rejects missing hash", func(t *testing.T) {
		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"chain_id": 1,
			},
		})
		require.NoError(t, err)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "missing hash")
	})

	t.Run("rejects invalid base64", func(t *testing.T) {
		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     "not-valid-base64!!!",
				"chain_id": 1,
			},
		})
		require.NoError(t, err)
		assert.True(t, resp.IsError())
		assert.Contains(t, resp.Error().Error(), "base64")
	})

	t.Run("signature is recoverable with EIP-155", func(t *testing.T) {
		hash := hashKeccak256([]byte("recoverable test"))
		hashB64 := base64.StdEncoding.EncodeToString(hash)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     hashB64,
				"chain_id": 1,
			},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError())

		// Get the public key from response
		pubKeyHex := resp.Data["public_key"].(string)
		pubKeyBytes, _ := hex.DecodeString(pubKeyHex)
		expectedPubKey, _ := ParsePublicKey(pubKeyBytes)

		// Construct signature for recovery
		rHex := resp.Data["r"].(string)
		sHex := resp.Data["s"].(string)
		vInt := resp.Data["v_int"].(int64)

		rBytes, _ := hex.DecodeString(rHex)
		sBytes, _ := hex.DecodeString(sHex)

		sig := make([]byte, 65)
		copy(sig[0:32], rBytes)
		copy(sig[32:64], sBytes)
		sig[64] = byte(vInt)

		// Recover public key
		recoveredPubKey, err := RecoverPubKeyFromSignature(hash, sig, big.NewInt(1))
		require.NoError(t, err)

		assert.True(t, recoveredPubKey.IsEqual(expectedPubKey))
	})

	t.Run("signature is recoverable with legacy", func(t *testing.T) {
		hash := hashKeccak256([]byte("legacy recoverable"))
		hashB64 := base64.StdEncoding.EncodeToString(hash)

		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash": hashB64,
			},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError())

		// Get the public key from response
		pubKeyHex := resp.Data["public_key"].(string)
		pubKeyBytes, _ := hex.DecodeString(pubKeyHex)
		expectedPubKey, _ := ParsePublicKey(pubKeyBytes)

		// Construct signature for recovery
		rHex := resp.Data["r"].(string)
		sHex := resp.Data["s"].(string)
		vInt := resp.Data["v_int"].(int64)

		rBytes, _ := hex.DecodeString(rHex)
		sBytes, _ := hex.DecodeString(sHex)

		sig := make([]byte, 65)
		copy(sig[0:32], rBytes)
		copy(sig[32:64], sBytes)
		sig[64] = byte(vInt)

		// Recover public key with nil chainID (legacy)
		recoveredPubKey, err := RecoverPubKeyFromSignature(hash, sig, nil)
		require.NoError(t, err)

		assert.True(t, recoveredPubKey.IsEqual(expectedPubKey))
	})

	t.Run("different chain IDs produce different v values", func(t *testing.T) {
		hash := hashKeccak256([]byte("chain id test"))
		hashB64 := base64.StdEncoding.EncodeToString(hash)

		// Sign with chain_id=1
		resp1, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     hashB64,
				"chain_id": 1,
			},
		})
		require.NoError(t, err)
		require.False(t, resp1.IsError())

		// Sign with chain_id=10
		resp10, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/test-evm-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     hashB64,
				"chain_id": 10,
			},
		})
		require.NoError(t, err)
		require.False(t, resp10.IsError())

		v1 := resp1.Data["v_int"].(int64)
		v10 := resp10.Data["v_int"].(int64)

		// v values should reflect different chain IDs
		// chainId=1: v=37 or 38
		// chainId=10: v=55 or 56
		assert.NotEqual(t, v1/2, v10/2, "v values should differ for different chain IDs")
	})
}

func TestPathSignEVMDefinition(t *testing.T) {
	b, _ := getTestBackend(t)

	paths := pathSignEVM(b)
	require.Len(t, paths, 1)

	t.Run("has correct path pattern", func(t *testing.T) {
		assert.Contains(t, paths[0].Pattern, "sign-evm/")
	})

	t.Run("has required fields", func(t *testing.T) {
		assert.Contains(t, paths[0].Fields, "name")
		assert.Contains(t, paths[0].Fields, "hash")
		assert.Contains(t, paths[0].Fields, "chain_id")
	})

	t.Run("has update operation", func(t *testing.T) {
		assert.NotNil(t, paths[0].Operations[logical.UpdateOperation])
	})
}

func TestPathKeys_EthAddress(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	t.Run("key creation returns eth_address", func(t *testing.T) {
		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/eth-test-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"exportable": true,
			},
		})
		require.NoError(t, err)
		require.False(t, resp.IsError())

		ethAddr := resp.Data["eth_address"].(string)
		assert.Len(t, ethAddr, 42)
		assert.Equal(t, "0x", ethAddr[:2])

		// Also verify Cosmos address is still present
		cosmosAddr := resp.Data["address"].(string)
		assert.Len(t, cosmosAddr, 40) // 20 bytes hex = 40 chars
	})

	t.Run("key read returns eth_address", func(t *testing.T) {
		// First create a key
		_, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/eth-read-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"exportable": true,
			},
		})
		require.NoError(t, err)

		// Then read it
		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "keys/eth-read-key",
			Storage:   storage,
		})
		require.NoError(t, err)
		require.False(t, resp.IsError())

		ethAddr := resp.Data["eth_address"].(string)
		assert.Len(t, ethAddr, 42)
		assert.Equal(t, "0x", ethAddr[:2])
	})

	t.Run("key list returns eth_address in key_info", func(t *testing.T) {
		// Create another key
		_, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/eth-list-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"exportable": true,
			},
		})
		require.NoError(t, err)

		// List keys
		resp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.ListOperation,
			Path:      "keys/",
			Storage:   storage,
		})
		require.NoError(t, err)
		require.False(t, resp.IsError())

		// Keys should still be present
		keys := resp.Data["keys"].([]string)
		assert.Contains(t, keys, "eth-list-key")

		// Key info should have eth_address
		keyInfo := resp.Data["key_info"].(map[string]interface{})
		listKeyInfo := keyInfo["eth-list-key"].(map[string]interface{})
		ethAddr := listKeyInfo["eth_address"].(string)
		assert.Len(t, ethAddr, 42)
		assert.Equal(t, "0x", ethAddr[:2])
	})

	t.Run("eth_address matches derived address", func(t *testing.T) {
		// Create a key
		createResp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "keys/eth-derive-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"exportable": true,
			},
		})
		require.NoError(t, err)
		require.False(t, createResp.IsError())

		ethAddrFromCreate := createResp.Data["eth_address"].(string)
		pubKeyHex := createResp.Data["public_key"].(string)

		// Manually derive eth address from public key
		pubKeyBytes, err := hex.DecodeString(pubKeyHex)
		require.NoError(t, err)
		pubKey, err := ParsePublicKey(pubKeyBytes)
		require.NoError(t, err)
		derivedAddr := deriveEthereumAddress(pubKey)
		derivedAddrFormatted := formatEthereumAddress(derivedAddr)

		assert.Equal(t, derivedAddrFormatted, ethAddrFromCreate)
	})
}

func TestEVMSigningConsistency(t *testing.T) {
	b, storage := getTestBackend(t)
	ctx := context.Background()

	// Create a key
	createResp, err := b.HandleRequest(ctx, &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "keys/consistency-key",
		Storage:   storage,
		Data: map[string]interface{}{
			"exportable": true,
		},
	})
	require.NoError(t, err)

	ethAddrFromKey := createResp.Data["eth_address"].(string)

	t.Run("sign-evm returns same eth_address as key creation", func(t *testing.T) {
		hash := hashKeccak256([]byte("consistency test"))
		hashB64 := base64.StdEncoding.EncodeToString(hash)

		signResp, err := b.HandleRequest(ctx, &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "sign-evm/consistency-key",
			Storage:   storage,
			Data: map[string]interface{}{
				"hash":     hashB64,
				"chain_id": 1,
			},
		})
		require.NoError(t, err)
		require.False(t, signResp.IsError())

		ethAddrFromSign := signResp.Data["eth_address"].(string)
		assert.Equal(t, ethAddrFromKey, ethAddrFromSign)
	})
}

func BenchmarkPathSignEVM(b *testing.B) {
	ctx := context.Background()
	backend, storage := getTestBackendBench(b)

	// Create a key
	_, err := backend.HandleRequest(ctx, &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "keys/bench-evm-key",
		Storage:   storage,
		Data:      map[string]interface{}{},
	})
	if err != nil {
		b.Fatal(err)
	}

	hash := hashKeccak256([]byte("benchmark message"))
	hashB64 := base64.StdEncoding.EncodeToString(hash)

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "sign-evm/bench-evm-key",
		Storage:   storage,
		Data: map[string]interface{}{
			"hash":     hashB64,
			"chain_id": 10,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = backend.HandleRequest(ctx, req)
	}
}

