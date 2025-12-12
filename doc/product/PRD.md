# Product Requirements Document: OpenBao Keyring Backend for Celestia

## 1. Overview

This document defines the requirements for implementing a custom Cosmos-SDK `keyring.Keyring` backend that leverages OpenBao (the open-source fork of HashiCorp Vault) Transit engine for secure `secp256k1` signing operations. The keyring integrates with Celestia's Go client for transaction signing and broadcasting without storing private keys locally.

**Product Name:** POPSigner (formerly BanhBaoRing)

POPSigner is Point-of-Presence signing infrastructure—a distributed signing layer designed to live inline with execution, not behind an API queue.

### Minimum Version Requirements

| Package | Minimum Version |
|---------|----------------|
| **celestia-app** | v6.4.0 |
| **celestia-node** | v0.28.4 |

### 1.1 Target Users

| User Type | Use Case |
| --------- | -------- |
| **Rollup Teams** | Sequencer signing, prover operations, bridge operators |
| **Execution Bots** | Market makers, arbitrage, MEV operations |
| **Infrastructure Teams** | Backend services requiring inline signing |

### 1.2 Problem Statement

Teams running execution infrastructure face architectural challenges:

- Key material exposure on disk
- Risk of key extraction from memory
- Signing infrastructure distant from execution
- No centralized audit logging
- Vendor lock-in with proprietary solutions

### 1.3 Solution

Implement Point-of-Presence signing infrastructure that:

- Deploys inline with execution—same region, same rack
- Delegates all cryptographic operations to OpenBao
- Stores only public metadata locally
- Provides full `keyring.Keyring` interface compatibility
- Guarantees exit—keys exportable by default
- Supports worker-native architecture for parallel workloads

### 1.4 Core Principles

| Principle | Description |
|-----------|-------------|
| **Inline Signing** | Signing happens on the execution path, not behind a queue |
| **Sovereignty by Default** | Keys are remote, but you control them. Export anytime. Exit anytime. |
| **Neutral Anchor** | Recovery data anchored to neutral data availability |

### 1.5 Parallel Worker Support

POPSigner supports worker-native architecture for burst workloads:

```go
// Create signing workers
keys, _ := client.Keys.CreateBatch(ctx, CreateBatchRequest{
    Prefix: "blob-worker",
    Count:  4,
})

// Sign in parallel—no blocking
results, _ := client.Sign.SignBatch(ctx, BatchSignRequest{...})
```

**POPSigner MUST support:**
- Concurrent `Sign()` calls from multiple goroutines
- Thread-safe access to multiple keys simultaneously
- No head-of-line blocking between different key operations
- Worker-native parallel signing

---

## 2. Objectives

| Objective | Description |
| --------- | ----------- |
| **Placement** | Deploy inline with execution—same region as your systems |
| **Security** | Private keys never leave OpenBao; no local key storage |
| **Sovereignty** | Exit guarantee—keys exportable by default |
| **Compatibility** | Full `keyring.Keyring` interface compliance |
| **Integration** | Works with Celestia's `tx.Factory`, signing, and broadcasting |
| **Modularity** | Clean, reusable Go module for Celestia applications |
| **Auditability** | Leverage OpenBao's audit logging for all signing operations |

---

## 3. Functional Requirements

### 3.1 Keyring Interface Compliance

The backend **MUST** implement the `keyring.Keyring` interface from Celestia's cosmos-sdk fork. Required methods:

| Method | Description | Implementation |
| ------ | ----------- | -------------- |
| `Key(uid string) (*Record, error)` | Retrieve key metadata by name | Read from local metadata store |
| `List() ([]*Record, error)` | List all available keys | Enumerate local metadata store |
| `NewAccount(...)` | Create new key in OpenBao | POST to Transit, store metadata locally |
| `Sign(uid string, msg []byte, signMode) ([]byte, PubKey, error)` | Sign message bytes | Hash → OpenBao sign → DER to compact |
| `SignByAddress(...)` | Sign by address lookup | Resolve address → call Sign() |

