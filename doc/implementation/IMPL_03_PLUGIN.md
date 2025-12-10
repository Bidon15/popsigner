# Implementation Guide: OpenBao secp256k1 Plugin

**Agent ID:** 03  
**Component:** OpenBao Secrets Engine Plugin  
**Parallelizable:** ✅ Yes - Completely independent (server-side code)

---

## 1. Overview

This agent builds the custom OpenBao plugin that provides native secp256k1 key management and signing. This is the server-side component that runs inside OpenBao.

### 1.1 Required Skills

| Skill                     | Level    | Description                     |
| ------------------------- | -------- | ------------------------------- |
| **Go**                    | Advanced | Plugin architecture, interfaces |
| **OpenBao/Vault Plugins** | Advanced | Secrets engine development      |
| **Cryptography**          | Advanced | secp256k1, ECDSA, btcec library |
| **Security**              | Advanced | Key handling, memory safety     |

### 1.2 Files to Create

```
plugin/
├── cmd/
│   └── plugin/
│       └── main.go           # Plugin entrypoint
├── secp256k1/
│   ├── backend.go            # Backend factory
│   ├── path_keys.go          # Key CRUD operations
│   ├── path_sign.go          # Signing operations
│   ├── path_verify.go        # Verification
│   ├── path_import.go        # Key import
│   ├── path_export.go        # Key export
│   ├── crypto.go             # Crypto helpers
│   ├── types.go              # Internal types
│   └── *_test.go             # Unit tests
├── go.mod
└── go.sum
```

---

## 2. Detailed Specifications

### 2.1 plugin/cmd/plugin/main.go

```go
package main

import (
    "log"
    "os"

    "github.com/openbao/openbao/sdk/plugin"
    "github.com/Bidon15/banhbaoring/plugin/secp256k1"
)

func main() {
    meta := &plugin.ServeOpts{
        BackendFactoryFunc: secp256k1.Factory,
    }

    if err := plugin.Serve(meta); err != nil {
        log.Printf("plugin shutting down: %v", err)
        os.Exit(1)
    }
}
```

### 2.2 plugin/secp256k1/backend.go

```go
package secp256k1

import (
    "context"
    "sync"

    "github.com/openbao/openbao/sdk/framework"
    "github.com/openbao/openbao/sdk/logical"
)

// Factory creates a new backend instance
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
    b := &backend{
        keyCache: make(map[string]*keyEntry),
    }

    b.Backend = &framework.Backend{
        Help:        backendHelp,
        BackendType: logical.TypeLogical,
        Paths: framework.PathAppend(
            pathKeys(b),
            pathSign(b),
            pathVerify(b),
            pathImport(b),
            pathExport(b),
        ),
        Secrets:     []*framework.Secret{},
        Invalidate:  b.invalidate,
        PathsSpecial: &logical.Paths{
            SealWrapStorage: []string{"keys/"},
        },
    }

    if err := b.Setup(ctx, conf); err != nil {
        return nil, err
    }

    return b, nil
}

type backend struct {
    *framework.Backend
    cacheMu  sync.RWMutex
    keyCache map[string]*keyEntry
}

func (b *backend) invalidate(ctx context.Context, key string) {
    b.cacheMu.Lock()
    defer b.cacheMu.Unlock()

    // Clear cache on invalidation
    if key == "keys/" {
        b.keyCache = make(map[string]*keyEntry)
    }
}

const backendHelp = `
The secp256k1 secrets engine provides native secp256k1 key management
and signing for Cosmos/Celestia applications.

Keys are stored encrypted using OpenBao's storage encryption.
Private keys NEVER leave OpenBao - only signatures are returned.
`
```

### 2.3 plugin/secp256k1/types.go

```go
package secp256k1

import "time"

// keyEntry is stored in OpenBao's encrypted storage
type keyEntry struct {
    PrivateKey  []byte    `json:"private_key"`  // 32 bytes
    PublicKey   []byte    `json:"public_key"`   // 33 bytes compressed
    Exportable  bool      `json:"exportable"`
    CreatedAt   time.Time `json:"created_at"`
    Imported    bool      `json:"imported"`
}

// OutputFormat specifies signature output format
type OutputFormat string

const (
    OutputFormatCosmos   OutputFormat = "cosmos"   // R||S (64 bytes)
    OutputFormatDER      OutputFormat = "der"      // ASN.1 DER
    OutputFormatEthereum OutputFormat = "ethereum" // R||S||V (65 bytes)
)

// HashAlgorithm specifies hash algorithm
type HashAlgorithm string

const (
    HashAlgoSHA256    HashAlgorithm = "sha256"
    HashAlgoKeccak256 HashAlgorithm = "keccak256"
)
```

### 2.4 plugin/secp256k1/crypto.go

