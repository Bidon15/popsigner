# Implementation Guide: BaoStore (Metadata Storage)

**Agent ID:** 02  
**Component:** Local Metadata Storage Layer  
**Parallelizable:** ✅ Yes - Only depends on types (interface)

---

## 1. Overview

This agent is responsible for building the local metadata storage layer (`BaoStore`). This component persists key metadata (public keys, addresses, paths) to disk without storing any private key material. It provides thread-safe access to key information for the `BaoKeyring`.

### 1.1 Required Skills

| Skill           | Level        | Description                          |
| --------------- | ------------ | ------------------------------------ |
| **Go**          | Advanced     | Structs, interfaces, methods         |
| **File I/O**    | Advanced     | Atomic writes, file locking          |
| **JSON**        | Intermediate | Encoding/decoding, schema versioning |
| **Concurrency** | Advanced     | sync.RWMutex, thread safety          |
| **Testing**     | Advanced     | Table-driven tests, temp files       |

### 1.2 Files to Create

```
banhbaoring/
├── bao_store.go       # Metadata storage implementation
└── bao_store_test.go  # Unit tests
```

### 1.3 Dependencies from Agent 01

You will use these types from `types.go` (Agent 01):

```go
// From types.go
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
    Source      string    `json:"source"`
}

type StoreData struct {
    Version int                     `json:"version"`
    Keys    map[string]*KeyMetadata `json:"keys"`
}
```

If Agent 01 is not complete, create a local copy of these types.

---

## 2. Detailed Specifications

### 2.1 bao_store.go - Metadata Storage Implementation

