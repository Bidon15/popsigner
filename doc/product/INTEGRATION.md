# Celestia Integration Guide

This document describes how to integrate the BaoKeyring with Celestia's Go client for secure transaction signing using OpenBao Transit.

---

## 1. Overview

The BaoKeyring integrates with Celestia's client architecture at the keyring level, replacing local key storage with remote OpenBao Transit signing.

### 1.1 Integration Points

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Celestia Application                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    client.Client                              │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌───────────────┐    │  │
│  │  │  Blob   │  │  State  │  │ Header  │  │    ...        │    │  │
│  │  └────┬────┘  └────┬────┘  └─────────┘  └───────────────┘    │  │
│  │       │            │                                          │  │
│  │       │            ▼                                          │  │
│  │       │     ┌──────────────┐                                  │  │
│  │       │     │  tx.Factory  │◀────── Signing                   │  │
│  │       │     └──────┬───────┘                                  │  │
│  │       │            │                                          │  │
│  │       │            ▼                                          │  │
│  │       │     ┌──────────────┐     ┌─────────────────────────┐ │  │
│  │       │     │   Keyring    │────▶│     BaoKeyring          │ │  │
│  │       │     │  (interface) │     │  (implementation)       │ │  │
│  │       │     └──────────────┘     └────────────┬────────────┘ │  │
│  │       │                                       │               │  │
│  └───────┼───────────────────────────────────────┼───────────────┘  │
│          │                                       │                  │
└──────────┼───────────────────────────────────────┼──────────────────┘
           │                                       │
           ▼                                       ▼
    ┌──────────────┐                      ┌──────────────────┐
    │ Celestia DA  │                      │    OpenBao       │
    │ Bridge Node  │                      │  Transit Engine  │
    └──────────────┘                      └──────────────────┘
```

---

## 2. Prerequisites

### 2.1 Celestia Node

- **celestia-node v0.28.4** or higher (MINIMUM)
- **celestia-app v6.4.0** or higher (MINIMUM)
- Running Bridge or Light node for DA operations
- Running Core/Consensus node for transaction broadcasting

### 2.2 OpenBao Server

- OpenBao server with Transit engine enabled
- Authentication token with appropriate permissions
- TLS configured for production use

### 2.3 Go Dependencies

**⚠️ CRITICAL:** You MUST use Celestia's forks of cosmos-sdk and tendermint!

```go
// go.mod
module your-app

go 1.22

require (
    github.com/celestiaorg/celestia-app/v4 v4.0.0 // minimum v6.4.0 for production
    github.com/celestiaorg/celestia-node v0.28.4  // minimum v0.28.4
    github.com/cosmos/cosmos-sdk v0.50.13
    github.com/Bidon15/banhbaoring v0.1.0
)

// CRITICAL: Celestia uses forked versions of cosmos-sdk and tendermint
replace (
    // Use Celestia's cosmos-sdk fork
    github.com/cosmos/cosmos-sdk => github.com/celestiaorg/cosmos-sdk v1.25.0-sdk-v0.50.6
    
    // Use Celestia's tendermint fork (celestia-core)
    github.com/tendermint/tendermint => github.com/celestiaorg/celestia-core v1.41.0-tm-v0.34.29
    github.com/cometbft/cometbft => github.com/celestiaorg/celestia-core v1.41.0-tm-v0.34.29
    
    // Use Celestia's IBC fork
    github.com/cosmos/ibc-go/v8 => github.com/celestiaorg/ibc-go/v8 v8.5.1
    
    // Required protobuf replacement
    github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
    
    // LevelDB fix
    github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
)
```

**Note:** Import paths in code remain standard (`github.com/cosmos/cosmos-sdk/crypto/keyring`) but resolve to Celestia's fork via `replace` directives.

---

## 3. Configuration

### 3.1 BaoKeyring Configuration

```go
import (
    "github.com/Bidon15/banhbaoring"
)

// Configure the BaoKeyring
baoConfig := banhbaoring.Config{
    // OpenBao connection
    BaoAddr:      "https://bao.example.com:8200",
    BaoToken:     os.Getenv("BAO_TOKEN"),
    BaoNamespace: "",                    // Optional: Enterprise namespace
    TransitPath:  "transit",             // Transit engine mount path
    
    // Local metadata storage
    StorePath:    "/var/lib/celestia/keyring-bao/metadata.json",
    
    // HTTP settings
    HTTPTimeout:  30 * time.Second,
    TLSConfig:    nil,                   // Use system CA pool
}
```

### 3.2 Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `BAO_TOKEN` | OpenBao authentication token | Yes |
| `BAO_ADDR` | OpenBao server address | Yes |
| `BAO_NAMESPACE` | OpenBao namespace | No |
| `BAO_CACERT` | Path to CA certificate | Production |

### 3.3 TLS Configuration

```go
import (
    "crypto/tls"
    "crypto/x509"
    "os"
)

