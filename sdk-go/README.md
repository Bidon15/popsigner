# POPSigner Go SDK

Official Go SDK for the [POPSigner](https://popsigner.com) Control Plane API.

POPSigner is Point-of-Presence signing infrastructure backed by OpenBao. Keys remain remote. You remain sovereign.

## Installation

```bash
go get github.com/Bidon15/popsigner/sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    popsigner "github.com/Bidon15/popsigner/sdk-go"
    "github.com/google/uuid"
)

func main() {
    // Create a client with your API key
    client := popsigner.NewClient("psk_live_xxxxx")
    ctx := context.Background()

    // Create a key
    key, err := client.Keys.Create(ctx, popsigner.CreateKeyRequest{
        Name:        "my-sequencer-key",
        NamespaceID: uuid.MustParse("your-namespace-id"),
        Algorithm:   "secp256k1",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created key: %s (address: %s)\n", key.Name, key.Address)

    // Sign inline with your execution
    result, err := client.Sign.Sign(ctx, key.ID, []byte("Hello, World!"), false)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Signature: %x\n", result.Signature)
}
```

## Features

- **Key Management**: Create, list, get, delete, import, and export keys
- **Signing**: Sign messages inline with your execution path
- **Batch Operations**: Parallel signing for worker-native workloads
- **Organizations**: Manage organizations, members, and namespaces
- **Audit Logs**: Query audit logs with filtering and pagination
- **Exit Guarantee**: Export keys at any time—sovereignty by default

## Client Options

```go
// Custom base URL
client := popsigner.NewClient(apiKey, popsigner.WithBaseURL("https://custom.api.io"))

// Custom timeout
client := popsigner.NewClient(apiKey, popsigner.WithTimeout(60*time.Second))

// Custom HTTP client
httpClient := &http.Client{
    Transport: &http.Transport{MaxIdleConns: 100},
}
client := popsigner.NewClient(apiKey, popsigner.WithHTTPClient(httpClient))
```

## Key Management

### Create a Key

```go
key, err := client.Keys.Create(ctx, popsigner.CreateKeyRequest{
    Name:        "sequencer-main",
    NamespaceID: namespaceID,
    Algorithm:   "secp256k1",  // or "ed25519"
    Exportable:  true,         // exit guarantee—export anytime
    Metadata: map[string]string{
        "environment": "production",
    },
})
```

### Batch Create Keys (Parallel Workers Pattern)

```go
// Create worker keys in parallel
keys, err := client.Keys.CreateBatch(ctx, popsigner.CreateBatchRequest{
    Prefix:      "blob-worker",
    Count:       4,
    NamespaceID: namespaceID,
})
// Creates: blob-worker-1, blob-worker-2, blob-worker-3, blob-worker-4
```

### List Keys

```go
// List all keys
keys, err := client.Keys.List(ctx, nil)

// List keys in a specific namespace
keys, err := client.Keys.List(ctx, &popsigner.ListOptions{
    NamespaceID: &namespaceID,
})
```

### Import/Export Keys (Exit Guarantee)

```go
// Import a private key
key, err := client.Keys.Import(ctx, popsigner.ImportKeyRequest{
    Name:        "imported-key",
    NamespaceID: namespaceID,
    PrivateKey:  base64PrivateKey,  // base64-encoded
    Exportable:  true,
})

// Export a key—sovereignty by default
result, err := client.Keys.Export(ctx, keyID)
privateKey := result.PrivateKey  // base64-encoded
```

## Signing

### Sign Inline

```go
result, err := client.Sign.Sign(ctx, keyID, []byte("message"), false)
fmt.Printf("Signature: %x\n", result.Signature)
```

### Sign Pre-hashed Data

```go
// For blockchain transactions that require signing a hash
result, err := client.Sign.Sign(ctx, keyID, txHash, true)
```

### Batch Sign (Parallel Workers)

Batch signing for worker-native workloads:

```go
results, err := client.Sign.SignBatch(ctx, popsigner.BatchSignRequest{
    Requests: []popsigner.SignRequest{
        {KeyID: worker1, Data: tx1},
        {KeyID: worker2, Data: tx2},
        {KeyID: worker3, Data: tx3},
        {KeyID: worker4, Data: tx4},
    },
})

for i, r := range results {
    if r.Error != "" {
        log.Printf("Worker %d failed: %s", i, r.Error)
    } else {
        fmt.Printf("Worker %d signature: %x\n", i, r.Signature)
    }
}
```

## Organizations

```go
// Create an organization
org, err := client.Orgs.Create(ctx, popsigner.CreateOrgRequest{
    Name: "My Organization",
})

// Create a namespace
ns, err := client.Orgs.CreateNamespace(ctx, orgID, popsigner.CreateNamespaceRequest{
    Name:        "production",
    Description: "Production keys",
})

// Invite a member
invitation, err := client.Orgs.InviteMember(ctx, orgID, popsigner.InviteMemberRequest{
    Email: "user@example.com",
    Role:  popsigner.RoleOperator,
})

// Get plan limits
limits, err := client.Orgs.GetLimits(ctx, orgID)
fmt.Printf("Keys: %d/%d\n", limits.CurrentKeys, limits.MaxKeys)
```

## Audit Logs

```go
// List all audit logs
resp, err := client.Audit.List(ctx, nil)

// Filter by event type
resp, err := client.Audit.List(ctx, &popsigner.AuditFilter{
    Event: popsigner.Ptr(popsigner.AuditEventKeySigned),
    Limit: 50,
})

// Filter by resource
keyID := uuid.MustParse("...")
resp, err := client.Audit.List(ctx, &popsigner.AuditFilter{
    ResourceType: popsigner.Ptr(popsigner.ResourceTypeKey),
    ResourceID:   &keyID,
})

// Paginate through results
filter := &popsigner.AuditFilter{Limit: 100}
for {
    resp, err := client.Audit.List(ctx, filter)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, log := range resp.Logs {
        fmt.Printf("%s: %s\n", log.CreatedAt, log.Event)
    }
    
    if resp.NextCursor == "" {
        break
    }
    filter.Cursor = resp.NextCursor
}
```

## Error Handling

The SDK provides typed errors with helper methods:

```go
key, err := client.Keys.Get(ctx, keyID)
if err != nil {
    if apiErr, ok := popsigner.IsAPIError(err); ok {
        switch {
        case apiErr.IsNotFound():
            fmt.Println("Key not found")
        case apiErr.IsUnauthorized():
            fmt.Println("Invalid API key")
        case apiErr.IsForbidden():
            fmt.Println("Insufficient permissions")
        case apiErr.IsRateLimited():
            fmt.Println("Rate limit exceeded, retry later")
        case apiErr.IsValidationError():
            fmt.Printf("Validation error: %s\n", apiErr.Message)
        default:
            fmt.Printf("API error: %s (%s)\n", apiErr.Message, apiErr.Code)
        }
    } else {
        fmt.Printf("Network error: %v\n", err)
    }
}
```

## Examples

See the [examples](./examples) directory:

- [Basic Usage](./examples/basic/main.go) — Key management and signing
- [Parallel Workers](./examples/parallel-workers/main.go) — Batch operations

## API Reference

### Client

| Method | Description |
|--------|-------------|
| `NewClient(apiKey, ...opts)` | Create a new client |
| `WithBaseURL(url)` | Set custom API URL |
| `WithTimeout(duration)` | Set HTTP timeout |
| `WithHTTPClient(client)` | Set custom HTTP client |

### KeysService

| Method | Description |
|--------|-------------|
| `Create(ctx, req)` | Create a new key |
| `CreateBatch(ctx, req)` | Create multiple keys |
| `Get(ctx, keyID)` | Get a key by ID |
| `List(ctx, opts)` | List all keys |
| `Delete(ctx, keyID)` | Delete a key |
| `Import(ctx, req)` | Import a private key |
| `Export(ctx, keyID)` | Export a key (exit guarantee) |

### SignService

| Method | Description |
|--------|-------------|
| `Sign(ctx, keyID, data, prehashed)` | Sign data inline |
| `SignBatch(ctx, req)` | Sign multiple messages in parallel |

### OrgsService

| Method | Description |
|--------|-------------|
| `Create(ctx, req)` | Create an organization |
| `Get(ctx, orgID)` | Get an organization |
| `List(ctx)` | List organizations |
| `Update(ctx, orgID, req)` | Update an organization |
| `Delete(ctx, orgID)` | Delete an organization |
| `GetLimits(ctx, orgID)` | Get plan limits |
| `ListMembers(ctx, orgID)` | List members |
| `InviteMember(ctx, orgID, req)` | Invite a member |
| `RemoveMember(ctx, orgID, userID)` | Remove a member |
| `ListNamespaces(ctx, orgID)` | List namespaces |
| `CreateNamespace(ctx, orgID, req)` | Create a namespace |
| `GetNamespace(ctx, orgID, nsID)` | Get a namespace |
| `DeleteNamespace(ctx, orgID, nsID)` | Delete a namespace |

### AuditService

| Method | Description |
|--------|-------------|
| `List(ctx, filter)` | List audit logs with optional filters |
| `Get(ctx, logID)` | Get a specific audit log |

## License

MIT License - see [LICENSE](./LICENSE) for details.
