# Implementation Guide: Migration Import

**Agent ID:** 05A  
**Parent:** Agent 05 (Migration & CLI)  
**Component:** Key Import from Local Keyring  
**Parallelizable:** ✅ Yes - Uses 04 interfaces

---

## 1. Overview

Import keys from local Cosmos SDK keyrings to BaoKeyring.

### 1.1 Required Skills

| Skill      | Level    | Description              |
| ---------- | -------- | ------------------------ |
| **Go**     | Advanced | Crypto, RSA              |
| **Crypto** | Advanced | Key wrapping, RSA-OAEP   |

### 1.2 Files to Create

```
migration/
├── types.go
└── import.go
```

---

## 2. Specifications

### 2.1 types.go

**IMPORTANT:** cosmos-sdk imports are replaced by Celestia's fork via go.mod.

```go
package migration

import (
    "github.com/Bidon15/banhbaoring"
    // Replaced by celestiaorg/cosmos-sdk via go.mod
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
)

type ImportConfig struct {
    SourceKeyring     keyring.Keyring
    DestKeyring       *banhbaoring.BaoKeyring
    KeyName           string
    NewKeyName        string
    DeleteAfterImport bool
    Exportable        bool
    VerifyAfterImport bool
}

type ImportResult struct {
    KeyName    string
    Address    string
    PubKey     []byte
    BaoKeyPath string
    Verified   bool
}

type ImportError struct {
    KeyName string
    Error   error
}

type BatchImportConfig struct {
    SourceKeyring     keyring.Keyring
    DestKeyring       *banhbaoring.BaoKeyring
    KeyNames          []string
    DeleteAfterImport bool
    Exportable        bool
    VerifyAfterImport bool
}

type BatchImportResult struct {
    Successful []ImportResult
    Failed     []ImportError
}
```

### 2.2 import.go

```go
package migration

import (
    "context"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/pem"
    "errors"
    "fmt"
    "runtime"
)

func Import(ctx context.Context, cfg ImportConfig) (*ImportResult, error) {
    if cfg.SourceKeyring == nil || cfg.DestKeyring == nil {
        return nil, errors.New("keyrings required")
    }
    if cfg.KeyName == "" {
        return nil, errors.New("key name required")
    }
    
    destName := cfg.KeyName
    if cfg.NewKeyName != "" {
        destName = cfg.NewKeyName
    }
    
    // Export from source
    privKey, err := exportPrivateKey(cfg.SourceKeyring, cfg.KeyName)
    if err != nil {
        return nil, fmt.Errorf("export source: %w", err)
    }
    defer secureZero(privKey)
    
    // Get wrapping key
    wrappingKeyPEM, err := cfg.DestKeyring.GetWrappingKey()
    if err != nil {
        return nil, fmt.Errorf("get wrapping key: %w", err)
    }
    
    wrappingKey, err := parseRSAPublicKey(wrappingKeyPEM)
    if err != nil {
        return nil, err
    }
    
    // Wrap key
    wrapped, err := wrapKey(privKey, wrappingKey)
    if err != nil {
        return nil, err
    }
    
    // Import to destination
    record, err := cfg.DestKeyring.ImportKey(destName, wrapped, cfg.Exportable)
    if err != nil {
        return nil, err
    }
    
    result := &ImportResult{KeyName: destName}
    addr, _ := record.GetAddress()
    result.Address = addr.String()
    
    if cfg.VerifyAfterImport {
        result.Verified = verifyKey(ctx, cfg.DestKeyring, destName)
    }
    
    if cfg.DeleteAfterImport && result.Verified {
        cfg.SourceKeyring.Delete(cfg.KeyName)
    }
    
    return result, nil
}

func BatchImport(ctx context.Context, cfg BatchImportConfig) (*BatchImportResult, error) {
    result := &BatchImportResult{}
    
    keyNames := cfg.KeyNames
    if len(keyNames) == 0 {
        records, _ := cfg.SourceKeyring.List()
        for _, r := range records {
            keyNames = append(keyNames, r.Name)
        }
    }
    
    for _, name := range keyNames {
        importCfg := ImportConfig{
            SourceKeyring:     cfg.SourceKeyring,
            DestKeyring:       cfg.DestKeyring,
            KeyName:           name,
            DeleteAfterImport: cfg.DeleteAfterImport,
            Exportable:        cfg.Exportable,
            VerifyAfterImport: cfg.VerifyAfterImport,
        }
        
        res, err := Import(ctx, importCfg)
        if err != nil {
            result.Failed = append(result.Failed, ImportError{KeyName: name, Error: err})
            continue
        }
        result.Successful = append(result.Successful, *res)
    }
    
    return result, nil
}

// Helpers
func exportPrivateKey(kr keyring.Keyring, name string) ([]byte, error) {
    armor, err := kr.ExportPrivKeyArmor(name, "")
    return []byte(armor), err
}

func wrapKey(privKey []byte, pubKey *rsa.PublicKey) ([]byte, error) {
    return rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, privKey, nil)
}

func parseRSAPublicKey(pemData []byte) (*rsa.PublicKey, error) {
    block, _ := pem.Decode(pemData)
    if block == nil {
        return nil, errors.New("invalid PEM")
    }
    pub, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
        return nil, err
    }
    rsaPub, ok := pub.(*rsa.PublicKey)
    if !ok {
        return nil, errors.New("not RSA key")
    }
    return rsaPub, nil
}

func secureZero(b []byte) {
    for i := range b {
        b[i] = 0
    }
    runtime.KeepAlive(b)
}

func verifyKey(ctx context.Context, kr *banhbaoring.BaoKeyring, name string) bool {
    _, _, err := kr.Sign(name, []byte("verification"), 0)
    return err == nil
}
```

---

## 3. Deliverables

- [ ] Import single key from local keyring
- [ ] Batch import multiple keys
- [ ] Key wrapping with RSA-OAEP
- [ ] Verification after import
- [ ] Delete from source option
- [ ] Memory wiped (secureZero)