// Load CA certificate for OpenBao
func loadTLSConfig(caPath string) (*tls.Config, error) {
    caCert, err := os.ReadFile(caPath)
    if err != nil {
        return nil, err
    }
    
    caCertPool := x509.NewCertPool()
    if !caCertPool.AppendCertsFromPEM(caCert) {
        return nil, errors.New("failed to parse CA certificate")
    }
    
    return &tls.Config{
        RootCAs:    caCertPool,
        MinVersion: tls.VersionTLS12,
    }, nil
}

// Use in config
tlsConfig, err := loadTLSConfig("/etc/ssl/certs/bao-ca.pem")
if err != nil {
    log.Fatal(err)
}

baoConfig.TLSConfig = tlsConfig
```

---

## 4. Client Integration

### 4.1 Creating the BaoKeyring

```go
import (
    "context"
    "github.com/Bidon15/banhbaoring"
)

func createKeyring(ctx context.Context, cfg banhbaoring.Config) (*banhbaoring.BaoKeyring, error) {
    kr, err := banhbaoring.New(ctx, cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create BaoKeyring: %w", err)
    }
    
    return kr, nil
}
```

### 4.2 Creating a New Key

```go
func createKey(ctx context.Context, kr *banhbaoring.BaoKeyring, keyName string) error {
    // Create a new secp256k1 key in OpenBao
    record, err := kr.NewAccount(
        keyName,           // uid
        "",                // mnemonic (not used for OpenBao)
        "",                // bip39 passphrase (not used)
        "",                // hd path (not used)
        keyring.Secp256k1, // algorithm
    )
    if err != nil {
        return fmt.Errorf("failed to create key: %w", err)
    }
    
    addr, err := record.GetAddress()
    if err != nil {
        return err
    }
    
    fmt.Printf("Created key: %s\n", keyName)
    fmt.Printf("Address: %s\n", addr.String())
    
    return nil
}
```

### 4.3 Integrating with Celestia Client

```go
import (
    "context"
    
    "github.com/celestiaorg/celestia-node/api/client"
    "github.com/Bidon15/banhbaoring"
)

func createCelestiaClient(ctx context.Context) (*client.Client, error) {
    // Create BaoKeyring
    baoConfig := banhbaoring.Config{
        BaoAddr:     os.Getenv("BAO_ADDR"),
        BaoToken:    os.Getenv("BAO_TOKEN"),
        TransitPath: "transit",
        StorePath:   "./keyring-metadata.json",
        HTTPTimeout: 30 * time.Second,
    }
    
    kr, err := banhbaoring.New(ctx, baoConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create keyring: %w", err)
    }
    
    // Configure Celestia client with BaoKeyring
    clientCfg := client.Config{
        ReadConfig: client.ReadConfig{
            BridgeDAAddr: "http://localhost:26658",
            DAAuthToken:  os.Getenv("CELESTIA_NODE_AUTH_TOKEN"),
            EnableDATLS:  false,
        },
        SubmitConfig: client.SubmitConfig{
            DefaultKeyName: "my-celestia-key",
            Network:        "mocha-4",
            CoreGRPCConfig: client.CoreGRPCConfig{
                Addr:       "celestia-consensus.example.com:9090",
                TLSEnabled: true,
            },
        },
    }
    
    // Create client with custom keyring
    celestiaClient, err := client.NewWithKeyring(ctx, clientCfg, kr)
    if err != nil {
        return nil, fmt.Errorf("failed to create Celestia client: %w", err)
    }
    
    return celestiaClient, nil
}
```

---

## 5. Transaction Signing Flow

### 5.1 Signing a Blob Submission

```go
import (
    "context"
    
    "github.com/celestiaorg/celestia-node/api/client"
    "github.com/celestiaorg/celestia-node/blob"
    libshare "github.com/celestiaorg/go-square/v2/share"
)

