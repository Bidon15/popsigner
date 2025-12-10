# OpenBao secp256k1 Plugin API Reference

This document provides a reference for the custom secp256k1 OpenBao plugin API endpoints used by the BaoKeyring implementation.

---

## 1. Overview

The **secp256k1 plugin** is a custom OpenBao secrets engine that provides native secp256k1 key management and signing for Cosmos/Celestia applications.

**Key Security Feature:** Private keys **NEVER** leave OpenBao. Signing happens inside the plugin, and only signatures are returned.

### 1.1 Why a Custom Plugin?

OpenBao Transit doesn't support secp256k1 natively:

| Engine | secp256k1 | Key Exposure |
|--------|-----------|--------------|
| Transit (built-in) | ❌ No | N/A |
| **secp256k1 Plugin** | ✅ Yes | **Never leaves OpenBao** |

**Base URL:** `https://<bao-host>:8200/v1`

**Plugin Path:** `/secp256k1` (configurable)

**Authentication:** All requests require the `X-Vault-Token` header with a valid OpenBao token.

---

## 2. Authentication

### Request Headers

All API requests must include:

```http
X-Vault-Token: <your-bao-token>
Content-Type: application/json
```

Optional namespace header (for Enterprise/namespaced deployments):

```http
X-Vault-Namespace: <namespace>
```

---

## 3. Key Management

### 3.1 Create Key

Creates a new named encryption key of the specified type.

**Endpoint:** `POST /v1/transit/keys/<name>`

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | URL path - name of the key |
| `type` | string | No | Key type (default: `aes256-gcm96`) |
| `exportable` | bool | No | Allow key export (default: `false`) |
| `allow_plaintext_backup` | bool | No | Allow plaintext backup (default: `false`) |

**Key Types for Signing:**

| Type | Description | Use Case |
|------|-------------|----------|
| `ecdsa-p256` | ECDSA with P-256 curve | General ECDSA signing |
| `ecdsa-p384` | ECDSA with P-384 curve | Higher security ECDSA |
| `ecdsa-p521` | ECDSA with P-521 curve | Maximum security ECDSA |

> **Note:** OpenBao Transit does not natively support `secp256k1`. See section 7 for workarounds.

**Request:**

```bash
curl -X POST \
  -H "X-Vault-Token: $BAO_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ecdsa-p256",
    "exportable": false
  }' \
  https://bao.example.com:8200/v1/transit/keys/my-celestia-key
```

**Response (Success - 204 No Content):**

No body returned on success.

**Response (Error - 400):**

```json
{
  "errors": ["key already exists"]
}
```

---

### 3.2 Read Key

Returns information about a named encryption key.

**Endpoint:** `GET /v1/transit/keys/<name>`

**Request:**

```bash
curl -X GET \
  -H "X-Vault-Token: $BAO_TOKEN" \
  https://bao.example.com:8200/v1/transit/keys/my-celestia-key
```

**Response:**

```json
{
  "request_id": "abc123",
  "lease_id": "",
  "renewable": false,
  "lease_duration": 0,
  "data": {
    "name": "my-celestia-key",
    "type": "ecdsa-p256",
    "deletion_allowed": false,
    "derived": false,
    "exportable": false,
    "allow_plaintext_backup": false,
    "keys": {
      "1": {
        "name": "P-256",
        "public_key": "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...\n-----END PUBLIC KEY-----",
        "creation_time": "2025-01-10T12:00:00Z"
      }
    },
    "min_decryption_version": 1,
    "min_encryption_version": 0,
    "latest_version": 1,
    "supports_encryption": false,
    "supports_decryption": false,
    "supports_derivation": false,
    "supports_signing": true
  },
  "wrap_info": null,
  "warnings": null,
  "auth": null
}
```

**Public Key Format:**

The `public_key` field contains a PEM-encoded public key. For ECDSA keys, this is in SubjectPublicKeyInfo format:

```
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...base64...
-----END PUBLIC KEY-----
```

**Parsing the Public Key (Go):**

```go
import (
    "crypto/x509"
    "encoding/pem"
)

func ParsePublicKey(pemData string) (*ecdsa.PublicKey, error) {
    block, _ := pem.Decode([]byte(pemData))
    if block == nil {
        return nil, errors.New("failed to decode PEM block")
    }
    
    pub, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
        return nil, err
    }
    
    ecdsaPub, ok := pub.(*ecdsa.PublicKey)
    if !ok {
        return nil, errors.New("not an ECDSA public key")
    }
    
    return ecdsaPub, nil
}
```

