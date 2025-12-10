# BanhBaoRing Product Documentation

> 🔔 **BanhBaoRing** - Named after the distinctive "ring ring!" of Vietnamese bánh bao street vendors. Just as that familiar sound signals trusted, reliable service arriving at your door, BanhBaoRing signals secure, reliable key management arriving in your infrastructure.

---

## Product Overview

BanhBaoRing is a complete key management SaaS platform for Celestia and Cosmos ecosystems, providing HSM-level security without the complexity.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         BANHBAORING PLATFORM                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │  Web Dashboard  │  │ Control Plane   │  │  K8s Operator   │             │
│  │  (PRD_DASHBOARD)│  │ (PRD_CONTROL)   │  │  (PRD_OPERATOR) │             │
│  │                 │  │                 │  │                 │             │
│  │  User-facing UI │  │  Multi-tenant   │  │  One-command    │             │
│  │  5-min onboard  │  │  API + Billing  │  │  deployment     │             │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘             │
│           │                    │                    │                       │
│           └────────────────────┼────────────────────┘                       │
│                                │                                            │
│                                ▼                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    CORE LIBRARY (Phases 0-4)                        │   │
│  │                                                                     │   │
│  │  ┌─────────────┐  ┌──────────────────┐  ┌───────────────────────┐  │   │
│  │  │ BaoKeyring  │  │ secp256k1 Plugin │  │ CLI (banhbaoring)     │  │   │
│  │  │ (Go lib)    │  │ (OpenBao plugin) │  │                       │  │   │
│  │  └─────────────┘  └──────────────────┘  └───────────────────────┘  │   │
│  │                                                                     │   │
│  │  Documented in: ARCHITECTURE.md, PLUGIN_DESIGN.md, API_REFERENCE   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Document Index

### Core Library (Already Built - Phases 0-4)

| Document | Description |
|----------|-------------|
| [PRD.md](./PRD.md) | Original product requirements for core library |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Technical architecture & component design |
| [PLUGIN_DESIGN.md](./PLUGIN_DESIGN.md) | OpenBao secp256k1 plugin specification |
| [API_REFERENCE.md](./API_REFERENCE.md) | Plugin API endpoints reference |
| [INTEGRATION.md](./INTEGRATION.md) | Celestia/Cosmos integration guide |
| [MIGRATION.md](./MIGRATION.md) | Key migration procedures |
| [DEPLOYMENT.md](./DEPLOYMENT.md) | Kubernetes deployment guide |

### SaaS Platform (New - Phases 5-7)

| Document | Description | Status |
|----------|-------------|--------|
| [PRD_CONTROL_PLANE.md](./PRD_CONTROL_PLANE.md) | Multi-tenant API, billing (Stripe + crypto) | 📝 PRD Ready |
| [PRD_DASHBOARD.md](./PRD_DASHBOARD.md) | Web dashboard, UX, 5-min onboarding | 📝 PRD Ready |
| [PRD_OPERATOR.md](./PRD_OPERATOR.md) | K8s operator for one-command deployment | 📝 PRD Ready |

---

## Platform Layers

### Layer 1: Core Library ✅ (Phases 0-4)
The foundation - a Go library implementing `keyring.Keyring` interface with OpenBao backend.

**Key Features:**
- `BaoKeyring` - Drop-in replacement for Cosmos SDK keyrings
- `secp256k1` OpenBao plugin - Native signing inside vault
- Key migration tools - Import/export between keyrings
- CLI tool - Command-line key management

**Status:** Implementation complete (17 agents across 4 phases)

---

### Layer 2: Control Plane API 📝 (Phase 5)
Multi-tenant backend API that wraps the core library.

**Key Features:**
- Multi-tenant isolation (organizations, namespaces)
- Authentication (OAuth, API keys, wallet connect)
- Role-based access control (RBAC)
- Billing (Stripe + stablecoin payments)
- Audit logging & compliance
- Webhooks

**Billing:**
- Stripe integration (cards, ACH, SEPA)
- Crypto payments (USDC, USDT, TIA)

**Timeline:** ~9 weeks

---

### Layer 3: Web Dashboard 📝 (Phase 6)
User-facing web application for key management.