func submitBlob(ctx context.Context, c *client.Client, data []byte) (uint64, error) {
    // Create namespace
    namespace, err := libshare.NewV0Namespace([]byte("example-ns"))
    if err != nil {
        return 0, fmt.Errorf("failed to create namespace: %w", err)
    }
    
    // Create blob
    b, err := blob.NewBlob(libshare.ShareVersionZero, namespace, data, nil)
    if err != nil {
        return 0, fmt.Errorf("failed to create blob: %w", err)
    }
    
    // Submit blob (signing happens via BaoKeyring)
    // The client internally:
    // 1. Builds the MsgPayForBlobs transaction
    // 2. Gets sign bytes
    // 3. Calls keyring.Sign() which uses OpenBao
    // 4. Broadcasts the signed transaction
    height, err := c.Blob.Submit(ctx, []*blob.Blob{b}, nil)
    if err != nil {
        return 0, fmt.Errorf("failed to submit blob: %w", err)
    }
    
    return height, nil
}
```

### 5.2 Manual Transaction Signing

For more control over the signing process:

```go
import (
    "github.com/cosmos/cosmos-sdk/client/tx"
    "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

func signTransaction(
    ctx context.Context,
    kr keyring.Keyring,
    txBuilder client.TxBuilder,
    keyName string,
) error {
    // Get key record
    record, err := kr.Key(keyName)
    if err != nil {
        return fmt.Errorf("key not found: %w", err)
    }
    
    pubKey, err := record.GetPubKey()
    if err != nil {
        return err
    }
    
    // Get sign bytes
    signBytes, err := txBuilder.GetSignBytes()
    if err != nil {
        return fmt.Errorf("failed to get sign bytes: %w", err)
    }
    
    // Sign using BaoKeyring (calls OpenBao Transit)
    signature, _, err := kr.Sign(keyName, signBytes, signing.SignMode_SIGN_MODE_DIRECT)
    if err != nil {
        return fmt.Errorf("signing failed: %w", err)
    }
    
    // Set signature on transaction
    sigData := signing.SingleSignatureData{
        SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
        Signature: signature,
    }
    
    sig := signing.SignatureV2{
        PubKey:   pubKey,
        Data:     &sigData,
        Sequence: 0, // Get from account
    }
    
    err = txBuilder.SetSignatures(sig)
    if err != nil {
        return fmt.Errorf("failed to set signature: %w", err)
    }
    
    return nil
}
```

---

## 6. Complete Example

### 6.1 Full Application Example

```go
// example/main.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
    
    "github.com/celestiaorg/celestia-node/api/client"
    "github.com/celestiaorg/celestia-node/blob"
    libshare "github.com/celestiaorg/go-square/v2/share"
    
    "github.com/Bidon15/banhbaoring"
)

func main() {
    ctx := context.Background()
    
    // Step 1: Create BaoKeyring
    kr, err := createBaoKeyring(ctx)
    if err != nil {
        log.Fatalf("Failed to create keyring: %v", err)
    }
    
    // Step 2: Ensure key exists
    keyName := "celestia-signer"
    if err := ensureKeyExists(ctx, kr, keyName); err != nil {
        log.Fatalf("Failed to ensure key: %v", err)
    }
    
    // Step 3: Create Celestia client
    celestiaClient, err := createCelestiaClient(ctx, kr, keyName)
    if err != nil {
        log.Fatalf("Failed to create Celestia client: %v", err)
    }
    
    // Step 4: Submit a blob
    data := []byte("Hello from OpenBao-signed transaction!")
    height, err := submitBlob(ctx, celestiaClient, data)
    if err != nil {
        log.Fatalf("Failed to submit blob: %v", err)
    }
    
    fmt.Printf("Blob submitted at height: %d\n", height)
    
    // Step 5: Retrieve and verify
    retrievedBlob, err := retrieveBlob(ctx, celestiaClient, height)
    if err != nil {
        log.Fatalf("Failed to retrieve blob: %v", err)
    }
    
    fmt.Printf("Retrieved data: %s\n", string(retrievedBlob.Data()))
}

func createBaoKeyring(ctx context.Context) (*banhbaoring.BaoKeyring, error) {
    cfg := banhbaoring.Config{
        BaoAddr:     getEnvOrDefault("BAO_ADDR", "http://localhost:8200"),
        BaoToken:    os.Getenv("BAO_TOKEN"),
        TransitPath: "transit",
        StorePath:   getEnvOrDefault("KEYRING_PATH", "./keyring-metadata.json"),
        HTTPTimeout: 30 * time.Second,
    }
    
    if cfg.BaoToken == "" {
        return nil, fmt.Errorf("BAO_TOKEN environment variable required")
    }
    
    return banhbaoring.New(ctx, cfg)
}

