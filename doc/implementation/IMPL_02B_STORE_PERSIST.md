# Implementation Guide: BaoStore Persistence

**Agent ID:** 02B  
**Parent:** Agent 02 (Storage Layer)  
**Component:** File I/O and Atomic Writes  
**Parallelizable:** ✅ Yes - Integrates with 02A

---

## 1. Overview

File persistence with atomic writes (temp file + rename pattern).

### 1.1 Required Skills

| Skill        | Level    | Description          |
| ------------ | -------- | -------------------- |
| **Go**       | Advanced | File I/O, os package |
| **File I/O** | Advanced | Atomic writes, fsync |

### 1.2 Methods to Add

```
NewBaoStore() - constructor with load
load() - read from disk
syncLocked() - atomic write
Sync() - public flush
Close() - cleanup
```

---

## 2. Specifications

```go
package banhbaoring

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
)

// NewBaoStore creates or opens a store.
func NewBaoStore(path string) (*BaoStore, error) {
    store := &BaoStore{
        path: path,
        data: &StoreData{
            Version: DefaultStoreVersion,
            Keys:    make(map[string]*KeyMetadata),
        },
    }

    // Create directory
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
        return nil, fmt.Errorf("create directory: %w", err)
    }

    // Load existing
    if err := store.load(); err != nil && !os.IsNotExist(err) {
        return nil, err
    }

    return store, nil
}

// load reads store from disk.
func (s *BaoStore) load() error {
    f, err := os.Open(s.path)
    if err != nil {
        return err
    }
    defer f.Close()

    data, err := io.ReadAll(f)
    if err != nil {
        return fmt.Errorf("read file: %w", err)
    }

    if len(data) == 0 {
        return nil
    }

    var storeData StoreData
    if err := json.Unmarshal(data, &storeData); err != nil {
        return fmt.Errorf("%w: %v", ErrStoreCorrupted, err)
    }

    if storeData.Version > DefaultStoreVersion {
        return fmt.Errorf("%w: unsupported version %d", ErrStoreCorrupted, storeData.Version)
    }

    if storeData.Keys == nil {
        storeData.Keys = make(map[string]*KeyMetadata)
    }

    s.data = &storeData
    s.dirty = false
    return nil
}

// syncLocked writes atomically. Must hold lock.
func (s *BaoStore) syncLocked() error {
    if !s.dirty {
        return nil
    }

    data, err := json.MarshalIndent(s.data, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal: %w", err)
    }

    tmpPath := s.path + ".tmp"

    f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
    if err != nil {
        return fmt.Errorf("%w: create temp: %v", ErrStorePersist, err)
    }

    if _, err := f.Write(data); err != nil {
        f.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("%w: write: %v", ErrStorePersist, err)
    }

    if err := f.Sync(); err != nil {
        f.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("%w: fsync: %v", ErrStorePersist, err)
    }

    if err := f.Close(); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("%w: close: %v", ErrStorePersist, err)
    }

    if err := os.Rename(tmpPath, s.path); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("%w: rename: %v", ErrStorePersist, err)
    }

    s.dirty = false
    return nil
}

// Sync flushes to disk.
func (s *BaoStore) Sync() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.syncLocked()
}

// Close syncs and releases.
func (s *BaoStore) Close() error {
    return s.Sync()
}

// Path returns store path.
func (s *BaoStore) Path() string {
    return s.path
}
```

---

## 3. Unit Tests

- Test atomic write leaves no temp file
- Test corrupted file detection
- Test version validation
- Test persistence across restart
- Test directory creation

---

## 4. Deliverables

- [ ] Atomic write pattern (temp + rename)
- [ ] fsync before rename
- [ ] File permissions 0600
- [ ] Directory permissions 0700
- [ ] Corrupted file handling
