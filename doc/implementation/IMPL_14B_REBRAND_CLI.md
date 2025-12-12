# Agent Task: Rebrand CLI

> **Parallel Execution:** ✅ Can run independently
> **Dependencies:** None (but coordinate with IMPL_14A for imports)
> **Estimated Time:** 1 hour

---

## Objective

Rename CLI from `banhbao` to `popsigner`.

---

## Scope

### Directory Rename

```
cmd/banhbao/           →  cmd/popsigner/
```

### Files to Modify

| File | Changes |
|------|---------|
| `cmd/popsigner/main.go` | Binary name, help text, imports |
| `cmd/popsigner/keys.go` | Help text, error messages |
| `cmd/popsigner/migrate.go` | Help text, error messages |
| `cmd/popsigner/commands_test.go` | Test updates |
| `Makefile` | Build targets |

### Files to Keep Unchanged

- `cmd/baokey/` - Internal OpenBao tool, keep as-is

---

## Implementation

### Step 1: Rename Directory

```bash
mv cmd/banhbao cmd/popsigner
```

### Step 2: Update cmd/popsigner/main.go

```go
// Before
package main

import (
    "github.com/Bidon15/banhbaoring"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "banhbao",
        Short: "BanhBaoRing CLI for key management",
        // ...
    }
}

// After
package main

import (
    popsigner "github.com/Bidon15/banhbaoring"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "popsigner",
        Short: "POPSigner CLI - Point-of-Presence signing",
        Long: `POPSigner CLI for key management and migration.

POPSigner is Point-of-Presence signing infrastructure.
Deploy inline with execution. Keys remain remote. You remain sovereign.`,
        // ...
    }
}
```

### Step 3: Update cmd/popsigner/keys.go

```go
// Update command descriptions
var keysCmd = &cobra.Command{
    Use:   "keys",
    Short: "Manage signing keys",
    Long:  "Create, list, and manage keys in POPSigner.",
}

// Update any error messages
fmt.Println("popsigner: key created successfully")
```

### Step 4: Update cmd/popsigner/migrate.go

```go
// Update command descriptions
var migrateCmd = &cobra.Command{
    Use:   "migrate",
    Short: "Import and export keys",
    Long:  "Migrate keys to/from POPSigner.",
}

// Update any error messages
fmt.Println("popsigner: migration complete")
```

### Step 5: Update Environment Variable Names

```go
// Before
token := os.Getenv("BANHBAO_TOKEN")
addr := os.Getenv("BANHBAO_ADDR")

// After
token := os.Getenv("POPSIGNER_API_KEY")
addr := os.Getenv("POPSIGNER_ADDR")
```

### Step 6: Update Makefile

```makefile
# Before
.PHONY: build-cli
build-cli:
	go build -o banhbao ./cmd/banhbao

# After
.PHONY: build-cli
build-cli:
	go build -o popsigner ./cmd/popsigner

# Update install target
.PHONY: install
install:
	go install ./cmd/popsigner
```

---

## Verification

```bash
# Build CLI
go build -o popsigner ./cmd/popsigner

# Test help output
./popsigner --help
./popsigner keys --help
./popsigner migrate --help

# Run tests
go test ./cmd/popsigner/...

# Check for remaining references
grep -r "banhbao" ./cmd/popsigner/
```

---

## Checklist

```
□ Rename cmd/banhbao/ → cmd/popsigner/
□ cmd/popsigner/main.go - Use, Short, Long, imports
□ cmd/popsigner/keys.go - descriptions, messages
□ cmd/popsigner/migrate.go - descriptions, messages
□ cmd/popsigner/commands_test.go - test updates
□ Environment variables: BANHBAO_* → POPSIGNER_*
□ Makefile - build targets
□ go build ./cmd/popsigner passes
□ go test ./cmd/popsigner/... passes
□ No remaining "banhbao" references
```

---

## Output

After completion, the CLI is available as `popsigner` command.