### 3.2 Remote Key Management

#### 3.2.1 Key Generation

- **MUST** use OpenBao Transit engine to generate `secp256k1` keys
- **MUST** retrieve and store public key locally after generation
- **MUST** derive and store Cosmos address (bech32) locally

#### 3.2.2 Key Import (Migration TO POPSigner)

Users with existing local keyrings need to migrate their keys to POPSigner infrastructure.

- **MUST** support importing keys from existing Cosmos SDK keyring backends:
  - `file` backend (encrypted keystore)
  - `test` backend (plaintext, for development)
  - `os` backend (system keychain)
- **MUST** securely transfer private key material to OpenBao Transit
- **MUST** never expose or log imported key material during migration
- **MUST** delete local private key copy after successful import (with user confirmation)
- **MUST** update local metadata store after import
- **SHOULD** support batch migration of multiple keys

**Migration Flow:**

1. Read key from source keyring (requires passphrase for encrypted backends)
2. Wrap private key with OpenBao Transit wrapping key
3. Import wrapped key via `/v1/transit/keys/<name>/import`
4. Verify import by signing test data
5. Store metadata locally
6. Optionally delete source key

#### 3.2.3 Key Export (Exit Guarantee)

Users may need to exit POPSigner and export keys to local storage. **This is a first-class feature, not an exception.**

- **MUST** support exporting keys by default (exportable: true is the default)
- **MUST** support export to standard Cosmos SDK keyring formats
- **MUST** log export events for audit purposes
- **SHOULD** support exporting to encrypted file format

**Exit Guarantee Principles:**

- Export is not gated by approval workflows
- If POPSigner is unavailable, recovery data is anchored to neutral DA
- Users can always force exit

**Export Flow:**

1. Read key from OpenBao Transit
2. Decrypt wrapped key material
3. Create new entry in target keyring backend
4. Verify export by signing test data with new keyring
5. Optionally delete key from OpenBao

### 3.3 Remote Signing

The signing flow **MUST** follow this sequence:

1. Receive sign bytes from Cosmos SDK
2. Compute SHA-256 hash of sign bytes
3. Base64 encode the hash
4. Send POST request to OpenBao: `/v1/transit/sign/<key-name>`
5. Receive DER-encoded ECDSA signature
6. Parse DER signature to extract R and S integers
7. Convert to Cosmos compact format: `R || S` (64 bytes, zero-padded)
8. Return signature with public key

### 3.4 Local Metadata Storage

Store only non-sensitive metadata locally:

| Field | Type | Description |
| ----- | ---- | ----------- |
| `Name` | `string` | Key identifier (uid) |
| `PubKey` | `[]byte` | Compressed secp256k1 public key (33 bytes) |
| `Address` | `string` | Bech32-encoded Cosmos address |
| `BaoKeyPath` | `string` | OpenBao Transit key path |
| `Algorithm` | `string` | Always `"secp256k1"` |
| `Exportable` | `bool` | Whether key can be exported (default: true) |
| `CreatedAt` | `time.Time` | Key creation timestamp |
| `Source` | `string` | Origin: `"generated"` or `"imported"` |

### 3.5 Algorithm Support

- **MUST** support `secp256k1` (Cosmos/Tendermint standard)
- **MAY** support additional algorithms via plugin architecture

---

## 4. Non-Functional Requirements

### 4.1 Security

| Requirement | Description |
| ----------- | ----------- |
| **No Local Keys** | Private keys MUST never be stored on disk or in memory |
| **TLS Required** | All OpenBao communication MUST use HTTPS in production |
| **Token Security** | OpenBao tokens MUST be handled securely (env vars, not hardcoded) |
| **Audit Trail** | All signing operations logged by OpenBao |

### 4.2 Placement

| Requirement | Description |
| ----------- | ----------- |
| **Region Selection** | Deploy POPSigner in the same region as execution infrastructure |
| **Inline Execution** | Signing happens on the execution path |
| **Connection pooling** | HTTP client SHOULD reuse connections |
| **Timeout handling** | Configurable request timeouts |

