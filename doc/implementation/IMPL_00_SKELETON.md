# Implementation Guide: Project Skeleton

**Agent ID:** 00  
**Component:** Project Scaffolding & Directory Structure  
**Parallelizable:** ❌ No - Must complete FIRST before all other agents  
**Priority:** HIGHEST - Blocking all other agents

---

## 1. Overview

This agent creates the complete project skeleton including:

- Directory structure
- Go module files (`go.mod`)
- Interface definitions (contracts for other agents)
- Placeholder files with TODO markers

### 1.1 Required Skills

| Skill       | Level        | Description                   |
| ----------- | ------------ | ----------------------------- |
| **Go**      | Intermediate | Modules, packages, interfaces |
| **Project** | Advanced     | Go project layout conventions |

### 1.2 Execution Order

```
Agent 00 (Skeleton) ──────► ALL OTHER AGENTS
         │
         └── Must complete first!
```

---

## 2. Directory Structure to Create

```
banhbaoring/
├── go.mod                          # Main module
├── go.sum
├── README.md                       # Quick start
├── doc/                            # Documentation (exists)
│
├── types.go                        # → Agent 01A
├── errors.go                       # → Agent 01B
├── bao_client.go                   # → Agent 01C
├── bao_store.go                    # → Agent 02A, 02B
├── bao_keyring.go                  # → Agent 04A, 04B, 04C
│
├── migration/
│   ├── doc.go                      # Package doc
│   ├── types.go                    # → Agent 05A
│   ├── import.go                   # → Agent 05A
│   └── export.go                   # → Agent 05B
│
├── cmd/
│   └── banhbao/
│       ├── main.go                 # → Agent 05C
│       ├── keys.go                 # → Agent 05C
│       └── migrate.go              # → Agent 05D
│
├── plugin/
│   ├── go.mod                      # Separate module for plugin
│   ├── go.sum
│   └── cmd/
│       └── plugin/
│           └── main.go             # → Agent 03A
│   └── secp256k1/
│       ├── doc.go                  # Package doc
│       ├── backend.go              # → Agent 03A
│       ├── types.go                # → Agent 03B
│       ├── path_keys.go            # → Agent 03B
│       ├── path_sign.go            # → Agent 03C
│       ├── path_verify.go          # → Agent 03C
│       ├── path_import.go          # → Agent 03D
│       ├── path_export.go          # → Agent 03D
│       └── crypto.go               # → Agent 03E
│
└── example/
    └── main.go                     # Usage example
```

---

## 3. Files to Create

### 3.1 go.mod (Main Module)

**CRITICAL:** Must use Celestia's forks, not upstream cosmos-sdk!

```go
module github.com/Bidon15/banhbaoring

go 1.22

require (
    github.com/celestiaorg/celestia-app/v6 v6.3.0
    github.com/cosmos/cosmos-sdk v0.50.13
    github.com/spf13/cobra v1.8.0
)

// CRITICAL: Celestia uses forked versions of cosmos-sdk and dependencies
// These replace directives are copied from celestia-app v6.3.0's go.mod
// Do NOT remove these replace directives!
replace (
    // Celestia's cosmos-sdk fork with keyring interface modifications
    cosmossdk.io/api => github.com/celestiaorg/cosmos-sdk/api v0.7.6
    // f48fea92e627 commit coincides with the v0.51.8 cosmos-sdk release
    cosmossdk.io/log => github.com/celestiaorg/cosmos-sdk/log v1.1.1-0.20251116153902-f48fea92e627
    cosmossdk.io/x/upgrade => github.com/celestiaorg/cosmos-sdk/x/upgrade v0.2.0

    // Use Celestia's cosmos-sdk fork - required for keyring interface compatibility
    github.com/cosmos/cosmos-sdk => github.com/celestiaorg/cosmos-sdk v0.51.8

    // Use Celestia's cometbft fork (celestia-core)
    github.com/cometbft/cometbft => github.com/celestiaorg/celestia-core v0.39.16

    // Use Celestia's IBC fork
    github.com/cosmos/ibc-go/v8 => github.com/celestiaorg/ibc-go/v8 v8.7.2

    // Use ledger-cosmos-go v0.16.0 because v0.15.0 causes "hidapi: unknown failure"
    github.com/cosmos/ledger-cosmos-go => github.com/cosmos/ledger-cosmos-go v0.16.0

    // LevelDB canonical version
    github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7

    // celestia-core(v0.34.x): used for multiplexing abci v1 requests
    github.com/tendermint/tendermint => github.com/celestiaorg/celestia-core v1.55.0-tm-v0.34.35
)
```

