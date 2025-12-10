package secp256k1

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// pathImport returns the path definitions for key import operations.
func pathImport(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "keys/" + framework.GenericNameRegex("name") + "/import",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the key to import",
					Required:    true,
				},
				"ciphertext": {
					Type:        framework.TypeString,
					Description: "Base64-encoded private key material (wrapped or raw)",
					Required:    true,
				},
				"exportable": {
					Type:        framework.TypeBool,
					Description: "Whether the imported key can be exported later",
					Default:     false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback:    b.pathKeyImport,
					Summary:     "Import an existing secp256k1 private key",
					Description: "Import an existing private key into OpenBao. The key material should be base64-encoded.",
				},
			},
			HelpSynopsis:    pathImportHelpSyn,
			HelpDescription: pathImportHelpDesc,
		},
	}
}

// pathKeyImport handles the key import operation.
func (b *backend) pathKeyImport(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("missing key name"), nil
	}

	ciphertext := data.Get("ciphertext").(string)
	if ciphertext == "" {
		return logical.ErrorResponse("missing ciphertext"), nil
	}

	exportable := data.Get("exportable").(bool)

	// Check if key already exists
	existing, err := req.Storage.Get(ctx, "keys/"+name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return logical.ErrorResponse("key already exists"), nil
	}

	// Decode the base64-encoded key material
	// In production, this would be RSA-OAEP encrypted with a wrapping key
	// For now, we support raw base64-encoded private keys
	privateKeyBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return logical.ErrorResponse("invalid ciphertext: not valid base64"), nil
	}
	defer secureZero(privateKeyBytes)

	// Validate that it's a valid secp256k1 private key
	if err := validatePrivateKeyBytes(privateKeyBytes); err != nil {
		return logical.ErrorResponse("invalid secp256k1 key: %s", err.Error()), nil
	}

	// Parse the private key to extract the public key
	privKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	if privKey == nil {
		return logical.ErrorResponse("failed to parse private key"), nil
	}

	// Create the key entry
	entry := &keyEntry{
		PrivateKey: make([]byte, len(privateKeyBytes)),
		PublicKey:  privKey.PubKey().SerializeCompressed(),
		Exportable: exportable,
		CreatedAt:  time.Now().UTC(),
		Imported:   true,
	}
	copy(entry.PrivateKey, privateKeyBytes)

	// Store the key
	storageEntry, err := logical.StorageEntryJSON("keys/"+name, entry)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(ctx, storageEntry); err != nil {
		return nil, err
	}

	// Update cache
	b.cacheMu.Lock()
	b.keyCache[name] = entry
	b.cacheMu.Unlock()

	return &logical.Response{
		Data: map[string]interface{}{
			"name":       name,
			"public_key": hex.EncodeToString(entry.PublicKey),
			"address":    hex.EncodeToString(deriveCosmosAddress(entry.PublicKey)),
			"exportable": exportable,
			"imported":   true,
			"created_at": entry.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// validatePrivateKeyBytes validates that the given bytes form a valid secp256k1 private key.
func validatePrivateKeyBytes(keyBytes []byte) error {
	// Private keys must be exactly 32 bytes
	_, err := ParsePrivateKey(keyBytes)
	return err
}

const pathImportHelpSyn = `Import an existing secp256k1 private key`

const pathImportHelpDesc = `
This endpoint allows you to import an existing secp256k1 private key into OpenBao.

The key material should be provided as a base64-encoded 32-byte raw private key.

Example:
  $ bao write secp256k1/keys/mykey/import \
      ciphertext="<base64-encoded-private-key>" \
      exportable=false

The 'exportable' flag determines whether the imported key can be exported
later using the export endpoint. Once set, this flag cannot be changed.

WARNING: Only import keys from trusted sources. Importing a compromised
key will result in compromised signatures.
`
