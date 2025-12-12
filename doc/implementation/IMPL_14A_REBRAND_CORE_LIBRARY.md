# Agent Task: Rebrand Core Library

> **Parallel Execution:** ✅ Can run independently
> **Dependencies:** None
> **Estimated Time:** 1-2 hours

---

## Objective

Rename package from `banhbaoring` to `popsigner` in the core library files.

---

## Scope

### Files to Modify

| File | Change |
|------|--------|
| `bao_client.go` | `package banhbaoring` → `package popsigner` |
| `bao_client_test.go` | `package banhbaoring` → `package popsigner` |
| `bao_keyring.go` | `package banhbaoring` → `package popsigner` |
| `bao_keyring_test.go` | `package banhbaoring` → `package popsigner` |
| `bao_keyring_parallel_test.go` | `package banhbaoring` → `package popsigner` |
| `bao_store.go` | `package banhbaoring` → `package popsigner` |
| `bao_store_test.go` | `package banhbaoring` → `package popsigner` |
| `types.go` | `package banhbaoring` → `package popsigner` |
| `types_test.go` | `package banhbaoring` → `package popsigner` |
| `errors.go` | `package banhbaoring` → `package popsigner` |
| `errors_test.go` | `package banhbaoring` → `package popsigner` |
| `go.mod` | Update module description |
| `example/main.go` | Update import path |

### Files to Keep Unchanged

- Filenames stay as-is (`bao_client.go`, `bao_keyring.go`, etc.)
- Internal OpenBao logic unchanged

---

## Implementation

### Step 1: Update Package Declarations

For each `.go` file in the root directory:

```go
// Before
package banhbaoring

// After
package popsigner
```

### Step 2: Update go.mod

```go
// Before
module github.com/Bidon15/banhbaoring

// After
module github.com/Bidon15/popsigner
```

**Note:** If keeping the repo name, just update comments:

```go
module github.com/Bidon15/banhbaoring

// POPSigner - Point-of-Presence signing infrastructure
// (formerly BanhBaoRing)
```

### Step 3: Update example/main.go

```go
// Before
import (
    "github.com/Bidon15/banhbaoring"
)

// After
import (
    popsigner "github.com/Bidon15/banhbaoring"
)
```

### Step 4: Update Any Error Messages

Search for "banhbao" in error messages and update:

```go
// Before
return fmt.Errorf("banhbaoring: failed to connect")

// After
return fmt.Errorf("popsigner: failed to connect")
```

---

## Verification

```bash
# Build the library
go build ./...

# Run tests
go test ./...

# Check for any remaining references
grep -r "banhbaoring" --include="*.go" .
```

---

## Checklist

```
□ bao_client.go - package declaration
□ bao_client_test.go - package declaration
□ bao_keyring.go - package declaration
□ bao_keyring_test.go - package declaration
□ bao_keyring_parallel_test.go - package declaration
□ bao_store.go - package declaration
□ bao_store_test.go - package declaration
□ types.go - package declaration
□ types_test.go - package declaration
□ errors.go - package declaration + error messages
□ errors_test.go - package declaration
□ example/main.go - import alias
□ go.mod - module description/comments
□ go build ./... passes
□ go test ./... passes
□ No remaining "banhbaoring" references in .go files
```

---

## Output

After completion, notify that core library is ready and other agents can update their imports.