**Why these replaces matter:**

- `celestiaorg/cosmos-sdk v0.51.8`: Contains Celestia-specific keyring modifications
- `celestiaorg/celestia-core v0.39.16`: Celestia's cometbft fork
- `celestiaorg/celestia-core v1.55.0-tm-v0.34.35`: Celestia's tendermint fork for ABCI v1
- The keyring interface MUST match Celestia's expectations

### 3.2 plugin/go.mod (Plugin Module)

The plugin runs inside OpenBao (server-side) and does NOT need Celestia dependencies.
It only needs btcec for secp256k1 crypto and the OpenBao SDK v2.

```go
module github.com/Bidon15/banhbaoring/plugin

go 1.22

require (
    github.com/btcsuite/btcd/btcec/v2 v2.3.2
    github.com/openbao/openbao/sdk/v2 v2.5.0
    golang.org/x/crypto v0.21.0
)

// Plugin is self-contained - no Celestia dependencies needed
// Crypto operations happen inside OpenBao using btcec
```

**Note:** The OpenBao SDK uses v2 module path (`github.com/openbao/openbao/sdk/v2`)

### 3.3 types.go (Interface Stubs)

```go
// Package banhbaoring provides a Cosmos SDK keyring implementation
// backed by OpenBao for secure secp256k1 signing.
//
// IMPORTANT: This package uses Celestia's fork of cosmos-sdk.
// The keyring interface comes from github.com/cosmos/cosmos-sdk/crypto/keyring
// but is replaced by github.com/celestiaorg/cosmos-sdk via go.mod replace.
package banhbaoring

import (
    "crypto/tls"
    "time"
)

// TODO(01A): Implement all types below

// Algorithm constants
const (
    AlgorithmSecp256k1 = "secp256k1"
    DefaultSecp256k1Path = "secp256k1"
    DefaultHTTPTimeout   = 30 * time.Second
    DefaultStoreVersion  = 1
)

// Source constants
const (
    SourceGenerated = "generated"
    SourceImported  = "imported"
    SourceSynced    = "synced"
)

// Config holds configuration for BaoKeyring.
type Config struct {
    BaoAddr       string
    BaoToken      string
    BaoNamespace  string
    Secp256k1Path string
    StorePath     string
    HTTPTimeout   time.Duration
    TLSConfig     *tls.Config
    SkipTLSVerify bool
}

// KeyMetadata contains locally stored key information.
type KeyMetadata struct {
    UID         string    `json:"uid"`
    Name        string    `json:"name"`
    PubKeyBytes []byte    `json:"pub_key"`
    PubKeyType  string    `json:"pub_key_type"`
    Address     string    `json:"address"`
    BaoKeyPath  string    `json:"bao_key_path"`
    Algorithm   string    `json:"algorithm"`
    Exportable  bool      `json:"exportable"`
    CreatedAt   time.Time `json:"created_at"`
    Source      string    `json:"source"`
}

// KeyInfo represents public key information from OpenBao.
type KeyInfo struct {
    Name       string    `json:"name"`
    PublicKey  string    `json:"public_key"`
    Address    string    `json:"address"`
    Exportable bool      `json:"exportable"`
    CreatedAt  time.Time `json:"created_at"`
}

// KeyOptions configures key creation.
type KeyOptions struct {
    Exportable bool
}

// SignRequest for OpenBao signing.
type SignRequest struct {
    Input        string `json:"input"`
    Prehashed    bool   `json:"prehashed"`
    HashAlgo     string `json:"hash_algorithm,omitempty"`
    OutputFormat string `json:"output_format,omitempty"`
}

// SignResponse from OpenBao signing.
type SignResponse struct {
    Signature  string `json:"signature"`
    PublicKey  string `json:"public_key"`
    KeyVersion int    `json:"key_version"`
}

// StoreData is the persisted store format.
type StoreData struct {
    Version int                     `json:"version"`
    Keys    map[string]*KeyMetadata `json:"keys"`
}
```

