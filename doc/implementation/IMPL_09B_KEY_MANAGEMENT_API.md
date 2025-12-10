# Implementation: Key Management API

## Agent: 09B - Key Management API

> **Phase 5.2** - Can run in parallel with 09A, 09C after Phase 5.1 completes.

---

## 1. Overview

Implement the Control Plane Key Management API that wraps the core BaoKeyring library.

> **Critical:** This is where rollup teams manage their sequencer, prover, and bridge operator keys.

---

## 2. Scope

| Feature | Included |
|---------|----------|
| Create key | ✅ |
| List keys | ✅ |
| Get key | ✅ |
| Delete key | ✅ |
| Sign | ✅ |
| **Batch create** | ✅ (parallel workers) |
| **Batch sign** | ✅ (parallel workers) |
| Import key | ✅ |
| Export key | ✅ |

---

## 3. Integration with Core Library

```go
import banhbaoring "github.com/Bidon15/banhbaoring"

// The Control Plane wraps BaoKeyring, adding:
// - Multi-tenant isolation (per-org OpenBao namespaces)
// - Quota enforcement
// - Audit logging
// - API key authentication
```

---

## 4. Service

**File:** `internal/service/key_service.go`

```go
package service

import (
    "context"
    "encoding/base64"
    "fmt"
    "sync"

    "github.com/google/uuid"

    banhbaoring "github.com/Bidon15/banhbaoring"
    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
    "github.com/Bidon15/banhbaoring/control-plane/internal/repository"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type KeyService interface {
    // Single key operations
    Create(ctx context.Context, req CreateKeyRequest) (*models.Key, error)
    Get(ctx context.Context, orgID, keyID uuid.UUID) (*models.Key, error)
    List(ctx context.Context, orgID uuid.UUID, namespaceID *uuid.UUID) ([]*models.Key, error)
    Delete(ctx context.Context, orgID, keyID uuid.UUID) error
    Sign(ctx context.Context, orgID, keyID uuid.UUID, data []byte, prehashed bool) (*SignResponse, error)
    
    // Batch operations for parallel workers
    CreateBatch(ctx context.Context, req CreateBatchRequest) ([]*models.Key, error)
    SignBatch(ctx context.Context, req SignBatchRequest) ([]*SignResponse, error)
    
    // Import/Export
    Import(ctx context.Context, req ImportKeyRequest) (*models.Key, error)
    Export(ctx context.Context, orgID, keyID uuid.UUID) (string, error)
}

type CreateKeyRequest struct {
    OrgID       uuid.UUID `json:"-"`
    NamespaceID uuid.UUID `json:"namespace_id" validate:"required"`
    Name        string    `json:"name" validate:"required,min=1,max=100"`
    Algorithm   string    `json:"algorithm" validate:"omitempty,oneof=secp256k1"`
    Exportable  bool      `json:"exportable"`
    Metadata    map[string]string `json:"metadata,omitempty"`
}

type CreateBatchRequest struct {
    OrgID       uuid.UUID `json:"-"`
    NamespaceID uuid.UUID `json:"namespace_id" validate:"required"`
    Prefix      string    `json:"prefix" validate:"required,min=1,max=50"`
    Count       int       `json:"count" validate:"required,min=1,max=100"`
    Exportable  bool      `json:"exportable"`
}

type SignRequest struct {
    KeyID     uuid.UUID `json:"key_id" validate:"required"`
    Data      string    `json:"data" validate:"required"` // base64
    Prehashed bool      `json:"prehashed"`
}

type SignBatchRequest struct {
    OrgID    uuid.UUID     `json:"-"`
    Requests []SignRequest `json:"requests" validate:"required,min=1,max=100"`
}

type SignResponse struct {
    KeyID     uuid.UUID `json:"key_id"`
    Signature string    `json:"signature"` // base64
    PublicKey string    `json:"public_key"` // hex
    Error     string    `json:"error,omitempty"`
}

type ImportKeyRequest struct {
    OrgID       uuid.UUID `json:"-"`
    NamespaceID uuid.UUID `json:"namespace_id" validate:"required"`
    Name        string    `json:"name" validate:"required"`
    PrivateKey  string    `json:"private_key" validate:"required"` // base64
    Exportable  bool      `json:"exportable"`
}

type keyService struct {
    keyRepo     repository.KeyRepository
    orgRepo     repository.OrgRepository
    auditRepo   repository.AuditRepository
    usageRepo   repository.UsageRepository
    baoKeyring  *banhbaoring.BaoKeyring
}

func NewKeyService(
    keyRepo repository.KeyRepository,
    orgRepo repository.OrgRepository,
    auditRepo repository.AuditRepository,
    usageRepo repository.UsageRepository,
    baoKeyring *banhbaoring.BaoKeyring,
) KeyService {
    return &keyService{
        keyRepo:    keyRepo,
        orgRepo:    orgRepo,
        auditRepo:  auditRepo,
        usageRepo:  usageRepo,
        baoKeyring: baoKeyring,
    }
}

func (s *keyService) Create(ctx context.Context, req CreateKeyRequest) (*models.Key, error) {
    // Check quota
    if err := s.checkKeyQuota(ctx, req.OrgID); err != nil {
        return nil, err
    }

    // Verify namespace belongs to org
    ns, err := s.orgRepo.GetNamespace(ctx, req.NamespaceID)
    if err != nil || ns == nil || ns.OrgID != req.OrgID {
        return nil, apierrors.NewNotFoundError("Namespace")
    }

    // Generate unique OpenBao key name
    baoKeyName := fmt.Sprintf("%s_%s_%s", req.OrgID, req.NamespaceID, req.Name)

    // Create in BaoKeyring
    record, err := s.baoKeyring.NewAccountWithOptions(baoKeyName, banhbaoring.KeyOptions{
        Exportable: req.Exportable,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create key in OpenBao: %w", err)
    }

    // Get public key and address
    pubKey := record.GetPubKey()
    address, err := sdk.Bech32ifyAddressBytes("celestia", pubKey.Address())
    if err != nil {
        address = pubKey.Address().String()
    }

    // Save metadata to database
    key := &models.Key{
        ID:          uuid.New(),
        OrgID:       req.OrgID,
        NamespaceID: req.NamespaceID,
        Name:        req.Name,
        PublicKey:   pubKey.Bytes(),
        Address:     address,
        Algorithm:   "secp256k1",
        BaoKeyPath:  baoKeyName,
        Exportable:  req.Exportable,
        Metadata:    req.Metadata,
    }

    if err := s.keyRepo.Create(ctx, key); err != nil {
        // Cleanup OpenBao key on failure
        _ = s.baoKeyring.Delete(baoKeyName)
        return nil, err
    }

    // Audit log
    s.auditLog(ctx, req.OrgID, "key.created", key.ID)

    return key, nil
}

func (s *keyService) CreateBatch(ctx context.Context, req CreateBatchRequest) ([]*models.Key, error) {
    // Check quota for all keys
    limits, err := s.getOrgLimits(ctx, req.OrgID)
    if err != nil {
        return nil, err
    }

    currentCount, err := s.keyRepo.CountByOrg(ctx, req.OrgID)
    if err != nil {
        return nil, err
    }

    if limits.Keys > 0 && currentCount+req.Count > limits.Keys {
        return nil, apierrors.ErrQuotaExceeded
    }

    // Create keys in parallel
    keys := make([]*models.Key, req.Count)
    errs := make([]error, req.Count)
    var wg sync.WaitGroup

    for i := 0; i < req.Count; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            name := fmt.Sprintf("%s-%d", req.Prefix, idx+1)
            key, err := s.Create(ctx, CreateKeyRequest{
                OrgID:       req.OrgID,
                NamespaceID: req.NamespaceID,
                Name:        name,
                Exportable:  req.Exportable,
            })
            keys[idx] = key
            errs[idx] = err
        }(i)
    }
    wg.Wait()

    // Collect results (partial success is possible)
    var result []*models.Key
    for i, key := range keys {
        if errs[i] == nil && key != nil {
            result = append(result, key)
        }
    }

    if len(result) == 0 {
        return nil, fmt.Errorf("batch create failed: %v", errs[0])
    }

    return result, nil
}

func (s *keyService) Sign(ctx context.Context, orgID, keyID uuid.UUID, data []byte, prehashed bool) (*SignResponse, error) {
    // Get key
    key, err := s.keyRepo.GetByID(ctx, keyID)
    if err != nil {
        return nil, err
    }
    if key == nil || key.OrgID != orgID || key.DeletedAt != nil {
        return nil, apierrors.NewNotFoundError("Key")
    }

    // Check signature quota
    if err := s.checkSignatureQuota(ctx, orgID); err != nil {
        return nil, err
    }

    // Sign via BaoKeyring
    sig, pubKey, err := s.baoKeyring.Sign(key.BaoKeyPath, data, signing.SignMode_SIGN_MODE_DIRECT)
    if err != nil {
        return nil, fmt.Errorf("signing failed: %w", err)
    }

    // Increment usage counter
    s.incrementUsage(ctx, orgID, "signatures", 1)

    // Audit log
    s.auditLog(ctx, orgID, "key.signed", keyID)

    return &SignResponse{
        KeyID:     keyID,
        Signature: base64.StdEncoding.EncodeToString(sig),
        PublicKey: fmt.Sprintf("%x", pubKey.Bytes()),
    }, nil
}

func (s *keyService) SignBatch(ctx context.Context, req SignBatchRequest) ([]*SignResponse, error) {
    // Check quota for all signatures
    if err := s.checkSignatureQuota(ctx, req.OrgID); err != nil {
        return nil, err
    }

    // Sign in parallel (no head-of-line blocking!)
    results := make([]*SignResponse, len(req.Requests))
    var wg sync.WaitGroup

    for i, signReq := range req.Requests {
        wg.Add(1)
        go func(idx int, r SignRequest) {
            defer wg.Done()

            data, err := base64.StdEncoding.DecodeString(r.Data)
            if err != nil {
                results[idx] = &SignResponse{KeyID: r.KeyID, Error: "invalid base64"}
                return
            }

            resp, err := s.Sign(ctx, req.OrgID, r.KeyID, data, r.Prehashed)
            if err != nil {
                results[idx] = &SignResponse{KeyID: r.KeyID, Error: err.Error()}
                return
            }
            results[idx] = resp
        }(i, signReq)
    }
    wg.Wait()

    return results, nil
}

func (s *keyService) List(ctx context.Context, orgID uuid.UUID, namespaceID *uuid.UUID) ([]*models.Key, error) {
    if namespaceID != nil {
        return s.keyRepo.ListByNamespace(ctx, *namespaceID)
    }
    return s.keyRepo.ListByOrg(ctx, orgID)
}

func (s *keyService) Get(ctx context.Context, orgID, keyID uuid.UUID) (*models.Key, error) {
    key, err := s.keyRepo.GetByID(ctx, keyID)
    if err != nil {
        return nil, err
    }
    if key == nil || key.OrgID != orgID || key.DeletedAt != nil {
        return nil, apierrors.NewNotFoundError("Key")
    }
    return key, nil
}

func (s *keyService) Delete(ctx context.Context, orgID, keyID uuid.UUID) error {
    key, err := s.keyRepo.GetByID(ctx, keyID)
    if err != nil {
        return err
    }
    if key == nil || key.OrgID != orgID {
        return apierrors.NewNotFoundError("Key")
    }

    // Delete from OpenBao
    if err := s.baoKeyring.Delete(key.BaoKeyPath); err != nil {
        return fmt.Errorf("failed to delete from OpenBao: %w", err)
    }

    // Soft delete in database
    if err := s.keyRepo.SoftDelete(ctx, keyID); err != nil {
        return err
    }

    s.auditLog(ctx, orgID, "key.deleted", keyID)
    return nil
}

// Helper methods
func (s *keyService) checkKeyQuota(ctx context.Context, orgID uuid.UUID) error {
    limits, err := s.getOrgLimits(ctx, orgID)
    if err != nil {
        return err
    }
    if limits.Keys < 0 { // unlimited
        return nil
    }

    count, err := s.keyRepo.CountByOrg(ctx, orgID)
    if err != nil {
        return err
    }

    if count >= limits.Keys {
        return apierrors.ErrQuotaExceeded
    }
    return nil
}

func (s *keyService) checkSignatureQuota(ctx context.Context, orgID uuid.UUID) error {
    limits, err := s.getOrgLimits(ctx, orgID)
    if err != nil {
        return err
    }
    if limits.SignaturesPerMonth < 0 { // unlimited
        return nil
    }

    usage, err := s.usageRepo.GetCurrentPeriod(ctx, orgID, "signatures")
    if err != nil {
        return err
    }

    if usage >= limits.SignaturesPerMonth {
        return apierrors.ErrQuotaExceeded
    }
    return nil
}

func (s *keyService) getOrgLimits(ctx context.Context, orgID uuid.UUID) (*models.PlanLimits, error) {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return nil, err
    }
    limits := models.PlanLimitsMap[org.Plan]
    return &limits, nil
}

func (s *keyService) incrementUsage(ctx context.Context, orgID uuid.UUID, metric string, value int64) {
    go func() {
        _ = s.usageRepo.Increment(context.Background(), orgID, metric, value)
    }()
}

func (s *keyService) auditLog(ctx context.Context, orgID uuid.UUID, event string, resourceID uuid.UUID) {
    go func() {
        _ = s.auditRepo.Create(context.Background(), &models.AuditLog{
            OrgID:        orgID,
            Event:        event,
            ResourceType: "key",
            ResourceID:   resourceID,
        })
    }()
}
```

