# Implementation Guide: CLI Migration Commands

**Agent ID:** 05D  
**Parent:** Agent 05 (Migration & CLI)  
**Component:** Migration CLI Commands  
**Parallelizable:** ✅ Yes - Uses 05A, 05B

---

## 1. Overview

CLI commands for key migration: import, export.

### 1.1 Required Skills

| Skill     | Level        | Description       |
| --------- | ------------ | ----------------- |
| **Go**    | Intermediate | CLI patterns      |
| **Cobra** | Intermediate | Subcommands       |

### 1.2 Files to Create

```
cmd/banhbao/
└── migrate.go
```

---

## 2. Specifications

**IMPORTANT:** cosmos-sdk imports are replaced by Celestia's fork via go.mod.

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/Bidon15/banhbaoring/migration"
    // Replaced by celestiaorg/cosmos-sdk via go.mod
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
    "github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
    Use:   "migrate",
    Short: "Migrate keys between keyrings",
}

var migrateImportCmd = &cobra.Command{
    Use:   "import",
    Short: "Import from local keyring",
    RunE: func(cmd *cobra.Command, args []string) error {
        fromPath, _ := cmd.Flags().GetString("from")
        backend, _ := cmd.Flags().GetString("backend")
        keyName, _ := cmd.Flags().GetString("key-name")
        deleteAfter, _ := cmd.Flags().GetBool("delete-after-import")
        all, _ := cmd.Flags().GetBool("all")
        
        sourceKr, err := keyring.New("celestia", keyring.BackendType(backend), fromPath, os.Stdin, nil)
        if err != nil {
            return fmt.Errorf("open source: %w", err)
        }
        
        destKr, err := getKeyring()
        if err != nil {
            return err
        }
        defer destKr.Close()
        
        ctx := context.Background()
        
        if all {
            result, err := migration.BatchImport(ctx, migration.BatchImportConfig{
                SourceKeyring:     sourceKr,
                DestKeyring:       destKr,
                DeleteAfterImport: deleteAfter,
                VerifyAfterImport: true,
            })
            if err != nil {
                return err
            }
            
            fmt.Printf("Imported: %d, Failed: %d\n", len(result.Successful), len(result.Failed))
            for _, f := range result.Failed {
                fmt.Printf("  ✗ %s: %v\n", f.KeyName, f.Error)
            }
        } else {
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
            
            fmt.Printf("Imported: %s\nAddress: %s\nVerified: %v\n",
                result.KeyName, result.Address, result.Verified)
        }
        return nil
    },
}

var migrateExportCmd = &cobra.Command{
    Use:   "export",
    Short: "Export to local keyring",
    RunE: func(cmd *cobra.Command, args []string) error {
        toPath, _ := cmd.Flags().GetString("to")
        backend, _ := cmd.Flags().GetString("backend")
        keyName, _ := cmd.Flags().GetString("key-name")
        confirm, _ := cmd.Flags().GetString("confirm")
        
        if confirm != "I understand this compromises key security" {
            fmt.Println(migration.SecurityWarning(keyName, "", toPath))
            fmt.Println("\nUse --confirm 'I understand this compromises key security'")
            return nil
        }
        
        sourceKr, err := getKeyring()
        if err != nil {
            return err
        }
        defer sourceKr.Close()
        
        destKr, err := keyring.New("celestia", keyring.BackendType(backend), toPath, os.Stdin, nil)
        if err != nil {
            return fmt.Errorf("open destination: %w", err)
        }
        
        result, err := migration.Export(context.Background(), migration.ExportConfig{
            SourceKeyring:     sourceKr,
            DestKeyring:       destKr,
            KeyName:           keyName,
            VerifyAfterExport: true,
            Confirmed:         true,
        })
        if err != nil {
            return err
        }
        
        fmt.Printf("Exported: %s\nVerified: %v\n", result.KeyName, result.Verified)
        return nil
    },
}

func init() {
    migrateImportCmd.Flags().String("from", "", "Source keyring path")
    migrateImportCmd.Flags().String("backend", "file", "Source backend")
    migrateImportCmd.Flags().String("key-name", "", "Key to import")
    migrateImportCmd.Flags().Bool("delete-after-import", false, "Delete from source")
    migrateImportCmd.Flags().Bool("all", false, "Import all keys")
    
    migrateExportCmd.Flags().String("to", "", "Destination path")
    migrateExportCmd.Flags().String("backend", "file", "Destination backend")
    migrateExportCmd.Flags().String("key-name", "", "Key to export")
    migrateExportCmd.Flags().String("confirm", "", "Confirmation string")
    
    migrateCmd.AddCommand(migrateImportCmd, migrateExportCmd)
    rootCmd.AddCommand(migrateCmd)
}
```

---

## 3. Deliverables

- [ ] `banhbao migrate import --from <path> --key-name <name>`
- [ ] `banhbao migrate import --all` for batch import
- [ ] `banhbao migrate export --to <path> --key-name <name>`
- [ ] Export requires explicit --confirm flag
- [ ] Security warning displayed for exports