### 3.4 errors.go (Error Stubs)

```go
package banhbaoring

import "errors"

// TODO(01B): Implement all errors below

// Sentinel errors
var (
    ErrMissingBaoAddr   = errors.New("banhbaoring: BaoAddr is required")
    ErrMissingBaoToken  = errors.New("banhbaoring: BaoToken is required")
    ErrMissingStorePath = errors.New("banhbaoring: StorePath is required")

    ErrKeyNotFound      = errors.New("banhbaoring: key not found")
    ErrKeyExists        = errors.New("banhbaoring: key already exists")
    ErrKeyNotExportable = errors.New("banhbaoring: key is not exportable")

    ErrBaoConnection    = errors.New("banhbaoring: connection failed")
    ErrBaoAuth          = errors.New("banhbaoring: authentication failed")
    ErrBaoSealed        = errors.New("banhbaoring: OpenBao is sealed")
    ErrBaoUnavailable   = errors.New("banhbaoring: OpenBao unavailable")

    ErrSigningFailed    = errors.New("banhbaoring: signing failed")
    ErrInvalidSignature = errors.New("banhbaoring: invalid signature")
    ErrUnsupportedAlgo  = errors.New("banhbaoring: unsupported algorithm")
    ErrStorePersist     = errors.New("banhbaoring: persist failed")
    ErrStoreCorrupted   = errors.New("banhbaoring: store corrupted")
)
```

### 3.5 bao_client.go (Interface Stub)

```go
package banhbaoring

import "context"

// TODO(01C): Implement BaoClient

// BaoClient handles HTTP communication with OpenBao.
type BaoClient struct {
    // TODO: Add fields
}

// NewBaoClient creates a new client.
func NewBaoClient(cfg Config) (*BaoClient, error) {
    panic("TODO(01C): implement NewBaoClient")
}

// CreateKey creates a new secp256k1 key.
func (c *BaoClient) CreateKey(ctx context.Context, name string, opts KeyOptions) (*KeyInfo, error) {
    panic("TODO(01C): implement CreateKey")
}

// GetKey retrieves key info.
func (c *BaoClient) GetKey(ctx context.Context, name string) (*KeyInfo, error) {
    panic("TODO(01C): implement GetKey")
}

// ListKeys lists all keys.
func (c *BaoClient) ListKeys(ctx context.Context) ([]string, error) {
    panic("TODO(01C): implement ListKeys")
}

// DeleteKey deletes a key.
func (c *BaoClient) DeleteKey(ctx context.Context, name string) error {
    panic("TODO(01C): implement DeleteKey")
}

// Sign signs data.
func (c *BaoClient) Sign(ctx context.Context, keyName string, data []byte, prehashed bool) ([]byte, error) {
    panic("TODO(01C): implement Sign")
}

// Health checks OpenBao status.
func (c *BaoClient) Health(ctx context.Context) error {
    panic("TODO(01C): implement Health")
}
```

### 3.6 bao_store.go (Interface Stub)

```go
package banhbaoring

// TODO(02A, 02B): Implement BaoStore

// BaoStore manages local key metadata.
type BaoStore struct {
    // TODO: Add fields
}

// NewBaoStore creates or opens a store.
func NewBaoStore(path string) (*BaoStore, error) {
    panic("TODO(02B): implement NewBaoStore")
}

// Save stores key metadata.
func (s *BaoStore) Save(meta *KeyMetadata) error {
    panic("TODO(02A): implement Save")
}

// Get retrieves metadata by UID.
func (s *BaoStore) Get(uid string) (*KeyMetadata, error) {
    panic("TODO(02A): implement Get")
}

// GetByAddress retrieves metadata by address.
func (s *BaoStore) GetByAddress(address string) (*KeyMetadata, error) {
    panic("TODO(02A): implement GetByAddress")
}

// List returns all metadata.
func (s *BaoStore) List() ([]*KeyMetadata, error) {
    panic("TODO(02A): implement List")
}

// Delete removes metadata.
func (s *BaoStore) Delete(uid string) error {
    panic("TODO(02A): implement Delete")
}

// Rename changes the UID.
func (s *BaoStore) Rename(oldUID, newUID string) error {
    panic("TODO(02A): implement Rename")
}

// Has checks existence.
func (s *BaoStore) Has(uid string) bool {
    panic("TODO(02A): implement Has")
}

// Count returns key count.
func (s *BaoStore) Count() int {
    panic("TODO(02A): implement Count")
}

// Sync flushes to disk.
func (s *BaoStore) Sync() error {
    panic("TODO(02B): implement Sync")
}

// Close releases resources.
func (s *BaoStore) Close() error {
    panic("TODO(02B): implement Close")
}
```

