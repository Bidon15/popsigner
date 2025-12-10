# Implementation Guide: Core Types, Errors & BaoClient

**Agent ID:** 01  
**Component:** Foundation Layer (Types, Errors, HTTP Client)  
**Parallelizable:** ✅ Yes - No dependencies on other agents

---

## 1. Overview

This agent is responsible for building the foundation layer of the BanhBao client library. This includes all shared types, custom error definitions, and the HTTP client that communicates with the OpenBao secp256k1 plugin.

### 1.1 Required Skills

| Skill            | Level        | Description                                |
| ---------------- | ------------ | ------------------------------------------ |
| **Go**           | Advanced     | Idiomatic Go, interfaces, error handling   |
| **HTTP APIs**    | Advanced     | REST clients, connection pooling, timeouts |
| **TLS/Security** | Intermediate | TLS configuration, certificate handling    |
| **JSON**         | Intermediate | Marshaling/unmarshaling, custom types      |
| **Testing**      | Advanced     | Table-driven tests, HTTP mocking           |

### 1.2 Files to Create

```
banhbaoring/
├── types.go          # Shared types and constants
├── errors.go         # Custom error types
├── bao_client.go     # HTTP client for OpenBao
├── types_test.go     # Unit tests for types
├── errors_test.go    # Unit tests for errors
└── bao_client_test.go # Unit tests for client
```

---

## 2. Detailed Specifications

### 2.1 types.go - Shared Types and Constants

```go
// Package banhbaoring provides a Cosmos SDK keyring implementation
// backed by OpenBao Transit engine for secure secp256k1 signing.
package banhbaoring

import (
    "crypto/tls"
    "time"
)

// Algorithm constants
const (
    AlgorithmSecp256k1 = "secp256k1"

    // Default configuration values
    DefaultTransitPath    = "transit"
    DefaultSecp256k1Path  = "secp256k1"
    DefaultHTTPTimeout    = 30 * time.Second
    DefaultStoreVersion   = 1
)

// Config holds configuration for BaoKeyring initialization.
type Config struct {
    // BaoAddr is the OpenBao server address (e.g., "https://bao.example.com:8200")
    BaoAddr string

    // BaoToken is the OpenBao authentication token
    BaoToken string

    // BaoNamespace is the optional OpenBao namespace (Enterprise feature)
    BaoNamespace string

    // Secp256k1Path is the mount path for the secp256k1 plugin (default: "secp256k1")
    Secp256k1Path string

    // StorePath is the path to the local metadata store file
    StorePath string

    // HTTPTimeout is the timeout for HTTP requests (default: 30s)
    HTTPTimeout time.Duration

    // TLSConfig is optional custom TLS configuration
    TLSConfig *tls.Config

    // SkipTLSVerify disables TLS certificate verification (INSECURE - dev only)
    SkipTLSVerify bool
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
    if c.BaoAddr == "" {
        return ErrMissingBaoAddr
    }
    if c.BaoToken == "" {
        return ErrMissingBaoToken
    }
    if c.StorePath == "" {
        return ErrMissingStorePath
    }
    return nil
}

// WithDefaults returns a new Config with default values applied.
func (c Config) WithDefaults() Config {
    if c.Secp256k1Path == "" {
        c.Secp256k1Path = DefaultSecp256k1Path
    }
    if c.HTTPTimeout == 0 {
        c.HTTPTimeout = DefaultHTTPTimeout
    }
    return c
}

// KeyMetadata contains locally stored key information.
// This is persisted to disk and does NOT contain private key material.
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
    Source      string    `json:"source"` // "generated" or "imported"
}

// KeyInfo represents public key information returned from OpenBao.
type KeyInfo struct {
    Name       string    `json:"name"`
    PublicKey  string    `json:"public_key"`     // Hex-encoded compressed public key
    Address    string    `json:"address"`        // Bech32 Cosmos address
    Exportable bool      `json:"exportable"`
    CreatedAt  time.Time `json:"created_at"`
}

// SignRequest represents a signing request to OpenBao.
type SignRequest struct {
    Input        string `json:"input"`         // Base64-encoded data
    Prehashed    bool   `json:"prehashed"`     // If true, input is already hashed
    HashAlgo     string `json:"hash_algorithm,omitempty"` // sha256, keccak256
    OutputFormat string `json:"output_format,omitempty"`  // cosmos, der, ethereum
}

// SignResponse represents the signing response from OpenBao.
type SignResponse struct {
    Signature  string `json:"signature"`   // Base64-encoded signature
    PublicKey  string `json:"public_key"`  // Hex-encoded public key
    KeyVersion int    `json:"key_version"`
}

// KeyOptions configures key creation behavior.
type KeyOptions struct {
    Exportable bool   // Allow future export from OpenBao
    Derived    bool   // Use derived key (future)
}

// StoreData represents the persisted metadata store format.
type StoreData struct {
    Version int                     `json:"version"`
    Keys    map[string]*KeyMetadata `json:"keys"`
}
```

