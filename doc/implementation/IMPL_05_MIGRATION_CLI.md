# Implementation Guide: Migration & CLI

**Agent ID:** 05  
**Component:** Key Migration Utilities & CLI Tool  
**Parallelizable:** ✅ Yes - Uses interfaces from Agent 04  

---

## 1. Overview

This agent builds key migration utilities (import/export between keyrings) and the CLI tool for user interaction.

### 1.1 Required Skills

| Skill | Level | Description |
|-------|-------|-------------|
| **Go** | Advanced | Interfaces, io, crypto |
| **CLI Development** | Advanced | Cobra framework |
| **Cosmos SDK** | Intermediate | Keyring backends |
| **Security** | Advanced | Secure key handling, memory wiping |

### 1.2 Files to Create

```
banhbaoring/
├── migration/
│   ├── migration.go      # Import/export logic
│   ├── types.go          # Migration types
│   └── migration_test.go # Tests
└── cmd/
    └── banhbao/
        ├── main.go       # CLI entrypoint
        ├── keys.go       # Key commands
        ├── migrate.go    # Migration commands
        └── sign.go       # Sign commands
```

---

## 2. Migration Package

### 2.1 migration/types.go

**IMPORTANT:** All cosmos-sdk imports are replaced by Celestia's fork via go.mod.

```go
package migration

import (
    "github.com/Bidon15/banhbaoring"
    // Replaced by celestiaorg/cosmos-sdk via go.mod
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
)

// ImportConfig configures key import from local to OpenBao
type ImportConfig struct {
    SourceKeyring     keyring.Keyring
    DestKeyring       *banhbaoring.BaoKeyring
    KeyName           string
    NewKeyName        string   // Optional rename
    DeleteAfterImport bool
    Exportable        bool
    VerifyAfterImport bool
}

// ExportConfig configures key export from OpenBao to local
type ExportConfig struct {
    SourceKeyring     *banhbaoring.BaoKeyring
    DestKeyring       keyring.Keyring
    KeyName           string
    NewKeyName        string
    DeleteAfterExport bool
    VerifyAfterExport bool
    Confirmed         bool // User confirmed security implications
}

// ImportResult contains import operation result
type ImportResult struct {
    KeyName    string
    Address    string
    PubKey     []byte
    BaoKeyPath string
    Verified   bool
}

// ExportResult contains export operation result
type ExportResult struct {
    KeyName  string
    Address  string
    DestPath string
    Verified bool
}

// BatchImportConfig for importing multiple keys
type BatchImportConfig struct {
    SourceKeyring     keyring.Keyring
    DestKeyring       *banhbaoring.BaoKeyring
    KeyNames          []string // Empty = all keys
    DeleteAfterImport bool
    Exportable        bool
    VerifyAfterImport bool
}

// BatchImportResult contains batch import results
type BatchImportResult struct {
    Successful []ImportResult
    Failed     []ImportError
}

// ImportError represents a failed import
type ImportError struct {
    KeyName string
    Error   error
}
```

### 2.2 migration/migration.go

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

    "github.com/Bidon15/banhbaoring"
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
)

// Import migrates a key from local keyring to BaoKeyring
func Import(ctx context.Context, cfg ImportConfig) (*ImportResult, error) {
    if cfg.SourceKeyring == nil {
        return nil, errors.New("source keyring is required")
    }
    if cfg.DestKeyring == nil {
        return nil, errors.New("destination keyring is required")
    }
    if cfg.KeyName == "" {
        return nil, errors.New("key name is required")
    }
    
    destName := cfg.KeyName
    if cfg.NewKeyName != "" {
        destName = cfg.NewKeyName
    }
    
    // Export private key from source
    privKey, err := exportPrivateKey(cfg.SourceKeyring, cfg.KeyName)
    if err != nil {
        return nil, fmt.Errorf("failed to export from source: %w", err)
    }
    defer secureZero(privKey)
    
    // Get wrapping key from OpenBao
    wrappingKeyPEM, err := cfg.DestKeyring.GetWrappingKey()
    if err != nil {
        return nil, fmt.Errorf("failed to get wrapping key: %w", err)
    }
    
    wrappingKey, err := parseRSAPublicKey(wrappingKeyPEM)
    if err != nil {
        return nil, fmt.Errorf("failed to parse wrapping key: %w", err)
    }
    
    // Wrap private key
    wrappedKey, err := wrapKey(privKey, wrappingKey)
    if err != nil {
        return nil, fmt.Errorf("failed to wrap key: %w", err)
    }
    
    // Import to OpenBao
    record, err := cfg.DestKeyring.ImportKey(destName, wrappedKey, cfg.Exportable)
    if err != nil {
        return nil, fmt.Errorf("failed to import to OpenBao: %w", err)
    }
    
    result := &ImportResult{
        KeyName: destName,
    }
    
    addr, _ := record.GetAddress()
    result.Address = addr.String()
    
    pubKey, _ := record.GetPubKey()
    result.PubKey = pubKey.Bytes()
    
    // Verify by signing test data
    if cfg.VerifyAfterImport {
        result.Verified = verifyImport(ctx, cfg.DestKeyring, destName)
    }
    
    // Delete from source if requested
    if cfg.DeleteAfterImport && result.Verified {
        _ = cfg.SourceKeyring.Delete(cfg.KeyName)
    }
    
    return result, nil
}

