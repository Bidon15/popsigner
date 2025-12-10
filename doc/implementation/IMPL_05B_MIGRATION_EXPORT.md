# Implementation Guide: Migration Export

**Agent ID:** 05B  
**Parent:** Agent 05 (Migration & CLI)  
**Component:** Key Export from BaoKeyring  
**Parallelizable:** ✅ Yes - Uses 04 interfaces

---

## 1. Overview

Export keys from BaoKeyring to local Cosmos SDK keyrings (for exportable keys only).

### 1.1 Required Skills

| Skill      | Level    | Description           |
| ---------- | -------- | --------------------- |
| **Go**     | Advanced | Crypto operations     |
| **Security** | Advanced | Security warnings   |

### 1.2 Files to Create

```
migration/
└── export.go
```

---

## 2. Specifications

**IMPORTANT:** cosmos-sdk imports are replaced by Celestia's fork via go.mod.

```go
package migration

import (
    "context"
    "errors"
    "fmt"

    "github.com/Bidon15/banhbaoring"
    // Replaced by celestiaorg/cosmos-sdk via go.mod
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
)

type ExportConfig struct {
    SourceKeyring     *banhbaoring.BaoKeyring
    DestKeyring       keyring.Keyring
    KeyName           string
    NewKeyName        string
    DeleteAfterExport bool
    VerifyAfterExport bool
    Confirmed         bool
}

type ExportResult struct {
    KeyName  string
    Address  string
    DestPath string
    Verified bool
}

var ErrExportNotConfirmed = errors.New("export requires user confirmation")

func Export(ctx context.Context, cfg ExportConfig) (*ExportResult, error) {
    if !cfg.Confirmed {
        return nil, ErrExportNotConfirmed
    }
    
    // Check exportable
    meta, err := cfg.SourceKeyring.GetMetadata(cfg.KeyName)
    if err != nil {
        return nil, err
    }
    if !meta.Exportable {
        return nil, banhbaoring.ErrKeyNotExportable
    }
    
    destName := cfg.KeyName
    if cfg.NewKeyName != "" {
        destName = cfg.NewKeyName
    }
    
    // Export from OpenBao
    privKey, err := cfg.SourceKeyring.ExportKey(cfg.KeyName)
    if err != nil {
        return nil, fmt.Errorf("export from OpenBao: %w", err)
    }
    defer secureZero(privKey)
    
    // Import to destination
    _, err = cfg.DestKeyring.ImportPrivKey(destName, string(privKey), "")
    if err != nil {
        return nil, fmt.Errorf("import to local: %w", err)
    }
    
    result := &ExportResult{
        KeyName: destName,
        Address: meta.Address,
    }
    
    if cfg.VerifyAfterExport {
        result.Verified = verifyLocalKey(cfg.DestKeyring, destName)
    }
    
    if cfg.DeleteAfterExport && result.Verified {
        cfg.SourceKeyring.Delete(cfg.KeyName)
    }
    
    return result, nil
}

func ValidateExport(ctx context.Context, cfg ExportConfig) error {
    meta, err := cfg.SourceKeyring.GetMetadata(cfg.KeyName)
    if err != nil {
        return fmt.Errorf("key not found: %w", err)
    }
    if !meta.Exportable {
        return banhbaoring.ErrKeyNotExportable
    }
    return nil
}

func verifyLocalKey(kr keyring.Keyring, name string) bool {
    _, _, err := kr.Sign(name, []byte("verification"), 0)
    return err == nil
}

// SecurityWarning returns the warning text for exports
func SecurityWarning(keyName, address, destPath string) string {
    return fmt.Sprintf(`
╔════════════════════════════════════════════════════════════════════╗
║                     ⚠️  SECURITY WARNING  ⚠️                        ║
╠════════════════════════════════════════════════════════════════════╣
║                                                                    ║
║  You are about to EXPORT a private key from OpenBao.               ║
║                                                                    ║
║  This action will:                                                 ║
║  • Copy the private key to local storage                          ║
║  • Reduce security (key no longer protected by OpenBao)           ║
║  • Create a potential attack vector                               ║
║                                                                    ║
║  Key: %s
║  Address: %s
║  Destination: %s
║                                                                    ║
╚════════════════════════════════════════════════════════════════════╝
`, keyName, address, destPath)
}
```

---

## 3. Deliverables

- [ ] Export checks `Confirmed` flag
- [ ] Export validates key is exportable
- [ ] Verification after export
- [ ] Delete from OpenBao option
- [ ] Security warning function
- [ ] Memory wiped (secureZero)