---

## 5. Handler

**File:** `internal/handler/key_handler.go`

```go
package handler

import (
    "encoding/base64"
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/go-playground/validator/v10"

    "github.com/Bidon15/banhbaoring/control-plane/internal/middleware"
    "github.com/Bidon15/banhbaoring/control-plane/internal/service"
    "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type KeyHandler struct {
    keyService service.KeyService
    validate   *validator.Validate
}

func NewKeyHandler(keyService service.KeyService) *KeyHandler {
    return &KeyHandler{
        keyService: keyService,
        validate:   validator.New(),
    }
}

func (h *KeyHandler) Routes() chi.Router {
    r := chi.NewRouter()

    r.With(middleware.RequireScope("keys:read")).Get("/", h.List)
    r.With(middleware.RequireScope("keys:write")).Post("/", h.Create)
    r.With(middleware.RequireScope("keys:write")).Post("/batch", h.CreateBatch)
    r.With(middleware.RequireScope("keys:read")).Get("/{id}", h.Get)
    r.With(middleware.RequireScope("keys:write")).Delete("/{id}", h.Delete)
    r.With(middleware.RequireScope("keys:sign")).Post("/{id}/sign", h.Sign)

    // Batch sign at /v1/sign/batch for convenience
    return r
}

func (h *KeyHandler) Create(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    var req service.CreateKeyRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }
    req.OrgID = orgID

    if err := h.validate.Struct(req); err != nil {
        response.Error(w, apierrors.NewValidationError("", err.Error()))
        return
    }

    key, err := h.keyService.Create(r.Context(), req)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.Created(w, key)
}

func (h *KeyHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    var req service.CreateBatchRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }
    req.OrgID = orgID

    if err := h.validate.Struct(req); err != nil {
        response.Error(w, apierrors.NewValidationError("", err.Error()))
        return
    }

    keys, err := h.keyService.CreateBatch(r.Context(), req)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.Created(w, map[string]any{"keys": keys})
}

func (h *KeyHandler) List(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    var nsID *uuid.UUID
    if nsStr := r.URL.Query().Get("namespace_id"); nsStr != "" {
        id, err := uuid.Parse(nsStr)
        if err == nil {
            nsID = &id
        }
    }

    keys, err := h.keyService.List(r.Context(), orgID, nsID)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.OK(w, keys)
}

func (h *KeyHandler) Get(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)
    keyID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    key, err := h.keyService.Get(r.Context(), orgID, keyID)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.OK(w, key)
}

func (h *KeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)
    keyID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    if err := h.keyService.Delete(r.Context(), orgID, keyID); err != nil {
        response.Error(w, err)
        return
    }

    response.NoContent(w)
}

func (h *KeyHandler) Sign(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)
    keyID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    var req struct {
        Data      string `json:"data"`
        Prehashed bool   `json:"prehashed"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    data, err := base64.StdEncoding.DecodeString(req.Data)
    if err != nil {
        response.Error(w, apierrors.NewValidationError("data", "invalid base64"))
        return
    }

    result, err := h.keyService.Sign(r.Context(), orgID, keyID, data, req.Prehashed)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.OK(w, result)
}
```

---

## 6. Batch Sign Handler

**File:** `internal/handler/sign_handler.go`

```go
package handler

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"

    "github.com/Bidon15/banhbaoring/control-plane/internal/middleware"
    "github.com/Bidon15/banhbaoring/control-plane/internal/service"
    "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type SignHandler struct {
    keyService service.KeyService
}