// Export migrates a key from BaoKeyring to local keyring
func Export(ctx context.Context, cfg ExportConfig) (*ExportResult, error) {
    if !cfg.Confirmed {
        return nil, errors.New("export requires user confirmation")
    }
    
    // Check if exportable
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
        return nil, fmt.Errorf("failed to export from OpenBao: %w", err)
    }
    defer secureZero(privKey)
    
    // Import to destination keyring
    // Note: Exact method depends on destination backend
    _, err = importPrivateKey(cfg.DestKeyring, destName, privKey)
    if err != nil {
        return nil, fmt.Errorf("failed to import to destination: %w", err)
    }
    
    result := &ExportResult{
        KeyName: destName,
        Address: meta.Address,
    }
    
    // Verify
    if cfg.VerifyAfterExport {
        result.Verified = verifyExport(cfg.DestKeyring, destName)
    }
    
    // Delete from OpenBao if requested
    if cfg.DeleteAfterExport && result.Verified {
        _ = cfg.SourceKeyring.Delete(cfg.KeyName)
    }
    
    return result, nil
}

// BatchImport imports multiple keys
func BatchImport(ctx context.Context, cfg BatchImportConfig) (*BatchImportResult, error) {
    result := &BatchImportResult{}
    
    keyNames := cfg.KeyNames
    if len(keyNames) == 0 {
        // Import all keys
        records, err := cfg.SourceKeyring.List()
        if err != nil {
            return nil, err
        }
        for _, r := range records {
            keyNames = append(keyNames, r.Name)
        }
    }
    
    for _, keyName := range keyNames {
        importCfg := ImportConfig{
            SourceKeyring:     cfg.SourceKeyring,
            DestKeyring:       cfg.DestKeyring,
            KeyName:           keyName,
            DeleteAfterImport: cfg.DeleteAfterImport,
            Exportable:        cfg.Exportable,
            VerifyAfterImport: cfg.VerifyAfterImport,
        }
        
        importResult, err := Import(ctx, importCfg)
        if err != nil {
            result.Failed = append(result.Failed, ImportError{
                KeyName: keyName,
                Error:   err,
            })
            continue
        }
        
        result.Successful = append(result.Successful, *importResult)
    }
    
    return result, nil
}

// ValidateImport checks if import would succeed without executing
func ValidateImport(ctx context.Context, cfg ImportConfig) error {
    // Check source key exists
    _, err := cfg.SourceKeyring.Key(cfg.KeyName)
    if err != nil {
        return fmt.Errorf("source key not found: %w", err)
    }
    
    // Check destination doesn't exist
    destName := cfg.KeyName
    if cfg.NewKeyName != "" {
        destName = cfg.NewKeyName
    }
    
    _, err = cfg.DestKeyring.Key(destName)
    if err == nil {
        return fmt.Errorf("key %q already exists in destination", destName)
    }
    
    return nil
}

// Helper functions

func exportPrivateKey(kr keyring.Keyring, keyName string) ([]byte, error) {
    // This requires accessing the underlying key material
    // Implementation depends on keyring backend
    armor, err := kr.ExportPrivKeyArmor(keyName, "")
    if err != nil {
        return nil, err
    }
    
    return []byte(armor), nil
}

func importPrivateKey(kr keyring.Keyring, keyName string, privKey []byte) (*keyring.Record, error) {
    return kr.ImportPrivKey(keyName, string(privKey), "")
}

func wrapKey(privKey []byte, wrappingKey *rsa.PublicKey) ([]byte, error) {
    return rsa.EncryptOAEP(sha256.New(), rand.Reader, wrappingKey, privKey, nil)
}

