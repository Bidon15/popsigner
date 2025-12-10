# Implementation Guide: Types & Constants

**Agent ID:** 01A  
**Parent:** Agent 01 (Foundation Layer)  
**Component:** Shared Types and Constants  
**Parallelizable:** ✅ Yes - Foundation, no dependencies

---

## 1. Overview

Define all shared types, constants, and configuration structures used across the codebase.

### 1.1 Required Skills

| Skill      | Level        | Description                    |
| ---------- | ------------ | ------------------------------ |
| **Go**     | Intermediate | Structs, interfaces, constants |
| **Design** | Intermediate | API design, type safety        |

### 1.2 Files to Create

```
banhbaoring/
└── types.go
└── types_test.go
```

---

## 2. Specifications

### 2.1 types.go

```go
package banhbaoring

import (
    "crypto/tls"
    "time"
)

// Algorithm constants
const (
    AlgorithmSecp256k1 = "secp256k1"
)

// Default configuration values
const (
    DefaultSecp256k1Path = "secp256k1"
    DefaultHTTPTimeout   = 30 * time.Second
    DefaultStoreVersion  = 1
)

// Source constants for key origin tracking
const (
    SourceGenerated = "generated"
    SourceImported  = "imported"
    SourceSynced    = "synced"
)

// Config holds configuration for BaoKeyring initialization.
type Config struct {
    BaoAddr       string        // OpenBao server address
    BaoToken      string        // OpenBao authentication token
    BaoNamespace  string        // Optional: OpenBao namespace
    Secp256k1Path string        // Plugin mount path (default: "secp256k1")
    StorePath     string        // Path to local metadata store
    HTTPTimeout   time.Duration // HTTP request timeout
    TLSConfig     *tls.Config   // Optional: custom TLS config
    SkipTLSVerify bool          // INSECURE: skip TLS verification
}

// WithDefaults returns Config with default values applied.
func (c Config) WithDefaults() Config {
    if c.Secp256k1Path == "" {
        c.Secp256k1Path = DefaultSecp256k1Path
    }
    if c.HTTPTimeout == 0 {
        c.HTTPTimeout = DefaultHTTPTimeout
    }
    return c
}

// KeyMetadata contains locally stored key information (no private keys).
type KeyMetadata struct {
    UID         string    `json:"uid"`
    Name        string    `json:"name"`
    PubKeyBytes []byte    `json:"pub_key"`
    PubKeyType  string    `json:"pub_key_type"`
    Address     string    `json:"address"`
    BaoKeyPath  string    `json:"bao_key_path"`
    Algorithm   string    `json:"algorithm"`
    Exportable  bool      `json:"exportable"`
    CreatedAt   time.Time `json:"created_at"`
    Source      string    `json:"source"`
}

// KeyInfo represents public key information from OpenBao.
type KeyInfo struct {
    Name       string    `json:"name"`
    PublicKey  string    `json:"public_key"`
    Address    string    `json:"address"`
    Exportable bool      `json:"exportable"`
    CreatedAt  time.Time `json:"created_at"`
}

// KeyOptions configures key creation behavior.
type KeyOptions struct {
    Exportable bool
}

// SignRequest represents a signing request.
type SignRequest struct {
    Input        string `json:"input"`
    Prehashed    bool   `json:"prehashed"`
    HashAlgo     string `json:"hash_algorithm,omitempty"`
    OutputFormat string `json:"output_format,omitempty"`
}

// SignResponse represents a signing response.
type SignResponse struct {
    Signature  string `json:"signature"`
    PublicKey  string `json:"public_key"`
    KeyVersion int    `json:"key_version"`
}

// StoreData represents the persisted metadata store format.
type StoreData struct {
    Version int                     `json:"version"`
    Keys    map[string]*KeyMetadata `json:"keys"`
}
```

---

## 3. Unit Tests

```go
func TestConfig_WithDefaults(t *testing.T) {
    cfg := Config{BaoAddr: "http://localhost:8200"}
    cfg = cfg.WithDefaults()

    assert.Equal(t, DefaultSecp256k1Path, cfg.Secp256k1Path)
    assert.Equal(t, DefaultHTTPTimeout, cfg.HTTPTimeout)
}

func TestConfig_WithDefaults_PreservesExisting(t *testing.T) {
    cfg := Config{
        Secp256k1Path: "custom-path",
        HTTPTimeout:   60 * time.Second,
    }
    cfg = cfg.WithDefaults()

    assert.Equal(t, "custom-path", cfg.Secp256k1Path)
    assert.Equal(t, 60*time.Second, cfg.HTTPTimeout)
}
```

---

## 4. Deliverables

- [ ] `types.go` with all types and constants
- [ ] `types_test.go` with unit tests
- [ ] All tests pass