### 2.2 errors.go - Custom Error Types

```go
package banhbaoring

import (
    "errors"
    "fmt"
)

// Sentinel errors for common failure conditions.
var (
    // Configuration errors
    ErrMissingBaoAddr   = errors.New("banhbaoring: BaoAddr is required")
    ErrMissingBaoToken  = errors.New("banhbaoring: BaoToken is required")
    ErrMissingStorePath = errors.New("banhbaoring: StorePath is required")

    // Key errors
    ErrKeyNotFound    = errors.New("banhbaoring: key not found")
    ErrKeyExists      = errors.New("banhbaoring: key already exists")
    ErrKeyNotExportable = errors.New("banhbaoring: key is not exportable")

    // OpenBao connection errors
    ErrBaoConnection  = errors.New("banhbaoring: failed to connect to OpenBao")
    ErrBaoAuth        = errors.New("banhbaoring: OpenBao authentication failed")
    ErrBaoSealed      = errors.New("banhbaoring: OpenBao is sealed")
    ErrBaoUnavailable = errors.New("banhbaoring: OpenBao is unavailable")

    // Operation errors
    ErrSigningFailed     = errors.New("banhbaoring: signing operation failed")
    ErrInvalidSignature  = errors.New("banhbaoring: invalid signature format")
    ErrUnsupportedAlgo   = errors.New("banhbaoring: unsupported algorithm")
    ErrStorePersist      = errors.New("banhbaoring: failed to persist metadata")
    ErrStoreCorrupted    = errors.New("banhbaoring: metadata store is corrupted")
)

// BaoError represents an error returned by OpenBao API.
type BaoError struct {
    StatusCode int
    Errors     []string
    RequestID  string
}

// Error implements the error interface.
func (e *BaoError) Error() string {
    if len(e.Errors) == 0 {
        return fmt.Sprintf("OpenBao error (HTTP %d)", e.StatusCode)
    }
    return fmt.Sprintf("OpenBao error (HTTP %d): %s", e.StatusCode, e.Errors[0])
}

// Is allows errors.Is to match against sentinel errors.
func (e *BaoError) Is(target error) bool {
    switch e.StatusCode {
    case 403:
        return errors.Is(target, ErrBaoAuth)
    case 404:
        return errors.Is(target, ErrKeyNotFound)
    case 503:
        return errors.Is(target, ErrBaoSealed)
    default:
        return false
    }
}

// NewBaoError creates a new BaoError from HTTP response details.
func NewBaoError(statusCode int, errors []string, requestID string) *BaoError {
    return &BaoError{
        StatusCode: statusCode,
        Errors:     errors,
        RequestID:  requestID,
    }
}

// KeyError wraps an error with key context.
type KeyError struct {
    KeyName string
    Op      string // Operation that failed
    Err     error
}

// Error implements the error interface.
func (e *KeyError) Error() string {
    return fmt.Sprintf("%s key %q: %v", e.Op, e.KeyName, e.Err)
}

// Unwrap returns the underlying error.
func (e *KeyError) Unwrap() error {
    return e.Err
}

// WrapKeyError wraps an error with key operation context.
func WrapKeyError(op, keyName string, err error) error {
    if err == nil {
        return nil
    }
    return &KeyError{
        KeyName: keyName,
        Op:      op,
        Err:     err,
    }
}
```

