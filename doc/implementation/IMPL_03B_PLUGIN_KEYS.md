# Implementation Guide: Plugin Key Paths

**Agent ID:** 03B  
**Parent:** Agent 03 (OpenBao Plugin)  
**Component:** Key CRUD Operations  
**Parallelizable:** ✅ Yes - Uses 03A backend, 03E crypto

---

## 1. Overview

Key management paths: create, read, delete, list.

### 1.1 Required Skills

| Skill           | Level    | Description         |
| --------------- | -------- | ------------------- |
| **Go**          | Advanced | HTTP handlers       |
| **OpenBao SDK** | Advanced | Path definitions    |

### 1.2 Files to Create

```
plugin/secp256k1/
├── path_keys.go
└── types.go
```

---

## 2. Specifications

### 2.1 types.go

```go
package secp256k1

import "time"

type keyEntry struct {
    PrivateKey  []byte    `json:"private_key"`
    PublicKey   []byte    `json:"public_key"`
    Exportable  bool      `json:"exportable"`
    CreatedAt   time.Time `json:"created_at"`
    Imported    bool      `json:"imported"`
}
```

### 2.2 path_keys.go

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
                "name":       {Type: framework.TypeString, Required: true},
                "exportable": {Type: framework.TypeBool, Default: false},
            },
            Operations: map[logical.Operation]framework.OperationHandler{
                logical.CreateOperation: &framework.PathOperation{Callback: b.pathKeyCreate},
                logical.ReadOperation:   &framework.PathOperation{Callback: b.pathKeyRead},
                logical.DeleteOperation: &framework.PathOperation{Callback: b.pathKeyDelete},
            },
        },
        {
            Pattern: "keys/?$",
            Operations: map[logical.Operation]framework.OperationHandler{
                logical.ListOperation: &framework.PathOperation{Callback: b.pathKeysList},
            },
        },
    }
}

func (b *backend) pathKeyCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    exportable := data.Get("exportable").(bool)
    
    // Check exists
    if existing, _ := req.Storage.Get(ctx, "keys/"+name); existing != nil {
        return logical.ErrorResponse("key already exists"), nil
    }
    
    // Generate key
    privKey, err := btcec.NewPrivateKey()
    if err != nil {
        return nil, err
    }
    
    entry := &keyEntry{
        PrivateKey: privKey.Serialize(),
        PublicKey:  privKey.PubKey().SerializeCompressed(),
        Exportable: exportable,
        CreatedAt:  time.Now().UTC(),
    }
    
    // Store
    storageEntry, _ := logical.StorageEntryJSON("keys/"+name, entry)
    if err := req.Storage.Put(ctx, storageEntry); err != nil {
        return nil, err
    }
    
    return &logical.Response{
        Data: map[string]interface{}{
            "name":       name,
            "public_key": hex.EncodeToString(entry.PublicKey),
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
    
    return &logical.Response{
        Data: map[string]interface{}{
            "name":       name,
            "public_key": hex.EncodeToString(entry.PublicKey),
            "address":    hex.EncodeToString(deriveCosmosAddress(entry.PublicKey)),
            "exportable": entry.Exportable,
            "created_at": entry.CreatedAt.Format(time.RFC3339),
        },
    }, nil
}

func (b *backend) pathKeyDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    
    if err := req.Storage.Delete(ctx, "keys/"+name); err != nil {
        return nil, err
    }
    
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

func (b *backend) getKey(ctx context.Context, storage logical.Storage, name string) (*keyEntry, error) {
    // Check cache
    b.cacheMu.RLock()
    if entry, ok := b.keyCache[name]; ok {
        b.cacheMu.RUnlock()
        return entry, nil
    }
    b.cacheMu.RUnlock()
    
    // Load from storage
    raw, err := storage.Get(ctx, "keys/"+name)
    if err != nil || raw == nil {
        return nil, err
    }
    
    var entry keyEntry
    if err := raw.DecodeJSON(&entry); err != nil {
        return nil, err
    }
    
    // Cache
    b.cacheMu.Lock()
    b.keyCache[name] = &entry
    b.cacheMu.Unlock()
    
    return &entry, nil
}
```

---

## 3. Deliverables

- [ ] Create key generates secp256k1
- [ ] Read returns public info only
- [ ] Delete removes from storage and cache
- [ ] List returns all key names
- [ ] Caching implemented

