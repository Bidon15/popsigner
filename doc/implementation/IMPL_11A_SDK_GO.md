# Implementation: Go SDK

## Agent: 11A - Go SDK

> **Phase 7** - Can run in parallel with 11B after Control Plane API is stable.

---

## 1. Overview

Create the official Go SDK for the BanhBaoRing Control Plane API.

**Package:** `github.com/banhbaoring/sdk-go`

---

## 2. Features

| Feature | Priority |
|---------|----------|
| Authentication (API keys) | P0 |
| Key management (CRUD) | P0 |
| Signing operations | P0 |
| Batch operations | P0 |
| Organizations | P1 |
| Audit logs | P1 |
| Webhooks | P2 |
| Billing | P2 |

---

## 3. Project Structure

```
sdk-go/
├── banhbaoring.go          # Main client
├── auth.go                 # Auth methods
├── keys.go                 # Key management
├── sign.go                 # Signing operations
├── orgs.go                 # Organizations
├── audit.go                # Audit logs
├── webhooks.go             # Webhooks
├── billing.go              # Billing
├── errors.go               # Error types
├── types.go                # Shared types
├── http.go                 # HTTP client wrapper
├── examples/
│   ├── basic/main.go
│   ├── parallel-workers/main.go
│   └── celestia-integration/main.go
├── go.mod
├── go.sum
└── README.md
```

---

## 4. Client Implementation

**File:** `banhbaoring.go`

```go
package banhbaoring

import (
    "context"
    "net/http"
    "time"
)

const (
    DefaultBaseURL = "https://api.banhbaoring.io"
    DefaultTimeout = 30 * time.Second
)

// Client is the BanhBaoRing API client.
type Client struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client

    // Services
    Keys     *KeysService
    Sign     *SignService
    Orgs     *OrgsService
    Audit    *AuditService
    Webhooks *WebhooksService
    Billing  *BillingService
}

// Option configures the client.
type Option func(*Client)

// WithBaseURL sets a custom API base URL.
func WithBaseURL(url string) Option {
    return func(c *Client) {
        c.baseURL = url
    }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
    return func(c *Client) {
        c.httpClient = httpClient
    }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) Option {
    return func(c *Client) {
        c.httpClient.Timeout = timeout
    }
}

// NewClient creates a new BanhBaoRing API client.
//
// Example:
//
//	client := banhbaoring.NewClient("bbr_live_xxxxx")
//	key, err := client.Keys.Create(ctx, banhbaoring.CreateKeyRequest{
//	    Name:      "sequencer",
//	    Namespace: "production",
//	})
func NewClient(apiKey string, opts ...Option) *Client {
    c := &Client{
        apiKey:  apiKey,
        baseURL: DefaultBaseURL,
        httpClient: &http.Client{
            Timeout: DefaultTimeout,
        },
    }

    for _, opt := range opts {
        opt(c)
    }

    // Initialize services
    c.Keys = &KeysService{client: c}
    c.Sign = &SignService{client: c}
    c.Orgs = &OrgsService{client: c}
    c.Audit = &AuditService{client: c}
    c.Webhooks = &WebhooksService{client: c}
    c.Billing = &BillingService{client: c}

    return c
}
```

---

## 5. Keys Service

**File:** `keys.go`

