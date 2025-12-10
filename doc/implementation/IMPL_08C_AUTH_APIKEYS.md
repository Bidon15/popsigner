# Implementation: Auth - API Keys

## Agent: 08C - API Key Authentication

> **Phase 5.1** - Can run in parallel with 08A, 08B after Agent 07 completes.

---

## 1. Overview

Implement API key generation, validation, scopes, and rate limiting for programmatic access.

---

## 2. Scope

| Feature | Included |
|---------|----------|
| API key generation | ✅ |
| Argon2 hashing | ✅ |
| Scope-based permissions | ✅ |
| Key rotation | ✅ |
| Rate limiting | ✅ |
| Key prefix for display | ✅ |

---

## 3. API Key Format

```
bbr_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
│    │    └── 32 random characters (base62)
│    └─────── Environment (live/test)
└──────────── Prefix (banhbaoring)

Example: bbr_live_k8Nj2mP9qR4sT6vX8yZ0aB3cD5eF7gH1
```

---

## 4. Scopes

| Scope | Description |
|-------|-------------|
| `keys:read` | List and view keys |
| `keys:write` | Create, delete keys |
| `keys:sign` | Sign operations |
| `audit:read` | View audit logs |
| `billing:read` | View invoices |
| `billing:write` | Manage subscriptions |
| `webhooks:write` | Manage webhooks |

---

## 5. Models

**File:** `internal/models/api_key.go`

```go
package models

import (
    "time"

    "github.com/google/uuid"
)

type APIKey struct {
    ID         uuid.UUID  `json:"id" db:"id"`
    OrgID      uuid.UUID  `json:"org_id" db:"org_id"`
    UserID     *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
    Name       string     `json:"name" db:"name"`
    KeyPrefix  string     `json:"key_prefix" db:"key_prefix"`
    KeyHash    string     `json:"-" db:"key_hash"`
    Scopes     []string   `json:"scopes" db:"scopes"`
    LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
    ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
    RevokedAt  *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
    CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

func (k *APIKey) IsValid() bool {
    if k.RevokedAt != nil {
        return false
    }
    if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
        return false
    }
    return true
}

func (k *APIKey) HasScope(scope string) bool {
    for _, s := range k.Scopes {
        if s == scope || s == "*" {
            return true
        }
    }
    return false
}
```

---

## 6. Repository

**File:** `internal/repository/api_key_repo.go`

