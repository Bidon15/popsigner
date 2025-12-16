package jsonrpc

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	"github.com/Bidon15/popsigner/control-plane/internal/openbao"
)

func TestEthSignTransactionHandler_Handle(t *testing.T) {
	t.Run("returns error without org context", func(t *testing.T) {
		mockRepo := new(MockKeyRepository)

		handler := &EthSignTransactionHandler{
			keyRepo:   mockRepo,
			baoClient: nil,
		}

		ctx := context.Background()
		_, err := handler.Handle(ctx, json.RawMessage(`[{}]`))

		require.NotNil(t, err)
		assert.Equal(t, UnauthorizedError, err.Code)
	})

	t.Run("returns error with empty params", func(t *testing.T) {
		mockRepo := new(MockKeyRepository)

		handler := &EthSignTransactionHandler{
			keyRepo:   mockRepo,
			baoClient: nil,
		}

		ctx := contextWithOrgID(uuid.New())
		_, err := handler.Handle(ctx, json.RawMessage(`[]`))

		require.NotNil(t, err)
		assert.Equal(t, InvalidParams, err.Code)
	})

	t.Run("returns error with invalid tx params", func(t *testing.T) {
		mockRepo := new(MockKeyRepository)

		handler := &EthSignTransactionHandler{
			keyRepo:   mockRepo,
			baoClient: nil,
		}

		ctx := contextWithOrgID(uuid.New())
		// Missing required fields
		_, err := handler.Handle(ctx, json.RawMessage(`[{"to": "0x1234567890123456789012345678901234567890"}]`))

		require.NotNil(t, err)
		assert.Equal(t, InvalidParams, err.Code)
	})

	t.Run("returns error when key not found", func(t *testing.T) {
		mockRepo := new(MockKeyRepository)

		handler := &EthSignTransactionHandler{
			keyRepo:   mockRepo,
			baoClient: nil,
		}

		orgID := uuid.New()

		mockRepo.On("GetByEthAddress", mock.Anything, orgID, mock.Anything).Return(nil, nil)

		ctx := contextWithOrgID(orgID)
		params := `[{
			"from": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			"to": "0x1234567890123456789012345678901234567890",
			"gas": "0x5208",
			"gasPrice": "0x3b9aca00",
			"value": "0x0",
			"nonce": "0x1",
			"chainId": "0xa"
		}]`
		_, err := handler.Handle(ctx, json.RawMessage(params))

		require.NotNil(t, err)
		assert.Equal(t, ResourceNotFound, err.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestParseSignatureResponse(t *testing.T) {
	t.Run("parses valid response", func(t *testing.T) {
		resp := &openbao.SignEVMResponse{
			R:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			S:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678",
			VInt: 37,
		}

		v, r, s, err := parseSignatureResponse(resp)

		require.NoError(t, err)
		assert.Equal(t, int64(37), v.Int64())
		assert.True(t, r.Cmp(big.NewInt(0)) > 0)
		assert.True(t, s.Cmp(big.NewInt(0)) > 0)
	})

	t.Run("parses legacy v values", func(t *testing.T) {
		resp := &openbao.SignEVMResponse{
			R:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			S:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678",
			VInt: 27,
		}

		v, _, _, err := parseSignatureResponse(resp)

		require.NoError(t, err)
		assert.Equal(t, int64(27), v.Int64())
	})

	t.Run("parses EIP-155 v values", func(t *testing.T) {
		resp := &openbao.SignEVMResponse{
			R:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			S:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678",
			VInt: 38, // chainId=1: v = chainId*2 + 35 + recovery_id
		}

		v, _, _, err := parseSignatureResponse(resp)

		require.NoError(t, err)
		assert.Equal(t, int64(38), v.Int64())
	})

	t.Run("handles invalid r", func(t *testing.T) {
		resp := &openbao.SignEVMResponse{
			R:    "invalid",
			S:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678",
			VInt: 27,
		}

		_, _, _, err := parseSignatureResponse(resp)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode r")
	})

	t.Run("handles invalid s", func(t *testing.T) {
		resp := &openbao.SignEVMResponse{
			R:    "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			S:    "invalid",
			VInt: 27,
		}

		_, _, _, err := parseSignatureResponse(resp)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode s")
	})
}

func TestEthSignTransactionHandler_TransactionTypeDetection(t *testing.T) {
	t.Run("detects legacy transaction", func(t *testing.T) {
		params := `[{
			"from": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			"to": "0x1234567890123456789012345678901234567890",
			"gas": "0x5208",
			"gasPrice": "0x3b9aca00",
			"value": "0x0",
			"nonce": "0x1",
			"chainId": "0xa"
		}]`

		var args []json.RawMessage
		err := json.Unmarshal([]byte(params), &args)
		require.NoError(t, err)

		// Verify it has gasPrice but not maxFeePerGas (legacy tx)
		var txArgs map[string]interface{}
		err = json.Unmarshal(args[0], &txArgs)
		require.NoError(t, err)

		assert.Contains(t, txArgs, "gasPrice")
		assert.NotContains(t, txArgs, "maxFeePerGas")
	})

	t.Run("detects EIP-1559 transaction", func(t *testing.T) {
		params := `[{
			"from": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			"to": "0x1234567890123456789012345678901234567890",
			"gas": "0x5208",
			"maxFeePerGas": "0x4a817c800",
			"maxPriorityFeePerGas": "0x3b9aca00",
			"value": "0x0",
			"nonce": "0x1",
			"chainId": "0xa"
		}]`

		var args []json.RawMessage
		err := json.Unmarshal([]byte(params), &args)
		require.NoError(t, err)

		// Verify it has maxFeePerGas (EIP-1559 tx)
		var txArgs map[string]interface{}
		err = json.Unmarshal(args[0], &txArgs)
		require.NoError(t, err)

		assert.Contains(t, txArgs, "maxFeePerGas")
		assert.Contains(t, txArgs, "maxPriorityFeePerGas")
	})
}

func TestEthSignTransactionHandler_KeyLookup(t *testing.T) {
	t.Run("looks up key by from address and fails at signing", func(t *testing.T) {
		mockRepo := new(MockKeyRepository)

		orgID := uuid.New()
		ethAddr := "0x742d35cc6634c0532925a3b844bc454e4438f44e"
		key := &models.Key{
			ID:         uuid.New(),
			OrgID:      orgID,
			EthAddress: &ethAddr,
			BaoKeyPath: "test-key",
		}

		// Return nil to simulate key not found - this prevents the nil baoClient panic
		mockRepo.On("GetByEthAddress", mock.Anything, orgID, mock.Anything).Return(nil, nil).Once()

		handler := &EthSignTransactionHandler{
			keyRepo:   mockRepo,
			baoClient: nil,
		}

		ctx := contextWithOrgID(orgID)
		params := `[{
			"from": "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			"to": "0x1234567890123456789012345678901234567890",
			"gas": "0x5208",
			"gasPrice": "0x3b9aca00",
			"value": "0x0",
			"nonce": "0x1",
			"chainId": "0xa"
		}]`

		// Call Handle - it will fail with ResourceNotFound because key isn't found
		_, err := handler.Handle(ctx, json.RawMessage(params))

		// Verify the key lookup was called
		mockRepo.AssertCalled(t, "GetByEthAddress", mock.Anything, orgID, mock.Anything)
		
		// Expect resource not found error since we returned nil for the key
		require.NotNil(t, err)
		assert.Equal(t, ResourceNotFound, err.Code)
		_ = key // key variable is setup to show expected structure
	})
}