func NewSignHandler(keyService service.KeyService) *SignHandler {
    return &SignHandler{keyService: keyService}
}

func (h *SignHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.With(middleware.RequireScope("keys:sign")).Post("/batch", h.SignBatch)
    return r
}

func (h *SignHandler) SignBatch(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    var req service.SignBatchRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }
    req.OrgID = orgID

    if len(req.Requests) == 0 || len(req.Requests) > 100 {
        response.Error(w, apierrors.NewValidationError("requests", "must be 1-100 items"))
        return
    }

    results, err := h.keyService.SignBatch(r.Context(), req)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.OK(w, map[string]any{"signatures": results})
}
```

---

## 7. Deliverables

| File | Description |
|------|-------------|
| `internal/models/key.go` | Key model |
| `internal/repository/key_repo.go` | Database operations |
| `internal/service/key_service.go` | Business logic with batch ops |
| `internal/handler/key_handler.go` | Key HTTP handlers |
| `internal/handler/sign_handler.go` | Batch sign handler |

---

## 8. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/keys` | Create key |
| POST | `/v1/keys/batch` | Create N keys (parallel workers) |
| GET | `/v1/keys` | List keys |
| GET | `/v1/keys/{id}` | Get key |
| DELETE | `/v1/keys/{id}` | Delete key |
| POST | `/v1/keys/{id}/sign` | Sign data |
| POST | `/v1/sign/batch` | Batch sign (parallel) |
| POST | `/v1/keys/import` | Import key |
| POST | `/v1/keys/{id}/export` | Export key |

---

## 9. Success Criteria

- [ ] Create key stores in both OpenBao and PostgreSQL
- [ ] Batch create creates N keys in parallel
- [ ] Sign works end-to-end
- [ ] Batch sign executes in parallel (not sequential!)
- [ ] Quota enforcement works
- [ ] Audit logging works
- [ ] Tests pass

---

## 10. Agent Prompt

```
You are Agent 09B - Key Management API. Implement the Control Plane key management.

Read the spec: doc/implementation/IMPL_09B_KEY_MANAGEMENT_API.md

CRITICAL: This wraps the core BaoKeyring library. Keys must be stored in BOTH OpenBao (via BaoKeyring) AND PostgreSQL (metadata).

Deliverables:
1. Key service with Create, List, Get, Delete, Sign
2. CreateBatch for parallel worker key creation
3. SignBatch for parallel signing (no head-of-line blocking!)
4. Quota enforcement (keys per org, signatures per month)
5. Audit logging
6. HTTP handlers
7. Tests

Dependencies: Agents 07, 08A/08B/08C must complete first.

Test: go test ./internal/... -v
```