```go
package banhbaoring

import (
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sync"
)

// BaoStore manages local key metadata persistence.
// It stores only public information (no private keys).
// Thread-safe for concurrent access.
type BaoStore struct {
    path     string
    data     *StoreData
    mu       sync.RWMutex
    dirty    bool
}

// NewBaoStore creates or opens a metadata store at the given path.
func NewBaoStore(path string) (*BaoStore, error) {
    store := &BaoStore{
        path: path,
        data: &StoreData{
            Version: DefaultStoreVersion,
            Keys:    make(map[string]*KeyMetadata),
        },
    }

    // Ensure directory exists
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0700); err != nil {
        return nil, fmt.Errorf("failed to create store directory: %w", err)
    }

    // Try to load existing store
    if err := store.load(); err != nil {
        if !os.IsNotExist(err) {
            return nil, err
        }
        // File doesn't exist, will be created on first save
    }

    return store, nil
}

// Save persists a key's metadata to the store.
func (s *BaoStore) Save(meta *KeyMetadata) error {
    if meta == nil {
        return errors.New("metadata cannot be nil")
    }
    if meta.UID == "" {
        return errors.New("metadata UID is required")
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    // Check for duplicate
    if existing, exists := s.data.Keys[meta.UID]; exists {
        // Allow update if it's the same key (same address)
        if existing.Address != meta.Address {
            return fmt.Errorf("%w: %s", ErrKeyExists, meta.UID)
        }
    }

    // Store metadata
    s.data.Keys[meta.UID] = meta
    s.dirty = true

    return s.syncLocked()
}

// Get retrieves metadata by UID.
func (s *BaoStore) Get(uid string) (*KeyMetadata, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    meta, exists := s.data.Keys[uid]
    if !exists {
        return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, uid)
    }

    // Return a copy to prevent external modification
    return copyMetadata(meta), nil
}

// GetByAddress retrieves metadata by Cosmos address.
func (s *BaoStore) GetByAddress(address string) (*KeyMetadata, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for _, meta := range s.data.Keys {
        if meta.Address == address {
            return copyMetadata(meta), nil
        }
    }

    return nil, fmt.Errorf("%w: address %s", ErrKeyNotFound, address)
}

// List returns all stored metadata.
func (s *BaoStore) List() ([]*KeyMetadata, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    result := make([]*KeyMetadata, 0, len(s.data.Keys))
    for _, meta := range s.data.Keys {
        result = append(result, copyMetadata(meta))
    }

    return result, nil
}

// Delete removes metadata by UID.
func (s *BaoStore) Delete(uid string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.data.Keys[uid]; !exists {
        return fmt.Errorf("%w: %s", ErrKeyNotFound, uid)
    }

    delete(s.data.Keys, uid)
    s.dirty = true

    return s.syncLocked()
}

// Rename updates the UID of stored metadata.
func (s *BaoStore) Rename(oldUID, newUID string) error {
    if oldUID == newUID {
        return nil
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    // Check old exists
    meta, exists := s.data.Keys[oldUID]
    if !exists {
        return fmt.Errorf("%w: %s", ErrKeyNotFound, oldUID)
    }

    // Check new doesn't exist
    if _, exists := s.data.Keys[newUID]; exists {
        return fmt.Errorf("%w: %s", ErrKeyExists, newUID)
    }

    // Rename
    meta.UID = newUID
    meta.Name = newUID
    s.data.Keys[newUID] = meta
    delete(s.data.Keys, oldUID)
    s.dirty = true

    return s.syncLocked()
}

// Has checks if a key exists in the store.
func (s *BaoStore) Has(uid string) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()

    _, exists := s.data.Keys[uid]
    return exists
}

// Count returns the number of keys in the store.
func (s *BaoStore) Count() int {
    s.mu.RLock()
    defer s.mu.RUnlock()

    return len(s.data.Keys)
}

// Sync persists in-memory state to disk.
func (s *BaoStore) Sync() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    return s.syncLocked()
}

// Path returns the store file path.
func (s *BaoStore) Path() string {
    return s.path
}

// Close ensures all data is persisted and releases resources.
func (s *BaoStore) Close() error {
    return s.Sync()
}

// Internal methods

// load reads the store from disk.
func (s *BaoStore) load() error {
    f, err := os.Open(s.path)
    if err != nil {
        return err
    }
    defer f.Close()

    data, err := io.ReadAll(f)
    if err != nil {
        return fmt.Errorf("failed to read store file: %w", err)
    }

    // Handle empty file
    if len(data) == 0 {
        return nil
    }

    var storeData StoreData
    if err := json.Unmarshal(data, &storeData); err != nil {
        return fmt.Errorf("%w: %v", ErrStoreCorrupted, err)
    }

    // Validate version
    if storeData.Version > DefaultStoreVersion {
        return fmt.Errorf("%w: unsupported version %d", ErrStoreCorrupted, storeData.Version)
    }

    // Initialize keys map if nil
    if storeData.Keys == nil {
        storeData.Keys = make(map[string]*KeyMetadata)
    }

    s.data = &storeData
    s.dirty = false

    return nil
}

// syncLocked writes the store to disk atomically.
// Must be called with mu held.
func (s *BaoStore) syncLocked() error {
    if !s.dirty {
        return nil
    }

    data, err := json.MarshalIndent(s.data, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal store data: %w", err)
    }

    // Write atomically using temp file + rename
    tmpPath := s.path + ".tmp"

    f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
    if err != nil {
        return fmt.Errorf("%w: failed to create temp file: %v", ErrStorePersist, err)
    }

    if _, err := f.Write(data); err != nil {
        f.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("%w: failed to write temp file: %v", ErrStorePersist, err)
    }

    // Ensure data is flushed to disk
    if err := f.Sync(); err != nil {
        f.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("%w: failed to sync temp file: %v", ErrStorePersist, err)
    }

    if err := f.Close(); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("%w: failed to close temp file: %v", ErrStorePersist, err)
    }

    // Atomic rename
    if err := os.Rename(tmpPath, s.path); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("%w: failed to rename temp file: %v", ErrStorePersist, err)
    }

    s.dirty = false
    return nil
}

// copyMetadata creates a deep copy of KeyMetadata.
func copyMetadata(meta *KeyMetadata) *KeyMetadata {
    if meta == nil {
        return nil
    }

    copy := *meta
    if meta.PubKeyBytes != nil {
        copy.PubKeyBytes = make([]byte, len(meta.PubKeyBytes))
        copy(copy.PubKeyBytes, meta.PubKeyBytes)
    }

    return &copy
}
```