func parseRSAPublicKey(pemData []byte) (*rsa.PublicKey, error) {
    block, _ := pem.Decode(pemData)
    if block == nil {
        return nil, errors.New("failed to decode PEM block")
    }
    
    pub, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
        return nil, err
    }
    
    rsaPub, ok := pub.(*rsa.PublicKey)
    if !ok {
        return nil, errors.New("not an RSA public key")
    }
    
    return rsaPub, nil
}

func secureZero(b []byte) {
    for i := range b {
        b[i] = 0
    }
    runtime.KeepAlive(b)
}

func verifyImport(ctx context.Context, kr *banhbaoring.BaoKeyring, keyName string) bool {
    testMsg := []byte("verification test message")
    _, _, err := kr.Sign(keyName, testMsg, 0)
    return err == nil
}

func verifyExport(kr keyring.Keyring, keyName string) bool {
    testMsg := []byte("verification test message")
    _, _, err := kr.Sign(keyName, testMsg, 0)
    return err == nil
}
```

---

## 3. CLI Tool

### 3.1 cmd/banhbao/main.go

```go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "banhbao",
    Short: "BanhBao - OpenBao keyring management for Celestia",
    Long: `BanhBao provides secure key management using OpenBao Transit engine.
    
Keys are stored in OpenBao and never leave the secure boundary.
Only signatures are returned to the client.`,
}

var (
    baoAddr  string
    baoToken string
    storePath string
)

func init() {
    rootCmd.PersistentFlags().StringVar(&baoAddr, "bao-addr", "", "OpenBao address (or BAO_ADDR env)")
    rootCmd.PersistentFlags().StringVar(&baoToken, "bao-token", "", "OpenBao token (or BAO_TOKEN env)")
    rootCmd.PersistentFlags().StringVar(&storePath, "store-path", "./keyring-metadata.json", "Local metadata store path")
    
    rootCmd.AddCommand(keysCmd)
    rootCmd.AddCommand(migrateCmd)
    rootCmd.AddCommand(signCmd)
    rootCmd.AddCommand(versionCmd)
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("banhbao v0.1.0")
    },
}
```

### 3.2 cmd/banhbao/keys.go

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/Bidon15/banhbaoring"
    "github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
    Use:   "keys",
    Short: "Manage keys",
}

var keysCreateCmd = &cobra.Command{
    Use:   "create <name>",
    Short: "Create a new key",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        keyName := args[0]
        exportable, _ := cmd.Flags().GetBool("exportable")
        
        kr, err := getKeyring()
        if err != nil {
            return err
        }
        defer kr.Close()
        
        record, err := kr.NewAccountWithOptions(keyName, banhbaoring.KeyOptions{
            Exportable: exportable,
        })
        if err != nil {
            return err
        }
        
        addr, _ := record.GetAddress()
        fmt.Printf("Created key: %s\n", keyName)
        fmt.Printf("Address: %s\n", addr.String())
        return nil
    },
}

var keysListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all keys",
    RunE: func(cmd *cobra.Command, args []string) error {
        kr, err := getKeyring()
        if err != nil {
            return err
        }
        defer kr.Close()
        
        records, err := kr.List()
        if err != nil {
            return err
        }
        
        if len(records) == 0 {
            fmt.Println("No keys found")
            return nil
        }
        
        fmt.Printf("%-20s %-50s\n", "NAME", "ADDRESS")
        for _, r := range records {
            addr, _ := r.GetAddress()
            fmt.Printf("%-20s %-50s\n", r.Name, addr.String())
        }
        return nil
    },
}

var keysShowCmd = &cobra.Command{
    Use:   "show <name>",
    Short: "Show key details",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        keyName := args[0]
        
        kr, err := getKeyring()
        if err != nil {
            return err
        }
        defer kr.Close()
        
        meta, err := kr.GetMetadata(keyName)
        if err != nil {
            return err
        }
        
        fmt.Printf("Name:       %s\n", meta.Name)
        fmt.Printf("Address:    %s\n", meta.Address)
        fmt.Printf("Algorithm:  %s\n", meta.Algorithm)
        fmt.Printf("Exportable: %v\n", meta.Exportable)
        fmt.Printf("Created:    %s\n", meta.CreatedAt.Format("2006-01-02 15:04:05"))
        fmt.Printf("Source:     %s\n", meta.Source)
        return nil
    },
}

var keysDeleteCmd = &cobra.Command{
    Use:   "delete <name>",
    Short: "Delete a key",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        keyName := args[0]
        yes, _ := cmd.Flags().GetBool("yes")
        
        if !yes {
            fmt.Printf("Are you sure you want to delete key %q? [y/N]: ", keyName)
            var response string
            fmt.Scanln(&response)
            if response != "y" && response != "Y" {
                fmt.Println("Aborted")
                return nil
            }
        }
        
        kr, err := getKeyring()
        if err != nil {
            return err
        }
        defer kr.Close()
        
        if err := kr.Delete(keyName); err != nil {
            return err
        }
        
        fmt.Printf("Deleted key: %s\n", keyName)
        return nil
    },
}

func init() {
    keysCreateCmd.Flags().Bool("exportable", false, "Allow key export")
    keysDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
    
    keysCmd.AddCommand(keysCreateCmd)
    keysCmd.AddCommand(keysListCmd)
    keysCmd.AddCommand(keysShowCmd)
    keysCmd.AddCommand(keysDeleteCmd)
}

func getKeyring() (*banhbaoring.BaoKeyring, error) {
    addr := baoAddr
    if addr == "" {
        addr = os.Getenv("BAO_ADDR")
    }
    
    token := baoToken
    if token == "" {
        token = os.Getenv("BAO_TOKEN")
    }
    
    if addr == "" || token == "" {
        return nil, fmt.Errorf("BAO_ADDR and BAO_TOKEN are required")
    }
    
    cfg := banhbaoring.Config{
        BaoAddr:   addr,
        BaoToken:  token,
        StorePath: storePath,
    }
    
    return banhbaoring.New(context.Background(), cfg)
}
```

