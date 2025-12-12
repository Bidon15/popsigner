# Key Migration Guide

This document provides detailed instructions for migrating keys between local Cosmos SDK keyrings and the BaoKeyring (OpenBao Transit).

---

## 1. Overview

### 1.1 Migration Scenarios

| Scenario | From | To | Use Case |
| -------- | ---- | -- | -------- |
| **Import** | Local keyring | BaoKeyring | Upgrade security, centralize key management |
| **Export** | BaoKeyring | Local keyring | Offboard from BanhBao, disaster recovery |

### 1.2 Security Considerations

**Import (Local → OpenBao):**
- Private key is temporarily in memory during transfer
- Key is encrypted in transit using OpenBao wrapping key
- After successful import, delete local key copy

**Export (OpenBao → Local):**
- Only works for keys created with `exportable: true`
- Compromises security model (key leaves OpenBao vault)
- Should be rare, deliberate operation
- Requires explicit user confirmation

---

## 2. Import: Local Keyring to BaoKeyring

### 2.1 Supported Source Backends

| Backend | Description | Passphrase Required |
| ------- | ----------- | ------------------- |
| `file` | Encrypted keystore in `~/.celestia-app/` | Yes |
| `test` | Plaintext keystore (development only) | No |
| `os` | System keychain (macOS Keychain, etc.) | System auth |

### 2.2 Import Flow Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Import Flow                                   │
└──────────────────────────────────────────────────────────────────────┘

┌─────────────────┐     ┌─────────────────┐     ┌─────────────────────┐
│  Local Keyring  │     │   BaoKeyring    │     │     OpenBao         │
│  (file/test/os) │     │   Migration     │     │   Transit Engine    │
└────────┬────────┘     └────────┬────────┘     └──────────┬──────────┘
         │                       │                         │
         │  1. Export key        │                         │
         │◀──────────────────────│                         │
         │                       │                         │
         │  Private key bytes    │                         │
         │──────────────────────▶│                         │
         │                       │                         │
         │                       │  2. Get wrapping key    │
         │                       │────────────────────────▶│
         │                       │                         │
         │                       │  Wrapping public key    │
         │                       │◀────────────────────────│
         │                       │                         │
         │                       │  3. Wrap private key    │
         │                       │  (RSA-OAEP encrypt)     │
         │                       │─────────┐               │
         │                       │◀────────┘               │
         │                       │                         │
         │                       │  4. Import wrapped key  │
         │                       │────────────────────────▶│
         │                       │                         │
         │                       │  Success                │
         │                       │◀────────────────────────│
         │                       │                         │
         │                       │  5. Get public key      │
         │                       │────────────────────────▶│
         │                       │                         │
         │                       │  Public key             │
         │                       │◀────────────────────────│
         │                       │                         │
         │                       │  6. Store metadata      │
         │                       │─────────┐               │
         │                       │◀────────┘               │
         │                       │                         │
         │  7. Delete local key  │                         │
         │  (with confirmation)  │                         │
         │◀──────────────────────│                         │
         │                       │                         │
```

### 2.3 CLI Usage

```bash
# Import single key
banhbao migrate import \
  --from ~/.celestia-app/keyring-file \
  --backend file \
  --key-name my-validator \
  --delete-after-import

# Import with passphrase from stdin
echo "my-passphrase" | banhbao migrate import \
  --from ~/.celestia-app/keyring-file \
  --backend file \
  --key-name my-validator \
  --passphrase-stdin

# Import all keys
banhbao migrate import \
  --from ~/.celestia-app/keyring-file \
  --backend file \
  --all

# Dry run (validate without importing)
banhbao migrate import \
  --from ~/.celestia-app/keyring-file \
  --backend file \
  --key-name my-validator \
  --dry-run
```

### 2.4 Go API Usage

```go
import (
    "context"
    "github.com/Bidon15/banhbaoring"
    "github.com/Bidon15/banhbaoring/migration"
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
)

