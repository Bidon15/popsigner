# Agent Task: Rebrand Plugin

> **Parallel Execution:** ✅ Can run independently
> **Dependencies:** None
> **Estimated Time:** 30 minutes

---

## Objective

Rename the OpenBao secp256k1 plugin binary and update related files.

---

## Scope

### Binary Rename

```
plugin/banhbaoring-secp256k1  →  plugin/popsigner-secp256k1
```

### Files to Modify

| File | Changes |
|------|---------|
| `plugin/Dockerfile` | Binary name, image labels |
| `plugin/go.mod` | Module description |
| `plugin/cmd/main.go` | Plugin registration name (if applicable) |
| `plugin/secp256k1/*.go` | Comments, error messages |
| `Makefile` (root) | Plugin build targets |
| `scripts/build-push.sh` | Image names |

---

## Implementation

### Step 1: Update plugin/Dockerfile

```dockerfile
# Before
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o banhbaoring-secp256k1 ./cmd

FROM alpine:3.19
COPY --from=builder /app/banhbaoring-secp256k1 /plugins/
ENTRYPOINT ["/plugins/banhbaoring-secp256k1"]

# After
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o popsigner-secp256k1 ./cmd

FROM alpine:3.19
LABEL org.opencontainers.image.title="POPSigner secp256k1 Plugin"
LABEL org.opencontainers.image.description="OpenBao plugin for secp256k1 signing"
COPY --from=builder /app/popsigner-secp256k1 /plugins/
ENTRYPOINT ["/plugins/popsigner-secp256k1"]
```

### Step 2: Update plugin/go.mod

```go
// Add description comment
module github.com/Bidon15/banhbaoring/plugin

// POPSigner secp256k1 Plugin for OpenBao
// Provides native secp256k1 signing operations
```

### Step 3: Update plugin/cmd/main.go

```go
// Before
func main() {
    // Plugin registration - keep path as secp256k1 (OpenBao path)
    plugin.Serve(&plugin.ServeOpts{
        BackendFactoryFunc: secp256k1.Factory,
    })
}

// After - add header comment
// POPSigner secp256k1 Plugin
//
// This OpenBao plugin provides native secp256k1 signing operations.
// Keys never leave the vault boundary.
package main

func main() {
    plugin.Serve(&plugin.ServeOpts{
        BackendFactoryFunc: secp256k1.Factory,
    })
}
```

### Step 4: Update Root Makefile

```makefile
# Before
.PHONY: build-plugin
build-plugin:
	cd plugin && go build -o banhbaoring-secp256k1 ./cmd

.PHONY: docker-plugin
docker-plugin:
	docker build -t ghcr.io/bidon15/banhbaoring-secp256k1:dev ./plugin

# After
.PHONY: build-plugin
build-plugin:
	cd plugin && go build -o popsigner-secp256k1 ./cmd

.PHONY: docker-plugin
docker-plugin:
	docker build -t ghcr.io/bidon15/popsigner-secp256k1:dev ./plugin
```

### Step 5: Update scripts/build-push.sh

```bash
# Before
PLUGIN_IMAGE="ghcr.io/bidon15/banhbaoring-secp256k1"

# After
PLUGIN_IMAGE="ghcr.io/bidon15/popsigner-secp256k1"
```

### Step 6: Delete Old Binary (if exists)

```bash
rm -f plugin/banhbaoring-secp256k1
```

---

## Verification

```bash
# Build plugin
cd plugin && go build -o popsigner-secp256k1 ./cmd

# Check binary exists
ls -la popsigner-secp256k1

# Build Docker image
docker build -t popsigner-secp256k1:test .

# Check for remaining references
grep -r "banhbaoring-secp256k1" ../
```

---

## Checklist

```
□ Update plugin/Dockerfile - binary name, labels
□ Update plugin/go.mod - description
□ Update plugin/cmd/main.go - header comments
□ Update root Makefile - build targets
□ Update scripts/build-push.sh - image names
□ Delete old binary if exists
□ go build ./cmd passes
□ docker build passes
□ No remaining "banhbaoring-secp256k1" references
```

---

## Output

After completion, the plugin binary is `popsigner-secp256k1`.

**Note:** The OpenBao mount path (`secp256k1`) stays unchanged - it's the algorithm name, not the product name.

