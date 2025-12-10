# Implementation: Parallel Worker Support

## Agent: 06 - Parallel Workers (Fee Grant Pattern)

> **Reference:** [Celestia Client Parallel Workers](https://github.com/celestiaorg/celestia-node/blob/main/api/client/readme.md)

---

## 1. Overview

Celestia rollups use parallel blob submission with multiple worker accounts and fee grants:

```go
cfg := client.Config{
    SubmitConfig: client.SubmitConfig{
        TxWorkerAccounts: 4,  // 4 workers signing in parallel
    },
}
```

This agent adds batch operations and ensures thread-safety for concurrent signing.

---

## 2. Current State Analysis

### ✅ Already Thread-Safe

| Component | Status | Details |
|-----------|--------|---------|
| `BaoStore` | ✅ Safe | Uses `sync.RWMutex`, reads don't block each other |
| `BaoClient` | ✅ Safe | HTTP connection pool, independent requests |
| `BaoKeyring.Sign()` | ✅ Safe* | *Works but not documented/tested |

### ❌ Missing Features

| Feature | Status | Priority |
|---------|--------|----------|
| `CreateBatch()` | ❌ Missing | P0 |
| `SignBatch()` | ❌ Missing | P0 |
| Concurrency tests | ❌ Missing | P0 |
| Thread-safety documentation | ❌ Missing | P1 |

---

## 3. Implementation Tasks

### 3.1 Add Batch Operations to BaoKeyring

**File:** `bao_keyring.go`

```go
// CreateBatchOptions configures batch key creation.
type CreateBatchOptions struct {
    Prefix     string // Key name prefix (e.g., "blob-worker")
    Count      int    // Number of keys to create (e.g., 4)
    Namespace  string // Optional namespace
    Exportable bool   // Whether keys are exportable
}

// CreateBatchResult contains results of batch key creation.
type CreateBatchResult struct {
    Keys   []*keyring.Record
    Errors []error // Per-key errors (nil if successful)
}

// CreateBatch creates multiple keys in parallel.
// This is optimized for the Celestia parallel worker pattern.
//
// Example:
//
//	results, err := kr.CreateBatch(ctx, CreateBatchOptions{
//	    Prefix: "blob-worker",
//	    Count:  4,
//	})
//	// Creates: blob-worker-1, blob-worker-2, blob-worker-3, blob-worker-4
func (k *BaoKeyring) CreateBatch(ctx context.Context, opts CreateBatchOptions) (*CreateBatchResult, error) {
    if opts.Count <= 0 || opts.Count > 100 {
        return nil, fmt.Errorf("count must be between 1 and 100")
    }
    if opts.Prefix == "" {
        return nil, fmt.Errorf("prefix is required")
    }

    result := &CreateBatchResult{
        Keys:   make([]*keyring.Record, opts.Count),
        Errors: make([]error, opts.Count),
    }

    var wg sync.WaitGroup
    for i := 0; i < opts.Count; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            uid := fmt.Sprintf("%s-%d", opts.Prefix, idx+1)
            record, err := k.NewAccountWithOptions(uid, KeyOptions{
                Exportable: opts.Exportable,
            })
            result.Keys[idx] = record
            result.Errors[idx] = err
        }(i)
    }
    wg.Wait()

    // Check for any errors
    var errs []string
    for i, err := range result.Errors {
        if err != nil {
            errs = append(errs, fmt.Sprintf("%s-%d: %v", opts.Prefix, i+1, err))
        }
    }
    if len(errs) > 0 {
        return result, fmt.Errorf("batch create partial failure: %s", strings.Join(errs, "; "))
    }

    return result, nil
}
```

### 3.2 Add SignBatch Operation

**File:** `bao_keyring.go`

```go
// SignRequest represents a single signing request in a batch.
type SignRequest struct {
    UID      string           // Key UID
    Msg      []byte           // Message to sign
    SignMode signing.SignMode // Sign mode
}

// SignResult represents a single signing result.
type SignResult struct {
    UID       string             // Key UID
    Signature []byte             // 64-byte Cosmos signature
    PubKey    cryptotypes.PubKey // Public key
    Error     error              // nil if successful
}

// SignBatch signs multiple messages in parallel.
// Each request can use a different key - perfect for parallel workers.
//
// Performance: Signing 4 messages takes ~200ms (not 4 × 200ms = 800ms).
//
// Example:
//
//	results := kr.SignBatch(ctx, []SignRequest{
//	    {UID: "worker-1", Msg: tx1},
//	    {UID: "worker-2", Msg: tx2},
//	    {UID: "worker-3", Msg: tx3},
//	    {UID: "worker-4", Msg: tx4},
//	})
func (k *BaoKeyring) SignBatch(ctx context.Context, requests []SignRequest) []SignResult {
    if len(requests) == 0 {
        return nil
    }

    results := make([]SignResult, len(requests))
    var wg sync.WaitGroup

    for i, req := range requests {
        wg.Add(1)
        go func(idx int, r SignRequest) {
            defer wg.Done()
            sig, pubKey, err := k.Sign(r.UID, r.Msg, r.SignMode)
            results[idx] = SignResult{
                UID:       r.UID,
                Signature: sig,
                PubKey:    pubKey,
                Error:     err,
            }
        }(i, req)
    }
    wg.Wait()

    return results
}
```

### 3.3 Add Thread-Safety Documentation

**File:** `bao_keyring.go` (update struct comment)

```go
// BaoKeyring implements keyring.Keyring using OpenBao.
//
// Thread Safety:
// BaoKeyring is safe for concurrent use by multiple goroutines.
// This is critical for Celestia's parallel worker pattern where
// multiple blob submissions happen concurrently with different keys.
//
// The underlying BaoStore uses sync.RWMutex for metadata access,
// and BaoClient uses HTTP connection pooling for parallel requests.
//
// Example (parallel workers):
//
//	var wg sync.WaitGroup
//	for _, worker := range workers {
//	    wg.Add(1)
//	    go func(uid string, tx []byte) {
//	        defer wg.Done()
//	        sig, _, _ := kr.Sign(uid, tx, signing.SignMode_SIGN_MODE_DIRECT)
//	        // Use signature...
//	    }(worker.UID, txBytes)
//	}
//	wg.Wait()
type BaoKeyring struct {
    client *BaoClient
    store  *BaoStore
}
```

---

## 4. Test Requirements

### 4.1 Concurrency Tests

**File:** `bao_keyring_parallel_test.go`

```go
package banhbaoring

import (
    "context"
    "sync"
    "sync/atomic"
    "testing"
    "time"

    "github.com/cosmos/cosmos-sdk/types/tx/signing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestSign_Concurrent verifies that Sign() is safe for concurrent use.
// This is critical for Celestia's parallel worker pattern.
func TestSign_Concurrent(t *testing.T) {
    // Setup mock server that handles concurrent requests
    var requestCount int64
    handler := func(w http.ResponseWriter, r *http.Request) {
        atomic.AddInt64(&requestCount, 1)
        // Simulate signing latency
        time.Sleep(50 * time.Millisecond)
        
        if strings.Contains(r.URL.Path, "/sign/") {
            sig := make([]byte, 64)
            _ = json.NewEncoder(w).Encode(map[string]interface{}{
                "data": SignResponse{
                    Signature: base64.StdEncoding.EncodeToString(sig),
                },
            })
            return
        }
        w.WriteHeader(http.StatusOK)
    }

    kr, server := setupTestKeyring(t, handler)
    defer server.Close()

    // Create 4 worker keys
    for i := 1; i <= 4; i++ {
        uid := fmt.Sprintf("worker-%d", i)
        _ = kr.store.Save(&KeyMetadata{
            UID:         uid,
            PubKeyBytes: testPubKeyBytes(),
            Address:     fmt.Sprintf("celestia1worker%d", i),
        })
    }

    // Sign concurrently with 4 workers
    const numWorkers = 4
    const signsPerWorker = 10
    
    var wg sync.WaitGroup
    errors := make(chan error, numWorkers*signsPerWorker)

    start := time.Now()
    
    for w := 1; w <= numWorkers; w++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            uid := fmt.Sprintf("worker-%d", workerID)
            for i := 0; i < signsPerWorker; i++ {
                msg := []byte(fmt.Sprintf("message-%d-%d", workerID, i))
                _, _, err := kr.Sign(uid, msg, signing.SignMode_SIGN_MODE_DIRECT)
                if err != nil {
                    errors <- err
                }
            }
        }(w)
    }
    wg.Wait()
    close(errors)

    elapsed := time.Since(start)

    // Check no errors
    for err := range errors {
        t.Errorf("sign error: %v", err)
    }

    // Verify all requests were made
    assert.Equal(t, int64(numWorkers*signsPerWorker), requestCount)

    // Performance check: parallel should be faster than sequential
    // Sequential would be: 40 signs × 50ms = 2000ms
    // Parallel (4 workers) should be: ~10 batches × 50ms = ~500ms
    t.Logf("Elapsed: %v for %d signs (%d workers)", elapsed, numWorkers*signsPerWorker, numWorkers)
    assert.Less(t, elapsed, 1500*time.Millisecond, 
        "parallel signing should be faster than sequential")
}

// TestSign_NoHeadOfLineBlocking verifies that slow signs don't block others.
func TestSign_NoHeadOfLineBlocking(t *testing.T) {
    // Worker 1 is slow (500ms), workers 2-4 are fast (50ms)
    handler := func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.URL.Path, "worker-1") {
            time.Sleep(500 * time.Millisecond) // Slow
        } else {
            time.Sleep(50 * time.Millisecond) // Fast
        }
        
        sig := make([]byte, 64)
        _ = json.NewEncoder(w).Encode(map[string]interface{}{
            "data": SignResponse{
                Signature: base64.StdEncoding.EncodeToString(sig),
            },
        })
    }

    kr, server := setupTestKeyring(t, handler)
    defer server.Close()

    // Create 4 workers
    for i := 1; i <= 4; i++ {
        _ = kr.store.Save(&KeyMetadata{
            UID:         fmt.Sprintf("worker-%d", i),
            PubKeyBytes: testPubKeyBytes(),
            Address:     fmt.Sprintf("addr%d", i),
        })
    }

    // Sign concurrently
    completionTimes := make([]time.Duration, 4)
    var wg sync.WaitGroup
    start := time.Now()

    for i := 1; i <= 4; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            _, _, _ = kr.Sign(fmt.Sprintf("worker-%d", idx), []byte("msg"), 
                signing.SignMode_SIGN_MODE_DIRECT)
            completionTimes[idx-1] = time.Since(start)
        }(i)
    }
    wg.Wait()

    // Fast workers (2,3,4) should complete in ~50-100ms
    // Slow worker (1) should complete in ~500ms
    // But fast workers should NOT wait for slow worker!
    for i := 1; i <= 3; i++ {
        assert.Less(t, completionTimes[i], 200*time.Millisecond,
            "worker-%d should not be blocked by slow worker-1", i+1)
    }
    
    t.Logf("Completion times: worker-1=%v, worker-2=%v, worker-3=%v, worker-4=%v",
        completionTimes[0], completionTimes[1], completionTimes[2], completionTimes[3])
}

// TestCreateBatch_Success tests batch key creation.
func TestCreateBatch_Success(t *testing.T) {
    keyCount := 0
    handler := func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/keys/") {
            keyCount++
            resp := map[string]interface{}{
                "data": map[string]interface{}{
                    "name":       r.URL.Path,
                    "public_key": fmt.Sprintf("02%064d", keyCount),
                    "address":    fmt.Sprintf("addr%d", keyCount),
                },
            }
            _ = json.NewEncoder(w).Encode(resp)
            return
        }
        w.WriteHeader(http.StatusOK)
    }

    kr, server := setupTestKeyring(t, handler)
    defer server.Close()

    result, err := kr.CreateBatch(context.Background(), CreateBatchOptions{
        Prefix: "blob-worker",
        Count:  4,
    })

    require.NoError(t, err)
    assert.Len(t, result.Keys, 4)
    
    // Verify all keys were created
    for i, key := range result.Keys {
        require.NotNil(t, key, "key %d should not be nil", i)
        assert.Contains(t, key.Name, "blob-worker")
    }
    assert.Nil(t, result.Errors[0])
    assert.Nil(t, result.Errors[1])
    assert.Nil(t, result.Errors[2])
    assert.Nil(t, result.Errors[3])
}

// TestSignBatch_Success tests batch signing.
func TestSignBatch_Success(t *testing.T) {
    handler := func(w http.ResponseWriter, r *http.Request) {
        sig := make([]byte, 64)
        _ = json.NewEncoder(w).Encode(map[string]interface{}{
            "data": SignResponse{
                Signature: base64.StdEncoding.EncodeToString(sig),
            },
        })
    }

    kr, server := setupTestKeyring(t, handler)
    defer server.Close()

    // Create 4 workers
    for i := 1; i <= 4; i++ {
        _ = kr.store.Save(&KeyMetadata{
            UID:         fmt.Sprintf("worker-%d", i),
            PubKeyBytes: testPubKeyBytes(),
            Address:     fmt.Sprintf("addr%d", i),
        })
    }

    requests := []SignRequest{
        {UID: "worker-1", Msg: []byte("tx1"), SignMode: signing.SignMode_SIGN_MODE_DIRECT},
        {UID: "worker-2", Msg: []byte("tx2"), SignMode: signing.SignMode_SIGN_MODE_DIRECT},
        {UID: "worker-3", Msg: []byte("tx3"), SignMode: signing.SignMode_SIGN_MODE_DIRECT},
        {UID: "worker-4", Msg: []byte("tx4"), SignMode: signing.SignMode_SIGN_MODE_DIRECT},
    }

    results := kr.SignBatch(context.Background(), requests)

    assert.Len(t, results, 4)
    for i, r := range results {
        assert.Nil(t, r.Error, "request %d should succeed", i)
        assert.Len(t, r.Signature, 64, "signature %d should be 64 bytes", i)
    }
}

// TestSignBatch_Performance verifies batch is faster than sequential.
func TestSignBatch_Performance(t *testing.T) {
    const signLatency = 50 * time.Millisecond
    
    handler := func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(signLatency)
        sig := make([]byte, 64)
        _ = json.NewEncoder(w).Encode(map[string]interface{}{
            "data": SignResponse{
                Signature: base64.StdEncoding.EncodeToString(sig),
            },
        })
    }

    kr, server := setupTestKeyring(t, handler)
    defer server.Close()

    // Create workers
    for i := 1; i <= 4; i++ {
        _ = kr.store.Save(&KeyMetadata{
            UID:         fmt.Sprintf("worker-%d", i),
            PubKeyBytes: testPubKeyBytes(),
            Address:     fmt.Sprintf("addr%d", i),
        })
    }

    requests := make([]SignRequest, 4)
    for i := 0; i < 4; i++ {
        requests[i] = SignRequest{
            UID:      fmt.Sprintf("worker-%d", i+1),
            Msg:      []byte(fmt.Sprintf("tx%d", i)),
            SignMode: signing.SignMode_SIGN_MODE_DIRECT,
        }
    }

    start := time.Now()
    results := kr.SignBatch(context.Background(), requests)
    elapsed := time.Since(start)

    // All should succeed
    for _, r := range results {
        assert.Nil(t, r.Error)
    }

    // Sequential would be: 4 × 50ms = 200ms
    // Parallel should be: ~50-100ms (all execute concurrently)
    t.Logf("Batch sign of 4 requests took: %v", elapsed)
    assert.Less(t, elapsed, 150*time.Millisecond,
        "batch signing should execute in parallel")
}
```

---

## 5. Files to Modify/Create

| File | Action | Description |
|------|--------|-------------|
| `bao_keyring.go` | Modify | Add `CreateBatch()`, `SignBatch()`, update docs |
| `bao_keyring_parallel_test.go` | Create | Concurrency tests |
| `types.go` | Modify | Add `CreateBatchOptions`, `SignRequest`, etc. |

---

## 6. Success Criteria

- [ ] `CreateBatch()` creates N keys in parallel
- [ ] `SignBatch()` signs N messages in parallel
- [ ] Concurrent `Sign()` calls don't block each other
- [ ] All concurrency tests pass
- [ ] No race conditions (`go test -race`)
- [ ] Thread-safety documented in code comments
- [ ] Performance: 4 parallel signs complete in <250ms (not 4×200ms)

---

## 7. Agent Prompt

```
You are Agent 06. Your task is to add parallel worker support to BaoKeyring.

Read the implementation spec: doc/implementation/IMPL_06_PARALLEL_WORKERS.md

Reference: Celestia parallel workers pattern
https://github.com/celestiaorg/celestia-node/blob/main/api/client/readme.md

Key deliverables:
1. CreateBatch(ctx, CreateBatchOptions) - create N worker keys in parallel
2. SignBatch(ctx, []SignRequest) - sign N messages in parallel  
3. Thread-safety documentation on BaoKeyring struct
4. Concurrency tests proving parallel Sign() works
5. Performance tests proving no head-of-line blocking

Files to modify:
- bao_keyring.go (add methods, update docs)
- types.go (add new types)
- bao_keyring_parallel_test.go (create new test file)

CRITICAL: Run `go test -race ./...` to verify no race conditions.

Test command: go test -race -v ./... && golangci-lint run
```

---

## 8. Dependencies

This agent depends on:
- Phase 4 complete (BaoKeyring implemented)
- No external dependencies

This agent blocks:
- Phase 5 Control Plane API (needs batch operations)

---

## 9. Estimated Effort

| Task | Time |
|------|------|
| Add batch methods | 2 hours |
| Write concurrency tests | 3 hours |
| Race condition debugging | 2 hours |
| Documentation | 1 hour |
| **Total** | **~1 day** |