func importFromLocalKeyring(ctx context.Context) error {
    // Open source keyring (local file backend)
    sourceKr, err := keyring.New(
        "celestia",
        keyring.BackendFile,
        "/home/user/.celestia-app",
        os.Stdin, // For passphrase input
        nil,
    )
    if err != nil {
        return fmt.Errorf("failed to open source keyring: %w", err)
    }
    
    // Create BaoKeyring as destination
    baoCfg := banhbaoring.Config{
        BaoAddr:     os.Getenv("BAO_ADDR"),
        BaoToken:    os.Getenv("BAO_TOKEN"),
        TransitPath: "transit",
        StorePath:   "./keyring-metadata.json",
    }
    
    destKr, err := banhbaoring.New(ctx, baoCfg)
    if err != nil {
        return fmt.Errorf("failed to create BaoKeyring: %w", err)
    }
    
    // Configure migration
    migrateCfg := migration.ImportConfig{
        SourceKeyring:     sourceKr,
        DestKeyring:       destKr,
        KeyName:           "my-validator",
        DeleteAfterImport: true,         // Delete from source after success
        Exportable:        false,        // Key cannot be exported from OpenBao
        VerifyAfterImport: true,         // Sign test data to verify
    }
    
    // Execute migration
    result, err := migration.Import(ctx, migrateCfg)
    if err != nil {
        return fmt.Errorf("migration failed: %w", err)
    }
    
    fmt.Printf("Successfully imported key: %s\n", result.KeyName)
    fmt.Printf("New address: %s\n", result.Address)
    fmt.Printf("OpenBao path: %s\n", result.BaoKeyPath)
    
    return nil
}
```

### 2.5 Batch Import

```go
func batchImport(ctx context.Context, sourceKr keyring.Keyring, destKr *banhbaoring.BaoKeyring) error {
    // List all keys in source keyring
    keys, err := sourceKr.List()
    if err != nil {
        return err
    }
    
    results := make([]migration.ImportResult, 0, len(keys))
    errors := make([]error, 0)
    
    for _, key := range keys {
        cfg := migration.ImportConfig{
            SourceKeyring:     sourceKr,
            DestKeyring:       destKr,
            KeyName:           key.Name,
            DeleteAfterImport: false, // Don't delete until all succeed
            Exportable:        false,
            VerifyAfterImport: true,
        }
        
        result, err := migration.Import(ctx, cfg)
        if err != nil {
            errors = append(errors, fmt.Errorf("failed to import %s: %w", key.Name, err))
            continue
        }
        
        results = append(results, result)
    }
    
    // Report results
    fmt.Printf("Imported %d/%d keys successfully\n", len(results), len(keys))
    
    if len(errors) > 0 {
        return fmt.Errorf("some imports failed: %v", errors)
    }
    
    return nil
}
```

---

## 3. Export: BaoKeyring to Local Keyring

### 3.1 Prerequisites

**Key must be exportable:**
- Created with `exportable: true` flag
- User has `read` permission on `transit/export/encryption-key/<name>`

**Check if key is exportable:**

```bash
banhbao keys show my-key --check-exportable
```

### 3.2 Export Flow Diagram

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Export Flow                                   │
└──────────────────────────────────────────────────────────────────────┘

┌─────────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     OpenBao         │     │   BaoKeyring    │     │  Local Keyring  │
│   Transit Engine    │     │   Migration     │     │  (destination)  │
└──────────┬──────────┘     └────────┬────────┘     └────────┬────────┘
           │                         │                       │
           │  1. Export key          │                       │
           │◀────────────────────────│                       │
           │                         │                       │
           │  Wrapped key material   │                       │
           │────────────────────────▶│                       │
           │                         │                       │
           │                         │  2. Decrypt key       │
           │                         │  (with wrapping key)  │
           │                         │─────────┐             │
           │                         │◀────────┘             │
           │                         │                       │
           │                         │  3. Create local key  │
           │                         │──────────────────────▶│
           │                         │                       │
           │                         │  Success              │
           │                         │◀──────────────────────│
           │                         │                       │
           │                         │  4. Verify signature  │
           │                         │──────────────────────▶│
           │                         │                       │
           │                         │  Signature matches    │
           │                         │◀──────────────────────│
           │                         │                       │
           │  5. Delete from OpenBao │                       │
           │  (optional)             │                       │
           │◀────────────────────────│                       │
           │                         │                       │
```

### 3.3 CLI Usage

```bash
# Export single key (requires confirmation)
banhbao migrate export \
  --to ~/.celestia-app/keyring-file \
  --backend file \
  --key-name my-validator

# Export with explicit confirmation
banhbao migrate export \
  --to ~/.celestia-app/keyring-file \
  --backend file \
  --key-name my-validator \
  --confirm "I understand this compromises key security"

# Export and delete from OpenBao
banhbao migrate export \
  --to ~/.celestia-app/keyring-file \
  --backend file \
  --key-name my-validator \
  --delete-after-export

# Export to encrypted file
banhbao migrate export \
  --to ./exported-key.json \
  --format encrypted-json \
  --key-name my-validator
```