### 2.3 bao_client.go - HTTP Client for OpenBao

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

// BaoClient handles HTTP communication with the OpenBao secp256k1 plugin.
type BaoClient struct {
    httpClient    *http.Client
    baseURL       string
    token         string
    namespace     string
    secp256k1Path string
}

// NewBaoClient creates a new OpenBao client.
func NewBaoClient(cfg Config) (*BaoClient, error) {
    cfg = cfg.WithDefaults()

    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    // Configure TLS
    tlsConfig := cfg.TLSConfig
    if tlsConfig == nil {
        tlsConfig = &tls.Config{
            MinVersion: tls.VersionTLS12,
        }
    }
    if cfg.SkipTLSVerify {
        tlsConfig.InsecureSkipVerify = true
    }

    // Create HTTP client with connection pooling
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
        TLSClientConfig:     tlsConfig,
    }

    httpClient := &http.Client{
        Timeout:   cfg.HTTPTimeout,
        Transport: transport,
    }

    // Normalize base URL
    baseURL := strings.TrimSuffix(cfg.BaoAddr, "/")

    return &BaoClient{
        httpClient:    httpClient,
        baseURL:       baseURL,
        token:         cfg.BaoToken,
        namespace:     cfg.BaoNamespace,
        secp256k1Path: cfg.Secp256k1Path,
    }, nil
}

// CreateKey creates a new secp256k1 key in OpenBao.
func (c *BaoClient) CreateKey(ctx context.Context, name string, opts KeyOptions) (*KeyInfo, error) {
    path := fmt.Sprintf("/v1/%s/keys/%s", c.secp256k1Path, name)

    body := map[string]interface{}{
        "exportable": opts.Exportable,
    }

    resp, err := c.post(ctx, path, body)
    if err != nil {
        return nil, WrapKeyError("create", name, err)
    }

    var result struct {
        Data KeyInfo `json:"data"`
    }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, WrapKeyError("create", name, fmt.Errorf("failed to parse response: %w", err))
    }

    return &result.Data, nil
}

// GetKey retrieves key information from OpenBao.
func (c *BaoClient) GetKey(ctx context.Context, name string) (*KeyInfo, error) {
    path := fmt.Sprintf("/v1/%s/keys/%s", c.secp256k1Path, name)

    resp, err := c.get(ctx, path)
    if err != nil {
        return nil, WrapKeyError("get", name, err)
    }

    var result struct {
        Data KeyInfo `json:"data"`
    }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, WrapKeyError("get", name, fmt.Errorf("failed to parse response: %w", err))
    }

    return &result.Data, nil
}

// ListKeys lists all keys in the secp256k1 engine.
func (c *BaoClient) ListKeys(ctx context.Context) ([]string, error) {
    path := fmt.Sprintf("/v1/%s/keys", c.secp256k1Path)

    resp, err := c.list(ctx, path)
    if err != nil {
        return nil, fmt.Errorf("list keys: %w", err)
    }

    var result struct {
        Data struct {
            Keys []string `json:"keys"`
        } `json:"data"`
    }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, fmt.Errorf("list keys: failed to parse response: %w", err)
    }

    return result.Data.Keys, nil
}

// DeleteKey deletes a key from OpenBao.
func (c *BaoClient) DeleteKey(ctx context.Context, name string) error {
    // First enable deletion
    configPath := fmt.Sprintf("/v1/%s/keys/%s/config", c.secp256k1Path, name)
    _, err := c.post(ctx, configPath, map[string]interface{}{
        "deletion_allowed": true,
    })
    if err != nil {
        return WrapKeyError("delete", name, fmt.Errorf("failed to enable deletion: %w", err))
    }

    // Then delete
    path := fmt.Sprintf("/v1/%s/keys/%s", c.secp256k1Path, name)
    if err := c.delete(ctx, path); err != nil {
        return WrapKeyError("delete", name, err)
    }

    return nil
}

