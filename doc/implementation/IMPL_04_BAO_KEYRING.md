# Implementation Guide: BaoKeyring

**Agent ID:** 04  
**Component:** Cosmos SDK Keyring Implementation  
**Parallelizable:** ✅ Yes - Uses interfaces from Agent 01 & 02  

---

## 1. Overview

This agent builds the main `BaoKeyring` struct that implements the Cosmos SDK `keyring.Keyring` interface. It orchestrates the `BaoClient` and `BaoStore` to provide seamless integration with Celestia applications.

### 1.1 Required Skills

| Skill | Level | Description |
|-------|-------|-------------|
| **Go** | Advanced | Interfaces, composition |
| **Cosmos SDK** | Advanced | keyring.Keyring interface |
| **Cryptography** | Intermediate | Public key handling, address derivation |
| **Integration** | Advanced | Combining multiple components |

### 1.2 Files to Create

```
banhbaoring/
├── bao_keyring.go       # Main keyring implementation
└── bao_keyring_test.go  # Unit tests
```

### 1.3 Dependencies

- **Agent 01:** `BaoClient`, `Config`, `KeyMetadata`, errors
- **Agent 02:** `BaoStore`

If other agents are not complete, mock their interfaces.

---

## 2. Interface to Implement

The `BaoKeyring` must implement `keyring.Keyring` from Cosmos SDK:

```go
// From github.com/cosmos/cosmos-sdk/crypto/keyring
type Keyring interface {
    Key(uid string) (*Record, error)
    KeyByAddress(address sdk.Address) (*Record, error)
    List() ([]*Record, error)
    
    NewAccount(uid, mnemonic, bip39Passphrase, hdPath string, algo SignatureAlgo) (*Record, error)
    
    Sign(uid string, msg []byte, signMode signing.SignMode) ([]byte, types.PubKey, error)
    SignByAddress(address sdk.Address, msg []byte, signMode signing.SignMode) ([]byte, types.PubKey, error)
    
    Delete(uid string) error
    Rename(fromUID, toUID string) error
    
    // Backend returns the backend type (for us: "openbao")
    Backend() string
}
```

---

## 3. Detailed Specifications

### 3.1 bao_keyring.go

**IMPORTANT:** All cosmos-sdk imports are replaced by Celestia's fork (`celestiaorg/cosmos-sdk`)
via go.mod replace directives. The import paths remain the same but resolve to Celestia's code.

