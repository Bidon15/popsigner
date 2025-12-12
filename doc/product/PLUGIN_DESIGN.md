# OpenBao secp256k1 Plugin Design

This document describes the design of the custom OpenBao Transit plugin that adds native secp256k1 signing support for Cosmos/Celestia compatibility.

---

## 1. Overview

### 1.1 Why a Plugin?

OpenBao Transit doesn't natively support secp256k1 (the curve used by Bitcoin, Ethereum, Cosmos, and Celestia). Our options were:

| Approach | Security | Complexity | Key Exposure |
|----------|----------|------------|--------------|
| **Hybrid (decrypt + sign locally)** | Good | Low | Key in app memory |
| **Custom Plugin** | **Excellent** | Medium | **Key never leaves OpenBao** |

**We chose the Plugin approach** because:
- Private keys **NEVER** leave OpenBao
- Signing happens inside the secure boundary
- Only the signature is returned to the caller
- Maximum security for production rollups

### 1.2 Security Model Comparison

```
┌─────────────────────────────────────────────────────────────────────┐
│                   Hybrid Approach (AWS KMS style)                   │
└─────────────────────────────────────────────────────────────────────┘

  App Memory                    OpenBao
  ┌─────────────────┐          ┌─────────────────┐
  │                 │  decrypt │                 │
  │  🔓 Plaintext   │◀─────────│  🔒 Encrypted   │
  │  private key    │          │  private key    │
  │                 │          │                 │
  │  Sign locally   │          └─────────────────┘
  │  ⚠️ Key exposed │
  └─────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                   Plugin Approach (BanhBao)                         │
└─────────────────────────────────────────────────────────────────────┘

  App Memory                    OpenBao + secp256k1 Plugin
  ┌─────────────────┐          ┌─────────────────────────────┐
  │                 │  sign    │                             │
  │  📝 Message     │─────────▶│  🔒 Private key (sealed)    │
  │                 │          │                             │
  │  ✅ Signature   │◀─────────│  Sign inside OpenBao        │
  │  (only output)  │          │  ✅ Key never exposed       │
  └─────────────────┘          └─────────────────────────────┘
```

---

## 2. Plugin Architecture

### 2.1 Component Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        OpenBao Server                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Built-in Transit Engine                   │   │
│  │                    (P-256, P-384, P-521)                     │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              secp256k1 Plugin (bao-plugin-secp256k1)        │   │
│  │  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │   │
│  │  │  Key Storage  │  │   Signing     │  │  Key Import   │   │   │
│  │  │  (encrypted)  │  │   (btcec)     │  │  (wrapped)    │   │   │
│  │  └───────────────┘  └───────────────┘  └───────────────┘   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Encrypted Storage (Raft)                  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 Plugin Registration

```go
// plugin/main.go
package main

import (
    "github.com/openbao/openbao/sdk/plugin"
    "github.com/Bidon15/banhbaoring/plugin/secp256k1"
)

func main() {
    plugin.Serve(&plugin.ServeOpts{
        BackendFactoryFunc: secp256k1.Factory,
    })
}
```

### 2.3 Backend Factory

```go
// plugin/secp256k1/backend.go
package secp256k1

import (
    "context"
    
    "github.com/openbao/openbao/sdk/framework"
    "github.com/openbao/openbao/sdk/logical"
)

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
    b := &backend{}
    b.Backend = &framework.Backend{
        BackendType: logical.TypeLogical,
        Help:        "secp256k1 signing engine for Cosmos/Celestia",
        Paths: []*framework.Path{
            pathKeys(b),
            pathSign(b),
            pathVerify(b),
            pathExport(b),
            pathImport(b),
        },
        Secrets:     []*framework.Secret{},
        Invalidate:  b.invalidate,
    }
    
    if err := b.Setup(ctx, conf); err != nil {
        return nil, err
    }
    
    return b, nil
}

type backend struct {
    *framework.Backend
}
```

---

## 3. API Endpoints

### 3.1 Mount the Plugin

```bash
# Register plugin
bao plugin register -sha256=<SHA256> secret bao-plugin-secp256k1

# Enable at path
bao secrets enable -path=secp256k1 bao-plugin-secp256k1
```

### 3.2 Endpoint Overview

| Method | Path | Description |
|--------|------|-------------|
| POST | `/secp256k1/keys/:name` | Create new key |
| GET | `/secp256k1/keys/:name` | Read key (public only) |
| LIST | `/secp256k1/keys` | List all keys |
| DELETE | `/secp256k1/keys/:name` | Delete key |
| POST | `/secp256k1/sign/:name` | Sign data |
| POST | `/secp256k1/verify/:name` | Verify signature |
| POST | `/secp256k1/keys/:name/import` | Import existing key |
| GET | `/secp256k1/export/:name` | Export key (if allowed) |

