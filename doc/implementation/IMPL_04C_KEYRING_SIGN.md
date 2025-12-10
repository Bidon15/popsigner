# Implementation Guide: BaoKeyring Signing

**Agent ID:** 04C  
**Parent:** Agent 04 (BaoKeyring)  
**Component:** Signing Operations  
**Parallelizable:** ✅ Yes - Uses 04A core

---

## 1. Overview

Signing methods: Sign, SignByAddress.

### 1.1 Required Skills

| Skill      | Level        | Description        |
| ---------- | ------------ | ------------------ |
| **Go**     | Advanced     | Crypto operations  |
| **Crypto** | Intermediate | SHA-256 hashing    |

### 1.2 Methods to Implement

```
Sign()
SignByAddress()
```

---

## 2. Specifications

**IMPORTANT:** These imports use standard cosmos-sdk paths but are REPLACED 
by Celestia's forks via go.mod replace directives.

```go
package banhbaoring

import (
    "context"
    "crypto/sha256"

    // All cosmos-sdk imports are replaced by celestiaorg/cosmos-sdk via go.mod
    "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
    cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// Sign signs message bytes using OpenBao.
func (k *BaoKeyring) Sign(uid string, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
    meta, err := k.store.Get(uid)
    if err != nil {
        return nil, nil, err
    }
    
    // Hash the message
    hash := sha256.Sum256(msg)
    
    // Sign via OpenBao (returns 64-byte Cosmos format)
    ctx := context.Background()
    sig, err := k.client.Sign(ctx, uid, hash[:], true)
    if err != nil {
        return nil, nil, err
    }
    
    pubKey := &secp256k1.PubKey{Key: meta.PubKeyBytes}
    return sig, pubKey, nil
}

// SignByAddress signs using the key at address.
func (k *BaoKeyring) SignByAddress(address sdk.Address, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
    meta, err := k.store.GetByAddress(address.String())
    if err != nil {
        return nil, nil, err
    }
    return k.Sign(meta.UID, msg, signMode)
}
```

---

## 3. Unit Tests

```go
func TestBaoKeyring_Sign(t *testing.T) {
    kr := setupTestKeyring(t)
    defer kr.Close()
    
    kr.NewAccount("sign-key", "", "", "", nil)
    
    msg := []byte("test message")
    sig, pubKey, err := kr.Sign("sign-key", msg, signing.SignMode_SIGN_MODE_DIRECT)
    
    require.NoError(t, err)
    assert.Len(t, sig, 64)
    assert.NotNil(t, pubKey)
}

func TestBaoKeyring_SignByAddress(t *testing.T) {
    kr := setupTestKeyring(t)
    defer kr.Close()
    
    record, _ := kr.NewAccount("addr-key", "", "", "", nil)
    addr, _ := record.GetAddress()
    
    sig, _, err := kr.SignByAddress(addr, []byte("test"), 0)
    require.NoError(t, err)
    assert.Len(t, sig, 64)
}
```

---

## 4. Deliverables

- [ ] `Sign` hashes message with SHA-256
- [ ] `Sign` calls OpenBao with prehashed=true
- [ ] `Sign` returns 64-byte signature + pubkey
- [ ] `SignByAddress` looks up key by address
- [ ] Both methods tested

