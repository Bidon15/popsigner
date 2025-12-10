# Implementation Guide: Plugin Backend

**Agent ID:** 03A  
**Parent:** Agent 03 (OpenBao Plugin)  
**Component:** Backend Factory & Registration  
**Parallelizable:** ✅ Yes - Server-side, independent

---

## 1. Overview

Plugin entrypoint and backend factory that registers paths.

### 1.1 Required Skills

| Skill                | Level    | Description             |
| -------------------- | -------- | ----------------------- |
| **Go**               | Advanced | Plugin architecture     |
| **OpenBao SDK**      | Advanced | Secrets engine patterns |

### 1.2 Files to Create

```
plugin/
├── cmd/plugin/main.go
└── secp256k1/backend.go
```

---

## 2. Specifications

### 2.1 main.go

```go
package main

import (
    "log"
    "os"
    
    "github.com/openbao/openbao/sdk/plugin"
    "github.com/Bidon15/banhbaoring/plugin/secp256k1"
)

func main() {
    if err := plugin.Serve(&plugin.ServeOpts{
        BackendFactoryFunc: secp256k1.Factory,
    }); err != nil {
        log.Printf("plugin error: %v", err)
        os.Exit(1)
    }
}
```

### 2.2 backend.go

```go
package secp256k1

import (
    "context"
    "sync"
    
    "github.com/openbao/openbao/sdk/framework"
    "github.com/openbao/openbao/sdk/logical"
)

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
        PathsSpecial: &logical.Paths{
            SealWrapStorage: []string{"keys/"},
        },
        Invalidate: b.invalidate,
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
    if key == "keys/" {
        b.cacheMu.Lock()
        b.keyCache = make(map[string]*keyEntry)
        b.cacheMu.Unlock()
    }
}

const backendHelp = `
The secp256k1 secrets engine provides native secp256k1 key management
and signing for Cosmos/Celestia. Private keys never leave OpenBao.
`
```

---

## 3. Deliverables

- [ ] `main.go` plugin entrypoint
- [ ] `backend.go` with Factory function
- [ ] Paths registered (keys, sign, verify, import, export)
- [ ] SealWrap configured for key storage
- [ ] Cache invalidation handler