### 3.3 Create Key

**Endpoint:** `POST /v1/secp256k1/keys/:name`

**Request:**
```json
{
  "exportable": false,
  "derived": false
}
```

**Response:**
```json
{
  "data": {
    "name": "my-celestia-key",
    "public_key": "02a1b2c3d4e5f6...",
    "public_key_hex": "02a1b2c3d4e5f6...",
    "address": "celestia1abc123...",
    "address_hex": "0x...",
    "exportable": false,
    "created_at": "2025-01-10T12:00:00Z"
  }
}
```

**Implementation:**
```go
// plugin/secp256k1/path_keys.go
func (b *backend) pathKeyCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    exportable := data.Get("exportable").(bool)
    
    // Generate secp256k1 key pair
    privateKey, err := btcec.NewPrivateKey()
    if err != nil {
        return nil, err
    }
    
    // Store encrypted (OpenBao handles encryption)
    entry := &keyEntry{
        PrivateKey:  privateKey.Serialize(),
        PublicKey:   privateKey.PubKey().SerializeCompressed(),
        Exportable:  exportable,
        CreatedAt:   time.Now(),
    }
    
    // Persist to storage
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
            "name":        name,
            "public_key":  hex.EncodeToString(entry.PublicKey),
            "address":     deriveCosmosAddress(entry.PublicKey),
            "exportable":  exportable,
            "created_at":  entry.CreatedAt,
        },
    }, nil
}
```

### 3.4 Sign Data

**Endpoint:** `POST /v1/secp256k1/sign/:name`

**Request:**
```json
{
  "input": "base64-encoded-data",
  "prehashed": true,
  "hash_algorithm": "sha256",
  "output_format": "cosmos"
}
```

**Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `input` | string | Yes | Base64-encoded data to sign |
| `prehashed` | bool | No | If true, input is already hashed (default: false) |
| `hash_algorithm` | string | No | Hash algo if not prehashed: `sha256`, `keccak256` |
| `output_format` | string | No | `cosmos` (R\|\|S 64 bytes), `der`, `ethereum` (default: cosmos) |

**Response:**
```json
{
  "data": {
    "signature": "base64-encoded-signature",
    "public_key": "02a1b2c3d4e5f6...",
    "key_version": 1
  }
}
```

**Implementation:**
```go
// plugin/secp256k1/path_sign.go
func (b *backend) pathSign(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    inputB64 := data.Get("input").(string)
    prehashed := data.Get("prehashed").(bool)
    outputFormat := data.Get("output_format").(string)
    
    // Decode input
    input, err := base64.StdEncoding.DecodeString(inputB64)
    if err != nil {
        return logical.ErrorResponse("invalid base64 input"), nil
    }
    
    // Hash if needed
    var hash []byte
    if prehashed {
        hash = input
    } else {
        h := sha256.Sum256(input)
        hash = h[:]
    }
    
    // Retrieve key from storage
    entry, err := req.Storage.Get(ctx, "keys/"+name)
    if err != nil {
        return nil, err
    }
    if entry == nil {
        return logical.ErrorResponse("key not found"), nil
    }
    
    var keyEntry keyEntry
    if err := entry.DecodeJSON(&keyEntry); err != nil {
        return nil, err
    }
    
    // Sign with secp256k1
    privKey, _ := btcec.PrivKeyFromBytes(keyEntry.PrivateKey)
    sig := ecdsa.Sign(privKey, hash)
    
    // Format signature based on output_format
    var sigBytes []byte
    switch outputFormat {
    case "cosmos":
        sigBytes = formatCosmosSignature(sig)
    case "der":
        sigBytes = sig.Serialize()
    case "ethereum":
        sigBytes = formatEthereumSignature(sig, hash, privKey.PubKey())
    default:
        sigBytes = formatCosmosSignature(sig)
    }
    
    return &logical.Response{
        Data: map[string]interface{}{
            "signature":   base64.StdEncoding.EncodeToString(sigBytes),
            "public_key":  hex.EncodeToString(keyEntry.PublicKey),
            "key_version": 1,
        },
    }, nil
}

// formatCosmosSignature converts to R || S (64 bytes, low-S normalized)
func formatCosmosSignature(sig *ecdsa.Signature) []byte {
    r := sig.R()
    s := sig.S()
    
    // Normalize to low-S
    s = normalizeLowS(s)
    
    // Pad to 32 bytes each
    result := make([]byte, 64)
    rBytes := r.Bytes()
    sBytes := s.Bytes()
    
    copy(result[32-len(rBytes):32], rBytes)
    copy(result[64-len(sBytes):64], sBytes)
    
    return result
}

// normalizeLowS ensures S <= N/2 (BIP-62)
func normalizeLowS(s *btcec.ModNScalar) *btcec.ModNScalar {
    // secp256k1 curve order N
    // If S > N/2, return N - S
    if s.IsOverHalfOrder() {
        s.Negate()
    }
    return s
}
```