func ensureKeyExists(ctx context.Context, kr *banhbaoring.BaoKeyring, keyName string) error {
    // Check if key exists
    _, err := kr.Key(keyName)
    if err == nil {
        fmt.Printf("Using existing key: %s\n", keyName)
        return nil
    }
    
    // Create new key
    fmt.Printf("Creating new key: %s\n", keyName)
    record, err := kr.NewAccount(keyName, "", "", "", nil)
    if err != nil {
        return err
    }
    
    addr, _ := record.GetAddress()
    fmt.Printf("Key created with address: %s\n", addr.String())
    fmt.Println("Please fund this address before submitting transactions")
    
    return nil
}

func createCelestiaClient(
    ctx context.Context,
    kr *banhbaoring.BaoKeyring,
    keyName string,
) (*client.Client, error) {
    cfg := client.Config{
        ReadConfig: client.ReadConfig{
            BridgeDAAddr: getEnvOrDefault("CELESTIA_BRIDGE_ADDR", "http://localhost:26658"),
            DAAuthToken:  os.Getenv("CELESTIA_NODE_AUTH_TOKEN"),
            EnableDATLS:  false,
        },
        SubmitConfig: client.SubmitConfig{
            DefaultKeyName: keyName,
            Network:        getEnvOrDefault("CELESTIA_NETWORK", "mocha-4"),
            CoreGRPCConfig: client.CoreGRPCConfig{
                Addr:       getEnvOrDefault("CELESTIA_CORE_ADDR", "localhost:9090"),
                TLSEnabled: false,
            },
        },
    }
    
    return client.NewWithKeyring(ctx, cfg, kr)
}

func submitBlob(ctx context.Context, c *client.Client, data []byte) (uint64, error) {
    namespace, err := libshare.NewV0Namespace([]byte("baokeyring"))
    if err != nil {
        return 0, err
    }
    
    b, err := blob.NewBlob(libshare.ShareVersionZero, namespace, data, nil)
    if err != nil {
        return 0, err
    }
    
    return c.Blob.Submit(ctx, []*blob.Blob{b}, nil)
}

