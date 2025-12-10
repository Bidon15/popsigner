# Implementation Guide: Error Definitions

**Agent ID:** 01B  
**Parent:** Agent 01 (Foundation Layer)  
**Component:** Custom Error Types and Handling  
**Parallelizable:** ✅ Yes - No dependencies

---

## 1. Overview

Define all custom error types, sentinel errors, and error wrapping utilities.

### 1.1 Required Skills

| Skill             | Level        | Description                |
| ----------------- | ------------ | -------------------------- |
| **Go**            | Advanced     | Error interfaces, wrapping |
| **Error Design**  | Intermediate | Error hierarchies          |

### 1.2 Files to Create

```
banhbaoring/
└── errors.go
└── errors_test.go
```

---

## 2. Specifications

### 2.1 errors.go

```go
package banhbaoring

import (
    "errors"
    "fmt"
)

// Sentinel errors - Configuration
var (
    ErrMissingBaoAddr   = errors.New("banhbaoring: BaoAddr is required")
    ErrMissingBaoToken  = errors.New("banhbaoring: BaoToken is required")
    ErrMissingStorePath = errors.New("banhbaoring: StorePath is required")
)

// Sentinel errors - Keys
var (
    ErrKeyNotFound      = errors.New("banhbaoring: key not found")
    ErrKeyExists        = errors.New("banhbaoring: key already exists")
    ErrKeyNotExportable = errors.New("banhbaoring: key is not exportable")
)

// Sentinel errors - OpenBao
var (
    ErrBaoConnection  = errors.New("banhbaoring: failed to connect to OpenBao")
    ErrBaoAuth        = errors.New("banhbaoring: authentication failed")
    ErrBaoSealed      = errors.New("banhbaoring: OpenBao is sealed")
    ErrBaoUnavailable = errors.New("banhbaoring: OpenBao is unavailable")
)

// Sentinel errors - Operations
var (
    ErrSigningFailed    = errors.New("banhbaoring: signing failed")
    ErrInvalidSignature = errors.New("banhbaoring: invalid signature")
    ErrUnsupportedAlgo  = errors.New("banhbaoring: unsupported algorithm")
    ErrStorePersist     = errors.New("banhbaoring: failed to persist")
    ErrStoreCorrupted   = errors.New("banhbaoring: store corrupted")
)

// BaoError represents an OpenBao API error.
type BaoError struct {
    StatusCode int
    Errors     []string
    RequestID  string
}

func (e *BaoError) Error() string {
    if len(e.Errors) == 0 {
        return fmt.Sprintf("OpenBao error (HTTP %d)", e.StatusCode)
    }
    return fmt.Sprintf("OpenBao error (HTTP %d): %s", e.StatusCode, e.Errors[0])
}

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

func NewBaoError(statusCode int, errs []string, requestID string) *BaoError {
    return &BaoError{StatusCode: statusCode, Errors: errs, RequestID: requestID}
}

// KeyError wraps an error with key context.
type KeyError struct {
    KeyName string
    Op      string
    Err     error
}

func (e *KeyError) Error() string {
    return fmt.Sprintf("%s key %q: %v", e.Op, e.KeyName, e.Err)
}

func (e *KeyError) Unwrap() error {
    return e.Err
}

func WrapKeyError(op, keyName string, err error) error {
    if err == nil {
        return nil
    }
    return &KeyError{KeyName: keyName, Op: op, Err: err}
}

// ValidationError for config validation
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}
```

---

## 3. Unit Tests

```go
func TestBaoError_Error(t *testing.T) {
    tests := []struct {
        name     string
        err      *BaoError
        expected string
    }{
        {
            name:     "with message",
            err:      &BaoError{StatusCode: 403, Errors: []string{"denied"}},
            expected: "OpenBao error (HTTP 403): denied",
        },
        {
            name:     "without message",
            err:      &BaoError{StatusCode: 500},
            expected: "OpenBao error (HTTP 500)",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.expected, tt.err.Error())
        })
    }
}

func TestBaoError_Is(t *testing.T) {
    assert.True(t, errors.Is(&BaoError{StatusCode: 403}, ErrBaoAuth))
    assert.True(t, errors.Is(&BaoError{StatusCode: 404}, ErrKeyNotFound))
    assert.False(t, errors.Is(&BaoError{StatusCode: 403}, ErrKeyNotFound))
}

func TestKeyError_Unwrap(t *testing.T) {
    err := WrapKeyError("sign", "mykey", ErrSigningFailed)
    assert.True(t, errors.Is(err, ErrSigningFailed))
    assert.Contains(t, err.Error(), "mykey")
}
```

---

## 4. Deliverables

- [ ] `errors.go` with all error types
- [ ] `errors_test.go` with unit tests
- [ ] All sentinel errors categorized
- [ ] `errors.Is()` works correctly