```go
package banhbaoring

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "time"

    // These imports use standard cosmos-sdk paths but are REPLACED
    // by Celestia's fork (github.com/celestiaorg/cosmos-sdk) via go.mod.
    // This is required because Celestia has custom keyring modifications.
    "github.com/cosmos/cosmos-sdk/codec"
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
    "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
    cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

const (
    // BackendType is the keyring backend identifier
    BackendType = "openbao"
)

// BaoKeyring implements keyring.Keyring using OpenBao for signing
type BaoKeyring struct {
    client *BaoClient
    store  *BaoStore
    cdc    codec.Codec
}

// Verify BaoKeyring implements keyring.Keyring
var _ keyring.Keyring = (*BaoKeyring)(nil)

// New creates a new BaoKeyring instance
func New(ctx context.Context, cfg Config) (*BaoKeyring, error) {
    // Create HTTP client
    client, err := NewBaoClient(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create BaoClient: %w", err)
    }
    
    // Verify connectivity
    if err := client.Health(ctx); err != nil {
        return nil, fmt.Errorf("OpenBao health check failed: %w", err)
    }
    
    // Create metadata store
    store, err := NewBaoStore(cfg.StorePath)
    if err != nil {
        return nil, fmt.Errorf("failed to create BaoStore: %w", err)
    }
    
    return &BaoKeyring{
        client: client,
        store:  store,
    }, nil
}

// Backend returns the keyring backend type
func (k *BaoKeyring) Backend() string {
    return BackendType
}

// Key retrieves a key's Record by uid
func (k *BaoKeyring) Key(uid string) (*keyring.Record, error) {
    meta, err := k.store.Get(uid)
    if err != nil {
        return nil, err
    }
    
    return k.metadataToRecord(meta)
}

// KeyByAddress retrieves a key's Record by address
func (k *BaoKeyring) KeyByAddress(address sdk.Address) (*keyring.Record, error) {
    meta, err := k.store.GetByAddress(address.String())
    if err != nil {
        return nil, err
    }
    
    return k.metadataToRecord(meta)
}

// List returns all keys in the keyring
func (k *BaoKeyring) List() ([]*keyring.Record, error) {
    metas, err := k.store.List()
    if err != nil {
        return nil, err
    }
    
    records := make([]*keyring.Record, 0, len(metas))
    for _, meta := range metas {
        record, err := k.metadataToRecord(meta)
        if err != nil {
            continue // Skip invalid entries
        }
        records = append(records, record)
    }
    
    return records, nil
}

// NewAccount creates a new key in OpenBao and stores metadata locally
func (k *BaoKeyring) NewAccount(
    uid, mnemonic, bip39Passphrase, hdPath string,
    algo keyring.SignatureAlgo,
) (*keyring.Record, error) {
    // BaoKeyring generates keys in OpenBao, ignores mnemonic/hdPath
    // Validate algorithm
    if algo != nil && algo.Name() != string(keyring.Secp256k1Type) {
        return nil, fmt.Errorf("%w: only secp256k1 supported", ErrUnsupportedAlgo)
    }
    
    // Check if key already exists locally
    if k.store.Has(uid) {
        return nil, fmt.Errorf("%w: %s", ErrKeyExists, uid)
    }
    
    ctx := context.Background()
    
    // Create key in OpenBao
    keyInfo, err := k.client.CreateKey(ctx, uid, KeyOptions{
        Exportable: false,
    })
    if err != nil {
        return nil, err
    }
    
    // Parse public key
    pubKeyBytes, err := hex.DecodeString(keyInfo.PublicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to decode public key: %w", err)
    }
    
    // Create and save metadata
    meta := &KeyMetadata{
        UID:         uid,
        Name:        uid,
        PubKeyBytes: pubKeyBytes,
        PubKeyType:  "secp256k1",
        Address:     keyInfo.Address,
        BaoKeyPath:  fmt.Sprintf("%s/keys/%s", k.client.secp256k1Path, uid),
        Algorithm:   AlgorithmSecp256k1,
        Exportable:  false,
        CreatedAt:   time.Now().UTC(),
        Source:      "generated",
    }
    
    if err := k.store.Save(meta); err != nil {
        // Attempt cleanup on failure
        _ = k.client.DeleteKey(ctx, uid)
        return nil, err
    }
    
    return k.metadataToRecord(meta)
}

// NewAccountWithOptions creates a key with custom options
func (k *BaoKeyring) NewAccountWithOptions(uid string, opts KeyOptions) (*keyring.Record, error) {
    if k.store.Has(uid) {
        return nil, fmt.Errorf("%w: %s", ErrKeyExists, uid)
    }
    
    ctx := context.Background()
    
    keyInfo, err := k.client.CreateKey(ctx, uid, opts)
    if err != nil {
        return nil, err
    }
    
    pubKeyBytes, err := hex.DecodeString(keyInfo.PublicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to decode public key: %w", err)
    }
    
    meta := &KeyMetadata{
        UID:         uid,
        Name:        uid,
        PubKeyBytes: pubKeyBytes,
        PubKeyType:  "secp256k1",
        Address:     keyInfo.Address,
        BaoKeyPath:  fmt.Sprintf("%s/keys/%s", k.client.secp256k1Path, uid),
        Algorithm:   AlgorithmSecp256k1,
        Exportable:  opts.Exportable,
        CreatedAt:   time.Now().UTC(),
        Source:      "generated",
    }
    
    if err := k.store.Save(meta); err != nil {
        _ = k.client.DeleteKey(ctx, uid)
        return nil, err
    }
    
    return k.metadataToRecord(meta)
}

// Sign signs the given message bytes using the key identified by uid
func (k *BaoKeyring) Sign(uid string, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
    meta, err := k.store.Get(uid)
    if err != nil {
        return nil, nil, err
    }
    
    // Hash the message
    hash := sha256.Sum256(msg)
    
    // Sign via OpenBao (returns Cosmos format R||S)
    ctx := context.Background()
    sig, err := k.client.Sign(ctx, uid, hash[:], true)
    if err != nil {
        return nil, nil, err
    }
    
    // Build public key
    pubKey := &secp256k1.PubKey{Key: meta.PubKeyBytes}
    
    return sig, pubKey, nil
}

// SignByAddress signs using the key matching the given address
func (k *BaoKeyring) SignByAddress(address sdk.Address, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
    meta, err := k.store.GetByAddress(address.String())
    if err != nil {
        return nil, nil, err
    }
    
    return k.Sign(meta.UID, msg, signMode)
}

// Delete removes a key from both OpenBao and local store
func (k *BaoKeyring) Delete(uid string) error {
    ctx := context.Background()
    
    // Delete from OpenBao first
    if err := k.client.DeleteKey(ctx, uid); err != nil {
        return err
    }
    
    // Then delete local metadata
    return k.store.Delete(uid)
}

// Rename changes the uid of a key
func (k *BaoKeyring) Rename(fromUID, toUID string) error {
    // Note: OpenBao doesn't support rename, so we only rename locally
    // The BaoKeyPath remains the same
    return k.store.Rename(fromUID, toUID)
}

// GetMetadata returns the raw metadata for a key
func (k *BaoKeyring) GetMetadata(uid string) (*KeyMetadata, error) {
    return k.store.Get(uid)
}

// SyncFromRemote syncs keys from OpenBao to local store
// Useful when keys are created via web UI or other clients
func (k *BaoKeyring) SyncFromRemote(ctx context.Context) error {
    remoteKeys, err := k.client.ListKeys(ctx)
    if err != nil {
        return fmt.Errorf("failed to list remote keys: %w", err)
    }
    
    for _, keyName := range remoteKeys {
        // Skip if already in local store
        if k.store.Has(keyName) {
            continue
        }
        
        // Fetch key info
        keyInfo, err := k.client.GetKey(ctx, keyName)
        if err != nil {
            continue // Skip on error
        }
        
        pubKeyBytes, err := hex.DecodeString(keyInfo.PublicKey)
        if err != nil {
            continue
        }
        
        meta := &KeyMetadata{
            UID:         keyName,
            Name:        keyName,
            PubKeyBytes: pubKeyBytes,
            PubKeyType:  "secp256k1",
            Address:     keyInfo.Address,
            BaoKeyPath:  fmt.Sprintf("%s/keys/%s", k.client.secp256k1Path, keyName),
            Algorithm:   AlgorithmSecp256k1,
            Exportable:  keyInfo.Exportable,
            CreatedAt:   keyInfo.CreatedAt,
            Source:      "synced",
        }
        
        _ = k.store.Save(meta)
    }
    
    return nil
}

// Close releases resources
func (k *BaoKeyring) Close() error {
    return k.store.Close()
}

// Helper methods

func (k *BaoKeyring) metadataToRecord(meta *KeyMetadata) (*keyring.Record, error) {
    pubKey := &secp256k1.PubKey{Key: meta.PubKeyBytes}
    
    // Create record using keyring.NewLocalRecord
    return keyring.NewLocalRecord(meta.Name, pubKey, nil)
}
```