### 3.3 cmd/banhbao/migrate.go

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/Bidon15/banhbaoring/migration"
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
    "github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
    Use:   "migrate",
    Short: "Migrate keys between keyrings",
}

var migrateImportCmd = &cobra.Command{
    Use:   "import",
    Short: "Import key from local keyring to OpenBao",
    RunE: func(cmd *cobra.Command, args []string) error {
        fromPath, _ := cmd.Flags().GetString("from")
        backend, _ := cmd.Flags().GetString("backend")
        keyName, _ := cmd.Flags().GetString("key-name")
        deleteAfter, _ := cmd.Flags().GetBool("delete-after-import")
        all, _ := cmd.Flags().GetBool("all")
        
        // Open source keyring
        sourceKr, err := keyring.New("celestia", keyring.BackendType(backend), fromPath, os.Stdin, nil)
        if err != nil {
            return fmt.Errorf("failed to open source keyring: %w", err)
        }
        
        // Open destination (BaoKeyring)
        destKr, err := getKeyring()
        if err != nil {
            return err
        }
        defer destKr.Close()
        
        ctx := context.Background()
        
        if all {
            // Batch import
            result, err := migration.BatchImport(ctx, migration.BatchImportConfig{
                SourceKeyring:     sourceKr,
                DestKeyring:       destKr,
                DeleteAfterImport: deleteAfter,
                VerifyAfterImport: true,
            })
            if err != nil {
                return err
            }
            
            fmt.Printf("Imported %d keys successfully\n", len(result.Successful))
            if len(result.Failed) > 0 {
                fmt.Printf("Failed to import %d keys:\n", len(result.Failed))
                for _, f := range result.Failed {
                    fmt.Printf("  - %s: %v\n", f.KeyName, f.Error)
                }
            }
        } else {
            // Single import
            result, err := migration.Import(ctx, migration.ImportConfig{
                SourceKeyring:     sourceKr,
                DestKeyring:       destKr,
                KeyName:           keyName,
                DeleteAfterImport: deleteAfter,
                VerifyAfterImport: true,
            })
            if err != nil {
                return err
            }
            
            fmt.Printf("Imported key: %s\n", result.KeyName)
            fmt.Printf("Address: %s\n", result.Address)
            fmt.Printf("Verified: %v\n", result.Verified)
        }
        
        return nil
    },
}