func retrieveBlob(ctx context.Context, c *client.Client, height uint64) (*blob.Blob, error) {
    namespace, _ := libshare.NewV0Namespace([]byte("baokeyring"))
    
    blobs, err := c.Blob.GetAll(ctx, height, []libshare.Namespace{namespace})
    if err != nil {
        return nil, err
    }
    
    if len(blobs) == 0 {
        return nil, fmt.Errorf("no blobs found at height %d", height)
    }
    
    return blobs[0], nil
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

---

## 7. Testing

### 7.1 Local Testing with OpenBao Dev Mode

```bash
# Start OpenBao in dev mode
openbao server -dev -dev-root-token-id="dev-token"

# Enable Transit engine
export BAO_ADDR="http://127.0.0.1:8200"
export BAO_TOKEN="dev-token"

openbao secrets enable transit
```

### 7.2 Unit Testing with Mocks

```go
// Mock BaoClient for testing
type MockBaoClient struct {
    keys       map[string]*KeyInfo
    signatures map[string][]byte
}

func (m *MockBaoClient) CreateKey(name string, keyType string) error {
    // Generate test key
    privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    m.keys[name] = &KeyInfo{
        PublicKey: &privateKey.PublicKey,
    }
    return nil
}

func (m *MockBaoClient) Sign(keyName string, data []byte, prehashed bool) ([]byte, error) {
    // Sign with test key
    keyInfo := m.keys[keyName]
    // ... generate signature
    return signature, nil
}

// Test example
func TestBaoKeyring_Sign(t *testing.T) {
    mockClient := NewMockBaoClient()
    store := NewMemoryStore()
    
    kr := &BaoKeyring{
        client: mockClient,
        store:  store,
    }
    
    // Create key
    _, err := kr.NewAccount("test-key", "", "", "", nil)
    require.NoError(t, err)
    
    // Sign
    msg := []byte("test message")
    sig, pubKey, err := kr.Sign("test-key", msg, signing.SignMode_SIGN_MODE_DIRECT)
    require.NoError(t, err)
    require.Len(t, sig, 64)
    require.NotNil(t, pubKey)
}
```

### 7.3 Integration Testing

```go
// +build integration

func TestIntegration_FullFlow(t *testing.T) {
    if os.Getenv("BAO_ADDR") == "" {
        t.Skip("BAO_ADDR not set, skipping integration test")
    }
    
    ctx := context.Background()
    
    // Create real keyring
    cfg := banhbaoring.Config{
        BaoAddr:     os.Getenv("BAO_ADDR"),
        BaoToken:    os.Getenv("BAO_TOKEN"),
        TransitPath: "transit",
        StorePath:   t.TempDir() + "/metadata.json",
        HTTPTimeout: 30 * time.Second,
    }
    
    kr, err := banhbaoring.New(ctx, cfg)
    require.NoError(t, err)
    
    // Test key creation
    keyName := fmt.Sprintf("test-key-%d", time.Now().UnixNano())
    record, err := kr.NewAccount(keyName, "", "", "", nil)
    require.NoError(t, err)
    
    // Test signing
    msg := []byte("integration test message")
    sig, pubKey, err := kr.Sign(keyName, msg, signing.SignMode_SIGN_MODE_DIRECT)
    require.NoError(t, err)
    require.Len(t, sig, 64)
    
    // Verify signature
    hash := sha256.Sum256(msg)
    valid := ecdsa.VerifyASN1(pubKey.(*ecdsa.PublicKey), hash[:], sig)
    require.True(t, valid)
    
    // Cleanup
    err = kr.Delete(keyName)
    require.NoError(t, err)
}
```

---

## 8. Troubleshooting

### 8.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| "permission denied" | Invalid or expired token | Refresh BAO_TOKEN |
| "key not found" | Key doesn't exist in OpenBao | Create key first |
| "connection refused" | OpenBao not reachable | Check BAO_ADDR and network |
| "TLS handshake failure" | Certificate issues | Verify TLS config |
| "signature verification failed" | Wrong signature format | Check DER to compact conversion |

### 8.2 Debug Logging

```go
import "log/slog"

// Enable debug logging
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

cfg := banhbaoring.Config{
    // ...
    Logger: logger,
}
```

### 8.3 Verifying OpenBao Connectivity

```bash
# Test OpenBao connectivity
curl -s -H "X-Vault-Token: $BAO_TOKEN" $BAO_ADDR/v1/sys/health | jq

# List Transit keys
curl -s -H "X-Vault-Token: $BAO_TOKEN" -X LIST $BAO_ADDR/v1/transit/keys | jq

# Test signing
echo -n "test" | base64 > /tmp/input.txt
curl -s -H "X-Vault-Token: $BAO_TOKEN" \
  -d "{\"input\": \"$(cat /tmp/input.txt)\"}" \
  $BAO_ADDR/v1/transit/sign/my-key | jq
```

---

## 9. Production Considerations

### 9.1 High Availability

- Deploy OpenBao in HA mode with multiple nodes
- Use load balancer for OpenBao endpoints
- Implement retry logic with exponential backoff

### 9.2 Security Checklist

- [ ] TLS enabled for OpenBao communication
- [ ] Token with minimal required permissions
- [ ] Token rotation configured
- [ ] Audit logging enabled in OpenBao
- [ ] Network segmentation between app and OpenBao
- [ ] Secrets not logged or exposed in errors

### 9.3 Monitoring

```go
// Prometheus metrics (example)
var (
    signLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "baokeyring_sign_duration_seconds",
            Help:    "Time spent signing via OpenBao",
            Buckets: prometheus.DefBuckets,
        },
        []string{"key_name", "status"},
    )
    
    signTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "baokeyring_sign_total",
            Help: "Total number of signing operations",
        },
        []string{"key_name", "status"},
    )
)
```

---

## 10. References

- [Celestia Go Client Documentation](https://github.com/celestiaorg/celestia-node/blob/main/api/client/readme.md)
- [celestia-node v0.28.4+ Releases](https://github.com/celestiaorg/celestia-node/releases)
- [celestia-app v6.4.0+ Releases](https://github.com/celestiaorg/celestia-app/releases)
- [Cosmos SDK Keyring](https://docs.cosmos.network/main/user/run-node/keyring)
- [OpenBao Documentation](https://openbao.org/docs/)
- [OpenBao Transit Engine](https://openbao.org/docs/secrets/transit/)