### 3.7 bao_keyring.go (Interface Stub)

```go
package banhbaoring

import (
    "context"

    // These imports use upstream paths but are REPLACED by Celestia forks via go.mod
    // The actual code comes from github.com/celestiaorg/cosmos-sdk
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
    cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// TODO(04A, 04B, 04C): Implement BaoKeyring

const BackendType = "openbao"

// BaoKeyring implements keyring.Keyring using OpenBao.
type BaoKeyring struct {
    client *BaoClient
    store  *BaoStore
}

// Verify interface compliance
var _ keyring.Keyring = (*BaoKeyring)(nil)

// New creates a BaoKeyring.
func New(ctx context.Context, cfg Config) (*BaoKeyring, error) {
    panic("TODO(04A): implement New")
}

// Backend returns the backend type.
func (k *BaoKeyring) Backend() string {
    return BackendType
}

// Key retrieves a key by UID.
func (k *BaoKeyring) Key(uid string) (*keyring.Record, error) {
    panic("TODO(04B): implement Key")
}

// KeyByAddress retrieves a key by address.
func (k *BaoKeyring) KeyByAddress(address sdk.Address) (*keyring.Record, error) {
    panic("TODO(04B): implement KeyByAddress")
}

// List returns all keys.
func (k *BaoKeyring) List() ([]*keyring.Record, error) {
    panic("TODO(04B): implement List")
}

// NewAccount creates a key in OpenBao.
func (k *BaoKeyring) NewAccount(uid, mnemonic, bip39Passphrase, hdPath string, algo keyring.SignatureAlgo) (*keyring.Record, error) {
    panic("TODO(04B): implement NewAccount")
}

// Sign signs message bytes.
func (k *BaoKeyring) Sign(uid string, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
    panic("TODO(04C): implement Sign")
}

// SignByAddress signs using the key at address.
func (k *BaoKeyring) SignByAddress(address sdk.Address, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
    panic("TODO(04C): implement SignByAddress")
}

// Delete removes a key.
func (k *BaoKeyring) Delete(uid string) error {
    panic("TODO(04B): implement Delete")
}

// Rename changes the UID.
func (k *BaoKeyring) Rename(fromUID, toUID string) error {
    panic("TODO(04B): implement Rename")
}

// Close releases resources.
func (k *BaoKeyring) Close() error {
    panic("TODO(04A): implement Close")
}

// --- Extended methods for migration ---

// GetMetadata returns raw metadata.
func (k *BaoKeyring) GetMetadata(uid string) (*KeyMetadata, error) {
    panic("TODO(04B): implement GetMetadata")
}

// NewAccountWithOptions creates a key with options.
func (k *BaoKeyring) NewAccountWithOptions(uid string, opts KeyOptions) (*keyring.Record, error) {
    panic("TODO(04B): implement NewAccountWithOptions")
}

// ImportKey imports a wrapped key.
func (k *BaoKeyring) ImportKey(uid string, wrappedKey []byte, exportable bool) (*keyring.Record, error) {
    panic("TODO(04B): implement ImportKey")
}

// ExportKey exports a key (if exportable).
func (k *BaoKeyring) ExportKey(uid string) ([]byte, error) {
    panic("TODO(04B): implement ExportKey")
}

// GetWrappingKey gets the RSA wrapping key.
func (k *BaoKeyring) GetWrappingKey() ([]byte, error) {
    panic("TODO(04B): implement GetWrappingKey")
}
```

### 3.8 migration/doc.go

