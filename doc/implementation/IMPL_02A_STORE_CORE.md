# Implementation Guide: BaoStore Core

**Agent ID:** 02A  
**Parent:** Agent 02 (Storage Layer)  
**Component:** Core CRUD Operations  
**Parallelizable:** ✅ Yes - Uses types from 01A

---

## 1. Overview

Core CRUD operations for the metadata store: Save, Get, List, Delete, Rename.

### 1.1 Required Skills

| Skill           | Level    | Description              |
| --------------- | -------- | ------------------------ |
| **Go**          | Advanced | Structs, methods         |
| **Concurrency** | Advanced | sync.RWMutex             |

### 1.2 Files to Create

```
banhbaoring/
└── bao_store.go (core struct + CRUD methods)
```

---

## 2. Specifications

```go
package banhbaoring

import (
    "fmt"
    "sync"
)

// BaoStore manages local key metadata.
type BaoStore struct {
    path  string
    data  *StoreData
    mu    sync.RWMutex
    dirty bool
}

// Save stores key metadata.
func (s *BaoStore) Save(meta *KeyMetadata) error {
    if meta == nil {
        return fmt.Errorf("metadata cannot be nil")
    }
    if meta.UID == "" {
        return fmt.Errorf("metadata UID is required")
    }
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if existing, exists := s.data.Keys[meta.UID]; exists {
        if existing.Address != meta.Address {
            return fmt.Errorf("%w: %s", ErrKeyExists, meta.UID)
        }
    }
    
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
    return copyMetadata(meta), nil
}

// GetByAddress retrieves metadata by address.
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

// List returns all metadata.
func (s *BaoStore) List() ([]*KeyMetadata, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    result := make([]*KeyMetadata, 0, len(s.data.Keys))
    for _, meta := range s.data.Keys {
        result = append(result, copyMetadata(meta))
    }
    return result, nil
}

// Delete removes metadata.
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

// Rename changes the UID.
func (s *BaoStore) Rename(oldUID, newUID string) error {
    if oldUID == newUID {
        return nil
    }
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    meta, exists := s.data.Keys[oldUID]
    if !exists {
        return fmt.Errorf("%w: %s", ErrKeyNotFound, oldUID)
    }
    if _, exists := s.data.Keys[newUID]; exists {
        return fmt.Errorf("%w: %s", ErrKeyExists, newUID)
    }
    
    meta.UID = newUID
    meta.Name = newUID
    s.data.Keys[newUID] = meta
    delete(s.data.Keys, oldUID)
    s.dirty = true
    return s.syncLocked()
}

// Has checks existence.
func (s *BaoStore) Has(uid string) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data.Keys[uid] != nil
}

// Count returns key count.
func (s *BaoStore) Count() int {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return len(s.data.Keys)
}

func copyMetadata(meta *KeyMetadata) *KeyMetadata {
    if meta == nil {
        return nil
    }
    cp := *meta
    if meta.PubKeyBytes != nil {
        cp.PubKeyBytes = make([]byte, len(meta.PubKeyBytes))
        copy(cp.PubKeyBytes, meta.PubKeyBytes)
    }
    return &cp
}
```

---

## 3. Unit Tests

- Test Save with valid/invalid metadata
- Test Get returns copy not reference
- Test GetByAddress finds correct key
- Test Delete removes and persists
- Test Rename updates UID correctly
- Test concurrent access is safe

---

## 4. Deliverables

- [ ] Core CRUD methods implemented
- [ ] Thread-safe with RWMutex
- [ ] Returns copies, not references
- [ ] All error cases handled