---

## 3. Additional Features

### 3.1 Store Backup/Restore

```go
// Backup creates a backup of the store at the given path.
func (s *BaoStore) Backup(backupPath string) error {
    s.mu.RLock()
    defer s.mu.RUnlock()

    data, err := json.MarshalIndent(s.data, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal store data: %w", err)
    }

    if err := os.WriteFile(backupPath, data, 0600); err != nil {
        return fmt.Errorf("failed to write backup: %w", err)
    }

    return nil
}

// Restore loads the store from a backup file.
func (s *BaoStore) Restore(backupPath string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    data, err := os.ReadFile(backupPath)
    if err != nil {
        return fmt.Errorf("failed to read backup: %w", err)
    }

    var storeData StoreData
    if err := json.Unmarshal(data, &storeData); err != nil {
        return fmt.Errorf("failed to parse backup: %w", err)
    }

    s.data = &storeData
    s.dirty = true

    return s.syncLocked()
}
```

### 3.2 Import/Export UIDs

```go
// UIDs returns all key UIDs in the store.
func (s *BaoStore) UIDs() []string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    uids := make([]string, 0, len(s.data.Keys))
    for uid := range s.data.Keys {
        uids = append(uids, uid)
    }
    return uids
}

// Addresses returns all addresses in the store.
func (s *BaoStore) Addresses() []string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    addresses := make([]string, 0, len(s.data.Keys))
    for _, meta := range s.data.Keys {
        addresses = append(addresses, meta.Address)
    }
    return addresses
}
```

### 3.3 Iteration Helper

```go
// ForEach iterates over all keys and calls fn for each.
// If fn returns an error, iteration stops and the error is returned.
func (s *BaoStore) ForEach(fn func(*KeyMetadata) error) error {
    s.mu.RLock()
    defer s.mu.RUnlock()

    for _, meta := range s.data.Keys {
        if err := fn(copyMetadata(meta)); err != nil {
            return err
        }
    }
    return nil
}
```

---

## 4. Unit Test Requirements

### 4.1 bao_store_test.go