```go
package secp256k1

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "math/big"
    "runtime"

    "github.com/btcsuite/btcd/btcec/v2"
    "github.com/btcsuite/btcd/btcec/v2/ecdsa"
    "golang.org/x/crypto/ripemd160"
    "golang.org/x/crypto/sha3"
)

// generateKeyPair creates a new secp256k1 key pair
func generateKeyPair() (*btcec.PrivateKey, error) {
    return btcec.NewPrivateKey()
}

// signHash signs a 32-byte hash with the private key
func signHash(privKey *btcec.PrivateKey, hash []byte) (*ecdsa.Signature, error) {
    if len(hash) != 32 {
        return nil, fmt.Errorf("hash must be 32 bytes, got %d", len(hash))
    }
    return ecdsa.Sign(privKey, hash), nil
}

// formatCosmosSignature formats signature as R||S (64 bytes)
func formatCosmosSignature(sig *ecdsa.Signature) []byte {
    r := sig.R()
    s := sig.S()

    // Normalize to low-S
    s = normalizeLowS(s)

    result := make([]byte, 64)
    r.PutBytesUnchecked(result[:32])
    s.PutBytesUnchecked(result[32:])

    return result
}

// normalizeLowS ensures S <= N/2 (BIP-62)
func normalizeLowS(s *btcec.ModNScalar) *btcec.ModNScalar {
    if s.IsOverHalfOrder() {
        s.Negate()
    }
    return s
}

// hashSHA256 computes SHA-256 hash
func hashSHA256(data []byte) []byte {
    h := sha256.Sum256(data)
    return h[:]
}

// hashKeccak256 computes Keccak-256 hash (Ethereum)
func hashKeccak256(data []byte) []byte {
    h := sha3.NewLegacyKeccak256()
    h.Write(data)
    return h.Sum(nil)
}

// deriveCosmosAddress derives a Cosmos address from public key
func deriveCosmosAddress(pubKey []byte) []byte {
    // SHA256 of public key
    sha := sha256.Sum256(pubKey)
    // RIPEMD160 of SHA256
    rip := ripemd160.New()
    rip.Write(sha[:])
    return rip.Sum(nil)
}

// secureZero wipes sensitive data from memory
func secureZero(b []byte) {
    for i := range b {
        b[i] = 0
    }
    runtime.KeepAlive(b)
}

// publicKeyHex returns hex-encoded compressed public key
func publicKeyHex(pubKey *btcec.PublicKey) string {
    return hex.EncodeToString(pubKey.SerializeCompressed())
}
```

### 2.5 plugin/secp256k1/path_keys.go

