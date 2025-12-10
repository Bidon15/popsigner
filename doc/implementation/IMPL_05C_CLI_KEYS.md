# Implementation Guide: CLI Keys Commands

**Agent ID:** 05C  
**Parent:** Agent 05 (Migration & CLI)  
**Component:** Key Management CLI  
**Parallelizable:** ✅ Yes - Uses 04 BaoKeyring

---

## 1. Overview

CLI commands for key management: create, list, show, delete.

### 1.1 Required Skills

| Skill     | Level        | Description        |
| --------- | ------------ | ------------------ |
| **Go**    | Intermediate | CLI patterns       |
| **Cobra** | Intermediate | Command framework  |

### 1.2 Files to Create

```
cmd/banhbao/
├── main.go
└── keys.go
```

---

## 2. Specifications

### 2.1 main.go

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/Bidon15/banhbaoring"
    "github.com/spf13/cobra"
)

var (
    baoAddr   string
    baoToken  string
    storePath string
)

var rootCmd = &cobra.Command{
    Use:   "banhbao",
    Short: "BanhBao - OpenBao keyring for Celestia",
}

func init() {
    rootCmd.PersistentFlags().StringVar(&baoAddr, "bao-addr", "", "OpenBao address")
    rootCmd.PersistentFlags().StringVar(&baoToken, "bao-token", "", "OpenBao token")
    rootCmd.PersistentFlags().StringVar(&storePath, "store-path", "./keyring.json", "Metadata store path")
    
    rootCmd.AddCommand(keysCmd)
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
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
        return nil, fmt.Errorf("BAO_ADDR and BAO_TOKEN required")
    }
    
    return banhbaoring.New(context.Background(), banhbaoring.Config{
        BaoAddr:   addr,
        BaoToken:  token,
        StorePath: storePath,
    })
}
```

### 2.2 keys.go

```go
package main

import (
    "fmt"
    
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
        name := args[0]
        exportable, _ := cmd.Flags().GetBool("exportable")
        
        kr, err := getKeyring()
        if err != nil {
            return err
        }
        defer kr.Close()
        
        record, err := kr.NewAccountWithOptions(name, banhbaoring.KeyOptions{
            Exportable: exportable,
        })
        if err != nil {
            return err
        }
        
        addr, _ := record.GetAddress()
        fmt.Printf("Created key: %s\nAddress: %s\n", name, addr.String())
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
        kr, err := getKeyring()
        if err != nil {
            return err
        }
        defer kr.Close()
        
        meta, err := kr.GetMetadata(args[0])
        if err != nil {
            return err
        }
        
        fmt.Printf("Name:       %s\n", meta.Name)
        fmt.Printf("Address:    %s\n", meta.Address)
        fmt.Printf("Exportable: %v\n", meta.Exportable)
        fmt.Printf("Created:    %s\n", meta.CreatedAt.Format("2006-01-02 15:04:05"))
        return nil
    },
}

var keysDeleteCmd = &cobra.Command{
    Use:   "delete <name>",
    Short: "Delete a key",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        yes, _ := cmd.Flags().GetBool("yes")
        
        if !yes {
            fmt.Printf("Delete key %q? [y/N]: ", args[0])
            var response string
            fmt.Scanln(&response)
            if response != "y" && response != "Y" {
                return nil
            }
        }
        
        kr, err := getKeyring()
        if err != nil {
            return err
        }
        defer kr.Close()
        
        return kr.Delete(args[0])
    },
}

func init() {
    keysCreateCmd.Flags().Bool("exportable", false, "Allow export")
    keysDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
    
    keysCmd.AddCommand(keysCreateCmd, keysListCmd, keysShowCmd, keysDeleteCmd)
}
```

---

## 3. Deliverables

- [ ] `banhbao keys create <name>` with --exportable flag
- [ ] `banhbao keys list` shows all keys
- [ ] `banhbao keys show <name>` shows details
- [ ] `banhbao keys delete <name>` with confirmation
- [ ] Environment variables: BAO_ADDR, BAO_TOKEN