```go
package banhbaoring

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
)

// KeysService handles key management operations.
type KeysService struct {
    client *Client
}

// Key represents a cryptographic key.
type Key struct {
    ID          uuid.UUID         `json:"id"`
    Name        string            `json:"name"`
    NamespaceID uuid.UUID         `json:"namespace_id"`
    PublicKey   string            `json:"public_key"`
    Address     string            `json:"address"`
    Algorithm   string            `json:"algorithm"`
    Exportable  bool              `json:"exportable"`
    Metadata    map[string]string `json:"metadata,omitempty"`
    CreatedAt   time.Time         `json:"created_at"`
}

// CreateKeyRequest is the request for creating a key.
type CreateKeyRequest struct {
    Name        string            `json:"name"`
    NamespaceID uuid.UUID         `json:"namespace_id"`
    Algorithm   string            `json:"algorithm,omitempty"`
    Exportable  bool              `json:"exportable,omitempty"`
    Metadata    map[string]string `json:"metadata,omitempty"`
}

// CreateBatchRequest creates multiple keys at once.
type CreateBatchRequest struct {
    Prefix      string    `json:"prefix"`
    Count       int       `json:"count"`
    NamespaceID uuid.UUID `json:"namespace_id"`
    Exportable  bool      `json:"exportable,omitempty"`
}

// Create creates a new key.
func (s *KeysService) Create(ctx context.Context, req CreateKeyRequest) (*Key, error) {
    var key Key
    err := s.client.post(ctx, "/v1/keys", req, &key)
    return &key, err
}

// CreateBatch creates multiple keys in parallel.
// This is optimized for Celestia's parallel worker pattern.
//
// Example:
//
//	keys, err := client.Keys.CreateBatch(ctx, banhbaoring.CreateBatchRequest{
//	    Prefix:      "blob-worker",
//	    Count:       4,
//	    NamespaceID: prodNamespace,
//	})
//	// Creates: blob-worker-1, blob-worker-2, blob-worker-3, blob-worker-4
func (s *KeysService) CreateBatch(ctx context.Context, req CreateBatchRequest) ([]*Key, error) {
    var resp struct {
        Keys []*Key `json:"keys"`
    }
    err := s.client.post(ctx, "/v1/keys/batch", req, &resp)
    return resp.Keys, err
}

// Get retrieves a key by ID.
func (s *KeysService) Get(ctx context.Context, keyID uuid.UUID) (*Key, error) {
    var key Key
    err := s.client.get(ctx, fmt.Sprintf("/v1/keys/%s", keyID), &key)
    return &key, err
}

// List returns all keys, optionally filtered by namespace.
func (s *KeysService) List(ctx context.Context, namespaceID *uuid.UUID) ([]*Key, error) {
    path := "/v1/keys"
    if namespaceID != nil {
        path = fmt.Sprintf("/v1/keys?namespace_id=%s", namespaceID)
    }
    var keys []*Key
    err := s.client.get(ctx, path, &keys)
    return keys, err
}

// Delete deletes a key.
func (s *KeysService) Delete(ctx context.Context, keyID uuid.UUID) error {
    return s.client.delete(ctx, fmt.Sprintf("/v1/keys/%s", keyID))
}
```

---

## 6. Sign Service

**File:** `sign.go`

```go
package banhbaoring

import (
    "context"
    "encoding/base64"
    "fmt"

    "github.com/google/uuid"
)

// SignService handles signing operations.
type SignService struct {
    client *Client
}

// SignRequest is the request for signing data.
type SignRequest struct {
    KeyID     uuid.UUID `json:"key_id"`
    Data      []byte    `json:"-"`
    Prehashed bool      `json:"prehashed,omitempty"`
}

// SignResponse is the response from a sign operation.
type SignResponse struct {
    KeyID     uuid.UUID `json:"key_id"`
    Signature []byte    `json:"-"`
    PublicKey string    `json:"public_key"`
}

// BatchSignRequest signs multiple messages in parallel.
type BatchSignRequest struct {
    Requests []SignRequest `json:"requests"`
}

// Sign signs data with a key.
func (s *SignService) Sign(ctx context.Context, keyID uuid.UUID, data []byte, prehashed bool) (*SignResponse, error) {
    req := map[string]any{
        "data":      base64.StdEncoding.EncodeToString(data),
        "prehashed": prehashed,
    }

    var resp struct {
        Signature string `json:"signature"`
        PublicKey string `json:"public_key"`
    }

    err := s.client.post(ctx, fmt.Sprintf("/v1/keys/%s/sign", keyID), req, &resp)
    if err != nil {
        return nil, err
    }

    sig, err := base64.StdEncoding.DecodeString(resp.Signature)
    if err != nil {
        return nil, fmt.Errorf("invalid signature encoding: %w", err)
    }

    return &SignResponse{
        KeyID:     keyID,
        Signature: sig,
        PublicKey: resp.PublicKey,
    }, nil
}

// SignBatch signs multiple messages in parallel.
// This is critical for Celestia's parallel blob submission pattern.
//
// Example:
//
//	results, err := client.Sign.SignBatch(ctx, banhbaoring.BatchSignRequest{
//	    Requests: []banhbaoring.SignRequest{
//	        {KeyID: worker1, Data: tx1},
//	        {KeyID: worker2, Data: tx2},
//	        {KeyID: worker3, Data: tx3},
//	        {KeyID: worker4, Data: tx4},
//	    },
//	})
//	// All 4 sign in parallel - completes in ~200ms, not 800ms!
func (s *SignService) SignBatch(ctx context.Context, req BatchSignRequest) ([]*SignResponse, error) {
    // Convert to API format
    apiReq := struct {
        Requests []map[string]any `json:"requests"`
    }{
        Requests: make([]map[string]any, len(req.Requests)),
    }

    for i, r := range req.Requests {
        apiReq.Requests[i] = map[string]any{
            "key_id":    r.KeyID,
            "data":      base64.StdEncoding.EncodeToString(r.Data),
            "prehashed": r.Prehashed,
        }
    }

    var resp struct {
        Signatures []struct {
            KeyID     uuid.UUID `json:"key_id"`
            Signature string    `json:"signature"`
            PublicKey string    `json:"public_key"`
            Error     string    `json:"error,omitempty"`
        } `json:"signatures"`
    }

    if err := s.client.post(ctx, "/v1/sign/batch", apiReq, &resp); err != nil {
        return nil, err
    }

    results := make([]*SignResponse, len(resp.Signatures))
    for i, sig := range resp.Signatures {
        if sig.Error != "" {
            continue // Skip errors, caller can check nil entries
        }
        sigBytes, _ := base64.StdEncoding.DecodeString(sig.Signature)
        results[i] = &SignResponse{
            KeyID:     sig.KeyID,
            Signature: sigBytes,
            PublicKey: sig.PublicKey,
        }
    }

    return results, nil
}
```

