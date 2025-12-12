# Agent Task: Rebrand SDK-Go

> **Parallel Execution:** ✅ Can run independently
> **Dependencies:** None
> **Estimated Time:** 1-2 hours

---

## Objective

Rename SDK-Go from `banhbaoring` to `popsigner`.

---

## Scope

### Files to Rename

```
sdk-go/banhbaoring.go       →  sdk-go/popsigner.go
sdk-go/banhbaoring_test.go  →  sdk-go/popsigner_test.go
```

### Files to Modify (Content Only)

| File | Changes |
|------|---------|
| `sdk-go/popsigner.go` | Package name, types, comments |
| `sdk-go/popsigner_test.go` | Package name, tests |
| `sdk-go/http.go` | Header names, comments |
| `sdk-go/keys.go` | Comments |
| `sdk-go/sign.go` | Comments |
| `sdk-go/orgs.go` | Comments |
| `sdk-go/audit.go` | Comments |
| `sdk-go/types.go` | Comments |
| `sdk-go/errors.go` | Error messages |
| `sdk-go/go.mod` | Module path/description |
| `sdk-go/examples/basic/main.go` | Imports, comments |
| `sdk-go/examples/parallel-workers/main.go` | Imports, comments |

---

## Implementation

### Step 1: Rename Files

```bash
cd sdk-go
mv banhbaoring.go popsigner.go
mv banhbaoring_test.go popsigner_test.go
```

### Step 2: Update sdk-go/popsigner.go

```go
// Before
// Package banhbaoring provides a Go SDK for the BanhBaoRing Control Plane API.
package banhbaoring

// Client is the BanhBaoRing API client.
type Client struct {
    // ...
}

// After
// Package popsigner provides a Go SDK for the POPSigner Control Plane API.
//
// POPSigner is Point-of-Presence signing infrastructure.
// Deploy inline with execution. Keys remain remote. You remain sovereign.
package popsigner

// Client is the POPSigner API client.
type Client struct {
    // ...
}
```

### Step 3: Update sdk-go/http.go

```go
// Before
req.Header.Set("X-BanhBaoRing-API-Key", c.apiKey)

// After
req.Header.Set("X-POPSigner-API-Key", c.apiKey)
```

### Step 4: Update API Key Validation

```go
// Before
if !strings.HasPrefix(apiKey, "bbr_") {
    return nil, fmt.Errorf("invalid API key format: must start with bbr_")
}

// After
if !strings.HasPrefix(apiKey, "psk_") {
    return nil, fmt.Errorf("invalid API key format: must start with psk_")
}
```

### Step 5: Update sdk-go/errors.go

```go
// Before
return fmt.Errorf("banhbaoring: %w", err)

// After
return fmt.Errorf("popsigner: %w", err)
```

### Step 6: Update sdk-go/go.mod

```go
module github.com/popsigner/sdk-go

// Or if keeping repo structure:
module github.com/Bidon15/banhbaoring/sdk-go

// POPSigner Go SDK
```

### Step 7: Update Examples

```go
// sdk-go/examples/basic/main.go

// Before
import "github.com/banhbaoring/sdk-go"
client := banhbaoring.NewClient("bbr_live_xxx")

// After
import "github.com/popsigner/sdk-go"
// or
import popsigner "github.com/Bidon15/banhbaoring/sdk-go"

client := popsigner.NewClient("psk_live_xxx")
```

### Step 8: Update All Package Declarations

For every `.go` file in sdk-go/:

```go
// Before
package banhbaoring

// After
package popsigner
```

---

## Verification

```bash
cd sdk-go

# Build
go build ./...

# Test
go test ./...

# Check for remaining references
grep -r "banhbaoring" --include="*.go" .
grep -r "bbr_" --include="*.go" .
```

---

## Checklist

```
□ Rename banhbaoring.go → popsigner.go
□ Rename banhbaoring_test.go → popsigner_test.go
□ Update package declarations in all files
□ Update popsigner.go - package docs, types
□ Update http.go - header name
□ Update API key prefix validation (bbr_ → psk_)
□ Update errors.go - error prefixes
□ Update go.mod
□ Update examples/basic/main.go
□ Update examples/parallel-workers/main.go
□ go build ./... passes
□ go test ./... passes
□ No remaining "banhbaoring" or "bbr_" references
```

---

## Output

After completion, the SDK is usable as:

```go
import popsigner "github.com/Bidon15/banhbaoring/sdk-go"

client := popsigner.NewClient("psk_live_xxx")
```