var migrateExportCmd = &cobra.Command{
    Use:   "export",
    Short: "Export key from OpenBao to local keyring",
    RunE: func(cmd *cobra.Command, args []string) error {
        toPath, _ := cmd.Flags().GetString("to")
        backend, _ := cmd.Flags().GetString("backend")
        keyName, _ := cmd.Flags().GetString("key-name")
        confirm, _ := cmd.Flags().GetString("confirm")
        
        if confirm != "I understand this compromises key security" {
            fmt.Println("⚠️  SECURITY WARNING")
            fmt.Println("Exporting keys compromises security.")
            fmt.Println("Use --confirm 'I understand this compromises key security' to proceed.")
            return nil
        }
        
        // Open source (BaoKeyring)
        sourceKr, err := getKeyring()
        if err != nil {
            return err
        }
        defer sourceKr.Close()
        
        // Open destination keyring
        destKr, err := keyring.New("celestia", keyring.BackendType(backend), toPath, os.Stdin, nil)
        if err != nil {
            return fmt.Errorf("failed to open destination keyring: %w", err)
        }
        
        ctx := context.Background()
        
        result, err := migration.Export(ctx, migration.ExportConfig{
            SourceKeyring:     sourceKr,
            DestKeyring:       destKr,
            KeyName:           keyName,
            VerifyAfterExport: true,
            Confirmed:         true,
        })
        if err != nil {
            return err
        }
        
        fmt.Printf("Exported key: %s\n", result.KeyName)
        fmt.Printf("Verified: %v\n", result.Verified)
        return nil
    },
}

func init() {
    migrateImportCmd.Flags().String("from", "", "Source keyring path")
    migrateImportCmd.Flags().String("backend", "file", "Source keyring backend")
    migrateImportCmd.Flags().String("key-name", "", "Key name to import")
    migrateImportCmd.Flags().Bool("delete-after-import", false, "Delete from source after import")
    migrateImportCmd.Flags().Bool("all", false, "Import all keys")
    
    migrateExportCmd.Flags().String("to", "", "Destination keyring path")
    migrateExportCmd.Flags().String("backend", "file", "Destination keyring backend")
    migrateExportCmd.Flags().String("key-name", "", "Key name to export")
    migrateExportCmd.Flags().String("confirm", "", "Confirmation string")
    
    migrateCmd.AddCommand(migrateImportCmd)
    migrateCmd.AddCommand(migrateExportCmd)
}
```

---

## 4. Unit Test Requirements

### 4.1 migration/migration_test.go

```go
func TestImport(t *testing.T) {
    // Setup mock keyrings
    sourceKr := setupMockLocalKeyring(t)
    destKr := setupMockBaoKeyring(t)
    
    // Create key in source
    sourceKr.NewAccount("import-test", "", "", "", nil)
    
    result, err := Import(context.Background(), ImportConfig{
        SourceKeyring:     sourceKr,
        DestKeyring:       destKr,
        KeyName:           "import-test",
        VerifyAfterImport: true,
    })
    
    require.NoError(t, err)
    require.Equal(t, "import-test", result.KeyName)
    require.True(t, result.Verified)
}

func TestExport_RequiresConfirmation(t *testing.T) {
    _, err := Export(context.Background(), ExportConfig{
        Confirmed: false,
    })
    
    require.Error(t, err)
    require.Contains(t, err.Error(), "confirmation")
}

func TestExport_RejectsNonExportable(t *testing.T) {
    sourceKr := setupMockBaoKeyring(t)
    destKr := setupMockLocalKeyring(t)
    
    // Create non-exportable key
    sourceKr.NewAccount("no-export", "", "", "", nil)
    
    _, err := Export(context.Background(), ExportConfig{
        SourceKeyring: sourceKr,
        DestKeyring:   destKr,
        KeyName:       "no-export",
        Confirmed:     true,
    })
    
    require.Error(t, err)
}
```

---

## 5. Success Criteria

### 5.1 Migration Package

- [ ] Import transfers key from local to OpenBao
- [ ] Export transfers key from OpenBao to local
- [ ] Key wrapping uses RSA-OAEP
- [ ] Memory is wiped after use (secureZero)
- [ ] Verification signs test data
- [ ] BatchImport handles multiple keys
- [ ] Export requires explicit confirmation
- [ ] Export rejects non-exportable keys

### 5.2 CLI Tool

- [ ] `banhbao keys create` creates keys
- [ ] `banhbao keys list` lists keys
- [ ] `banhbao keys show` shows details
- [ ] `banhbao keys delete` with confirmation
- [ ] `banhbao migrate import` imports keys
- [ ] `banhbao migrate export` with security warning
- [ ] Environment variables supported
- [ ] Helpful error messages

---

## 6. Dependencies

```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/cosmos/cosmos-sdk v0.50.x
)
```

---

## 7. Deliverables Checklist

- [ ] `migration/migration.go` - Import/export logic
- [ ] `migration/types.go` - Types
- [ ] `migration/migration_test.go` - Tests
- [ ] `cmd/banhbao/main.go` - CLI entrypoint
- [ ] `cmd/banhbao/keys.go` - Key commands
- [ ] `cmd/banhbao/migrate.go` - Migration commands
- [ ] All tests pass
- [ ] CLI builds and runs
- [ ] Security warnings displayed for export