```go
package banhbaoring

import (
    "os"
    "path/filepath"
    "sync"
    "testing"
    "time"
)

func TestNewBaoStore(t *testing.T) {
    t.Run("creates new store", func(t *testing.T) {
        path := filepath.Join(t.TempDir(), "store.json")

        store, err := NewBaoStore(path)
        if err != nil {
            t.Fatal(err)
        }
        defer store.Close()

        if store.Count() != 0 {
            t.Error("new store should be empty")
        }
    })

    t.Run("loads existing store", func(t *testing.T) {
        path := filepath.Join(t.TempDir(), "store.json")

        // Create store with data
        store1, err := NewBaoStore(path)
        if err != nil {
            t.Fatal(err)
        }

        meta := &KeyMetadata{
            UID:     "test-key",
            Name:    "test-key",
            Address: "celestia1test...",
        }
        if err := store1.Save(meta); err != nil {
            t.Fatal(err)
        }
        store1.Close()

        // Reopen store
        store2, err := NewBaoStore(path)
        if err != nil {
            t.Fatal(err)
        }
        defer store2.Close()

        if store2.Count() != 1 {
            t.Error("store should have 1 key")
        }

        loaded, err := store2.Get("test-key")
        if err != nil {
            t.Fatal(err)
        }
        if loaded.Address != meta.Address {
            t.Error("address mismatch")
        }
    })

    t.Run("creates parent directories", func(t *testing.T) {
        path := filepath.Join(t.TempDir(), "subdir", "nested", "store.json")

        store, err := NewBaoStore(path)
        if err != nil {
            t.Fatal(err)
        }
        store.Close()

        if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
            t.Error("parent directories should be created")
        }
    })
}

func TestBaoStore_Save(t *testing.T) {
    store, _ := NewBaoStore(filepath.Join(t.TempDir(), "store.json"))
    defer store.Close()

    t.Run("saves new key", func(t *testing.T) {
        meta := &KeyMetadata{
            UID:         "key1",
            Name:        "key1",
            Address:     "celestia1abc...",
            PubKeyBytes: []byte{0x02, 0xab, 0xcd},
            Algorithm:   "secp256k1",
            CreatedAt:   time.Now(),
        }

        err := store.Save(meta)
        if err != nil {
            t.Fatal(err)
        }

        if !store.Has("key1") {
            t.Error("key should exist after save")
        }
    })

    t.Run("rejects nil metadata", func(t *testing.T) {
        err := store.Save(nil)
        if err == nil {
            t.Error("should reject nil metadata")
        }
    })

    t.Run("rejects empty UID", func(t *testing.T) {
        err := store.Save(&KeyMetadata{})
        if err == nil {
            t.Error("should reject empty UID")
        }
    })

    t.Run("rejects duplicate UID with different address", func(t *testing.T) {
        meta := &KeyMetadata{
            UID:     "dup-key",
            Name:    "dup-key",
            Address: "celestia1first...",
        }
        store.Save(meta)

        meta2 := &KeyMetadata{
            UID:     "dup-key",
            Name:    "dup-key",
            Address: "celestia1second...",
        }
        err := store.Save(meta2)
        if err == nil {
            t.Error("should reject duplicate UID with different address")
        }
    })
}

func TestBaoStore_Get(t *testing.T) {
    store, _ := NewBaoStore(filepath.Join(t.TempDir(), "store.json"))
    defer store.Close()

    meta := &KeyMetadata{
        UID:     "test-key",
        Name:    "test-key",
        Address: "celestia1test...",
    }
    store.Save(meta)

    t.Run("returns existing key", func(t *testing.T) {
        got, err := store.Get("test-key")
        if err != nil {
            t.Fatal(err)
        }
        if got.Address != meta.Address {
            t.Error("address mismatch")
        }
    })

    t.Run("returns copy not reference", func(t *testing.T) {
        got, _ := store.Get("test-key")
        got.Address = "modified"

        got2, _ := store.Get("test-key")
        if got2.Address == "modified" {
            t.Error("should return copy, not reference")
        }
    })

    t.Run("returns error for missing key", func(t *testing.T) {
        _, err := store.Get("nonexistent")
        if err == nil {
            t.Error("should return error for missing key")
        }
    })
}

func TestBaoStore_GetByAddress(t *testing.T) {
    store, _ := NewBaoStore(filepath.Join(t.TempDir(), "store.json"))
    defer store.Close()

    store.Save(&KeyMetadata{
        UID:     "key1",
        Address: "celestia1addr1...",
    })
    store.Save(&KeyMetadata{
        UID:     "key2",
        Address: "celestia1addr2...",
    })

    t.Run("finds key by address", func(t *testing.T) {
        got, err := store.GetByAddress("celestia1addr2...")
        if err != nil {
            t.Fatal(err)
        }
        if got.UID != "key2" {
            t.Error("wrong key returned")
        }
    })

    t.Run("returns error for missing address", func(t *testing.T) {
        _, err := store.GetByAddress("celestia1unknown...")
        if err == nil {
            t.Error("should return error for missing address")
        }
    })
}

func TestBaoStore_List(t *testing.T) {
    store, _ := NewBaoStore(filepath.Join(t.TempDir(), "store.json"))
    defer store.Close()

    store.Save(&KeyMetadata{UID: "key1", Address: "addr1"})
    store.Save(&KeyMetadata{UID: "key2", Address: "addr2"})
    store.Save(&KeyMetadata{UID: "key3", Address: "addr3"})

    keys, err := store.List()
    if err != nil {
        t.Fatal(err)
    }

    if len(keys) != 3 {
        t.Errorf("expected 3 keys, got %d", len(keys))
    }
}

func TestBaoStore_Delete(t *testing.T) {
    store, _ := NewBaoStore(filepath.Join(t.TempDir(), "store.json"))
    defer store.Close()

    store.Save(&KeyMetadata{UID: "to-delete", Address: "addr"})

    t.Run("deletes existing key", func(t *testing.T) {
        err := store.Delete("to-delete")
        if err != nil {
            t.Fatal(err)
        }

        if store.Has("to-delete") {
            t.Error("key should be deleted")
        }
    })

    t.Run("returns error for missing key", func(t *testing.T) {
        err := store.Delete("nonexistent")
        if err == nil {
            t.Error("should return error for missing key")
        }
    })
}

func TestBaoStore_Rename(t *testing.T) {
    store, _ := NewBaoStore(filepath.Join(t.TempDir(), "store.json"))
    defer store.Close()

    store.Save(&KeyMetadata{UID: "old-name", Address: "addr"})

    t.Run("renames key", func(t *testing.T) {
        err := store.Rename("old-name", "new-name")
        if err != nil {
            t.Fatal(err)
        }

        if store.Has("old-name") {
            t.Error("old name should not exist")
        }
        if !store.Has("new-name") {
            t.Error("new name should exist")
        }

        meta, _ := store.Get("new-name")
        if meta.Address != "addr" {
            t.Error("address should be preserved")
        }
    })

    t.Run("no-op for same name", func(t *testing.T) {
        store.Save(&KeyMetadata{UID: "same", Address: "addr"})
        err := store.Rename("same", "same")
        if err != nil {
            t.Error("same name should be no-op")
        }
    })

    t.Run("rejects rename to existing", func(t *testing.T) {
        store.Save(&KeyMetadata{UID: "source", Address: "addr1"})
        store.Save(&KeyMetadata{UID: "target", Address: "addr2"})

        err := store.Rename("source", "target")
        if err == nil {
            t.Error("should reject rename to existing key")
        }
    })
}

func TestBaoStore_Concurrency(t *testing.T) {
    store, _ := NewBaoStore(filepath.Join(t.TempDir(), "store.json"))
    defer store.Close()

    var wg sync.WaitGroup

    // Concurrent writes
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            store.Save(&KeyMetadata{
                UID:     fmt.Sprintf("key-%d", i),
                Address: fmt.Sprintf("addr-%d", i),
            })
        }(i)
    }

    // Concurrent reads
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            store.List()
            store.Count()
        }()
    }

    wg.Wait()

    if store.Count() != 100 {
        t.Errorf("expected 100 keys, got %d", store.Count())
    }
}

func TestBaoStore_Persistence(t *testing.T) {
    path := filepath.Join(t.TempDir(), "store.json")

    // Create and populate store
    store1, _ := NewBaoStore(path)
    for i := 0; i < 10; i++ {
        store1.Save(&KeyMetadata{
            UID:     fmt.Sprintf("key-%d", i),
            Address: fmt.Sprintf("addr-%d", i),
        })
    }
    store1.Close()

    // Verify file exists and has content
    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatal(err)
    }
    if len(data) == 0 {
        t.Error("store file should not be empty")
    }

    // Reopen and verify
    store2, _ := NewBaoStore(path)
    defer store2.Close()

    if store2.Count() != 10 {
        t.Errorf("expected 10 keys after reload, got %d", store2.Count())
    }
}

func TestBaoStore_CorruptedFile(t *testing.T) {
    path := filepath.Join(t.TempDir(), "store.json")

    // Write corrupted data
    os.WriteFile(path, []byte("not valid json"), 0600)

    _, err := NewBaoStore(path)
    if err == nil {
        t.Error("should return error for corrupted file")
    }
}

func TestBaoStore_AtomicWrite(t *testing.T) {
    path := filepath.Join(t.TempDir(), "store.json")

    store, _ := NewBaoStore(path)
    defer store.Close()

    // Save initial data
    store.Save(&KeyMetadata{UID: "key1", Address: "addr1"})

    // Verify no temp file remains
    tmpPath := path + ".tmp"
    if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
        t.Error("temp file should not remain after save")
    }
}
```