### 3.4 Go API Usage

```go
import (
    "context"
    "github.com/Bidon15/banhbaoring"
    "github.com/Bidon15/banhbaoring/migration"
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
)

func exportToLocalKeyring(ctx context.Context) error {
    // Create BaoKeyring as source
    baoCfg := banhbaoring.Config{
        BaoAddr:     os.Getenv("BAO_ADDR"),
        BaoToken:    os.Getenv("BAO_TOKEN"),
        TransitPath: "transit",
        StorePath:   "./keyring-metadata.json",
    }
    
    sourceKr, err := banhbaoring.New(ctx, baoCfg)
    if err != nil {
        return fmt.Errorf("failed to create BaoKeyring: %w", err)
    }
    
    // Check if key is exportable
    keyInfo, err := sourceKr.Key("my-validator")
    if err != nil {
        return err
    }
    
    meta, _ := sourceKr.GetMetadata("my-validator")
    if !meta.Exportable {
        return fmt.Errorf("key %q is not exportable", "my-validator")
    }
    
    // Open destination keyring
    destKr, err := keyring.New(
        "celestia",
        keyring.BackendFile,
        "/home/user/.celestia-app-backup",
        os.Stdin,
        nil,
    )
    if err != nil {
        return fmt.Errorf("failed to open destination keyring: %w", err)
    }
    
    // Configure export
    exportCfg := migration.ExportConfig{
        SourceKeyring:     sourceKr,
        DestKeyring:       destKr,
        KeyName:           "my-validator",
        DeleteAfterExport: false,        // Keep in OpenBao
        VerifyAfterExport: true,         // Sign test data to verify
        Confirm:           true,         // User has confirmed
    }
    
    // Execute export
    result, err := migration.Export(ctx, exportCfg)
    if err != nil {
        return fmt.Errorf("export failed: %w", err)
    }
    
    fmt.Printf("Successfully exported key: %s\n", result.KeyName)
    fmt.Printf("Exported to: %s\n", result.DestPath)
    
    return nil
}
```

### 3.5 Security Warning

When exporting keys, the CLI and API should display prominent warnings:

```
╔════════════════════════════════════════════════════════════════════╗
║                     ⚠️  SECURITY WARNING  ⚠️                        ║
╠════════════════════════════════════════════════════════════════════╣
║                                                                    ║
║  You are about to EXPORT a private key from OpenBao.               ║
║                                                                    ║
║  This action will:                                                 ║
║  • Copy the private key to local storage                          ║
║  • Reduce security (key no longer protected by OpenBao)           ║
║  • Create a potential attack vector on the local machine          ║
║                                                                    ║
║  Key: my-validator                                                 ║
║  Address: celestia1abc123...                                       ║
║  Destination: /home/user/.celestia-app-backup/keyring-file        ║
║                                                                    ║
║  Type "I understand the security implications" to continue:        ║
╚════════════════════════════════════════════════════════════════════╝
```

---

## 4. Creating Exportable Keys

If you anticipate needing to export keys in the future, create them with the exportable flag:

### 4.1 CLI

```bash
# Create exportable key
banhbao keys create my-key --exportable

# Note: Exportable keys are less secure
# Only use when export capability is required
```

### 4.2 Go API

```go
// Create exportable key
record, err := kr.NewAccountWithOptions("my-key", banhbaoring.KeyOptions{
    Algorithm:  keyring.Secp256k1,
    Exportable: true,
})
```

### 4.3 Default Behavior

| Scenario | Exportable Default | Rationale |
| -------- | ------------------ | --------- |
| New key creation | `false` | Maximum security |
| Import from local | `false` | User upgrading security |
| Web UI creation | `false` | Protect users from themselves |

---

## 5. Migration Verification

### 5.1 Pre-Migration Checklist

- [ ] Backup existing keyring
- [ ] Verify OpenBao connectivity
- [ ] Confirm key names don't conflict
- [ ] Test with non-production key first
- [ ] Document current key addresses

### 5.2 Post-Migration Verification

```bash
# Verify key was imported correctly
banhbao keys show my-validator

# Compare addresses
echo "Old address: celestia1abc..."
banhbao keys show my-validator --format json | jq -r '.address'

# Test signing
echo "test message" | banhbao sign --key my-validator

# Verify signature with original address
# (addresses should match)
```