```go
// Package migration provides key import/export between keyrings.
package migration
```

### 3.9 migration/types.go

```go
package migration

import (
    "github.com/Bidon15/banhbaoring"
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
)

// TODO(05A): Implement migration types

// ImportConfig configures key import.
type ImportConfig struct {
    SourceKeyring     keyring.Keyring
    DestKeyring       *banhbaoring.BaoKeyring
    KeyName           string
    NewKeyName        string
    DeleteAfterImport bool
    Exportable        bool
    VerifyAfterImport bool
}

// ImportResult contains import result.
type ImportResult struct {
    KeyName    string
    Address    string
    PubKey     []byte
    BaoKeyPath string
    Verified   bool
}

// ExportConfig configures key export.
type ExportConfig struct {
    SourceKeyring     *banhbaoring.BaoKeyring
    DestKeyring       keyring.Keyring
    KeyName           string
    NewKeyName        string
    DeleteAfterExport bool
    VerifyAfterExport bool
    Confirmed         bool
}

// ExportResult contains export result.
type ExportResult struct {
    KeyName  string
    Address  string
    DestPath string
    Verified bool
}

// BatchImportConfig for multiple keys.
type BatchImportConfig struct {
    SourceKeyring     keyring.Keyring
    DestKeyring       *banhbaoring.BaoKeyring
    KeyNames          []string
    DeleteAfterImport bool
    Exportable        bool
    VerifyAfterImport bool
}

// BatchImportResult contains batch results.
type BatchImportResult struct {
    Successful []ImportResult
    Failed     []ImportError
}

// ImportError for failed imports.
type ImportError struct {
    KeyName string
    Error   error
}
```

### 3.10 migration/import.go

```go
package migration

import "context"

// TODO(05A): Implement Import functions

// Import migrates a key from local to OpenBao.
func Import(ctx context.Context, cfg ImportConfig) (*ImportResult, error) {
    panic("TODO(05A): implement Import")
}

// BatchImport imports multiple keys.
func BatchImport(ctx context.Context, cfg BatchImportConfig) (*BatchImportResult, error) {
    panic("TODO(05A): implement BatchImport")
}
```

### 3.11 migration/export.go

```go
package migration

import "context"

// TODO(05B): Implement Export functions

// Export migrates a key from OpenBao to local.
func Export(ctx context.Context, cfg ExportConfig) (*ExportResult, error) {
    panic("TODO(05B): implement Export")
}

// SecurityWarning returns export warning text.
func SecurityWarning(keyName, address, destPath string) string {
    panic("TODO(05B): implement SecurityWarning")
}
```

### 3.12 cmd/banhbao/main.go

```go
package main

import (
    "fmt"
    "os"
)

// TODO(05C): Implement CLI

func main() {
    fmt.Println("TODO(05C): implement CLI")
    os.Exit(1)
}
```

### 3.13 plugin/secp256k1/doc.go

```go
// Package secp256k1 implements an OpenBao secrets engine for secp256k1 keys.
package secp256k1
```

### 3.14 plugin/secp256k1/backend.go

```go
package secp256k1

import (
    "context"

    "github.com/openbao/openbao/sdk/v2/logical"
)

// TODO(03A): Implement backend

// Factory creates a new backend.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
    panic("TODO(03A): implement Factory")
}
```

### 3.15 plugin/secp256k1/types.go

```go
package secp256k1

import "time"

// TODO(03B): Implement types

type keyEntry struct {
    PrivateKey  []byte    `json:"private_key"`
    PublicKey   []byte    `json:"public_key"`
    Exportable  bool      `json:"exportable"`
    CreatedAt   time.Time `json:"created_at"`
    Imported    bool      `json:"imported"`
}
```

### 3.16 plugin/secp256k1/crypto.go

```go
package secp256k1

// TODO(03E): Implement crypto helpers

func hashSHA256(data []byte) []byte {
    panic("TODO(03E): implement hashSHA256")
}

func deriveCosmosAddress(pubKey []byte) []byte {
    panic("TODO(03E): implement deriveCosmosAddress")
}

func secureZero(b []byte) {
    panic("TODO(03E): implement secureZero")
}
```