**Key Features:**
- 5-minute onboarding flow
- Key management UI (create, view, sign test)
- Usage analytics & audit log viewer
- Team management
- Billing & crypto payments

**USPs:**
- 🚀 5-minute signup to first signature
- 🔐 HSM-level security made simple
- 🌐 Web3 native (wallet login, crypto payments)
- 📊 Real-time monitoring

**Timeline:** ~7 weeks

---

### Layer 4: Kubernetes Operator 📝 (Phase 7)
One-command deployment of the entire stack.

**Key Features:**
- Single CRD deploys everything
- Auto-unseal (AWS KMS, GCP KMS, Azure KV)
- Built-in PostgreSQL & Redis
- Monitoring stack (Prometheus, Grafana)
- Automated backups to S3/GCS
- Tenant provisioning

**One-Command Deploy:**
```yaml
apiVersion: banhbaoring.io/v1
kind: BanhBaoRingCluster
metadata:
  name: production
spec:
  domain: keys.mycompany.com
  openbao:
    replicas: 3
    autoUnseal:
      provider: awskms
      keyId: alias/banhbaoring-unseal
```

**Timeline:** ~8 weeks

---

## Timeline Summary

| Phase | Component | Agents | Duration |
|-------|-----------|--------|----------|
| 0-4 | Core Library | 18 | ✅ Complete |
| 5 | Control Plane API | ~6 | 9 weeks |
| 6 | Web Dashboard | ~6 | 7 weeks |
| 7 | K8s Operator | ~4 | 8 weeks |
| **Total** | **Full Platform** | **~34** | **~24 weeks** |

---

## Target Users

> **🎯 Maximum Focus:** We serve exactly two user types. No validators. No dApp builders. Just rollups.

| User Segment          | The Pain                                              | BanhBaoRing Solution                    |
|-----------------------|-------------------------------------------------------|----------------------------------------|
| **Rollup Developers** | Sequencer keys in config files, no rotation, no audit | HSM-level security, zero-downtime rotation |
| **Rollup Operators**  | Bridge keys on single server, compliance nightmares   | Full audit trail, disaster recovery    |

### The Pain We Solve

Rollup teams know this pain:
- 🔓 Sequencer keys stored in plaintext `.env` files
- 💀 Bridge operator keys on a single server = single point of failure
- ⏰ Manual key rotation during incidents = downtime
- 📋 No audit trail of who signed what when
- 😰 Compliance asks "where are your keys?" and you point to a config file
- ⚡ **Parallel workers with fee grants** need concurrent signing from multiple accounts

**BanhBaoRing:** One API call to sign. Keys never leave the vault. Full audit trail. Sleep at night.

### Parallel Worker Support (Critical for Celestia)

> **Reference:** [Celestia Client Parallel Workers](https://github.com/celestiaorg/celestia-node/blob/main/api/client/readme.md)

Celestia rollups use parallel blob submission with multiple worker accounts:

```go
cfg := client.Config{
    SubmitConfig: client.SubmitConfig{
        TxWorkerAccounts: 4,  // 4 parallel workers
    },
}
```

**BanhBaoRing supports:**
- ⚡ Concurrent signing from multiple worker keys
- 📦 Batch key creation (create 4 workers in one call)
- 🚀 No head-of-line blocking (100+ signs/second)
- 🔧 Easy worker key management in dashboard

---

## Pricing Model

| Plan | Monthly | Keys | Signatures | Use Case |
|------|---------|------|------------|----------|
| **Free** | $0 | 3 | 10K/mo | Testing, small projects |
| **Pro** | $49 | 25 | 500K/mo | Production validators |
| **Enterprise** | Custom | Unlimited | Unlimited | Large teams, SLA |

**Payment Options:**
- 💳 Credit/debit cards (Stripe)
- 🏦 Bank transfer (ACH, SEPA)
- 🪙 Crypto (USDC, USDT, TIA)

---

## Next Steps

1. **Review PRDs** - Control Plane, Dashboard, Operator
2. **Prioritize** - Which layer to build first?
3. **Create Implementation Docs** - Break down into agent tasks
4. **Build** - Execute with parallel agents

---

## Quick Links

- **Core Library Implementation:** [`../implementation/README.md`](../implementation/README.md)
- **Repository:** `github.com/Bidon15/banhbaoring`

