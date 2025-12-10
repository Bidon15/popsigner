# Implementation Guide: BaoKeyring Key Operations

**Agent ID:** 04B  
**Parent:** Agent 04 (BaoKeyring)  
**Component:** Key Management Methods  
**Parallelizable:** ✅ Yes - Uses 04A core

---

## 1. Overview

Key management: NewAccount, Key, KeyByAddress, List, Delete, Rename.

### 1.1 Required Skills

| Skill          | Level    | Description          |
| -------------- | -------- | -------------------- |
| **Go**         | Advanced | Interfaces           |
| **Cosmos SDK** | Advanced | keyring.Record       |

### 1.2 Methods to Implement

```
NewAccount()
Key()
KeyByAddress()
List()
Delete()
Rename()
```

---

## 2. Specifications

**IMPORTANT:** These imports use standard cosmos-sdk paths but are REPLACED 
by Celestia's forks via go.mod replace directives.

```go
package banhbaoring

import (
    "context"
    "encoding/hex"
    "fmt"
    "time"

    // All cosmos-sdk imports are replaced by celestiaorg/cosmos-sdk via go.mod
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
    "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewAccount creates a key in OpenBao.
func (k *BaoKeyring) NewAccount(
    uid, mnemonic, bip39Passphrase, hdPath string,
    algo keyring.SignatureAlgo,
) (*keyring.Record, error) {
    // Validate algo
    if algo != nil && algo.Name() != string(keyring.Secp256k1Type) {
        return nil, ErrUnsupportedAlgo
    }
    
    if k.store.Has(uid) {
        return nil, fmt.Errorf("%w: %s", ErrKeyExists, uid)
    }
    
    ctx := context.Background()
    
    keyInfo, err := k.client.CreateKey(ctx, uid, KeyOptions{Exportable: false})
    if err != nil {
        return nil, err
    }
    
    pubKeyBytes, _ := hex.DecodeString(keyInfo.PublicKey)
    
    meta := &KeyMetadata{
        UID:         uid,
        Name:        uid,
        PubKeyBytes: pubKeyBytes,
        PubKeyType:  "secp256k1",
        Address:     keyInfo.Address,
        BaoKeyPath:  fmt.Sprintf("%s/keys/%s", k.client.secp256k1Path, uid),
        Algorithm:   AlgorithmSecp256k1,
        CreatedAt:   time.Now().UTC(),
        Source:      SourceGenerated,
    }
    
    if err := k.store.Save(meta); err != nil {
        k.client.DeleteKey(ctx, uid) // Cleanup
        return nil, err
    }
    
    return k.metadataToRecord(meta)
}

// Key retrieves a key by UID.
func (k *BaoKeyring) Key(uid string) (*keyring.Record, error) {
    meta, err := k.store.Get(uid)
    if err != nil {
        return nil, err
    }
    return k.metadataToRecord(meta)
}

// KeyByAddress retrieves a key by address.
func (k *BaoKeyring) KeyByAddress(address sdk.Address) (*keyring.Record, error) {
    meta, err := k.store.GetByAddress(address.String())
    if err != nil {
        return nil, err
    }
    return k.metadataToRecord(meta)
}

// List returns all keys.
func (k *BaoKeyring) List() ([]*keyring.Record, error) {
    metas, err := k.store.List()
    if err != nil {
        return nil, err
    }
    
    records := make([]*keyring.Record, 0, len(metas))
    for _, meta := range metas {
        if record, err := k.metadataToRecord(meta); err == nil {
            records = append(records, record)
        }
    }
    return records, nil
}

// Delete removes a key.
func (k *BaoKeyring) Delete(uid string) error {
    ctx := context.Background()
    
    if err := k.client.DeleteKey(ctx, uid); err != nil {
        return err
    }
    return k.store.Delete(uid)
}

// Rename changes the UID.
func (k *BaoKeyring) Rename(fromUID, toUID string) error {
    return k.store.Rename(fromUID, toUID)
}

// GetMetadata returns raw metadata.
func (k *BaoKeyring) GetMetadata(uid string) (*KeyMetadata, error) {
    return k.store.Get(uid)
}

func (k *BaoKeyring) metadataToRecord(meta *KeyMetadata) (*keyring.Record, error) {
    pubKey := &secp256k1.PubKey{Key: meta.PubKeyBytes}
    return keyring.NewLocalRecord(meta.Name, pubKey, nil)
}
```

---

## 3. Deliverables

- [ ] `NewAccount` creates in OpenBao + saves metadata
- [ ] `Key` retrieves by UID
- [ ] `KeyByAddress` retrieves by address
- [ ] `List` returns all keys
- [ ] `Delete` removes from both OpenBao and store
- [ ] `Rename` updates local metadata

