package secp256k1

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// pathSignEVM returns the path definitions for EVM signing operations.
func pathSignEVM(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "sign-evm/" + framework.GenericNameRegex("name"),
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the key to use for signing",
					Required:    true,
				},
				"hash": {
					Type:        framework.TypeString,
					Description: "Base64-encoded 32-byte hash to sign (typically Keccak256 of RLP-encoded transaction)",
					Required:    true,
				},
				"chain_id": {
					Type:        framework.TypeInt,
					Description: "EIP-155 chain ID (e.g., 1 for Ethereum mainnet, 10 for OP Mainnet). If 0 or omitted, uses legacy signing (v=27/28).",
					Default:     0,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback:    b.pathSignEVMWrite,
					Summary:     "Sign a hash with EIP-155 format for Ethereum transactions",
					Description: "Signs a 32-byte hash and returns v, r, s values suitable for Ethereum transaction signing.",
				},
			},
			HelpSynopsis:    pathSignEVMHelpSyn,
			HelpDescription: pathSignEVMHelpDesc,
		},
	}
}

// pathSignEVMWrite handles the EVM sign operation.
func (b *backend) pathSignEVMWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("missing key name"), nil
	}

	hashB64 := data.Get("hash").(string)
	if hashB64 == "" {
		return logical.ErrorResponse("missing hash"), nil
	}

	chainIDInt := data.Get("chain_id").(int)
	var chainID *big.Int
	if chainIDInt > 0 {
		chainID = big.NewInt(int64(chainIDInt))
	}

	// Decode the hash
	hash, err := base64.StdEncoding.DecodeString(hashB64)
	if err != nil {
		return logical.ErrorResponse("invalid hash: not valid base64"), nil
	}

	if len(hash) != 32 {
		return logical.ErrorResponse("hash must be 32 bytes, got %d", len(hash)), nil
	}

	// Get the key
	entry, err := b.getKey(ctx, req.Storage, name)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return logical.ErrorResponse("key not found"), nil
	}

	// Parse the private key
	privKey, err := ParsePrivateKey(entry.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Sign with appropriate method
	var v, r, s *big.Int
	if chainID != nil && chainID.Sign() > 0 {
		// EIP-155 signing
		v, r, s, err = SignEIP155(privKey, hash, chainID)
	} else {
		// Legacy signing (v=27/28)
		v, r, s, err = SignLegacy(privKey, hash)
	}
	if err != nil {
		return nil, fmt.Errorf("signing failed: %w", err)
	}

	// Format r and s as 32-byte hex strings (zero-padded)
	rHex := fmt.Sprintf("%064x", r)
	sHex := fmt.Sprintf("%064x", s)
	vHex := fmt.Sprintf("%x", v)

	// Also compute Ethereum address for convenience
	ethAddr := deriveEthereumAddress(privKey.PubKey())

	return &logical.Response{
		Data: map[string]interface{}{
			"v":           vHex,
			"r":           rHex,
			"s":           sHex,
			"v_int":       v.Int64(),
			"public_key":  hex.EncodeToString(entry.PublicKey),
			"eth_address": formatEthereumAddress(ethAddr),
		},
	}, nil
}

const pathSignEVMHelpSyn = `Sign a hash with EIP-155 format for Ethereum transactions`

const pathSignEVMHelpDesc = `
This endpoint signs a 32-byte hash using the specified secp256k1 key and
returns the signature in Ethereum-compatible format (v, r, s).

The hash should typically be the Keccak256 hash of an RLP-encoded unsigned
transaction. This endpoint does NOT hash the input - it expects a pre-computed
32-byte hash.

Parameters:
  hash      - Base64-encoded 32-byte hash to sign
  chain_id  - EIP-155 chain ID (optional, default: 0 for legacy)

Chain IDs:
  1    - Ethereum Mainnet
  10   - OP Mainnet
  420  - OP Goerli (deprecated)
  11155420 - OP Sepolia
  8453 - Base Mainnet
  84531 - Base Goerli (deprecated)
  84532 - Base Sepolia

Examples:
  # Sign with EIP-155 (OP Mainnet, chain_id=10):
  $ bao write secp256k1/sign-evm/mykey hash="<base64-hash>" chain_id=10

  # Sign with legacy format (v=27/28):
  $ bao write secp256k1/sign-evm/mykey hash="<base64-hash>"

Response:
  v           - Hex-encoded v value (includes chain_id for EIP-155)
  r           - Hex-encoded r value (32 bytes, zero-padded)
  s           - Hex-encoded s value (32 bytes, zero-padded)
  v_int       - Integer value of v (for convenience)
  public_key  - Hex-encoded compressed public key
  eth_address - EIP-55 checksummed Ethereum address

Usage with OP Stack:
  The op-batcher and op-proposer compute the transaction hash internally,
  then call this endpoint to get the v, r, s values for the signed transaction.
`