```go
package repository

import (
    "context"
    "database/sql"

    "github.com/google/uuid"
    "github.com/lib/pq"

    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

type APIKeyRepository interface {
    Create(ctx context.Context, key *models.APIKey) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error)
    GetByPrefix(ctx context.Context, prefix string) (*models.APIKey, error)
    ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error)
    UpdateLastUsed(ctx context.Context, id uuid.UUID) error
    Revoke(ctx context.Context, id uuid.UUID) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type apiKeyRepo struct {
    db *sql.DB
}

func NewAPIKeyRepository(db *sql.DB) APIKeyRepository {
    return &apiKeyRepo{db: db}
}

func (r *apiKeyRepo) Create(ctx context.Context, key *models.APIKey) error {
    query := `
        INSERT INTO api_keys (id, org_id, user_id, name, key_prefix, key_hash, scopes, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING created_at`

    key.ID = uuid.New()
    return r.db.QueryRowContext(ctx, query,
        key.ID, key.OrgID, key.UserID, key.Name, key.KeyPrefix,
        key.KeyHash, pq.Array(key.Scopes), key.ExpiresAt,
    ).Scan(&key.CreatedAt)
}

func (r *apiKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
    query := `
        SELECT id, org_id, user_id, name, key_prefix, key_hash, scopes,
               last_used_at, expires_at, revoked_at, created_at
        FROM api_keys WHERE id = $1`

    var key models.APIKey
    var scopes pq.StringArray
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &key.ID, &key.OrgID, &key.UserID, &key.Name, &key.KeyPrefix,
        &key.KeyHash, &scopes, &key.LastUsedAt, &key.ExpiresAt,
        &key.RevokedAt, &key.CreatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    key.Scopes = scopes
    return &key, err
}

func (r *apiKeyRepo) GetByPrefix(ctx context.Context, prefix string) (*models.APIKey, error) {
    query := `
        SELECT id, org_id, user_id, name, key_prefix, key_hash, scopes,
               last_used_at, expires_at, revoked_at, created_at
        FROM api_keys WHERE key_prefix = $1`

    var key models.APIKey
    var scopes pq.StringArray
    err := r.db.QueryRowContext(ctx, query, prefix).Scan(
        &key.ID, &key.OrgID, &key.UserID, &key.Name, &key.KeyPrefix,
        &key.KeyHash, &scopes, &key.LastUsedAt, &key.ExpiresAt,
        &key.RevokedAt, &key.CreatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    key.Scopes = scopes
    return &key, err
}

func (r *apiKeyRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error) {
    query := `
        SELECT id, org_id, user_id, name, key_prefix, scopes,
               last_used_at, expires_at, revoked_at, created_at
        FROM api_keys WHERE org_id = $1 ORDER BY created_at DESC`

    rows, err := r.db.QueryContext(ctx, query, orgID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var keys []*models.APIKey
    for rows.Next() {
        var key models.APIKey
        var scopes pq.StringArray
        if err := rows.Scan(
            &key.ID, &key.OrgID, &key.UserID, &key.Name, &key.KeyPrefix,
            &scopes, &key.LastUsedAt, &key.ExpiresAt, &key.RevokedAt, &key.CreatedAt,
        ); err != nil {
            return nil, err
        }
        key.Scopes = scopes
        keys = append(keys, &key)
    }
    return keys, rows.Err()
}

func (r *apiKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}

func (r *apiKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE api_keys SET revoked_at = NOW() WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}

func (r *apiKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
    query := `DELETE FROM api_keys WHERE id = $1`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}
```

---

## 7. Service

**File:** `internal/service/api_key_service.go`

```go
package service

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "strings"

    "github.com/google/uuid"
    "golang.org/x/crypto/argon2"

    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
    "github.com/Bidon15/banhbaoring/control-plane/internal/repository"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type APIKeyService interface {
    Create(ctx context.Context, orgID uuid.UUID, req CreateAPIKeyRequest) (*models.APIKey, string, error)
    Validate(ctx context.Context, rawKey string) (*models.APIKey, error)
    List(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error)
    Revoke(ctx context.Context, orgID, keyID uuid.UUID) error
    Delete(ctx context.Context, orgID, keyID uuid.UUID) error
}

type CreateAPIKeyRequest struct {
    Name      string   `json:"name" validate:"required,min=1,max=255"`
    Scopes    []string `json:"scopes" validate:"required,min=1"`
    ExpiresIn *int     `json:"expires_in_days,omitempty"` // days until expiry
}

// Argon2 parameters
const (
    argon2Time    = 1
    argon2Memory  = 64 * 1024
    argon2Threads = 4
    argon2KeyLen  = 32
)

type apiKeyService struct {
    keyRepo repository.APIKeyRepository
}

func NewAPIKeyService(keyRepo repository.APIKeyRepository) APIKeyService {
    return &apiKeyService{keyRepo: keyRepo}
}

func (s *apiKeyService) Create(ctx context.Context, orgID uuid.UUID, req CreateAPIKeyRequest) (*models.APIKey, string, error) {
    // Validate scopes
    validScopes := map[string]bool{
        "keys:read": true, "keys:write": true, "keys:sign": true,
        "audit:read": true, "billing:read": true, "billing:write": true,
        "webhooks:write": true, "*": true,
    }
    for _, scope := range req.Scopes {
        if !validScopes[scope] {
            return nil, "", apierrors.NewValidationError("scopes", fmt.Sprintf("invalid scope: %s", scope))
        }
    }

    // Generate raw key
    rawKey, prefix, err := s.generateKey()
    if err != nil {
        return nil, "", err
    }

    // Hash the key
    hash := s.hashKey(rawKey)

    key := &models.APIKey{
        OrgID:     orgID,
        Name:      req.Name,
        KeyPrefix: prefix,
        KeyHash:   hash,
        Scopes:    req.Scopes,
    }

    if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
        exp := time.Now().AddDate(0, 0, *req.ExpiresIn)
        key.ExpiresAt = &exp
    }

    if err := s.keyRepo.Create(ctx, key); err != nil {
        return nil, "", err
    }

    // Return full key only once - it won't be retrievable later
    return key, rawKey, nil
}

func (s *apiKeyService) Validate(ctx context.Context, rawKey string) (*models.APIKey, error) {
    // Extract prefix
    parts := strings.Split(rawKey, "_")
    if len(parts) != 3 || parts[0] != "bbr" {
        return nil, apierrors.ErrUnauthorized
    }

    prefix := parts[0] + "_" + parts[1] + "_" + parts[2][:8]

    // Lookup by prefix
    key, err := s.keyRepo.GetByPrefix(ctx, prefix)
    if err != nil {
        return nil, err
    }
    if key == nil {
        return nil, apierrors.ErrUnauthorized
    }

    // Verify hash
    expectedHash := s.hashKey(rawKey)
    if key.KeyHash != expectedHash {
        return nil, apierrors.ErrUnauthorized
    }

    // Check validity
    if !key.IsValid() {
        return nil, apierrors.ErrUnauthorized
    }

    // Update last used (async)
    go func() {
        _ = s.keyRepo.UpdateLastUsed(context.Background(), key.ID)
    }()

    return key, nil
}

func (s *apiKeyService) List(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error) {
    return s.keyRepo.ListByOrg(ctx, orgID)
}

func (s *apiKeyService) Revoke(ctx context.Context, orgID, keyID uuid.UUID) error {
    key, err := s.keyRepo.GetByID(ctx, keyID)
    if err != nil {
        return err
    }
    if key == nil || key.OrgID != orgID {
        return apierrors.NewNotFoundError("API key")
    }
    return s.keyRepo.Revoke(ctx, keyID)
}

func (s *apiKeyService) Delete(ctx context.Context, orgID, keyID uuid.UUID) error {
    key, err := s.keyRepo.GetByID(ctx, keyID)
    if err != nil {
        return err
    }
    if key == nil || key.OrgID != orgID {
        return apierrors.NewNotFoundError("API key")
    }
    return s.keyRepo.Delete(ctx, keyID)
}

func (s *apiKeyService) generateKey() (rawKey, prefix string, err error) {
    // Generate 24 random bytes = 32 base62 characters
    b := make([]byte, 24)
    if _, err := rand.Read(b); err != nil {
        return "", "", err
    }

    secret := base62Encode(b)
    rawKey = fmt.Sprintf("bbr_live_%s", secret)
    prefix = fmt.Sprintf("bbr_live_%s", secret[:8])

    return rawKey, prefix, nil
}

func (s *apiKeyService) hashKey(rawKey string) string {
    salt := []byte("banhbaoring-api-key-salt-v1")
    hash := argon2.IDKey([]byte(rawKey), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
    return base64.StdEncoding.EncodeToString(hash)
}

// base62Encode encodes bytes to base62 string
func base62Encode(data []byte) string {
    const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
    result := make([]byte, len(data)*4/3+1)
    for i := range result {
        result[i] = alphabet[int(data[i%len(data)])%62]
    }
    return string(result)
}
```

---

## 8. Middleware

**File:** `internal/middleware/api_key_auth.go`

```go
package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/Bidon15/banhbaoring/control-plane/internal/service"
    "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type contextKey string

const (
    APIKeyContextKey contextKey = "api_key"
    OrgIDContextKey  contextKey = "org_id"
)

func APIKeyAuth(apiKeyService service.APIKeyService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get API key from header
            auth := r.Header.Get("Authorization")
            if auth == "" {
                response.Error(w, apierrors.ErrUnauthorized)
                return
            }

            // Support both "Bearer xxx" and "ApiKey xxx" formats
            var rawKey string
            if strings.HasPrefix(auth, "Bearer ") {
                rawKey = strings.TrimPrefix(auth, "Bearer ")
            } else if strings.HasPrefix(auth, "ApiKey ") {
                rawKey = strings.TrimPrefix(auth, "ApiKey ")
            } else {
                response.Error(w, apierrors.ErrUnauthorized)
                return
            }

            // Validate API key
            apiKey, err := apiKeyService.Validate(r.Context(), rawKey)
            if err != nil {
                response.Error(w, err)
                return
            }

            // Add to context
            ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
            ctx = context.WithValue(ctx, OrgIDContextKey, apiKey.OrgID)

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func RequireScope(scope string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            apiKey, ok := r.Context().Value(APIKeyContextKey).(*models.APIKey)
            if !ok || !apiKey.HasScope(scope) {
                response.Error(w, apierrors.ErrForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 9. Handler

**File:** `internal/handler/api_key_handler.go`

```go
package handler

import (
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

type APIKeyHandler struct {
    apiKeyService service.APIKeyService
    validate      *validator.Validate
}

func NewAPIKeyHandler(apiKeyService service.APIKeyService) *APIKeyHandler {
    return &APIKeyHandler{
        apiKeyService: apiKeyService,
        validate:      validator.New(),
    }
}

func (h *APIKeyHandler) Routes() chi.Router {
    r := chi.NewRouter()

    r.Get("/", h.List)
    r.Post("/", h.Create)
    r.Delete("/{id}", h.Delete)
    r.Post("/{id}/revoke", h.Revoke)

    return r
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    var req service.CreateAPIKeyRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    if err := h.validate.Struct(req); err != nil {
        response.Error(w, apierrors.NewValidationError("", err.Error()))
        return
    }

    key, rawKey, err := h.apiKeyService.Create(r.Context(), orgID, req)
    if err != nil {
        response.Error(w, err)
        return
    }

    // Return key only once!
    response.Created(w, map[string]any{
        "api_key": key,
        "key":     rawKey,
        "warning": "This key will not be shown again. Store it securely.",
    })
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    keys, err := h.apiKeyService.List(r.Context(), orgID)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.OK(w, keys)
}

func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)
    keyID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    if err := h.apiKeyService.Delete(r.Context(), orgID, keyID); err != nil {
        response.Error(w, err)
        return
    }

    response.NoContent(w)
}