// Sign signs data using the specified key.
// If prehashed is true, data is expected to be a 32-byte hash.
// Returns the signature in Cosmos compact format (R || S, 64 bytes).
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

    var result struct {
        Data SignResponse `json:"data"`
    }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, WrapKeyError("sign", keyName, fmt.Errorf("failed to parse response: %w", err))
    }

    sig, err := base64.StdEncoding.DecodeString(result.Data.Signature)
    if err != nil {
        return nil, WrapKeyError("sign", keyName, fmt.Errorf("failed to decode signature: %w", err))
    }

    // Validate signature length
    if len(sig) != 64 {
        return nil, WrapKeyError("sign", keyName, fmt.Errorf("%w: expected 64 bytes, got %d", ErrInvalidSignature, len(sig)))
    }

    return sig, nil
}

// Verify verifies a signature against the provided data.
func (c *BaoClient) Verify(ctx context.Context, keyName string, data, signature []byte, prehashed bool) (bool, error) {
    path := fmt.Sprintf("/v1/%s/verify/%s", c.secp256k1Path, keyName)

    body := map[string]interface{}{
        "input":     base64.StdEncoding.EncodeToString(data),
        "signature": base64.StdEncoding.EncodeToString(signature),
        "prehashed": prehashed,
    }

    resp, err := c.post(ctx, path, body)
    if err != nil {
        return false, WrapKeyError("verify", keyName, err)
    }

    var result struct {
        Data struct {
            Valid bool `json:"valid"`
        } `json:"data"`
    }
    if err := json.Unmarshal(resp, &result); err != nil {
        return false, WrapKeyError("verify", keyName, fmt.Errorf("failed to parse response: %w", err))
    }

    return result.Data.Valid, nil
}

// ImportKey imports an existing key into OpenBao.
func (c *BaoClient) ImportKey(ctx context.Context, name string, wrappedKey []byte, exportable bool) (*KeyInfo, error) {
    path := fmt.Sprintf("/v1/%s/keys/%s/import", c.secp256k1Path, name)

    body := map[string]interface{}{
        "ciphertext": base64.StdEncoding.EncodeToString(wrappedKey),
        "exportable": exportable,
    }

    resp, err := c.post(ctx, path, body)
    if err != nil {
        return nil, WrapKeyError("import", name, err)
    }

    var result struct {
        Data KeyInfo `json:"data"`
    }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, WrapKeyError("import", name, fmt.Errorf("failed to parse response: %w", err))
    }

    return &result.Data, nil
}

// ExportKey exports a key from OpenBao (only works for exportable keys).
func (c *BaoClient) ExportKey(ctx context.Context, name string) ([]byte, error) {
    path := fmt.Sprintf("/v1/%s/export/%s", c.secp256k1Path, name)

    resp, err := c.get(ctx, path)
    if err != nil {
        return nil, WrapKeyError("export", name, err)
    }

    var result struct {
        Data struct {
            Keys map[string]string `json:"keys"`
        } `json:"data"`
    }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, WrapKeyError("export", name, fmt.Errorf("failed to parse response: %w", err))
    }

    // Get the latest version
    keyData, ok := result.Data.Keys["1"]
    if !ok {
        return nil, WrapKeyError("export", name, errors.New("no key version found"))
    }

    return base64.StdEncoding.DecodeString(keyData)
}

// GetWrappingKey retrieves the public key used for wrapping imported keys.
func (c *BaoClient) GetWrappingKey(ctx context.Context) ([]byte, error) {
    path := fmt.Sprintf("/v1/%s/wrapping_key", c.secp256k1Path)

    resp, err := c.get(ctx, path)
    if err != nil {
        return nil, fmt.Errorf("get wrapping key: %w", err)
    }

    var result struct {
        Data struct {
            PublicKey string `json:"public_key"`
        } `json:"data"`
    }
    if err := json.Unmarshal(resp, &result); err != nil {
        return nil, fmt.Errorf("get wrapping key: failed to parse response: %w", err)
    }

    return []byte(result.Data.PublicKey), nil
}

