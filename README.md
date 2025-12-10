# 🍞 BanhBao Ring

**Secure Key Management for Celestia Rollups**

[![Go Reference](https://pkg.go.dev/badge/github.com/Bidon15/banhbaoring.svg)](https://pkg.go.dev/github.com/Bidon15/banhbaoring)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> Your private keys **never leave** the secure boundary. Ever.

---

## 🚀 Get Started in 2 Minutes

### Option 1: BanhBao Cloud (Recommended)

No infrastructure to deploy. Just sign up and start signing.

```bash
# 1. Get your API token at https://app.banhbao.io
# 2. Install the SDK
go get github.com/Bidon15/banhbaoring
```

```go
package main

import (
    "context"
    "os"
    
    "github.com/Bidon15/banhbaoring"
)

func main() {
    // Connect to BanhBao Cloud
    kr, _ := banhbaoring.NewCloud(ctx, banhbaoring.CloudConfig{
        APIToken: os.Getenv("BANHBAO_TOKEN"),  // From dashboard
    })
    
    // Create a key
    kr.NewAccount("my-rollup-signer", "", "", "", nil)
    
    // Sign transactions - keys never touch your servers
    sig, pubKey, _ := kr.Sign("my-rollup-signer", txBytes, signMode)
}
```

**That's it.** Your keys are secured in our infrastructure. You never see them.

---

### Option 2: Self-Hosted

Run your own OpenBao cluster for complete control.

```go
kr, _ := banhbaoring.New(ctx, banhbaoring.Config{
    BaoAddr:   "https://your-openbao.internal:8200",
    BaoToken:  os.Getenv("BAO_TOKEN"),
    StorePath: "./keyring-metadata.json",
})
```

See [Deployment Guide](doc/product/DEPLOYMENT.md) for Kubernetes setup.

---

## ✨ Why BanhBao?

```
┌─────────────────┐          ┌─────────────────────────────┐
│  Your Rollup    │  sign    │     BanhBao (Cloud/Self)    │
│                 │─────────▶│                             │
│  📝 Transaction │          │  🔒 Private key (sealed)    │
│                 │◀─────────│                             │
│  ✅ Signature   │          │  Key NEVER leaves here      │
└─────────────────┘          └─────────────────────────────┘
```

| | Local Keyring | AWS/GCP KMS | **BanhBao** |
|--|--------------|-------------|-------------|
| **Key exposure** | On disk 😰 | Decrypted in app | **Never exposed** |
| **secp256k1** | ✅ | ❌ | ✅ |
| **Setup time** | Minutes | Hours | **2 minutes** |
| **Self-hostable** | ✅ | ❌ | ✅ |
| **Managed option** | ❌ | ✅ | ✅ |

---

## 🌐 BanhBao Cloud

### Features

| Feature | Free Tier | Pro | Enterprise |
|---------|-----------|-----|------------|
| Keys | 3 | Unlimited | Unlimited |
| Signatures/month | 10,000 | 1M | Unlimited |
| Web Dashboard | ✅ | ✅ | ✅ |
| Key Migration Tools | ✅ | ✅ | ✅ |
| API Access | ✅ | ✅ | ✅ |
| Audit Logs | 7 days | 90 days | 1 year |
| SLA | - | 99.9% | 99.99% |
| Support | Community | Email | Dedicated |

### Dashboard Preview

```
┌─────────────────────────────────────────────────────────────────┐
│  BanhBao Dashboard                              [user@email.com]│
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Your Keys                                         [+ New Key]  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 🔑 my-rollup-signer                                     │   │
│  │    celestia1abc123...xyz789                             │   │
│  │    Created: 2 days ago  │  Signatures: 1,247            │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 🔑 backup-validator                                     │   │
│  │    celestia1def456...uvw012                             │   │
│  │    Created: 1 week ago  │  Signatures: 89               │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Quick Actions                                                  │
│  [Create Key]  [Import Key]  [View API Token]  [Audit Logs]    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Get Started

1. **Sign up** at [app.banhbao.io](https://app.banhbao.io)
2. **Create a key** in the dashboard (or via API)
3. **Copy your API token**
4. **Integrate** with 3 lines of code:

```go
kr, _ := banhbaoring.NewCloud(ctx, banhbaoring.CloudConfig{
    APIToken: os.Getenv("BANHBAO_TOKEN"),
})
```

---

## 🔗 Celestia Integration

Works seamlessly with the Celestia client:

```go
import (
    "github.com/celestiaorg/celestia-node/api/client"
    "github.com/Bidon15/banhbaoring"
)

func main() {
    // BanhBao Cloud
    kr, _ := banhbaoring.NewCloud(ctx, banhbaoring.CloudConfig{
        APIToken: os.Getenv("BANHBAO_TOKEN"),
    })
    
    // Plug into Celestia
    celestiaClient, _ := client.NewWithKeyring(ctx, clientConfig, kr)
    
    // Submit blobs - signing happens via BanhBao
    height, _ := celestiaClient.Blob.Submit(ctx, blobs, nil)
    
    fmt.Printf("Blob submitted at height %d\n", height)
}
```

See [Integration Guide](doc/product/INTEGRATION.md) for complete examples.

---

## 🔐 Security Model

### The Problem with Other Solutions

**Local Keyring:**
```
Your Server
┌─────────────────────────────────────┐
│  ~/.celestia-app/keyring-file      │
│  ┌─────────────────────────────┐   │
│  │ 🔓 Private Key on Disk      │ ← Attacker target
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

**AWS KMS / GCP Cloud KMS:**
```
Your Server                      Cloud KMS
┌─────────────────┐  decrypt   ┌─────────────────┐
│                 │◀───────────│  🔒 Encrypted   │
│  🔓 Key in RAM  │            │  Key            │
│  Sign locally   │            └─────────────────┘
│  ⚠️ Exposed!    │
└─────────────────┘
```

**BanhBao:**
```
Your Server                      BanhBao
┌─────────────────┐  sign req  ┌─────────────────────────┐
│                 │───────────▶│                         │
│  📝 TX bytes    │            │  🔒 Key SEALED          │
│                 │◀───────────│  Sign inside            │
│  ✅ Signature   │  signature │  Key never leaves       │
└─────────────────┘            └─────────────────────────┘
```

### Security Features

- **Zero key exposure** - Private keys never leave BanhBao
- **TLS everywhere** - All API calls encrypted
- **Audit logging** - Every signature logged with metadata
- **Access control** - Fine-grained API token permissions
- **SOC 2 Type II** - Enterprise compliance (Cloud)

---

## 🛠️ CLI Tool

```bash
# Install
go install github.com/Bidon15/banhbaoring/cmd/banhbao@latest

# Configure (Cloud)
export BANHBAO_TOKEN="your-api-token"

# Or configure (Self-hosted)
export BAO_ADDR="https://your-openbao:8200"
export BAO_TOKEN="your-token"

# Key management
banhbao keys create my-validator
banhbao keys list
banhbao keys show my-validator

# Sign a file
banhbao sign --key my-validator message.txt

# Health check
banhbao health
```

---

## 🔄 Migration

Already have keys? Migrate them to BanhBao:

```bash
# Import from local Celestia keyring
banhbao migrate import \
  --from ~/.celestia-app/keyring-file \
  --key-name my-validator

# Your key is now secured in BanhBao
# Delete local copy after verification
```

See [Migration Guide](doc/product/MIGRATION.md) for all options.

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Integration Guide](doc/product/INTEGRATION.md) | Celestia client integration |
| [Migration Guide](doc/product/MIGRATION.md) | Import/export keys |
| [API Reference](doc/product/API_REFERENCE.md) | REST API endpoints |
| [Deployment Guide](doc/product/DEPLOYMENT.md) | Self-hosted Kubernetes setup |
| [Architecture](doc/product/ARCHITECTURE.md) | Technical design |
| [Plugin Design](doc/product/PLUGIN_DESIGN.md) | OpenBao plugin details |
| [PRD](doc/product/PRD.md) | Product requirements |

---

## 📦 Installation

```bash
# Go SDK
go get github.com/Bidon15/banhbaoring

# CLI
go install github.com/Bidon15/banhbaoring/cmd/banhbao@latest
```

### Requirements

| Deployment | Requirements |
|------------|--------------|
| **Cloud** | Just an API token |
| **Self-hosted** | OpenBao + secp256k1 plugin, Kubernetes 1.25+ |

---

## 🧪 Local Development

```bash
# Start local OpenBao (dev mode)
docker run -d --name banhbao-dev \
  -p 8200:8200 \
  -e 'BAO_DEV_ROOT_TOKEN_ID=dev-token' \
  quay.io/openbao/openbao:2.0.0 server -dev

# Configure
export BAO_ADDR="http://localhost:8200"
export BAO_TOKEN="dev-token"

# Test
go test ./...
```

---

## 🗺️ Roadmap

| Phase | Status | Features |
|-------|--------|----------|
| **v0.1** | ✅ Done | Core keyring, secp256k1 plugin |
| **v0.2** | 🚧 Now | CLI tools, migration utilities |
| **v1.0** | 📋 Next | BanhBao Cloud launch |
| **v1.1** | 📋 Planned | Web dashboard, team management |
| **v2.0** | 📋 Future | Key rotation, threshold signatures |

---

## 💬 Community

- **Discord**: [Join our server](https://discord.gg/banhbao)
- **Twitter**: [@banhbao_io](https://twitter.com/banhbao_io)
- **GitHub Issues**: Bug reports and feature requests

---

## 🤝 Contributing

```bash
git clone https://github.com/Bidon15/banhbaoring.git
cd banhbaoring
go mod download
go test ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## 📄 License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

---

<p align="center">
  <b>🍞 BanhBao</b> — Enterprise key security, startup simplicity.
  <br><br>
  <a href="https://app.banhbao.io">Get Started Free</a> · 
  <a href="doc/product/INTEGRATION.md">Docs</a> · 
  <a href="https://discord.gg/banhbao">Discord</a>
</p>