---

### 3.3 List Keys

Lists all keys in the Transit engine.

**Endpoint:** `LIST /v1/transit/keys`

**Request:**

```bash
curl -X LIST \
  -H "X-Vault-Token: $BAO_TOKEN" \
  https://bao.example.com:8200/v1/transit/keys
```

Or using GET with `list=true`:

```bash
curl -X GET \
  -H "X-Vault-Token: $BAO_TOKEN" \
  "https://bao.example.com:8200/v1/transit/keys?list=true"
```

**Response:**

```json
{
  "request_id": "abc123",
  "data": {
    "keys": [
      "my-celestia-key",
      "another-key"
    ]
  }
}
```

---

### 3.4 Delete Key

Deletes a named encryption key. Keys must have `deletion_allowed` set to `true`.

**Endpoint:** `DELETE /v1/transit/keys/<name>`

**Pre-requisite:** Enable deletion

```bash
curl -X POST \
  -H "X-Vault-Token: $BAO_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"deletion_allowed": true}' \
  https://bao.example.com:8200/v1/transit/keys/my-celestia-key/config
```

**Delete Request:**

```bash
curl -X DELETE \
  -H "X-Vault-Token: $BAO_TOKEN" \
  https://bao.example.com:8200/v1/transit/keys/my-celestia-key
```

**Response (Success - 204 No Content):**

No body returned.

---

## 4. Signing Operations

### 4.1 Sign Data

Signs the provided input using the named key.

**Endpoint:** `POST /v1/transit/sign/<name>`

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | URL path - key name |
| `input` | string | Yes | Base64-encoded input data |
| `key_version` | int | No | Key version to use (default: latest) |
| `prehashed` | bool | No | If `true`, input is already hashed |
| `signature_algorithm` | string | No | Algorithm for RSA keys |
| `marshaling_algorithm` | string | No | `asn1` (DER) or `jws` |
| `hash_algorithm` | string | No | Hash algorithm if not prehashed |

**Marshaling Algorithms:**

| Value | Description |
|-------|-------------|
| `asn1` | DER-encoded ASN.1 signature (default) |
| `jws` | JWS-compatible format (R \|\| S, base64url) |

**Request (with pre-hashed data):**

```bash
# First, hash the data
HASH=$(echo -n "data to sign" | sha256sum | cut -d' ' -f1 | xxd -r -p | base64)

curl -X POST \
  -H "X-Vault-Token: $BAO_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"input\": \"$HASH\",
    \"prehashed\": true,
    \"hash_algorithm\": \"sha2-256\",
    \"marshaling_algorithm\": \"asn1\"
  }" \
  https://bao.example.com:8200/v1/transit/sign/my-celestia-key
```

**Request (raw data, OpenBao hashes):**

```bash
DATA=$(echo -n "data to sign" | base64)

curl -X POST \
  -H "X-Vault-Token: $BAO_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"input\": \"$DATA\",
    \"prehashed\": false,
    \"hash_algorithm\": \"sha2-256\",
    \"marshaling_algorithm\": \"asn1\"
  }" \
  https://bao.example.com:8200/v1/transit/sign/my-celestia-key
```

**Response:**

```json
{
  "request_id": "abc123",
  "data": {
    "signature": "vault:v1:MEUCIQDx...base64...",
    "key_version": 1
  }
}
```

**Signature Format:**

The signature is prefixed with `vault:v<version>:` followed by the base64-encoded signature:

```
vault:v1:MEUCIQDxR7p8NkKhL5c3RVf...
         └──────────────────────┘
              Base64 DER signature
```

**Parsing the Signature (Go):**

```go
func ParseBaoSignature(sigStr string) ([]byte, int, error) {
    // Format: "vault:v<version>:<base64-signature>"
    parts := strings.SplitN(sigStr, ":", 3)
    if len(parts) != 3 || parts[0] != "vault" {
        return nil, 0, errors.New("invalid signature format")
    }
    
    // Extract version
    versionStr := strings.TrimPrefix(parts[1], "v")
    version, err := strconv.Atoi(versionStr)
    if err != nil {
        return nil, 0, fmt.Errorf("invalid version: %w", err)
    }
    
    // Decode signature
    sig, err := base64.StdEncoding.DecodeString(parts[2])
    if err != nil {
        return nil, 0, fmt.Errorf("failed to decode signature: %w", err)
    }
    
    return sig, version, nil
}
```

