# Implementation Guide: Plugin Import/Export

**Agent ID:** 03D  
**Parent:** Agent 03 (OpenBao Plugin)  
**Component:** Key Import and Export  
**Parallelizable:** ✅ Yes - Uses 03B keys

---

## 1. Overview

Import existing keys and export (if allowed) from OpenBao.

### 1.1 Required Skills

| Skill      | Level    | Description               |
| ---------- | -------- | ------------------------- |
| **Go**     | Advanced | Crypto, RSA               |
| **Crypto** | Advanced | Key wrapping, RSA-OAEP    |

### 1.2 Files to Create

```
plugin/secp256k1/
├── path_import.go
└── path_export.go
```

---

## 2. Specifications

### 2.1 path_import.go

```go
package secp256k1

import (
    "context"
    "encoding/base64"
    "encoding/hex"
    "time"
    
    "github.com/btcsuite/btcd/btcec/v2"
    "github.com/openbao/openbao/sdk/framework"
    "github.com/openbao/openbao/sdk/logical"
)

func pathImport(b *backend) []*framework.Path {
    return []*framework.Path{
        {
            Pattern: "keys/" + framework.GenericNameRegex("name") + "/import",
            Fields: map[string]*framework.FieldSchema{
                "name":       {Type: framework.TypeString, Required: true},
                "ciphertext": {Type: framework.TypeString, Required: true},
                "exportable": {Type: framework.TypeBool, Default: false},
            },
            Operations: map[logical.Operation]framework.OperationHandler{
                logical.UpdateOperation: &framework.PathOperation{Callback: b.pathKeyImport},
            },
        },
    }
}

func (b *backend) pathKeyImport(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    ciphertext := data.Get("ciphertext").(string)
    exportable := data.Get("exportable").(bool)
    
    // Decrypt wrapped key (using OpenBao's wrapping mechanism)
    wrappedKey, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return logical.ErrorResponse("invalid ciphertext"), nil
    }
    
    // Note: In production, implement actual key unwrapping
    // This is simplified - real implementation needs RSA-OAEP decryption
    privateKeyBytes := wrappedKey // Placeholder
    defer secureZero(privateKeyBytes)
    
    // Validate key
    privKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
    if privKey == nil {
        return logical.ErrorResponse("invalid secp256k1 key"), nil
    }
    
    entry := &keyEntry{
        PrivateKey: privateKeyBytes,
        PublicKey:  privKey.PubKey().SerializeCompressed(),
        Exportable: exportable,
        CreatedAt:  time.Now().UTC(),
        Imported:   true,
    }
    
    storageEntry, _ := logical.StorageEntryJSON("keys/"+name, entry)
    if err := req.Storage.Put(ctx, storageEntry); err != nil {
        return nil, err
    }
    
    return &logical.Response{
        Data: map[string]interface{}{
            "name":       name,
            "public_key": hex.EncodeToString(entry.PublicKey),
            "address":    hex.EncodeToString(deriveCosmosAddress(entry.PublicKey)),
            "imported":   true,
        },
    }, nil
}
```

### 2.2 path_export.go

```go
package secp256k1

import (
    "context"
    "encoding/base64"
    
    "github.com/openbao/openbao/sdk/framework"
    "github.com/openbao/openbao/sdk/logical"
)

func pathExport(b *backend) []*framework.Path {
    return []*framework.Path{
        {
            Pattern: "export/" + framework.GenericNameRegex("name"),
            Fields: map[string]*framework.FieldSchema{
                "name": {Type: framework.TypeString, Required: true},
            },
            Operations: map[logical.Operation]framework.OperationHandler{
                logical.ReadOperation: &framework.PathOperation{Callback: b.pathKeyExport},
            },
        },
    }
}

func (b *backend) pathKeyExport(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    
    entry, err := b.getKey(ctx, req.Storage, name)
    if err != nil {
        return nil, err
    }
    if entry == nil {
        return logical.ErrorResponse("key not found"), nil
    }
    
    if !entry.Exportable {
        return logical.ErrorResponse("key is not exportable"), nil
    }
    
    // Return wrapped key
    // In production, wrap with caller's public key
    return &logical.Response{
        Data: map[string]interface{}{
            "keys": map[string]string{
                "1": base64.StdEncoding.EncodeToString(entry.PrivateKey),
            },
        },
    }, nil
}
```

---

## 3. Deliverables

- [ ] Import accepts wrapped key material
- [ ] Import validates secp256k1 key format
- [ ] Export checks exportable flag
- [ ] Export returns wrapped key
- [ ] Memory wiped after use (secureZero)

