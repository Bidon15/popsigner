# Implementation Guide: Plugin Crypto Helpers

**Agent ID:** 03E  
**Parent:** Agent 03 (OpenBao Plugin)  
**Component:** Cryptographic Utilities  
**Parallelizable:** ✅ Yes - Pure functions, no dependencies

---

## 1. Overview

Cryptographic helper functions: hashing, signature formatting, address derivation.

### 1.1 Required Skills

| Skill      | Level    | Description                    |
| ---------- | -------- | ------------------------------ |
| **Go**     | Advanced | Crypto packages                |
| **Crypto** | Advanced | secp256k1, SHA256, RIPEMD160   |

### 1.2 Files to Create

```
plugin/secp256k1/
└── crypto.go
└── crypto_test.go
```

---

## 2. Specifications

### 2.1 crypto.go

```go
package secp256k1

import (
    "crypto/sha256"
    "runtime"
    
    "github.com/btcsuite/btcd/btcec/v2"
    "github.com/btcsuite/btcd/btcec/v2/ecdsa"
    "golang.org/x/crypto/ripemd160"
    "golang.org/x/crypto/sha3"
)

// hashSHA256 computes SHA-256.
func hashSHA256(data []byte) []byte {
    h := sha256.Sum256(data)
    return h[:]
}

// hashKeccak256 computes Keccak-256 (Ethereum).
func hashKeccak256(data []byte) []byte {
    h := sha3.NewLegacyKeccak256()
    h.Write(data)
    return h.Sum(nil)
}

// deriveCosmosAddress derives address from compressed public key.
// Formula: RIPEMD160(SHA256(pubkey))
func deriveCosmosAddress(pubKey []byte) []byte {
    sha := sha256.Sum256(pubKey)
    rip := ripemd160.New()
    rip.Write(sha[:])
    return rip.Sum(nil)
}

// formatCosmosSignature formats as R||S (64 bytes) with low-S.
func formatCosmosSignature(sig *ecdsa.Signature) []byte {
    r := sig.R()
    s := sig.S()
    
    // Normalize to low-S (BIP-62)
    if s.IsOverHalfOrder() {
        s.Negate()
    }
    
    result := make([]byte, 64)
    r.PutBytesUnchecked(result[:32])
    s.PutBytesUnchecked(result[32:])
    return result
}

// parseCosmosSignature parses R||S format.
func parseCosmosSignature(sigBytes []byte) (*ecdsa.Signature, error) {
    if len(sigBytes) != 64 {
        return nil, fmt.Errorf("signature must be 64 bytes")
    }
    
    r := new(btcec.ModNScalar)
    s := new(btcec.ModNScalar)
    
    r.SetByteSlice(sigBytes[:32])
    s.SetByteSlice(sigBytes[32:])
    
    return ecdsa.NewSignature(r, s), nil
}

// secureZero wipes sensitive data from memory.
func secureZero(b []byte) {
    for i := range b {
        b[i] = 0
    }
    runtime.KeepAlive(b)
}
```

---

## 3. Unit Tests

```go
func TestHashSHA256(t *testing.T) {
    result := hashSHA256([]byte("test"))
    assert.Len(t, result, 32)
}

func TestDeriveCosmosAddress(t *testing.T) {
    // Known test vector
    pubKey, _ := hex.DecodeString("02a1b2c3...")
    addr := deriveCosmosAddress(pubKey)
    assert.Len(t, addr, 20) // RIPEMD160 output
}

func TestFormatCosmosSignature(t *testing.T) {
    privKey, _ := btcec.NewPrivateKey()
    hash := hashSHA256([]byte("test"))
    sig := ecdsa.Sign(privKey, hash)
    
    formatted := formatCosmosSignature(sig)
    assert.Len(t, formatted, 64)
    
    // Verify low-S
    s := new(btcec.ModNScalar)
    s.SetByteSlice(formatted[32:])
    assert.False(t, s.IsOverHalfOrder())
}

func TestSecureZero(t *testing.T) {
    data := []byte{1, 2, 3, 4, 5}
    secureZero(data)
    
    for _, b := range data {
        assert.Equal(t, byte(0), b)
    }
}
```

---

## 4. Deliverables

- [ ] `hashSHA256` returns 32 bytes
- [ ] `hashKeccak256` returns 32 bytes
- [ ] `deriveCosmosAddress` returns 20 bytes
- [ ] `formatCosmosSignature` returns 64 bytes, low-S
- [ ] `secureZero` wipes memory
- [ ] All functions have test coverage