---

## 7. Example: Parallel Workers

**File:** `examples/parallel-workers/main.go`

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "sync"

    "github.com/banhbaoring/sdk-go"
)

func main() {
    client := banhbaoring.NewClient(os.Getenv("BANHBAORING_API_KEY"))
    ctx := context.Background()

    // Create 4 worker keys for parallel blob submission
    keys, err := client.Keys.CreateBatch(ctx, banhbaoring.CreateBatchRequest{
        Prefix:      "blob-worker",
        Count:       4,
        NamespaceID: uuid.MustParse(os.Getenv("NAMESPACE_ID")),
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Created %d worker keys:\n", len(keys))
    for _, k := range keys {
        fmt.Printf("  - %s: %s\n", k.Name, k.Address)
    }

    // Simulate parallel blob transactions
    txs := [][]byte{
        []byte("blob-tx-1"),
        []byte("blob-tx-2"),
        []byte("blob-tx-3"),
        []byte("blob-tx-4"),
    }

    // Option 1: Use SignBatch API (recommended)
    results, err := client.Sign.SignBatch(ctx, banhbaoring.BatchSignRequest{
        Requests: []banhbaoring.SignRequest{
            {KeyID: keys[0].ID, Data: txs[0]},
            {KeyID: keys[1].ID, Data: txs[1]},
            {KeyID: keys[2].ID, Data: txs[2]},
            {KeyID: keys[3].ID, Data: txs[3]},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nSigned transactions (batch):")
    for i, r := range results {
        fmt.Printf("  - TX %d: sig=%x...\n", i+1, r.Signature[:8])
    }

    // Option 2: Manual parallel goroutines
    var wg sync.WaitGroup
    sigs := make([][]byte, 4)

    for i := 0; i < 4; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            resp, err := client.Sign.Sign(ctx, keys[idx].ID, txs[idx], false)
            if err != nil {
                log.Printf("Sign error: %v", err)
                return
            }
            sigs[idx] = resp.Signature
        }(i)
    }
    wg.Wait()

    fmt.Println("\nSigned transactions (goroutines):")
    for i, sig := range sigs {
        if sig != nil {
            fmt.Printf("  - TX %d: sig=%x...\n", i+1, sig[:8])
        }
    }
}
```

---

## 8. Deliverables

| File | Description |
|------|-------------|
| `banhbaoring.go` | Main client |
| `keys.go` | Key management |
| `sign.go` | Signing with batch support |
| `orgs.go` | Organization management |
| `audit.go` | Audit log queries |
| `http.go` | HTTP client wrapper |
| `errors.go` | Error types |
| `examples/*.go` | Usage examples |
| `README.md` | Documentation |

---

## 9. Success Criteria

- [ ] NewClient works with API key
- [ ] Keys.Create/Get/List/Delete work
- [ ] Sign.Sign works
- [ ] Sign.SignBatch works in parallel
- [ ] Keys.CreateBatch creates N keys
- [ ] Errors properly typed
- [ ] Examples compile and work
- [ ] README has quickstart

---

## 10. Agent Prompt

```
You are Agent 11A - Go SDK. Create the official Go SDK for BanhBaoRing.

Read: doc/implementation/IMPL_11A_SDK_GO.md

Deliverables:
1. Client with API key auth
2. KeysService (CRUD + CreateBatch)
3. SignService (Sign + SignBatch)
4. OrgsService, AuditService
5. Error types
6. Examples (basic, parallel-workers)
7. README

Package: github.com/banhbaoring/sdk-go

Test: go test ./... -v
```

