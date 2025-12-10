# Implementation Guide: BaoClient HTTP

**Agent ID:** 01C  
**Parent:** Agent 01 (Foundation Layer)  
**Component:** HTTP Client for OpenBao API  
**Parallelizable:** ✅ Yes - Uses types from 01A, errors from 01B

---

## 1. Overview

HTTP client implementation for communicating with the OpenBao secp256k1 plugin.

### 1.1 Required Skills

| Skill            | Level    | Description                    |
| ---------------- | -------- | ------------------------------ |
| **Go**           | Advanced | HTTP clients, interfaces       |
| **HTTP/REST**    | Advanced | Connection pooling, timeouts   |
| **TLS**          | Intermediate | Certificate handling        |

### 1.2 Files to Create

```
banhbaoring/
└── bao_client.go
└── bao_client_test.go
```

---

## 2. Specifications

### 2.1 bao_client.go

```go
package banhbaoring

import (
    "bytes"
    "context"
    "crypto/tls"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

// BaoClient handles HTTP communication with OpenBao.
type BaoClient struct {
    httpClient    *http.Client
    baseURL       string
    token         string
    namespace     string
    secp256k1Path string
}

// NewBaoClient creates a new client instance.
func NewBaoClient(cfg Config) (*BaoClient, error) {
    cfg = cfg.WithDefaults()
    
    // Build TLS config
    tlsConfig := cfg.TLSConfig
    if tlsConfig == nil {
        tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
    }
    if cfg.SkipTLSVerify {
        tlsConfig.InsecureSkipVerify = true
    }
    
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
        TLSClientConfig:     tlsConfig,
    }
    
    return &BaoClient{
        httpClient:    &http.Client{Timeout: cfg.HTTPTimeout, Transport: transport},
        baseURL:       strings.TrimSuffix(cfg.BaoAddr, "/"),
        token:         cfg.BaoToken,
        namespace:     cfg.BaoNamespace,
        secp256k1Path: cfg.Secp256k1Path,
    }, nil
}

// CreateKey creates a new secp256k1 key.
func (c *BaoClient) CreateKey(ctx context.Context, name string, opts KeyOptions) (*KeyInfo, error) {
    path := fmt.Sprintf("/v1/%s/keys/%s", c.secp256k1Path, name)
    body := map[string]interface{}{"exportable": opts.Exportable}
    
    resp, err := c.post(ctx, path, body)
    if err != nil {
        return nil, WrapKeyError("create", name, err)
    }
    
    var result struct{ Data KeyInfo `json:"data"` }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, WrapKeyError("create", name, err)
    }
    return &result.Data, nil
}

// GetKey retrieves key info.
func (c *BaoClient) GetKey(ctx context.Context, name string) (*KeyInfo, error) {
    path := fmt.Sprintf("/v1/%s/keys/%s", c.secp256k1Path, name)
    resp, err := c.get(ctx, path)
    if err != nil {
        return nil, WrapKeyError("get", name, err)
    }
    
    var result struct{ Data KeyInfo `json:"data"` }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, WrapKeyError("get", name, err)
    }
    return &result.Data, nil
}

// ListKeys lists all keys.
func (c *BaoClient) ListKeys(ctx context.Context) ([]string, error) {
    path := fmt.Sprintf("/v1/%s/keys", c.secp256k1Path)
    resp, err := c.list(ctx, path)
    if err != nil {
        return nil, err
    }
    
    var result struct{ Data struct{ Keys []string `json:"keys"` } `json:"data"` }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, err
    }
    return result.Data.Keys, nil
}

// DeleteKey deletes a key.
func (c *BaoClient) DeleteKey(ctx context.Context, name string) error {
    // Enable deletion first
    configPath := fmt.Sprintf("/v1/%s/keys/%s/config", c.secp256k1Path, name)
    c.post(ctx, configPath, map[string]interface{}{"deletion_allowed": true})
    
    path := fmt.Sprintf("/v1/%s/keys/%s", c.secp256k1Path, name)
    return c.delete(ctx, path)
}

// Sign signs data and returns 64-byte Cosmos signature.
func (c *BaoClient) Sign(ctx context.Context, keyName string, data []byte, prehashed bool) ([]byte, error) {
    path := fmt.Sprintf("/v1/%s/sign/%s", c.secp256k1Path, keyName)
    body := map[string]interface{}{
        "input":         base64.StdEncoding.EncodeToString(data),
        "prehashed":     prehashed,
        "output_format": "cosmos",
    }
    if !prehashed {
        body["hash_algorithm"] = "sha256"
    }
    
    resp, err := c.post(ctx, path, body)
    if err != nil {
        return nil, WrapKeyError("sign", keyName, err)
    }
    
    var result struct{ Data SignResponse `json:"data"` }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, WrapKeyError("sign", keyName, err)
    }
    
    sig, err := base64.StdEncoding.DecodeString(result.Data.Signature)
    if err != nil {
        return nil, WrapKeyError("sign", keyName, err)
    }
    if len(sig) != 64 {
        return nil, WrapKeyError("sign", keyName, ErrInvalidSignature)
    }
    return sig, nil
}

// Health checks OpenBao status.
func (c *BaoClient) Health(ctx context.Context) error {
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/sys/health", nil)
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return ErrBaoConnection
    }
    defer resp.Body.Close()
    
    switch resp.StatusCode {
    case 200:
        return nil
    case 503:
        return ErrBaoSealed
    default:
        return ErrBaoUnavailable
    }
}

// HTTP helpers
func (c *BaoClient) get(ctx context.Context, path string) ([]byte, error) {
    return c.doRequest(ctx, http.MethodGet, path, nil)
}

func (c *BaoClient) post(ctx context.Context, path string, body interface{}) ([]byte, error) {
    return c.doRequest(ctx, http.MethodPost, path, body)
}

func (c *BaoClient) delete(ctx context.Context, path string) error {
    _, err := c.doRequest(ctx, http.MethodDelete, path, nil)
    return err
}

func (c *BaoClient) list(ctx context.Context, path string) ([]byte, error) {
    return c.doRequest(ctx, "LIST", path, nil)
}

func (c *BaoClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
    var bodyReader io.Reader
    if body != nil {
        jsonBody, _ := json.Marshal(body)
        bodyReader = bytes.NewReader(jsonBody)
    }
    
    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
    if err != nil {
        return nil, ErrBaoConnection
    }
    
    req.Header.Set("X-Vault-Token", c.token)
    req.Header.Set("Content-Type", "application/json")
    if c.namespace != "" {
        req.Header.Set("X-Vault-Namespace", c.namespace)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, ErrBaoConnection
    }
    defer resp.Body.Close()
    
    respBody, _ := io.ReadAll(resp.Body)
    
    if resp.StatusCode >= 400 {
        var errResp struct{ Errors []string `json:"errors"` }
        json.Unmarshal(respBody, &errResp)
        return nil, NewBaoError(resp.StatusCode, errResp.Errors, "")
    }
    
    return respBody, nil
}
```

