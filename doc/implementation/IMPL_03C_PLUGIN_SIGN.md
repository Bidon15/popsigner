# Implementation Guide: Plugin Sign/Verify

**Agent ID:** 03C  
**Parent:** Agent 03 (OpenBao Plugin)  
**Component:** Signing and Verification  
**Parallelizable:** ✅ Yes - Uses 03B keys, 03E crypto

---

## 1. Overview

Sign and verify endpoints with Cosmos signature format.

### 1.1 Required Skills

| Skill      | Level    | Description                  |
| ---------- | -------- | ---------------------------- |
| **Go**     | Advanced | Crypto operations            |
| **Crypto** | Advanced | ECDSA, signature formats     |

### 1.2 Files to Create

```
plugin/secp256k1/
├── path_sign.go
└── path_verify.go
```

---

## 2. Specifications

### 2.1 path_sign.go

```go
package secp256k1

import (
    "context"
    "encoding/base64"
    "encoding/hex"
    "fmt"
    
    "github.com/btcsuite/btcd/btcec/v2"
    "github.com/btcsuite/btcd/btcec/v2/ecdsa"
    "github.com/openbao/openbao/sdk/framework"
    "github.com/openbao/openbao/sdk/logical"
)

func pathSign(b *backend) []*framework.Path {
    return []*framework.Path{
        {
            Pattern: "sign/" + framework.GenericNameRegex("name"),
            Fields: map[string]*framework.FieldSchema{
                "name":           {Type: framework.TypeString, Required: true},
                "input":          {Type: framework.TypeString, Required: true},
                "prehashed":      {Type: framework.TypeBool, Default: false},
                "hash_algorithm": {Type: framework.TypeString, Default: "sha256"},
                "output_format":  {Type: framework.TypeString, Default: "cosmos"},
            },
            Operations: map[logical.Operation]framework.OperationHandler{
                logical.UpdateOperation: &framework.PathOperation{Callback: b.pathSignWrite},
            },
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
        return logical.ErrorResponse("invalid base64"), nil
    }
    
    // Get key
    entry, err := b.getKey(ctx, req.Storage, name)
    if err != nil {
        return nil, err
    }
    if entry == nil {
        return logical.ErrorResponse("key not found"), nil
    }
    
    // Hash if needed
    var hash []byte
    if prehashed {
        if len(input) != 32 {
            return logical.ErrorResponse("prehashed must be 32 bytes"), nil
        }
        hash = input
    } else {
        switch hashAlgo {
        case "sha256":
            hash = hashSHA256(input)
        case "keccak256":
            hash = hashKeccak256(input)
        default:
            return logical.ErrorResponse("unsupported hash algorithm"), nil
        }
    }
    
    // Sign
    privKey, _ := btcec.PrivKeyFromBytes(entry.PrivateKey)
    sig := ecdsa.Sign(privKey, hash)
    
    // Format
    var sigBytes []byte
    switch outputFormat {
    case "cosmos":
        sigBytes = formatCosmosSignature(sig)
    case "der":
        sigBytes = sig.Serialize()
    default:
        sigBytes = formatCosmosSignature(sig)
    }
    
    return &logical.Response{
        Data: map[string]interface{}{
            "signature":   base64.StdEncoding.EncodeToString(sigBytes),
            "public_key":  hex.EncodeToString(entry.PublicKey),
            "key_version": 1,
        },
    }, nil
}
```

### 2.2 path_verify.go

```go
package secp256k1

import (
    "context"
    "encoding/base64"
    
    "github.com/btcsuite/btcd/btcec/v2"
    "github.com/btcsuite/btcd/btcec/v2/ecdsa"
    "github.com/openbao/openbao/sdk/framework"
    "github.com/openbao/openbao/sdk/logical"
)

func pathVerify(b *backend) []*framework.Path {
    return []*framework.Path{
        {
            Pattern: "verify/" + framework.GenericNameRegex("name"),
            Fields: map[string]*framework.FieldSchema{
                "name":      {Type: framework.TypeString, Required: true},
                "input":     {Type: framework.TypeString, Required: true},
                "signature": {Type: framework.TypeString, Required: true},
                "prehashed": {Type: framework.TypeBool, Default: false},
            },
            Operations: map[logical.Operation]framework.OperationHandler{
                logical.UpdateOperation: &framework.PathOperation{Callback: b.pathVerifyWrite},
            },
        },
    }
}

func (b *backend) pathVerifyWrite(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    inputB64 := data.Get("input").(string)
    sigB64 := data.Get("signature").(string)
    prehashed := data.Get("prehashed").(bool)
    
    input, _ := base64.StdEncoding.DecodeString(inputB64)
    sigBytes, _ := base64.StdEncoding.DecodeString(sigB64)
    
    entry, err := b.getKey(ctx, req.Storage, name)
    if err != nil || entry == nil {
        return logical.ErrorResponse("key not found"), nil
    }
    
    var hash []byte
    if prehashed {
        hash = input
    } else {
        hash = hashSHA256(input)
    }
    
    pubKey, _ := btcec.ParsePubKey(entry.PublicKey)
    
    // Parse signature (Cosmos format: R||S)
    if len(sigBytes) != 64 {
        return &logical.Response{Data: map[string]interface{}{"valid": false}}, nil
    }
    
    r := new(btcec.ModNScalar)
    s := new(btcec.ModNScalar)
    r.SetByteSlice(sigBytes[:32])
    s.SetByteSlice(sigBytes[32:])
    
    sig := ecdsa.NewSignature(r, s)
    valid := sig.Verify(hash, pubKey)
    
    return &logical.Response{
        Data: map[string]interface{}{"valid": valid},
    }, nil
}
```

---

## 3. Deliverables

- [ ] Sign returns 64-byte Cosmos format
- [ ] Low-S normalization applied
- [ ] Verify parses Cosmos signature
- [ ] Hash algorithms: sha256, keccak256
- [ ] Output formats: cosmos, der