---

## 5. Success Criteria

### 5.1 Functional Requirements

- [ ] `NewBaoStore` creates new store file if not exists
- [ ] `NewBaoStore` loads existing store file
- [ ] `NewBaoStore` creates parent directories
- [ ] `Save` persists new key metadata
- [ ] `Save` rejects nil metadata
- [ ] `Save` rejects empty UID
- [ ] `Save` allows update with same address
- [ ] `Save` rejects duplicate UID with different address
- [ ] `Get` returns existing key metadata
- [ ] `Get` returns copy, not reference
- [ ] `Get` returns error for missing key
- [ ] `GetByAddress` finds key by Cosmos address
- [ ] `List` returns all keys
- [ ] `Delete` removes key metadata
- [ ] `Delete` returns error for missing key
- [ ] `Rename` changes key UID
- [ ] `Has` checks key existence
- [ ] `Count` returns key count
- [ ] `Sync` flushes to disk
- [ ] `Close` syncs before closing

### 5.2 Non-Functional Requirements

- [ ] Thread-safe for concurrent access (RWMutex)
- [ ] Atomic writes (temp file + rename)
- [ ] File permissions are 0600 (owner read/write only)
- [ ] Directory permissions are 0700
- [ ] Returns copies to prevent external mutation

### 5.3 Test Coverage