### 4.3 Reliability

| Requirement | Description |
| ----------- | ----------- |
| **Error handling** | Clear error messages for OpenBao failures |
| **Retry logic** | Configurable retry for transient failures |
| **Health checks** | Method to verify OpenBao connectivity |

### 4.4 Code Quality

- Clean, idiomatic Go code
- Comprehensive documentation
- Unit tests with mocked OpenBao responses
- Integration tests against real OpenBao instance

---

## 5. Deliverables

### 5.1 Source Code

**Go Client Library:**

| File | Description |
| ---- | ----------- |
| `bao_keyring.go` | Main `BaoKeyring` struct implementing `keyring.Keyring` |
| `bao_store.go` | `BaoStore` for local metadata persistence |
| `bao_client.go` | `BaoClient` HTTP wrapper for secp256k1 plugin API |
| `migration.go` | Key import/export migration utilities |
| `types.go` | Shared types and constants |
| `errors.go` | Custom error types |

**OpenBao secp256k1 Plugin:**

| File | Description |
| ---- | ----------- |
| `plugin/cmd/plugin/main.go` | Plugin entrypoint |
| `plugin/secp256k1/backend.go` | Backend factory and registration |
| `plugin/secp256k1/path_keys.go` | Key creation, listing, deletion |
| `plugin/secp256k1/path_sign.go` | Signing operations (Cosmos format) |
| `plugin/secp256k1/path_import.go` | Key import from wrapped material |
| `plugin/secp256k1/path_export.go` | Key export (exit guarantee) |
| `plugin/secp256k1/crypto.go` | secp256k1 crypto helpers (btcec) |

### 5.2 CLI Tool

| File | Description |
| ---- | ----------- |
| `cmd/popsigner/*` | CLI tool for key and migration tasks |

The CLI **MUST** support:

1. Key creation, listing, and deletion
2. Import from local Cosmos SDK keyrings
3. Export to local keyring (exit guarantee)
4. Health check for OpenBao connectivity

### 5.3 Example Application

| File | Description |
| ---- | ----------- |
| `example/main.go` | Complete usage example with Celestia client |

The example **MUST** demonstrate:

1. Creating a new key via OpenBao
2. Configuring Celestia client with `BaoKeyring`
3. Signing a blob submission transaction
4. Broadcasting to Celestia network

### 5.4 Documentation

| File | Description |
| ---- | ----------- |
| `doc/PRD.md` | This document |
| `doc/ARCHITECTURE.md` | Technical design and component details |
| `doc/PLUGIN_DESIGN.md` | secp256k1 plugin design and implementation |
| `doc/API_REFERENCE.md` | OpenBao plugin API reference |
| `doc/INTEGRATION.md` | Celestia integration guide |
| `doc/MIGRATION.md` | Key migration guide (import/export) |
| `doc/DEPLOYMENT.md` | Kubernetes deployment and operations |
| `README.md` | Quick start and usage instructions |

---

## 6. Constraints

### 6.1 Technical Constraints

| Constraint | Description |
| ---------- | ----------- |
| **OpenBao Only** | Use OpenBao API paths; no HashiCorp Vault BSL dependencies |
| **Celestia Versions** | celestia-app ≥ v6.4.0, celestia-node ≥ v0.28.4 |
| **Go 1.21+** | Modern Go version for generics and improved error handling |
| **Cosmos SDK** | Compatible with cosmos-sdk keyring interface version used by Celestia |
| **Kubernetes** | OpenBao deployment requires Kubernetes 1.25+ cluster |

### 6.2 Signature Format

All signatures **MUST** match Cosmos expected format:

- 64 bytes total: `R (32 bytes) || S (32 bytes)`
- R and S are big-endian, zero-padded to 32 bytes
- Low-S normalization (S ≤ N/2) for signature malleability prevention

### 6.3 OpenBao Requirements

