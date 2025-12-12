package secp256k1

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// pathExport returns the path definitions for key export operations.
func pathExport(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "export/" + framework.GenericNameRegex("name"),
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the key to export",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback:    b.pathKeyExport,
					Summary:     "Export a secp256k1 private key",
					Description: "Export a private key if it was created with exportable=true.",
				},
			},
			HelpSynopsis:    pathExportHelpSyn,
			HelpDescription: pathExportHelpDesc,
		},
	}
}

// pathKeyExport handles the key export operation.
func (b *backend) pathKeyExport(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("missing key name"), nil
	}

	// Get the key from storage
	entry, err := b.getKey(ctx, req.Storage, name)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return logical.ErrorResponse("key not found"), nil
	}

	// Check if the key is exportable
	if !entry.Exportable {
		return logical.ErrorResponse("key is not exportable"), nil
	}

	// Return the key material as base64-encoded
	// In production, this could be wrapped with the caller's public key
	// for secure transport (RSA-OAEP wrapping)
	keyBase64 := base64.StdEncoding.EncodeToString(entry.PrivateKey)

	return &logical.Response{
		Data: map[string]interface{}{
			"name":       name,
			"public_key": hex.EncodeToString(entry.PublicKey),
			"address":    hex.EncodeToString(deriveCosmosAddress(entry.PublicKey)),
			"keys": map[string]string{
				"1": keyBase64,
			},
			"created_at": entry.CreatedAt.Format(time.RFC3339),
			"imported":   entry.Imported,
		},
	}, nil
}

const pathExportHelpSyn = `Export a secp256k1 private key`

const pathExportHelpDesc = `
This endpoint allows you to export a secp256k1 private key from OpenBao.

The key can only be exported if it was created or imported with the
'exportable' flag set to true. This is a security measure to prevent
accidental exposure of keys meant to be kept secure within OpenBao.

Example:
  $ bao read secp256k1/export/mykey

Response:
  Key          Value
  ---          -----
  name         mykey
  public_key   02abc...
  address      def...
  keys         map[1:<base64-encoded-private-key>]

The 'keys' field contains a map with version "1" containing the
base64-encoded raw private key material.

WARNING: Exporting a private key exposes it outside of OpenBao's
protection. Only export keys when absolutely necessary, and ensure
the receiving system provides adequate security.
`
