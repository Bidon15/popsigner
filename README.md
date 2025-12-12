# POPSigner

**Point-of-Presence Signing Infrastructure**

[![Go Reference](https://pkg.go.dev/badge/github.com/Bidon15/banhbaoring.svg)](https://pkg.go.dev/github.com/Bidon15/banhbaoring)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> POPSigner is a distributed signing layer designed to live inline with execution—not behind an API queue.

---

## What POPSigner Is

POPSigner is Point-of-Presence signing infrastructure. It deploys where your systems already run—the same region, the same rack, the same execution path.

**This isn't custody. This isn't MPC. This is signing at the point of execution.**

```
┌─────────────────────────────────────────────────────────────┐
│  YOUR INFRASTRUCTURE                                         │
│                                                              │
│  ┌──────────────┐    inline    ┌──────────────────────────┐ │
│  │  Execution   │ ───────────▶ │  POPSigner POP           │ │
│  │  (sequencer, │              │  (same region)           │ │
│  │   bot, etc.) │ ◀─────────── │                          │ │
│  └──────────────┘   signature  └──────────────────────────┘ │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Core Principles

| Principle | Description |
|-----------|-------------|
| **Inline Signing** | Signing happens on the execution path, not behind a queue |
| **Sovereignty by Default** | Keys are remote, but you control them. Export anytime. Exit anytime. |
| **Neutral Anchor** | Recovery data is anchored to neutral data availability. If we disappear, you don't. |

---

## Quick Start

### Option 1: POPSigner Cloud

Deploy without infrastructure. Connect and sign.

```bash
# Get your API key at https://popsigner.io
go get github.com/Bidon15/banhbaoring
```

```go
package main

import (
    "context"
    "os"
    
    popsigner "github.com/Bidon15/banhbaoring"
)

func main() {
    ctx := context.Background()
    
    // Connect to POPSigner
    client := popsigner.NewClient(os.Getenv("POPSIGNER_API_KEY"))
    
    // Create a key
    key, _ := client.Keys.Create(ctx, popsigner.CreateKeyRequest{
        Name: "sequencer-key",
    })
    
    // Sign inline with your execution
    sig, _ := client.Sign.Sign(ctx, key.ID, txBytes, false)
}
```

### Option 2: Self-Hosted

Run POPSigner on your own infrastructure. Full control. No dependencies.

```go
kr, _ := popsigner.New(ctx, popsigner.Config{
    BaoAddr:   "https://your-openbao.internal:8200",
    BaoToken:  os.Getenv("BAO_TOKEN"),
    StorePath: "./keyring-metadata.json",
})
```

See [Deployment Guide](doc/product/DEPLOYMENT.md) for Kubernetes setup.

---

## Why POPSigner

| | Local Keyring | Cloud KMS | **POPSigner** |
|--|--------------|-----------|---------------|
| **Key exposure** | On disk | Decrypted in app | **Never exposed** |
| **secp256k1** | ✅ | ❌ | ✅ |
| **Placement** | Local only | Their region | **Your region** |
| **Self-hostable** | ✅ | ❌ | ✅ |
| **Managed option** | ❌ | ✅ | ✅ |
| **Exit guarantee** | N/A | ❌ | **Always** |

---

## Exit Guarantee

POPSigner is designed with exit as a first-class primitive.

- **Key Export**: Your keys are exportable by default. No ceremony. No approval workflow.
- **Recovery Anchor**: Recovery data is anchored to neutral data availability infrastructure.
- **Force Exit**: If POPSigner is unavailable for any reason, you can force recovery. This is not gated.

---

## Plugin Architecture

POPSigner ships with `secp256k1`. But the plugin architecture is the actual product.

- Plugins are chain-agnostic
- Plugins are free
- Plugins don't require approval

```go
// Built-in secp256k1
sig, pubKey, _ := kr.Sign("my-key", signBytes, signMode)

// Your custom algorithm tomorrow
```

---

## Integration

### Celestia / Cosmos SDK

POPSigner implements the standard `keyring.Keyring` interface:

```go
import (
    "github.com/celestiaorg/celestia-node/api/client"
    popsigner "github.com/Bidon15/banhbaoring"
)

func main() {
    ctx := context.Background()
    
    // POPSigner as keyring
    kr, _ := popsigner.NewClient(os.Getenv("POPSIGNER_API_KEY"))
    
    // Plug into Celestia
    celestiaClient, _ := client.NewWithKeyring(ctx, clientConfig, kr)
    
    // Submit blobs—signing happens inline
    height, _ := celestiaClient.Blob.Submit(ctx, blobs, nil)
}
```

### Parallel Workers

POPSigner supports worker-native architecture for burst workloads:

```go
// Create signing workers
keys, _ := client.Keys.CreateBatch(ctx, popsigner.CreateBatchRequest{
    Prefix: "blob-worker",
    Count:  4,
})

// Sign in parallel—no blocking
results, _ := client.Sign.SignBatch(ctx, popsigner.BatchSignRequest{
    Requests: []popsigner.SignRequest{
        {KeyID: keys[0].ID, Data: tx1},
        {KeyID: keys[1].ID, Data: tx2},
        {KeyID: keys[2].ID, Data: tx3},
        {KeyID: keys[3].ID, Data: tx4},
    },
})
```

---

## CLI

```bash
# Install
go install github.com/Bidon15/banhbaoring/cmd/popsigner@latest

# Configure
export POPSIGNER_API_KEY="psk_xxx"

# Key management
popsigner keys create my-sequencer
popsigner keys list
popsigner keys show my-sequencer

# Sign
popsigner sign --key my-sequencer message.txt

# Health check
popsigner health
```

---

## Migration

### Import existing keys

```bash
popsigner migrate import \
  --from ~/.celestia-app/keyring-file \
  --key-name my-validator
```

### Export keys (exit guarantee)

```bash
popsigner migrate export \
  --to ./exported-keys \
  --key my-validator
```

See [Migration Guide](doc/product/MIGRATION.md) for all options.

---

## Documentation

| Document | Description |
|----------|-------------|
| [Integration Guide](doc/product/INTEGRATION.md) | Celestia client integration |
| [Migration Guide](doc/product/MIGRATION.md) | Import/export keys |
| [API Reference](doc/product/API_REFERENCE.md) | REST API endpoints |
| [Deployment Guide](doc/product/DEPLOYMENT.md) | Self-hosted Kubernetes setup |
| [Architecture](doc/product/ARCHITECTURE.md) | Technical design |
| [Plugin Design](doc/product/PLUGIN_DESIGN.md) | OpenBao plugin details |

---

## Installation

```bash
# Go SDK
go get github.com/Bidon15/banhbaoring

# CLI
go install github.com/Bidon15/banhbaoring/cmd/popsigner@latest
```

### Requirements

| Deployment | Requirements |
|------------|--------------|
| **Cloud** | API key only |
| **Self-hosted** | OpenBao + secp256k1 plugin, Kubernetes 1.25+ |

---

## Pricing

| Tier | Monthly | Description |
|------|---------|-------------|
| **Shared POPSigner** | €49 | Shared POP infrastructure. For experimentation. |
| **Priority POPSigner** | €499 | Priority lanes, region selection, 99.9% SLA |
| **Dedicated POPSigner** | €19,999 | Region-pinned, CPU isolation, 99.99% SLA |

Self-host option is always free. [View pricing details →](https://popsigner.io/pricing)

---

## About the Name

POPSigner (formerly BanhBaoRing) reflects a clearer articulation of what the system is: **Point-of-Presence signing infrastructure**.

The rename signals a shift from playful internal naming to category-defining infrastructure positioning.

---

## Contributing

```bash
git clone https://github.com/Bidon15/banhbaoring.git
cd banhbaoring
go mod download
go test ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

---

<p align="center">
  <b>POPSigner</b> — Signing at the point of execution.
  <br><br>
  <a href="https://popsigner.io">Deploy POPSigner</a> · 
  <a href="doc/product/INTEGRATION.md">Documentation</a> · 
  <a href="https://github.com/Bidon15/banhbaoring">GitHub</a>
</p>