### 3.5 Import Key

**Endpoint:** `POST /v1/secp256k1/keys/:name/import`

**Request:**
```json
{
  "ciphertext": "base64-wrapped-key",
  "exportable": false
}
```

**Note:** Key must be wrapped using OpenBao's wrapping key (RSA-OAEP).

**Implementation:**
```go
func (b *backend) pathKeyImport(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
    name := data.Get("name").(string)
    ciphertext := data.Get("ciphertext").(string)
    exportable := data.Get("exportable").(bool)
    
    // Decrypt using OpenBao's internal wrapping key
    wrappedKey, _ := base64.StdEncoding.DecodeString(ciphertext)
    privateKeyBytes, err := b.unwrapKey(ctx, req.Storage, wrappedKey)
    if err != nil {
        return nil, err
    }
    defer secureZero(privateKeyBytes)
    
    // Validate it's a valid secp256k1 key
    privKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
    if privKey == nil {
        return logical.ErrorResponse("invalid secp256k1 private key"), nil
    }
    
    // Store
    entry := &keyEntry{
        PrivateKey:  privateKeyBytes,
        PublicKey:   privKey.PubKey().SerializeCompressed(),
        Exportable:  exportable,
        CreatedAt:   time.Now(),
        Imported:    true,
    }
    
    storageEntry, _ := logical.StorageEntryJSON("keys/"+name, entry)
    if err := req.Storage.Put(ctx, storageEntry); err != nil {
        return nil, err
    }
    
    return &logical.Response{
        Data: map[string]interface{}{
            "name":       name,
            "public_key": hex.EncodeToString(entry.PublicKey),
            "address":    deriveCosmosAddress(entry.PublicKey),
            "imported":   true,
        },
    }, nil
}
```

---

## 4. Storage Schema

### 4.1 Key Entry

```go
// plugin/secp256k1/types.go
type keyEntry struct {
    PrivateKey  []byte    `json:"private_key"`  // 32 bytes, encrypted by OpenBao
    PublicKey   []byte    `json:"public_key"`   // 33 bytes compressed
    Exportable  bool      `json:"exportable"`
    CreatedAt   time.Time `json:"created_at"`
    Imported    bool      `json:"imported"`
    Version     int       `json:"version"`
}
```

### 4.2 Storage Paths

```
secp256k1/
├── keys/
│   ├── my-validator      # Key entry JSON
│   ├── my-sequencer      # Key entry JSON
│   └── ...
├── config/
│   └── settings          # Plugin configuration
└── archive/
    └── ...               # Deleted keys (soft delete)
```

---

## 5. Security Measures

### 5.1 Memory Protection

```go
// secureZero wipes sensitive data from memory
func secureZero(b []byte) {
    for i := range b {
        b[i] = 0
    }
    runtime.KeepAlive(b)
}

// All private key operations use defer secureZero
func (b *backend) pathSign(...) (*logical.Response, error) {
    keyEntry := getKey(...)
    defer secureZero(keyEntry.PrivateKey)
    
    // Sign...
}
```

### 5.2 Access Control Policy

```hcl
# banhbao-policy.hcl
# Allow key creation
path "secp256k1/keys/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Allow signing
path "secp256k1/sign/*" {
  capabilities = ["create", "update"]
}

# Deny export by default (override per-key if needed)
path "secp256k1/export/*" {
  capabilities = ["deny"]
}
```

### 5.3 Audit Logging

All operations are automatically logged by OpenBao:

```json
{
  "time": "2025-01-10T12:00:00Z",
  "type": "request",
  "auth": {
    "client_token": "hmac-sha256:xxx",
    "policies": ["banhbao"]
  },
  "request": {
    "path": "secp256k1/sign/my-validator",
    "operation": "update",
    "data": {
      "input": "hmac-sha256:yyy"
    }
  },
  "response": {
    "data": {
      "signature": "hmac-sha256:zzz"
    }
  }
}
```

---

## 6. Build & Deployment

### 6.1 Build Plugin

```bash
# Build the plugin binary
cd plugin
go build -o bao-plugin-secp256k1 ./cmd/plugin

# Calculate SHA256 for registration
sha256sum bao-plugin-secp256k1
```

### 6.2 Plugin Directory Structure

```
plugin/
├── cmd/
│   └── plugin/
│       └── main.go           # Plugin entrypoint
├── secp256k1/
│   ├── backend.go            # Backend factory
│   ├── path_keys.go          # Key management
│   ├── path_sign.go          # Signing operations
│   ├── path_verify.go        # Verification
│   ├── path_import.go        # Key import
│   ├── path_export.go        # Key export
│   ├── types.go              # Data structures
│   └── crypto.go             # Crypto helpers
├── go.mod
└── go.sum
```