---

### 4.2 Verify Signature

Verifies a signature against the provided input.

**Endpoint:** `POST /v1/transit/verify/<name>`

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | URL path - key name |
| `input` | string | Yes | Base64-encoded input data |
| `signature` | string | Yes | Signature to verify |
| `prehashed` | bool | No | If `true`, input is already hashed |
| `hash_algorithm` | string | No | Hash algorithm used |
| `marshaling_algorithm` | string | No | `asn1` or `jws` |

**Request:**

```bash
curl -X POST \
  -H "X-Vault-Token: $BAO_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "'"$HASH"'",
    "signature": "vault:v1:MEUCIQDx...",
    "prehashed": true,
    "hash_algorithm": "sha2-256"
  }' \
  https://bao.example.com:8200/v1/transit/verify/my-celestia-key
```

**Response:**

```json
{
  "request_id": "abc123",
  "data": {
    "valid": true
  }
}
```

---

## 5. Key Import

### 5.1 Import Key

Imports an existing key into the Transit engine.

**Endpoint:** `POST /v1/transit/keys/<name>/import`

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | URL path - key name |
| `ciphertext` | string | Yes | Wrapped key material |
| `type` | string | Yes | Key type |
| `allow_rotation` | bool | No | Allow key rotation |

**Key Wrapping:**

OpenBao requires key material to be wrapped using the Transit wrapping key:

1. Get the wrapping key: `GET /v1/transit/wrapping_key`
2. Wrap your key material with the wrapping key
3. Submit the wrapped key

**Get Wrapping Key:**

```bash
curl -X GET \
  -H "X-Vault-Token: $BAO_TOKEN" \
  https://bao.example.com:8200/v1/transit/wrapping_key
```

**Response:**

```json
{
  "data": {
    "public_key": "-----BEGIN PUBLIC KEY-----\nMIICIjANBg...\n-----END PUBLIC KEY-----"
  }
}
```

**Import Request:**

```bash
curl -X POST \
  -H "X-Vault-Token: $BAO_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "ciphertext": "<wrapped-key-base64>",
    "type": "ecdsa-p256"
  }' \
  https://bao.example.com:8200/v1/transit/keys/imported-key/import
```

---

## 6. DER Signature Parsing

### 6.1 DER Structure

ECDSA signatures from OpenBao are DER-encoded:

```
SEQUENCE (2 elements)
├── INTEGER r (variable length, may have leading 0x00)
└── INTEGER s (variable length, may have leading 0x00)
```

**Example DER (hex):**

```
30 45                          ; SEQUENCE, 69 bytes
   02 21                       ; INTEGER, 33 bytes
      00 AB CD ... (33 bytes)  ; R with leading zero
   02 20                       ; INTEGER, 32 bytes
      12 34 ... (32 bytes)     ; S
```

### 6.2 Parsing Implementation

```go
import (
    "encoding/asn1"
    "math/big"
)

// ECDSASignature represents a DER-encoded ECDSA signature
type ECDSASignature struct {
    R, S *big.Int
}

// ParseDERSignature parses a DER-encoded ECDSA signature
func ParseDERSignature(der []byte) (*ECDSASignature, error) {
    var sig ECDSASignature
    _, err := asn1.Unmarshal(der, &sig)
    if err != nil {
        return nil, fmt.Errorf("failed to parse DER signature: %w", err)
    }
    return &sig, nil
}

// ToCompact converts to 64-byte compact format (R || S)
func (s *ECDSASignature) ToCompact() []byte {
    compact := make([]byte, 64)
    
    // Copy R (32 bytes, zero-padded)
    rBytes := s.R.Bytes()
    copy(compact[32-len(rBytes):32], rBytes)
    
    // Copy S (32 bytes, zero-padded)
    sBytes := s.S.Bytes()
    copy(compact[64-len(sBytes):64], sBytes)
    
    return compact
}
```

---

## 7. secp256k1 Native Support (BanhBao Plugin)