// Health checks OpenBao health status.
func (c *BaoClient) Health(ctx context.Context) error {
    path := "/v1/sys/health"

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
    if err != nil {
        return fmt.Errorf("%w: %v", ErrBaoConnection, err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("%w: %v", ErrBaoConnection, err)
    }
    defer resp.Body.Close()

    switch resp.StatusCode {
    case 200:
        return nil // Healthy
    case 429:
        return ErrBaoSealed
    case 472, 473:
        return ErrBaoSealed
    case 501:
        return ErrBaoUnavailable
    case 503:
        return ErrBaoSealed
    default:
        return fmt.Errorf("%w: unexpected status %d", ErrBaoConnection, resp.StatusCode)
    }
}

// HTTP helper methods

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
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal request body: %w", err)
        }
        bodyReader = bytes.NewReader(jsonBody)
    }

    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
    if err != nil {
        return nil, fmt.Errorf("%w: failed to create request: %v", ErrBaoConnection, err)
    }

    // Set headers
    req.Header.Set("X-Vault-Token", c.token)
    req.Header.Set("Content-Type", "application/json")
    if c.namespace != "" {
        req.Header.Set("X-Vault-Namespace", c.namespace)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrBaoConnection, err)
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response body: %w", err)
    }

    // Handle error responses
    if resp.StatusCode >= 400 {
        var errResp struct {
            Errors    []string `json:"errors"`
            RequestID string   `json:"request_id"`
        }
        json.Unmarshal(respBody, &errResp) // Ignore parse errors

        return nil, NewBaoError(resp.StatusCode, errResp.Errors, errResp.RequestID)
    }

    return respBody, nil
}
```

---

## 3. Unit Test Requirements

### 3.1 types_test.go

Test cases for types:

```go
func TestConfig_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr error
    }{
        {
            name: "valid config",
            config: Config{
                BaoAddr:   "https://localhost:8200",
                BaoToken:  "test-token",
                StorePath: "/tmp/store.json",
            },
            wantErr: nil,
        },
        {
            name: "missing BaoAddr",
            config: Config{
                BaoToken:  "test-token",
                StorePath: "/tmp/store.json",
            },
            wantErr: ErrMissingBaoAddr,
        },
        {
            name: "missing BaoToken",
            config: Config{
                BaoAddr:   "https://localhost:8200",
                StorePath: "/tmp/store.json",
            },
            wantErr: ErrMissingBaoToken,
        },
        {
            name: "missing StorePath",
            config: Config{
                BaoAddr:  "https://localhost:8200",
                BaoToken: "test-token",
            },
            wantErr: ErrMissingStorePath,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

func TestConfig_WithDefaults(t *testing.T) {
    cfg := Config{
        BaoAddr:   "https://localhost:8200",
        BaoToken:  "test",
        StorePath: "/tmp/store.json",
    }

    cfg = cfg.WithDefaults()

    if cfg.Secp256k1Path != DefaultSecp256k1Path {
        t.Errorf("expected Secp256k1Path %q, got %q", DefaultSecp256k1Path, cfg.Secp256k1Path)
    }
    if cfg.HTTPTimeout != DefaultHTTPTimeout {
        t.Errorf("expected HTTPTimeout %v, got %v", DefaultHTTPTimeout, cfg.HTTPTimeout)
    }
}
```

### 3.2 errors_test.go

Test cases for errors:

```go
func TestBaoError_Error(t *testing.T) {
    tests := []struct {
        name     string
        err      *BaoError
        expected string
    }{
        {
            name: "with errors",
            err: &BaoError{
                StatusCode: 403,
                Errors:     []string{"permission denied"},
            },
            expected: "OpenBao error (HTTP 403): permission denied",
        },
        {
            name: "no errors",
            err: &BaoError{
                StatusCode: 500,
            },
            expected: "OpenBao error (HTTP 500)",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := tt.err.Error(); got != tt.expected {
                t.Errorf("Error() = %q, want %q", got, tt.expected)
            }
        })
    }
}

func TestBaoError_Is(t *testing.T) {
    tests := []struct {
        name   string
        err    *BaoError
        target error
        want   bool
    }{
        {
            name:   "403 is ErrBaoAuth",
            err:    &BaoError{StatusCode: 403},
            target: ErrBaoAuth,
            want:   true,
        },
        {
            name:   "404 is ErrKeyNotFound",
            err:    &BaoError{StatusCode: 404},
            target: ErrKeyNotFound,
            want:   true,
        },
        {
            name:   "503 is ErrBaoSealed",
            err:    &BaoError{StatusCode: 503},
            target: ErrBaoSealed,
            want:   true,
        },
        {
            name:   "403 is not ErrKeyNotFound",
            err:    &BaoError{StatusCode: 403},
            target: ErrKeyNotFound,
            want:   false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := errors.Is(tt.err, tt.target); got != tt.want {
                t.Errorf("errors.Is() = %v, want %v", got, tt.want)
            }
        })
    }
}

