package jsonrpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/sha3"

	"github.com/Bidon15/popsigner/control-plane/internal/ethereum"
	"github.com/Bidon15/popsigner/control-plane/internal/middleware"
	"github.com/Bidon15/popsigner/control-plane/internal/models"
	"github.com/Bidon15/popsigner/control-plane/internal/openbao"
	"github.com/Bidon15/popsigner/control-plane/internal/repository"
)

// EthSignHandler handles eth_sign and personal_sign requests.
type EthSignHandler struct {
	keyRepo   repository.KeyRepository
	baoClient *openbao.Client
	auditRepo repository.AuditRepository
	usageRepo repository.UsageRepository
}

// NewEthSignHandler creates a new eth_sign handler.
func NewEthSignHandler(keyRepo repository.KeyRepository, baoClient *openbao.Client, auditRepo repository.AuditRepository, usageRepo repository.UsageRepository) *EthSignHandler {
	return &EthSignHandler{
		keyRepo:   keyRepo,
		baoClient: baoClient,
		auditRepo: auditRepo,
		usageRepo: usageRepo,
	}
}

// HandleEthSign implements the eth_sign JSON-RPC method.
// Signs data with the standard Ethereum prefix: "\x19Ethereum Signed Message:\n" + len(message) + message
// Parameters: [address, data]
func (h *EthSignHandler) HandleEthSign(ctx context.Context, params json.RawMessage) (interface{}, *Error) {
	return h.signMessage(ctx, params, false)
}

// HandlePersonalSign implements the personal_sign JSON-RPC method.
// Same as eth_sign but with params in reversed order: [data, address]
func (h *EthSignHandler) HandlePersonalSign(ctx context.Context, params json.RawMessage) (interface{}, *Error) {
	return h.signMessage(ctx, params, true)
}

// signMessage handles both eth_sign and personal_sign.
// personalSign: true means params are [data, address], false means [address, data]
func (h *EthSignHandler) signMessage(ctx context.Context, params json.RawMessage, personalSign bool) (interface{}, *Error) {
	// Get org ID from context
	orgID := middleware.GetOrgIDFromContext(ctx)
	if orgID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, ErrUnauthorized("missing organization context")
	}

	// Parse params based on method
	var args []string
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, ErrInvalidParams(err.Error())
	}
	if len(args) < 2 {
		return nil, ErrInvalidParams("requires [address, data] or [data, address]")
	}

	var addressHex, dataHex string
	if personalSign {
		// personal_sign: [data, address]
		dataHex = args[0]
		addressHex = args[1]
	} else {
		// eth_sign: [address, data]
		addressHex = args[0]
		dataHex = args[1]
	}

	// Lookup key by address
	key, err := h.keyRepo.GetByEthAddress(ctx, orgID, addressHex)
	if err != nil {
		return nil, ErrInternal(fmt.Sprintf("failed to lookup key: %v", err))
	}
	if key == nil {
		return nil, ErrResourceNotFound(fmt.Sprintf("no key found for address %s", addressHex))
	}

	// Decode the data
	data, err := ethereum.DecodeBytes(dataHex)
	if err != nil {
		return nil, ErrInvalidParams(fmt.Sprintf("invalid data hex: %v", err))
	}

	// Apply Ethereum signed message prefix
	// "\x19Ethereum Signed Message:\n" + len(message) + message
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(data))
	prefixedData := append([]byte(prefix), data...)

	// Hash the prefixed message with Keccak256
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(prefixedData)
	hash := hasher.Sum(nil)

	// Sign via OpenBao (use legacy signing, v=27/28, chainID=0)
	hashB64 := base64.StdEncoding.EncodeToString(hash)
	signResp, err := h.baoClient.SignEVM(key.BaoKeyPath, hashB64, 0)
	if err != nil {
		return nil, ErrSigningFailed(err.Error())
	}

	// Construct signature in Ethereum format: r (32 bytes) || s (32 bytes) || v (1 byte)
	rBytes, _ := ethereum.DecodeBytes("0x" + signResp.R)
	sBytes, _ := ethereum.DecodeBytes("0x" + signResp.S)

	// Pad r and s to 32 bytes
	sig := make([]byte, 65)
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)
	sig[64] = byte(signResp.VInt)

	// Log audit and increment usage asynchronously
	go h.recordSignature(orgID, key.ID)

	return ethereum.EncodeBytes(sig), nil
}

// recordSignature logs the signing operation and increments usage counters.
func (h *EthSignHandler) recordSignature(orgID, keyID uuid.UUID) {
	ctx := context.Background()

	// Create audit log
	if h.auditRepo != nil {
		resourceType := models.ResourceTypeKey
		_ = h.auditRepo.Create(ctx, &models.AuditLog{
			ID:           uuid.New(),
			OrgID:        orgID,
			Event:        models.AuditEventKeySigned,
			ActorType:    models.ActorTypeAPIKey,
			ResourceType: &resourceType,
			ResourceID:   &keyID,
		})
	}

	// Increment signature usage
	if h.usageRepo != nil {
		_ = h.usageRepo.Increment(ctx, orgID, "signatures", 1)
	}
}

// HashPersonalMessage computes the hash of a personal message.
// This is useful for verifying signatures off-chain.
func HashPersonalMessage(data []byte) []byte {
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(data))
	prefixedData := append([]byte(prefix), data...)

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(prefixedData)
	return hasher.Sum(nil)
}

