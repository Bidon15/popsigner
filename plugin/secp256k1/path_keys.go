package secp256k1

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// pathKeys returns the path definitions for key CRUD operations.
func pathKeys(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "keys/" + framework.GenericNameRegex("name"),
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeString,
					Description: "Name of the key",
					Required:    true,
				},
				"exportable": {
					Type:        framework.TypeBool,
					Description: "Whether the key can be exported",
					Default:     false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{Callback: b.pathKeyCreate},
				logical.UpdateOperation: &framework.PathOperation{Callback: b.pathKeyCreate},
				logical.ReadOperation:   &framework.PathOperation{Callback: b.pathKeyRead},
				logical.DeleteOperation: &framework.PathOperation{Callback: b.pathKeyDelete},
			},
			ExistenceCheck:  b.pathKeyExistenceCheck,
			HelpSynopsis:    pathKeysHelpSyn,
			HelpDescription: pathKeysHelpDesc,
		},
		{
			Pattern: "keys/?$",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{Callback: b.pathKeysList},
			},
			HelpSynopsis:    pathKeysListHelpSyn,
			HelpDescription: pathKeysListHelpDesc,
		},
	}
}

// pathKeyExistenceCheck checks if a key exists.
func (b *backend) pathKeyExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	name := data.Get("name").(string)
	entry, err := b.getKey(ctx, req.Storage, name)
	if err != nil {
		return false, err
	}
	return entry != nil, nil
}

// pathKeyCreate handles key creation.
func (b *backend) pathKeyCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("missing key name"), nil
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

	// Generate new key
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	// Store both compressed and uncompressed public keys
	pubKeyCompressed := privKey.PubKey().SerializeCompressed()
	pubKeyUncompressed := privKey.PubKey().SerializeUncompressed()

	entry := &keyEntry{
		PrivateKey:            privKey.Serialize(),
		PublicKey:             pubKeyCompressed,
		PublicKeyUncompressed: pubKeyUncompressed,
		Exportable:            exportable,
		CreatedAt:             time.Now().UTC(),
		Imported:              false,
	}

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

	// Derive both addresses
	cosmosAddr := deriveCosmosAddress(entry.PublicKey)
	ethAddr := deriveEthereumAddress(privKey.PubKey())

	return &logical.Response{
		Data: map[string]interface{}{
			"name":        name,
			"public_key":  hex.EncodeToString(entry.PublicKey),
			"address":     hex.EncodeToString(cosmosAddr),
			"eth_address": formatEthereumAddress(ethAddr),
			"exportable":  exportable,
			"created_at":  entry.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// pathKeyRead handles reading key metadata.
func (b *backend) pathKeyRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("missing key name"), nil
	}

	entry, err := b.getKey(ctx, req.Storage, name)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return logical.ErrorResponse("key not found"), nil
	}

	// Derive addresses
	cosmosAddr := deriveCosmosAddress(entry.PublicKey)

	// Parse public key to derive Ethereum address
	pubKey, err := ParsePublicKey(entry.PublicKey)
	if err != nil {
		return nil, err
	}
	ethAddr := deriveEthereumAddress(pubKey)

	return &logical.Response{
		Data: map[string]interface{}{
			"name":        name,
			"public_key":  hex.EncodeToString(entry.PublicKey),
			"address":     hex.EncodeToString(cosmosAddr),
			"eth_address": formatEthereumAddress(ethAddr),
			"exportable":  entry.Exportable,
			"created_at":  entry.CreatedAt.Format(time.RFC3339),
			"imported":    entry.Imported,
		},
	}, nil
}

// pathKeyDelete handles key deletion.
func (b *backend) pathKeyDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("missing key name"), nil
	}

	if err := req.Storage.Delete(ctx, "keys/"+name); err != nil {
		return nil, err
	}

	// Remove from cache
	b.cacheMu.Lock()
	delete(b.keyCache, name)
	b.cacheMu.Unlock()

	return nil, nil
}

// pathKeysList handles listing all keys.
func (b *backend) pathKeysList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	keys, err := req.Storage.List(ctx, "keys/")
	if err != nil {
		return nil, err
	}
	// Ensure we return an empty slice, not nil
	if keys == nil {
		keys = []string{}
	}

	keyInfo := make(map[string]interface{}, len(keys))
	for _, name := range keys {
		entry, err := b.getKey(ctx, req.Storage, name)
		if err != nil {
			continue
		}
		if entry == nil {
			continue
		}

		// Derive addresses
		cosmosAddr := deriveCosmosAddress(entry.PublicKey)
		pubKey, _ := ParsePublicKey(entry.PublicKey)
		ethAddr := deriveEthereumAddress(pubKey)

		keyInfo[name] = map[string]interface{}{
			"name":        name,
			"public_key":  hex.EncodeToString(entry.PublicKey),
			"address":     hex.EncodeToString(cosmosAddr),
			"eth_address": formatEthereumAddress(ethAddr),
			"exportable":  entry.Exportable,
			"created_at":  entry.CreatedAt.Format(time.RFC3339),
		}
	}

	return logical.ListResponseWithInfo(keys, keyInfo), nil
}

const pathKeysHelpSyn = `Manage secp256k1 keys`

const pathKeysHelpDesc = `
This endpoint allows you to create, read, and delete secp256k1 keys.

To create a new key:
  $ bao write secp256k1/keys/mykey exportable=false

To read key metadata:
  $ bao read secp256k1/keys/mykey

To delete a key:
  $ bao delete secp256k1/keys/mykey
`

const pathKeysListHelpSyn = `List all secp256k1 keys`

const pathKeysListHelpDesc = `
This endpoint lists all secp256k1 keys stored in OpenBao.

  $ bao list secp256k1/keys/
`