```go
package secp256k1

import (
    "context"
    "encoding/hex"
    "fmt"
    "time"

    "github.com/btcsuite/btcd/btcec/v2"
    "github.com/openbao/openbao/sdk/framework"
    "github.com/openbao/openbao/sdk/logical"
)

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
                    Description: "Allow key export",
                    Default:     false,
                },
            },
            Operations: map[logical.Operation]framework.OperationHandler{
                logical.CreateOperation: &framework.PathOperation{
                    Callback: b.pathKeyCreate,
                    Summary:  "Create a new secp256k1 key",
                },
                logical.ReadOperation: &framework.PathOperation{
                    Callback: b.pathKeyRead,
                    Summary:  "Read key information (public only)",
                },
                logical.DeleteOperation: &framework.PathOperation{
                    Callback: b.pathKeyDelete,
                    Summary:  "Delete a key",
                },
            },
            ExistenceCheck: b.pathKeyExistenceCheck,
            HelpSynopsis:   "Manage secp256k1 keys",
        },
        {
            Pattern: "keys/?$",
            Operations: map[logical.Operation]framework.OperationHandler{
                logical.ListOperation: &framework.PathOperation{
                    Callback: b.pathKeysList,
                    Summary:  "List all keys",
                },
            },
            HelpSynopsis: "List secp256k1 keys",
        },
    }
}

func (b *backend) pathKeyCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    exportable := data.Get("exportable").(bool)

    // Check if key exists
    existing, err := req.Storage.Get(ctx, "keys/"+name)
    if err != nil {
        return nil, err
    }
    if existing != nil {
        return logical.ErrorResponse("key already exists"), nil
    }

    // Generate key pair
    privKey, err := generateKeyPair()
    if err != nil {
        return nil, fmt.Errorf("failed to generate key: %w", err)
    }

    entry := &keyEntry{
        PrivateKey:  privKey.Serialize(),
        PublicKey:   privKey.PubKey().SerializeCompressed(),
        Exportable:  exportable,
        CreatedAt:   time.Now().UTC(),
        Imported:    false,
    }

    // Persist to storage (encrypted by OpenBao)
    storageEntry, err := logical.StorageEntryJSON("keys/"+name, entry)
    if err != nil {
        return nil, err
    }
    if err := req.Storage.Put(ctx, storageEntry); err != nil {
        return nil, err
    }

    // Return public info only
    return &logical.Response{
        Data: map[string]interface{}{
            "name":       name,
            "public_key": publicKeyHex(privKey.PubKey()),
            "address":    hex.EncodeToString(deriveCosmosAddress(entry.PublicKey)),
            "exportable": exportable,
            "created_at": entry.CreatedAt.Format(time.RFC3339),
        },
    }, nil
}

func (b *backend) pathKeyRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)

    entry, err := b.getKey(ctx, req.Storage, name)
    if err != nil {
        return nil, err
    }
    if entry == nil {
        return logical.ErrorResponse("key not found"), nil
    }

    pubKey, _ := btcec.ParsePubKey(entry.PublicKey)

    return &logical.Response{
        Data: map[string]interface{}{
            "name":       name,
            "public_key": publicKeyHex(pubKey),
            "address":    hex.EncodeToString(deriveCosmosAddress(entry.PublicKey)),
            "exportable": entry.Exportable,
            "created_at": entry.CreatedAt.Format(time.RFC3339),
            "imported":   entry.Imported,
        },
    }, nil
}

func (b *backend) pathKeyDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)

    if err := req.Storage.Delete(ctx, "keys/"+name); err != nil {
        return nil, err
    }

    // Clear from cache
    b.cacheMu.Lock()
    delete(b.keyCache, name)
    b.cacheMu.Unlock()

    return nil, nil
}

func (b *backend) pathKeysList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    keys, err := req.Storage.List(ctx, "keys/")
    if err != nil {
        return nil, err
    }

    return logical.ListResponse(keys), nil
}

func (b *backend) pathKeyExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
    name := data.Get("name").(string)
    entry, err := req.Storage.Get(ctx, "keys/"+name)
    if err != nil {
        return false, err
    }
    return entry != nil, nil
}

func (b *backend) getKey(ctx context.Context, storage logical.Storage, name string) (*keyEntry, error) {
    // Check cache first
    b.cacheMu.RLock()
    if entry, ok := b.keyCache[name]; ok {
        b.cacheMu.RUnlock()
        return entry, nil
    }
    b.cacheMu.RUnlock()

    // Load from storage
    raw, err := storage.Get(ctx, "keys/"+name)
    if err != nil {
        return nil, err
    }
    if raw == nil {
        return nil, nil
    }

    var entry keyEntry
    if err := raw.DecodeJSON(&entry); err != nil {
        return nil, err
    }

    // Update cache
    b.cacheMu.Lock()
    b.keyCache[name] = &entry
    b.cacheMu.Unlock()

    return &entry, nil
}
```

### 2.6 plugin/secp256k1/path_sign.go

```go
package secp256k1

import (
    "context"
    "encoding/base64"
    "fmt"

    "github.com/btcsuite/btcd/btcec/v2"
    "github.com/openbao/openbao/sdk/framework"
    "github.com/openbao/openbao/sdk/logical"
)

func pathSign(b *backend) []*framework.Path {
    return []*framework.Path{
        {
            Pattern: "sign/" + framework.GenericNameRegex("name"),
            Fields: map[string]*framework.FieldSchema{
                "name": {
                    Type:        framework.TypeString,
                    Description: "Key name",
                    Required:    true,
                },
                "input": {
                    Type:        framework.TypeString,
                    Description: "Base64-encoded data to sign",
                    Required:    true,
                },
                "prehashed": {
                    Type:        framework.TypeBool,
                    Description: "If true, input is already hashed",
                    Default:     false,
                },
                "hash_algorithm": {
                    Type:        framework.TypeString,
                    Description: "Hash algorithm: sha256, keccak256",
                    Default:     "sha256",
                },
                "output_format": {
                    Type:        framework.TypeString,
                    Description: "Output format: cosmos, der, ethereum",
                    Default:     "cosmos",
                },
            },
            Operations: map[logical.Operation]framework.OperationHandler{
                logical.UpdateOperation: &framework.PathOperation{
                    Callback: b.pathSignWrite,
                    Summary:  "Sign data with secp256k1 key",
                },
            },
            HelpSynopsis: "Sign data using a secp256k1 key",
        },
    }
}

func (b *backend) pathSignWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    inputB64 := data.Get("input").(string)
    prehashed := data.Get("prehashed").(bool)
    hashAlgo := data.Get("hash_algorithm").(string)
    outputFormat := data.Get("output_format").(string)

    // Decode input
    input, err := base64.StdEncoding.DecodeString(inputB64)
    if err != nil {
        return logical.ErrorResponse("invalid base64 input: %v", err), nil
    }

    // Get key
    entry, err := b.getKey(ctx, req.Storage, name)
    if err != nil {
        return nil, err
    }
    if entry == nil {
        return logical.ErrorResponse("key not found"), nil
    }

    // Compute hash if needed
    var hash []byte
    if prehashed {
        if len(input) != 32 {
            return logical.ErrorResponse("prehashed input must be 32 bytes"), nil
        }
        hash = input
    } else {
        switch HashAlgorithm(hashAlgo) {
        case HashAlgoSHA256:
            hash = hashSHA256(input)
        case HashAlgoKeccak256:
            hash = hashKeccak256(input)
        default:
            return logical.ErrorResponse("unsupported hash algorithm: %s", hashAlgo), nil
        }
    }

    // Sign
    privKey, _ := btcec.PrivKeyFromBytes(entry.PrivateKey)
    sig, err := signHash(privKey, hash)
    if err != nil {
        return nil, fmt.Errorf("signing failed: %w", err)
    }

    // Format output
    var sigBytes []byte
    switch OutputFormat(outputFormat) {
    case OutputFormatCosmos:
        sigBytes = formatCosmosSignature(sig)
    case OutputFormatDER:
        sigBytes = sig.Serialize()
    default:
        sigBytes = formatCosmosSignature(sig)
    }

    return &logical.Response{
        Data: map[string]interface{}{
            "signature":   base64.StdEncoding.EncodeToString(sigBytes),
            "public_key":  publicKeyHex(privKey.PubKey()),
            "key_version": 1,
        },
    }, nil
}
```