---

## 4. Unit Test Requirements

### 4.1 bao_keyring_test.go

```go
func TestBaoKeyring_NewAccount(t *testing.T) {
    // Setup mock client and store
    kr := setupTestKeyring(t)
    defer kr.Close()
    
    record, err := kr.NewAccount("test-key", "", "", "", nil)
    require.NoError(t, err)
    require.Equal(t, "test-key", record.Name)
    require.NotNil(t, record.PubKey)
}

func TestBaoKeyring_Sign(t *testing.T) {
    kr := setupTestKeyring(t)
    defer kr.Close()
    
    // Create key
    _, err := kr.NewAccount("sign-test", "", "", "", nil)
    require.NoError(t, err)
    
    // Sign message
    msg := []byte("test message for signing")
    sig, pubKey, err := kr.Sign("sign-test", msg, signing.SignMode_SIGN_MODE_DIRECT)
    
    require.NoError(t, err)
    require.Len(t, sig, 64) // Cosmos format
    require.NotNil(t, pubKey)
}

func TestBaoKeyring_SignByAddress(t *testing.T) {
    kr := setupTestKeyring(t)
    defer kr.Close()
    
    record, _ := kr.NewAccount("addr-test", "", "", "", nil)
    addr, _ := record.GetAddress()
    
    sig, pubKey, err := kr.SignByAddress(addr, []byte("test"), signing.SignMode_SIGN_MODE_DIRECT)
    
    require.NoError(t, err)
    require.NotNil(t, sig)
    require.NotNil(t, pubKey)
}

func TestBaoKeyring_List(t *testing.T) {
    kr := setupTestKeyring(t)
    defer kr.Close()
    
    kr.NewAccount("key1", "", "", "", nil)
    kr.NewAccount("key2", "", "", "", nil)
    kr.NewAccount("key3", "", "", "", nil)
    
    keys, err := kr.List()
    require.NoError(t, err)
    require.Len(t, keys, 3)
}

func TestBaoKeyring_Delete(t *testing.T) {
    kr := setupTestKeyring(t)
    defer kr.Close()
    
    kr.NewAccount("to-delete", "", "", "", nil)
    
    err := kr.Delete("to-delete")
    require.NoError(t, err)
    
    _, err = kr.Key("to-delete")
    require.Error(t, err)
}

func TestBaoKeyring_Backend(t *testing.T) {
    kr := setupTestKeyring(t)
    defer kr.Close()
    
    require.Equal(t, "openbao", kr.Backend())
}
```