- [ ] > 85% code coverage
- [ ] Concurrent access tested
- [ ] Persistence across restart tested
- [ ] Corrupted file handling tested
- [ ] All error paths tested

---

## 6. Interface Contracts

Other agents depend on this interface. Do NOT change without coordination:

```go
// BaoStoreInterface defines the contract for Agent 4 (BaoKeyring)
type BaoStoreInterface interface {
    Save(meta *KeyMetadata) error
    Get(uid string) (*KeyMetadata, error)
    GetByAddress(address string) (*KeyMetadata, error)
    List() ([]*KeyMetadata, error)
    Delete(uid string) error
    Rename(oldUID, newUID string) error
    Has(uid string) bool
    Count() int
    Sync() error
    Close() error
}
```

---

## 7. Storage Format

### 7.1 JSON Schema

```json
{
  "version": 1,
  "keys": {
    "my-validator": {
      "uid": "my-validator",
      "name": "my-validator",
      "pub_key": "AqG3rFy...",
      "pub_key_type": "secp256k1",
      "address": "celestia1abc123...",
      "bao_key_path": "secp256k1/keys/my-validator",
      "algorithm": "secp256k1",
      "exportable": false,
      "created_at": "2025-01-10T12:00:00Z",
      "source": "generated"
    }
  }
}
```

### 7.2 Version Migration

If future versions need schema changes:

```go
// migrate performs schema migration if needed
func (s *BaoStore) migrate() error {
    if s.data.Version == DefaultStoreVersion {
        return nil // Already current
    }

    switch s.data.Version {
    case 0:
        // Migrate v0 → v1
        s.data.Version = 1
        s.dirty = true
    }

    return s.syncLocked()
}
```

---

## 8. Deliverables Checklist

- [ ] `bao_store.go` - Store implementation complete
- [ ] `bao_store_test.go` - Unit tests passing
- [ ] Thread safety verified with concurrent tests
- [ ] Atomic writes verified
- [ ] File permissions verified
- [ ] All tests pass: `go test ./...`
- [ ] No linter errors: `golangci-lint run`