---

## 3. Unit Test Requirements

### 3.1 Key Tests

```go
func TestPathKeyCreate(t *testing.T) {
    b, storage := getTestBackend(t)

    resp, err := b.HandleRequest(context.Background(), &logical.Request{
        Operation: logical.CreateOperation,
        Path:      "keys/test-key",
        Storage:   storage,
        Data: map[string]interface{}{
            "exportable": false,
        },
    })

    require.NoError(t, err)
    require.NotNil(t, resp)
    require.NotEmpty(t, resp.Data["public_key"])
    require.NotEmpty(t, resp.Data["address"])
}
```

### 3.2 Signing Tests

```go
func TestPathSign_Cosmos(t *testing.T) {
    b, storage := getTestBackend(t)

    // Create key
    b.HandleRequest(context.Background(), &logical.Request{
        Operation: logical.CreateOperation,
        Path:      "keys/sign-test",
        Storage:   storage,
    })

    // Sign
    hash := sha256.Sum256([]byte("test message"))
    resp, err := b.HandleRequest(context.Background(), &logical.Request{
        Operation: logical.UpdateOperation,
        Path:      "sign/sign-test",
        Storage:   storage,
        Data: map[string]interface{}{
            "input":         base64.StdEncoding.EncodeToString(hash[:]),
            "prehashed":     true,
            "output_format": "cosmos",
        },
    })

    require.NoError(t, err)
    sig, _ := base64.StdEncoding.DecodeString(resp.Data["signature"].(string))
    require.Len(t, sig, 64) // Cosmos format is R||S = 64 bytes
}
```

---

## 4. Success Criteria

- [ ] Plugin builds and registers with OpenBao
- [ ] Key creation generates valid secp256k1 keys
- [ ] Public keys are 33-byte compressed format
- [ ] Signatures are 64-byte Cosmos format (R||S)
- [ ] Low-S normalization is applied (BIP-62)
- [ ] Private keys never appear in responses
- [ ] Import/export work correctly
- [ ] All unit tests pass

---

## 5. Build & Deploy

```bash
# Build plugin
cd plugin
go build -o bao-plugin-secp256k1 ./cmd/plugin

# Calculate SHA256
sha256sum bao-plugin-secp256k1

# Register with OpenBao
bao plugin register -sha256=<SHA> secret bao-plugin-secp256k1
bao secrets enable -path=secp256k1 bao-plugin-secp256k1
```

---

## 6. Dependencies

```go
// plugin/go.mod
require (
    github.com/btcsuite/btcd/btcec/v2 v2.3.2
    github.com/openbao/openbao/sdk v0.11.0
    golang.org/x/crypto v0.18.0
)
```

---

## 7. Deliverables Checklist

- [ ] `plugin/cmd/plugin/main.go` - Entrypoint
- [ ] `plugin/secp256k1/backend.go` - Backend factory
- [ ] `plugin/secp256k1/path_keys.go` - Key management
- [ ] `plugin/secp256k1/path_sign.go` - Signing
- [ ] `plugin/secp256k1/path_verify.go` - Verification
- [ ] `plugin/secp256k1/crypto.go` - Crypto helpers
- [ ] All tests pass
- [ ] Plugin registers successfully
- [ ] Memory safety verified (secureZero usage)