### 7.1 Plugin Approach

BanhBao uses a **custom OpenBao plugin** that provides native secp256k1 support. This is superior to hybrid approaches:

| Approach | Security | Implementation |
|----------|----------|----------------|
| Hybrid (decrypt + sign locally) | Good | Key exposed in app memory |
| **Native Plugin (BanhBao)** | **Excellent** | **Key never leaves OpenBao** |

### 7.2 Plugin Endpoints

The secp256k1 plugin is mounted at `/secp256k1` and provides:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/secp256k1/keys/:name` | POST | Create secp256k1 key |
| `/secp256k1/keys/:name` | GET | Get public key info |
| `/secp256k1/keys` | LIST | List all keys |
| `/secp256k1/sign/:name` | POST | Sign with secp256k1 |
| `/secp256k1/verify/:name` | POST | Verify signature |
| `/secp256k1/keys/:name/import` | POST | Import wrapped key |
| `/secp256k1/export/:name` | GET | Export (if allowed) |

### 7.3 Signature Output Formats

The plugin supports multiple output formats:

| Format | Description | Use Case |
|--------|-------------|----------|
| `cosmos` | R \|\| S (64 bytes, low-S) | Celestia, Cosmos SDK |
| `der` | ASN.1 DER encoding | Generic ECDSA |
| `ethereum` | R \|\| S \|\| V (65 bytes) | Ethereum transactions |

**Default:** `cosmos` format for direct Celestia compatibility.

### 7.4 Alternative: Fork for Cloud KMS

Users who prefer AWS KMS or GCP Cloud KMS can fork the client library:

```go
// Alternative implementation using AWS KMS (user's responsibility)
type KMSKeyring struct {
    kmsClient *kms.Client
}

func (k *KMSKeyring) Sign(uid string, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
    // 1. Fetch encrypted key from KMS
    // 2. Decrypt (key now in app memory - less secure)
    // 3. Sign with btcec
    // 4. Return signature
}
```

**Note:** This hybrid approach exposes the key in application memory. The BanhBao plugin approach is more secure.

---

## 8. Error Responses

### 8.1 Common Error Codes

| HTTP Status | Error | Description |
|-------------|-------|-------------|
| 400 | Bad Request | Invalid parameters or malformed request |
| 403 | Forbidden | Permission denied or invalid token |
| 404 | Not Found | Key or path doesn't exist |
| 500 | Internal Server Error | Server-side error |
| 503 | Service Unavailable | OpenBao sealed or unavailable |

### 8.2 Error Response Format

```json
{
  "errors": [
    "1 error occurred:\n\t* permission denied"
  ]
}
```

### 8.3 Error Handling (Go)

```go
type BaoError struct {
    StatusCode int
    Errors     []string
}

func (e *BaoError) Error() string {
    return fmt.Sprintf("OpenBao error (HTTP %d): %s", e.StatusCode, strings.Join(e.Errors, "; "))
}

func handleResponse(resp *http.Response) error {
    if resp.StatusCode >= 400 {
        var errResp struct {
            Errors []string `json:"errors"`
        }
        json.NewDecoder(resp.Body).Decode(&errResp)
        return &BaoError{
            StatusCode: resp.StatusCode,
            Errors:     errResp.Errors,
        }
    }
    return nil
}
```

---

## 9. Rate Limiting & Performance

### 9.1 Best Practices

| Practice | Description |
|----------|-------------|
| Connection Pooling | Reuse HTTP connections to OpenBao |
| Batch Operations | Group multiple operations when possible |
| Caching | Cache public keys (they don't change) |
| Timeouts | Set appropriate request timeouts |

### 9.2 HTTP Client Configuration

```go
func NewBaoHTTPClient(timeout time.Duration) *http.Client {
    return &http.Client{
        Timeout: timeout,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
            TLSHandshakeTimeout: 10 * time.Second,
        },
    }
}
```

---

## 10. References

- [OpenBao Transit Secrets Engine](https://openbao.org/docs/secrets/transit/)
- [OpenBao Transit API Documentation](https://openbao.org/api-docs/secret/transit/)
- [ASN.1 DER Encoding](https://luca.ntop.org/Teaching/Appunti/asn1.html)
- [SEC 1: Elliptic Curve Cryptography](https://www.secg.org/sec1-v2.pdf)