### 6.3 Kubernetes Deployment

```yaml
# openbao-values.yaml (Helm)
server:
  volumes:
    - name: plugins
      emptyDir: {}
  
  volumeMounts:
    - name: plugins
      mountPath: /vault/plugins
  
  extraInitContainers:
    - name: plugin-installer
      image: ghcr.io/bidon15/bao-plugin-secp256k1:latest
      command: ["cp", "/bao-plugin-secp256k1", "/plugins/"]
      volumeMounts:
        - name: plugins
          mountPath: /plugins

  extraEnvironmentVars:
    BAO_PLUGIN_DIR: /vault/plugins
```

### 6.4 Register Plugin on Startup

```bash
# init-script.sh (run after unseal)
#!/bin/bash

# Register the plugin
bao plugin register \
  -sha256=$(sha256sum /vault/plugins/bao-plugin-secp256k1 | cut -d' ' -f1) \
  secret bao-plugin-secp256k1

# Enable at secp256k1 path
bao secrets enable -path=secp256k1 bao-plugin-secp256k1

# Verify
bao secrets list | grep secp256k1
```

---

## 7. Client Integration

### 7.1 Updated BaoClient

```go
// bao_client.go
type BaoClient struct {
    httpClient   *http.Client
    baseURL      string
    token        string
    secp256k1Path string  // "secp256k1" by default
}

// CreateKey creates a new secp256k1 key
func (c *BaoClient) CreateKey(name string, exportable bool) (*KeyInfo, error) {
    resp, err := c.post(fmt.Sprintf("/v1/%s/keys/%s", c.secp256k1Path, name), map[string]interface{}{
        "exportable": exportable,
    })
    if err != nil {
        return nil, err
    }
    
    return &KeyInfo{
        Name:      resp.Data["name"].(string),
        PublicKey: resp.Data["public_key"].(string),
        Address:   resp.Data["address"].(string),
    }, nil
}

// Sign signs data using the secp256k1 key
func (c *BaoClient) Sign(keyName string, data []byte, prehashed bool) ([]byte, error) {
    resp, err := c.post(fmt.Sprintf("/v1/%s/sign/%s", c.secp256k1Path, keyName), map[string]interface{}{
        "input":         base64.StdEncoding.EncodeToString(data),
        "prehashed":     prehashed,
        "output_format": "cosmos",
    })
    if err != nil {
        return nil, err
    }
    
    sigB64 := resp.Data["signature"].(string)
    return base64.StdEncoding.DecodeString(sigB64)
}
```

### 7.2 Simplified BaoKeyring

```go
// bao_keyring.go
func (k *BaoKeyring) Sign(uid string, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
    // Get metadata (pubkey cached locally)
    meta, err := k.store.Get(uid)
    if err != nil {
        return nil, nil, err
    }
    
    // Hash message
    hash := sha256.Sum256(msg)
    
    // Sign via OpenBao plugin - key NEVER leaves OpenBao
    sig, err := k.client.Sign(uid, hash[:], true)
    if err != nil {
        return nil, nil, err
    }
    
    // Signature is already in Cosmos format (R || S, 64 bytes)
    return sig, meta.GetPubKey(), nil
}
```

---

## 8. Comparison with Alternatives

| Feature | AWS KMS | GCP Cloud KMS | BanhBao Plugin |
|---------|---------|---------------|----------------|
| secp256k1 support | ❌ | ❌ | ✅ Native |
| Key leaves vault | Yes (decrypt) | Yes (decrypt) | **Never** |
| Open source | ❌ | ❌ | ✅ |
| Self-hostable | ❌ | ❌ | ✅ |
| Cosmos signature format | Manual | Manual | ✅ Built-in |
| Audit logging | CloudTrail | Cloud Audit | ✅ OpenBao |

---

## 9. Future Enhancements

| Enhancement | Description | Priority |
|-------------|-------------|----------|
| Key versioning | Support key rotation with version history | Medium |
| Batch signing | Sign multiple messages in one request | Low |
| Threshold signatures | M-of-N signing (future) | Low |
| Hardware backing | HSM support for plugin storage | Medium |
| keccak256 support | For Ethereum compatibility | Low |

---

## 10. References

- [OpenBao Plugin Development](https://openbao.org/docs/plugins/)
- [btcec library](https://github.com/btcsuite/btcd/tree/master/btcec)
- [BIP-62 Low-S Signatures](https://github.com/bitcoin/bips/blob/master/bip-0062.mediawiki)
- [Cosmos SDK Keyring](https://docs.cosmos.network/main/user/run-node/keyring)