func TestKeyError(t *testing.T) {
    err := WrapKeyError("sign", "my-key", ErrSigningFailed)

    if !strings.Contains(err.Error(), "my-key") {
        t.Error("expected error to contain key name")
    }
    if !strings.Contains(err.Error(), "sign") {
        t.Error("expected error to contain operation")
    }
    if !errors.Is(err, ErrSigningFailed) {
        t.Error("expected error to wrap ErrSigningFailed")
    }
}
```

### 3.3 bao_client_test.go

Test the HTTP client with mocked server:

```go
func TestBaoClient_CreateKey(t *testing.T) {
    // Mock server
    server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.Method != http.MethodPost {
            t.Errorf("expected POST, got %s", r.Method)
        }
        if !strings.Contains(r.URL.Path, "/secp256k1/keys/test-key") {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        if r.Header.Get("X-Vault-Token") != "test-token" {
            t.Error("missing or wrong token header")
        }

        // Return response
        json.NewEncoder(w).Encode(map[string]interface{}{
            "data": KeyInfo{
                Name:      "test-key",
                PublicKey: "02" + strings.Repeat("ab", 32),
                Address:   "celestia1test...",
            },
        })
    }))
    defer server.Close()

    client, err := NewBaoClient(Config{
        BaoAddr:       server.URL,
        BaoToken:      "test-token",
        StorePath:     "/tmp/test.json",
        SkipTLSVerify: true,
    })
    if err != nil {
        t.Fatal(err)
    }

    keyInfo, err := client.CreateKey(context.Background(), "test-key", KeyOptions{})
    if err != nil {
        t.Fatal(err)
    }

    if keyInfo.Name != "test-key" {
        t.Errorf("expected name test-key, got %s", keyInfo.Name)
    }
}

func TestBaoClient_Sign(t *testing.T) {
    expectedSig := make([]byte, 64)
    for i := range expectedSig {
        expectedSig[i] = byte(i)
    }

    server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]interface{}{
            "data": SignResponse{
                Signature:  base64.StdEncoding.EncodeToString(expectedSig),
                PublicKey:  "02" + strings.Repeat("ab", 32),
                KeyVersion: 1,
            },
        })
    }))
    defer server.Close()

    client, _ := NewBaoClient(Config{
        BaoAddr:       server.URL,
        BaoToken:      "test-token",
        StorePath:     "/tmp/test.json",
        SkipTLSVerify: true,
    })

    hash := make([]byte, 32)
    sig, err := client.Sign(context.Background(), "test-key", hash, true)
    if err != nil {
        t.Fatal(err)
    }

    if !bytes.Equal(sig, expectedSig) {
        t.Error("signature mismatch")
    }
}