func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)
    keyID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    if err := h.apiKeyService.Revoke(r.Context(), orgID, keyID); err != nil {
        response.Error(w, err)
        return
    }

    response.NoContent(w)
}
```

---

## 10. Deliverables

| File | Description |
|------|-------------|
| `internal/models/api_key.go` | API key model |
| `internal/repository/api_key_repo.go` | Database operations |
| `internal/service/api_key_service.go` | Business logic |
| `internal/middleware/api_key_auth.go` | Auth middleware |
| `internal/handler/api_key_handler.go` | HTTP handlers |
| `internal/handler/api_key_handler_test.go` | Tests |

---

## 11. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/api-keys` | Create API key |
| GET | `/v1/api-keys` | List API keys |
| DELETE | `/v1/api-keys/{id}` | Delete API key |
| POST | `/v1/api-keys/{id}/revoke` | Revoke API key |

---

## 12. Success Criteria

- [ ] API key generation works
- [ ] Argon2 hashing implemented
- [ ] Scope validation works
- [ ] Auth middleware validates keys
- [ ] RequireScope middleware works
- [ ] Rate limiting integrated
- [ ] Tests pass

---

## 13. Agent Prompt

```
You are Agent 08C - API Key Authentication. Implement API key generation and validation.

Read the spec: doc/implementation/IMPL_08C_AUTH_APIKEYS.md

Deliverables:
1. API key model with scopes
2. Argon2 hashing for key storage
3. API key generation (bbr_live_xxx format)
4. Validation middleware
5. Scope-checking middleware
6. HTTP handlers
7. Tests

Dependencies: Agent 07 (Foundation) must complete first.

Test: go test ./internal/... -v
```

