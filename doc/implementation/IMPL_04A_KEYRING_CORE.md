# Implementation Guide: BaoKeyring Core

**Agent ID:** 04A  
**Parent:** Agent 04 (BaoKeyring)  
**Component:** Keyring Interface Core  
**Parallelizable:** ✅ Yes - Uses interfaces from 01, 02

---

## 1. Overview

Core struct and constructor implementing `keyring.Keyring` interface.

### 1.1 Required Skills

| Skill          | Level    | Description              |
| -------------- | -------- | ------------------------ |
| **Go**         | Advanced | Interfaces, composition  |
| **Cosmos SDK** | Advanced | keyring.Keyring contract |

### 1.2 Files to Create

```
banhbaoring/
└── bao_keyring.go (core struct, constructor, Backend)
```

---

## 2. Specifications

**IMPORTANT:** These imports use standard cosmos-sdk paths but are REPLACED 
by Celestia's forks via go.mod replace directives.

```go
package banhbaoring

import (
    "context"
    "fmt"

    // Replaced by celestiaorg/cosmos-sdk via go.mod
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
)

const BackendType = "openbao"

// BaoKeyring implements keyring.Keyring using OpenBao.
type BaoKeyring struct {
    client *BaoClient
    store  *BaoStore
}

var _ keyring.Keyring = (*BaoKeyring)(nil)

// New creates a BaoKeyring instance.
func New(ctx context.Context, cfg Config) (*BaoKeyring, error) {
    if err := cfg.Validate(); err != nil {
        return nil, err
    }
    
    client, err := NewBaoClient(cfg)
    if err != nil {
        return nil, fmt.Errorf("create client: %w", err)
    }
    
    if err := client.Health(ctx); err != nil {
        return nil, fmt.Errorf("health check: %w", err)
    }
    
    store, err := NewBaoStore(cfg.StorePath)
    if err != nil {
        return nil, fmt.Errorf("create store: %w", err)
    }
    
    return &BaoKeyring{client: client, store: store}, nil
}

// Backend returns the keyring backend type.
func (k *BaoKeyring) Backend() string {
    return BackendType
}

// Close releases resources.
func (k *BaoKeyring) Close() error {
    return k.store.Close()
}

// Config validation
func (c *Config) Validate() error {
    if c.BaoAddr == "" {
        return ErrMissingBaoAddr
    }
    if c.BaoToken == "" {
        return ErrMissingBaoToken
    }
    if c.StorePath == "" {
        return ErrMissingStorePath
    }
    return nil
}
```

---

## 3. Deliverables

- [ ] `BaoKeyring` struct defined
- [ ] `New()` constructor with validation
- [ ] Health check on startup
- [ ] `Backend()` returns "openbao"
- [ ] `Close()` releases resources