func TestBaoClient_ErrorHandling(t *testing.T) {
    server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusForbidden)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "errors": []string{"permission denied"},
        })
    }))
    defer server.Close()

    client, _ := NewBaoClient(Config{
        BaoAddr:       server.URL,
        BaoToken:      "bad-token",
        StorePath:     "/tmp/test.json",
        SkipTLSVerify: true,
    })

    _, err := client.GetKey(context.Background(), "test-key")
    if !errors.Is(err, ErrBaoAuth) {
        t.Errorf("expected ErrBaoAuth, got %v", err)
    }
}

func TestBaoClient_Health(t *testing.T) {
    tests := []struct {
        name       string
        statusCode int
        wantErr    error
    }{
        {"healthy", 200, nil},
        {"sealed", 503, ErrBaoSealed},
        {"unavailable", 501, ErrBaoUnavailable},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.statusCode)
            }))
            defer server.Close()

            client, _ := NewBaoClient(Config{
                BaoAddr:       server.URL,
                BaoToken:      "test",
                StorePath:     "/tmp/test.json",
                SkipTLSVerify: true,
            })

            err := client.Health(context.Background())
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("Health() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## 4. Success Criteria

### 4.1 Functional Requirements

- [ ] `Config.Validate()` correctly validates all required fields
- [ ] `Config.WithDefaults()` applies default values
- [ ] All sentinel errors are defined and properly categorized
- [ ] `BaoError` implements error unwrapping correctly
- [ ] `KeyError` wraps errors with context
- [ ] `BaoClient` successfully creates keys
- [ ] `BaoClient` successfully retrieves keys
- [ ] `BaoClient` successfully lists keys
- [ ] `BaoClient` successfully deletes keys
- [ ] `BaoClient` successfully signs data (returns 64-byte Cosmos format)
- [ ] `BaoClient` successfully verifies signatures
- [ ] `BaoClient` handles import/export operations
- [ ] `BaoClient` reports health status correctly
- [ ] All HTTP errors are properly wrapped with `BaoError`

### 4.2 Non-Functional Requirements

- [ ] HTTP client uses connection pooling
- [ ] TLS configuration is properly applied
- [ ] Timeouts are configurable
- [ ] Context cancellation is respected
- [ ] No sensitive data (tokens) in error messages

### 4.3 Test Coverage

- [ ] > 80% code coverage
- [ ] All error paths tested
- [ ] HTTP client tested with mock server
- [ ] Table-driven tests for validation and error mapping

---

## 5. Interface Contracts

Other agents depend on these interfaces. Do NOT change these signatures without coordination:

```go
// BaoClientInterface defines the contract for Agent 4 (BaoKeyring)
type BaoClientInterface interface {
    CreateKey(ctx context.Context, name string, opts KeyOptions) (*KeyInfo, error)
    GetKey(ctx context.Context, name string) (*KeyInfo, error)
    ListKeys(ctx context.Context) ([]string, error)
    DeleteKey(ctx context.Context, name string) error
    Sign(ctx context.Context, keyName string, data []byte, prehashed bool) ([]byte, error)
    Verify(ctx context.Context, keyName string, data, signature []byte, prehashed bool) (bool, error)
    ImportKey(ctx context.Context, name string, wrappedKey []byte, exportable bool) (*KeyInfo, error)
    ExportKey(ctx context.Context, name string) ([]byte, error)
    GetWrappingKey(ctx context.Context) ([]byte, error)
    Health(ctx context.Context) error
}
```

---

## 6. Dependencies

### 6.1 External Dependencies

```go
// go.mod additions
require (
    // Standard library only for this component
)
```

### 6.2 Internal Dependencies

None - this is the foundation layer.

---

## 7. Deliverables Checklist

- [ ] `types.go` - All types implemented
- [ ] `errors.go` - All errors implemented
- [ ] `bao_client.go` - HTTP client implemented
- [ ] `types_test.go` - Unit tests passing
- [ ] `errors_test.go` - Unit tests passing
- [ ] `bao_client_test.go` - Unit tests passing
- [ ] All tests pass: `go test ./...`
- [ ] No linter errors: `golangci-lint run`
- [ ] Code reviewed for sensitive data exposure