### 3.17 plugin/cmd/plugin/main.go

```go
package main

import (
    "fmt"
    "os"
)

// TODO(03A): Implement plugin entrypoint

func main() {
    fmt.Println("TODO(03A): implement plugin")
    os.Exit(1)
}
```

---

## 4. Execution Script

Create this as `scripts/scaffold.sh`:

```bash
#!/bin/bash
set -e

ROOT=$(pwd)

echo "Creating directory structure..."

# Main directories
mkdir -p migration
mkdir -p cmd/banhbao
mkdir -p plugin/cmd/plugin
mkdir -p plugin/secp256k1
mkdir -p example

echo "Creating placeholder files..."

# Touch all files (agents will fill them)
touch types.go errors.go bao_client.go bao_store.go bao_keyring.go
touch migration/doc.go migration/types.go migration/import.go migration/export.go
touch cmd/banhbao/main.go cmd/banhbao/keys.go cmd/banhbao/migrate.go
touch plugin/secp256k1/doc.go plugin/secp256k1/backend.go plugin/secp256k1/types.go
touch plugin/secp256k1/path_keys.go plugin/secp256k1/path_sign.go
touch plugin/secp256k1/path_verify.go plugin/secp256k1/path_import.go
touch plugin/secp256k1/path_export.go plugin/secp256k1/crypto.go
touch plugin/cmd/plugin/main.go
touch example/main.go

echo "Creating go.mod files..."

# Main go.mod
cat > go.mod << 'EOF'
module github.com/Bidon15/banhbaoring

go 1.21

require (
    github.com/cosmos/cosmos-sdk v0.50.6
    github.com/spf13/cobra v1.8.0
)
EOF

# Plugin go.mod
cat > plugin/go.mod << 'EOF'
module github.com/Bidon15/banhbaoring/plugin

go 1.21

require (
    github.com/btcsuite/btcd/btcec/v2 v2.3.2
    github.com/openbao/openbao/sdk v0.11.0
    golang.org/x/crypto v0.21.0
)
EOF

echo "Scaffold complete!"
echo ""
echo "Files created:"
find . -name "*.go" -o -name "go.mod" | grep -v doc/ | sort
```

---

## 5. Deliverables Checklist

- [ ] All directories created
- [ ] `go.mod` for main module
- [ ] `plugin/go.mod` for plugin module
- [ ] All `.go` files created with stubs
- [ ] Each stub has `TODO(AGENT_ID)` markers
- [ ] `go build ./...` compiles (panics on use)
- [ ] No import cycles

---

## 6. Success Criteria

After Agent 00 completes:

```bash
# Directory structure exists
ls -la types.go errors.go bao_client.go bao_store.go bao_keyring.go
ls -la migration/
ls -la cmd/banhbao/
ls -la plugin/secp256k1/

# Go modules work
go mod tidy
cd plugin && go mod tidy

# Code compiles (with panics)
go build ./...
cd plugin && go build ./...
```

---

## 7. Agent Mapping

| File                              | Implementing Agent |
| --------------------------------- | ------------------ |
| `types.go`                        | 01A                |
| `errors.go`                       | 01B                |
| `bao_client.go`                   | 01C                |
| `bao_store.go`                    | 02A, 02B           |
| `bao_keyring.go`                  | 04A, 04B, 04C      |
| `migration/types.go`              | 05A                |
| `migration/import.go`             | 05A                |
| `migration/export.go`             | 05B                |
| `cmd/banhbao/main.go`             | 05C                |
| `cmd/banhbao/keys.go`             | 05C                |
| `cmd/banhbao/migrate.go`          | 05D                |
| `plugin/secp256k1/backend.go`     | 03A                |
| `plugin/secp256k1/types.go`       | 03B                |
| `plugin/secp256k1/path_keys.go`   | 03B                |
| `plugin/secp256k1/path_sign.go`   | 03C                |
| `plugin/secp256k1/path_verify.go` | 03C                |
| `plugin/secp256k1/path_import.go` | 03D                |
| `plugin/secp256k1/path_export.go` | 03D                |
| `plugin/secp256k1/crypto.go`      | 03E                |
| `plugin/cmd/plugin/main.go`       | 03A                |