---

## 5. Success Criteria

### 5.1 Functional Requirements

- [ ] Implements full `keyring.Keyring` interface
- [ ] `NewAccount` creates keys in OpenBao
- [ ] `NewAccount` stores metadata locally
- [ ] `Sign` hashes message with SHA-256
- [ ] `Sign` returns 64-byte Cosmos signature
- [ ] `SignByAddress` looks up key by address
- [ ] `Delete` removes from both OpenBao and local store
- [ ] `List` returns all keys
- [ ] `Key` and `KeyByAddress` return valid Records
- [ ] `SyncFromRemote` imports keys created elsewhere

### 5.2 Non-Functional Requirements

- [ ] Health check on initialization
- [ ] Graceful error handling
- [ ] Context propagation for cancellation
- [ ] No private keys in memory

### 5.3 Test Coverage

- [ ] > 80% code coverage
- [ ] All keyring methods tested
- [ ] Error paths tested
- [ ] Integration with mock client/store

---

## 6. Interface Contracts

For Agent 05 (Migration), expose these:

```go
// For migration package
func (k *BaoKeyring) ImportKey(uid string, privKey []byte, exportable bool) (*keyring.Record, error)
func (k *BaoKeyring) ExportKey(uid string) ([]byte, error)
func (k *BaoKeyring) GetWrappingKey() ([]byte, error)
```

---

## 7. Dependencies

```go
require (
    github.com/cosmos/cosmos-sdk v0.50.x
)
```

---

## 8. Deliverables Checklist

- [ ] `bao_keyring.go` - Keyring implementation
- [ ] `bao_keyring_test.go` - Unit tests
- [ ] Implements `keyring.Keyring` interface
- [ ] All tests pass
- [ ] No linter errors