---

## 3. Unit Tests

Use `httptest.NewTLSServer` to mock OpenBao responses:

```go
func TestBaoClient_CreateKey(t *testing.T) {
    server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "data": KeyInfo{Name: "test", PublicKey: "02ab..."},
        })
    }))
    defer server.Close()
    
    client, _ := NewBaoClient(Config{
        BaoAddr: server.URL, BaoToken: "test", SkipTLSVerify: true,
    })
    
    info, err := client.CreateKey(context.Background(), "test", KeyOptions{})
    require.NoError(t, err)
    assert.Equal(t, "test", info.Name)
}

func TestBaoClient_Sign(t *testing.T) {
    sig := make([]byte, 64)
    server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "data": SignResponse{Signature: base64.StdEncoding.EncodeToString(sig)},
        })
    }))
    defer server.Close()
    
    client, _ := NewBaoClient(Config{
        BaoAddr: server.URL, BaoToken: "test", SkipTLSVerify: true,
    })
    
    result, err := client.Sign(context.Background(), "key", make([]byte, 32), true)
    require.NoError(t, err)
    assert.Len(t, result, 64)
}

func TestBaoClient_ErrorHandling(t *testing.T) {
    server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(403)
        json.NewEncoder(w).Encode(map[string]interface{}{"errors": []string{"denied"}})
    }))
    defer server.Close()
    
    client, _ := NewBaoClient(Config{
        BaoAddr: server.URL, BaoToken: "bad", SkipTLSVerify: true,
    })
    
    _, err := client.GetKey(context.Background(), "test")
    assert.True(t, errors.Is(err, ErrBaoAuth))
}
```

---

## 4. Deliverables

- [ ] `bao_client.go` with all methods
- [ ] `bao_client_test.go` with mock server tests
- [ ] Connection pooling configured
- [ ] TLS properly handled
- [ ] All error paths tested

