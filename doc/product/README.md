# POPSigner Product Documentation

> **POPSigner** — Point-of-Presence signing infrastructure. Deploy inline with execution.

---

## Product Overview

POPSigner is a distributed signing layer designed to live inline with execution—not behind an API queue. Deploy next to your systems. Keys remain remote. You remain sovereign.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         POPSIGNER PLATFORM                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │  Web Dashboard  │  │ Control Plane   │  │  K8s Operator   │             │
│  │                 │  │                 │  │                 │             │
│  │  Management UI  │  │  Multi-tenant   │  │  One-command    │             │
│  │  Key operations │  │  API + Billing  │  │  deployment     │             │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘             │
│           │                    │                    │                       │
│           └────────────────────┼────────────────────┘                       │
│                                │                                            │
│                                ▼                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    CORE LIBRARY                                     │   │
│  │                                                                     │   │
│  │  ┌─────────────┐  ┌──────────────────┐  ┌───────────────────────┐  │   │
│  │  │ BaoKeyring  │  │ secp256k1 Plugin │  │ CLI (popsigner)       │  │   │
│  │  │ (Go lib)    │  │ (OpenBao plugin) │  │                       │  │   │
│  │  └─────────────┘  └──────────────────┘  └───────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Document Index

### Core Library

| Document | Description |
|----------|-------------|
| [PRD.md](./PRD.md) | Product requirements for core library |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Technical architecture & component design |
| [PLUGIN_DESIGN.md](./PLUGIN_DESIGN.md) | OpenBao secp256k1 plugin specification |
| [API_REFERENCE.md](./API_REFERENCE.md) | Plugin API endpoints reference |
| [INTEGRATION.md](./INTEGRATION.md) | Celestia/Cosmos integration guide |
| [MIGRATION.md](./MIGRATION.md) | Key migration procedures |
| [DEPLOYMENT.md](./DEPLOYMENT.md) | Kubernetes deployment guide |

### Platform Components

| Document | Description |
|----------|-------------|
| [PRD_CONTROL_PLANE.md](./PRD_CONTROL_PLANE.md) | Multi-tenant API, billing |
| [PRD_DASHBOARD.md](./PRD_DASHBOARD.md) | Web dashboard, UX |
| [PRD_OPERATOR.md](./PRD_OPERATOR.md) | K8s operator for deployment |

### Design

| Document | Description |
|----------|-------------|
| [DESIGN_SYSTEM.md](../design/DESIGN_SYSTEM.md) | Brand identity, components, layouts |

---

## Core Principles

### 1. Inline Signing

Signing happens on the execution path, not behind a queue. Your transactions don't wait.

### 2. Sovereignty by Default

Keys are remote, but you control them. Export at any time. Exit at any time. No lock-in, ever.

### 3. Neutral Anchor

Recovery data is anchored to neutral data availability. If POPSigner disappears, you don't.

---

## Platform Layers

### Layer 1: Core Library ✅

The foundation—a Go library implementing `keyring.Keyring` interface with OpenBao backend.

**Key Features:**
- `BaoKeyring` — Drop-in replacement for Cosmos SDK keyrings
- `secp256k1` OpenBao plugin — Native signing inside vault
- Key migration tools — Import/export between keyrings
- CLI tool — Command-line key management

---

### Layer 2: Control Plane API

Multi-tenant backend API that wraps the core library.

**Key Features:**
- Multi-tenant isolation (organizations, namespaces)
- Authentication (OAuth, API keys)
- Role-based access control (RBAC)
- Billing integration
- Audit logging
- Webhooks

---

### Layer 3: Web Dashboard

Management interface for key operations.

**Key Features:**
- Key management UI (create, view, export)
- Usage analytics & audit log viewer
- Team management
- Billing

---

### Layer 4: Kubernetes Operator

One-command deployment of the entire stack.

**Key Features:**
- Single CRD deploys everything
- Auto-unseal (AWS KMS, GCP KMS, Azure KV)
- Built-in PostgreSQL & Redis
- Monitoring stack (Prometheus, Grafana)
- Automated backups

**Deployment Example:**
```yaml
apiVersion: popsigner.com/v1
kind: POPSignerCluster
metadata:
  name: production
spec:
  domain: keys.mycompany.com
  openbao:
    replicas: 3
    autoUnseal:
      provider: awskms
      keyId: alias/popsigner-unseal
```

---

## Target Users

| User Segment | Use Case |
|--------------|----------|
| **Rollup Teams** | Sequencer signing, prover operations |
| **Execution Bots** | Market makers, arbitrage, MEV |
| **Infrastructure Teams** | Backend services requiring signing |

### What We Solve

- **Architectural distance** — Traditional remote signers introduce latency between execution and signing
- **Lock-in** — Proprietary APIs make migration difficult
- **No exit** — Keys trapped in vendor infrastructure
- **Algorithm gaps** — Cloud KMS doesn't support secp256k1

**POPSigner:** Open source. Plugin architecture. Sovereign by default. Exit guaranteed.

---

## Pricing Model

| Plan | Monthly | Description |
|------|---------|-------------|
| **Shared POPSigner** | €49 | Shared POP infrastructure, no SLA |
| **Priority POPSigner** | €499 | Priority lanes, region selection, 99.9% SLA |
| **Dedicated POPSigner** | €19,999 | Region-pinned, CPU isolation, 99.99% SLA |

**Self-host:** Always free. 100% open source.

**Payment Options:**
- Credit/debit cards
- Bank transfer (SEPA)

---

## About the Name

POPSigner (formerly BanhBaoRing) reflects a clearer articulation of what the system is: Point-of-Presence signing infrastructure.

The rename signals maturation from playful internal naming to category-defining infrastructure positioning.

---

## Quick Links

- **Implementation Docs:** [`../implementation/README.md`](../implementation/README.md)
- **Design System:** [`../design/DESIGN_SYSTEM.md`](../design/DESIGN_SYSTEM.md)
- **Repository:** `github.com/Bidon15/banhbaoring`