### 5.3 Rollback Procedure

If migration fails:

1. **Do NOT delete source keys** until verification passes
2. Check OpenBao logs for import errors
3. Verify key material is correct format
4. Contact support with error details

---

## 6. Troubleshooting

### 6.1 Common Import Errors

| Error | Cause | Solution |
| ----- | ----- | -------- |
| "key already exists" | Duplicate key name in OpenBao | Use different name or delete existing |
| "invalid key format" | Unsupported key type | Ensure secp256k1 key |
| "permission denied" | Insufficient OpenBao permissions | Check token policies |
| "wrapping key unavailable" | Transit engine misconfigured | Enable Transit wrapping |

### 6.2 Common Export Errors

| Error | Cause | Solution |
| ----- | ----- | -------- |
| "key not exportable" | Key created without export flag | Cannot export; create new exportable key |
| "export permission denied" | Missing export policy | Add `transit/export/*` permission |
| "destination exists" | Key name exists in target | Delete or use different name |

### 6.3 Debug Logging

```bash
# Enable debug logging for migration
export BANHBAO_LOG_LEVEL=debug

banhbao migrate import --from ... --key-name my-key
```

---

## 7. Best Practices

### 7.1 Import Best Practices

1. **Test first**: Import a test key before production keys
2. **Verify addresses**: Confirm addresses match before/after
3. **Backup source**: Keep source keyring backup until verified
4. **Use dry-run**: Preview import without executing
5. **Batch carefully**: Import keys one at a time for critical keys

### 7.2 Export Best Practices

1. **Avoid if possible**: Exporting reduces security
2. **Document reason**: Log why export was needed
3. **Encrypt destination**: Use encrypted keyring backend
4. **Rotate after export**: Consider rotating key after export
5. **Delete from OpenBao**: Remove from OpenBao after confirmed export

### 7.3 Security Best Practices

1. **Non-exportable by default**: Only enable export when truly needed
2. **Audit trail**: Log all migration events
3. **Separate environments**: Don't mix prod/test key namespaces
4. **Access control**: Limit who can import/export keys
5. **Key rotation**: Rotate keys periodically regardless of backend

---

## 8. API Reference

### 8.1 Migration Types

```go
// ImportConfig configures key import from local to OpenBao
type ImportConfig struct {
    SourceKeyring     keyring.Keyring
    DestKeyring       *BaoKeyring
    KeyName           string
    NewKeyName        string   // Optional: rename during import
    DeleteAfterImport bool
    Exportable        bool
    VerifyAfterImport bool
}

// ExportConfig configures key export from OpenBao to local
type ExportConfig struct {
    SourceKeyring     *BaoKeyring
    DestKeyring       keyring.Keyring
    KeyName           string
    NewKeyName        string   // Optional: rename during export
    DeleteAfterExport bool
    VerifyAfterExport bool
    Confirm           bool     // User has acknowledged security implications
}

// ImportResult contains the result of an import operation
type ImportResult struct {
    KeyName    string
    Address    string
    PubKey     []byte
    BaoKeyPath string
    Verified   bool
}

// ExportResult contains the result of an export operation
type ExportResult struct {
    KeyName   string
    Address   string
    DestPath  string
    Verified  bool
}
```

### 8.2 Migration Functions

```go
// Import migrates a key from local keyring to BaoKeyring
func Import(ctx context.Context, cfg ImportConfig) (*ImportResult, error)

// Export migrates a key from BaoKeyring to local keyring
func Export(ctx context.Context, cfg ExportConfig) (*ExportResult, error)

// BatchImport imports multiple keys
func BatchImport(ctx context.Context, cfg BatchImportConfig) ([]ImportResult, []error)

// ValidateImport checks if import would succeed without executing
func ValidateImport(ctx context.Context, cfg ImportConfig) error

// ValidateExport checks if export would succeed without executing
func ValidateExport(ctx context.Context, cfg ExportConfig) error
```

---

## 9. References

- [OpenBao Transit Key Wrapping](https://openbao.org/docs/secrets/transit/#key-wrapping)
- [OpenBao Transit Export](https://openbao.org/api-docs/secret/transit/#export-key)
- [Cosmos SDK Keyring](https://docs.cosmos.network/main/user/run-node/keyring)