| Requirement | Details |
| ----------- | ------- |
| secp256k1 plugin | Custom plugin providing native secp256k1 signing |
| Plugin path | Mounted at `/secp256k1` |
| Permissions | `create`, `read`, `update` on secp256k1/keys; `update` on sign |

### 6.4 Architecture Decision: Native Plugin

**Decision:** We use a custom OpenBao plugin for secp256k1 rather than a hybrid decrypt-and-sign approach.

| Approach | Security | Key Exposure |
|----------|----------|--------------|
| Hybrid (AWS KMS style) | Good | Key decrypted in app memory |
| **Native Plugin (POPSigner)** | **Excellent** | **Key NEVER leaves OpenBao** |

**Rationale:**
- Private keys remain sealed inside OpenBao at all times
- Only signatures are returned to callers
- Maximum security for production infrastructure

---

## 7. Dependencies

### 7.1 Go Modules

```go
require (
    github.com/celestiaorg/celestia-app/v4 v4.0.0
    github.com/celestiaorg/celestia-node v0.28.4
    github.com/cosmos/cosmos-sdk v0.50.13
    github.com/cosmos/cosmos-sdk/crypto/keyring
)
```

### 7.2 External Services

| Service | Purpose |
| ------- | ------- |
| OpenBao Server | Transit engine for key management and signing |
| Celestia Bridge Node | For blob submission (read operations) |
| Celestia Core Node | For transaction broadcasting |

---

## 8. Success Criteria

### 8.1 Functional Tests

- [ ] Create new key via OpenBao Transit
- [ ] List all keys from metadata store
- [ ] Retrieve single key by name
- [ ] Sign arbitrary bytes and verify signature
- [ ] Sign Cosmos transaction and broadcast successfully
- [ ] Import key from local file keyring to OpenBao
- [ ] Export key from OpenBao to local file keyring (exit guarantee)
- [ ] Batch import multiple keys

### 8.2 Integration Tests

- [ ] Full transaction flow with Celestia testnet (Mocha)
- [ ] Blob submission with OpenBao-signed transaction
- [ ] Key rotation scenario

### 8.3 Security Tests

- [ ] Verify no private key material in logs
- [ ] Verify no private key material on disk
- [ ] Verify TLS enforcement in production mode

---

## 9. Plugin Architecture

POPSigner ships with `secp256k1`. But the plugin architecture is the actual product.

### 9.1 Plugin Principles

- Plugins are chain-agnostic
- Plugins are free
- Plugins don't require approval

### 9.2 Creating Custom Plugins

See [PLUGIN_DESIGN.md](./PLUGIN_DESIGN.md) for implementation details.

---

## 10. Out of Scope (Phase 1)

The following are explicitly **NOT** in scope for Phase 1:

- HSM integration (future enhancement)
- Multi-signature support
- Threshold signatures
- Key sharding
- Other signature algorithms (ed25519, sr25519) - available via plugins

---

## 11. References

- [OpenBao Transit Secrets Engine](https://openbao.org/docs/secrets/transit/)
- [OpenBao Transit API](https://openbao.org/api-docs/secret/transit/)
- [Cosmos SDK Keyring Interface](https://pkg.go.dev/github.com/cosmos/cosmos-sdk/crypto/keyring)
- [Celestia Go Client](https://github.com/celestiaorg/celestia-node/blob/main/api/client/readme.md)
- [celestia-node v0.28.4+](https://github.com/celestiaorg/celestia-node/releases)
- [celestia-app v6.4.0+](https://github.com/celestiaorg/celestia-app/releases)

---

## 12. About the Name

POPSigner (formerly BanhBaoRing) reflects a clearer articulation of what the system is: Point-of-Presence signing infrastructure.

---

## 13. Revision History

| Version | Date | Author | Changes |
| ------- | ---- | ------ | ------- |
| 1.0 | 2025-01-XX | - | Initial PRD |
| 1.1 | 2025-01-XX | - | Added migration, exit guarantee, CLI sections |
| 2.0 | 2025-12-XX | - | Renamed to POPSigner, updated positioning |
