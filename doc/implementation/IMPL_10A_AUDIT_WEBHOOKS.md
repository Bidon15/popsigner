# Implementation: Audit Logs & Webhooks

## Agent: 10A - Audit & Webhooks

> **Phase 5.3** - Can run in parallel with 10B after Phase 5.2 completes.

---

## 1. Overview

Implement audit logging for all key operations and webhook delivery for event notifications.

---

## 2. Scope

| Feature | Included |
|---------|----------|
| Audit log creation | ✅ |
| Audit log querying | ✅ |
| Audit log retention | ✅ |
| Webhook configuration | ✅ |
| Webhook delivery | ✅ |
| Webhook signature verification | ✅ |
| Retry logic | ✅ |

---

## 3. Audit Events

| Event | Description |
|-------|-------------|
| `key.created` | New key created |
| `key.deleted` | Key deleted |
| `key.signed` | Signing operation |
| `key.exported` | Key exported |
| `key.imported` | Key imported |
| `auth.login` | User login |
| `auth.api_key_used` | API key authentication |
| `member.invited` | Team member added |
| `member.removed` | Team member removed |
| `billing.charge` | Payment processed |
| `quota.warning` | 80% usage |
| `quota.exceeded` | Quota exceeded |

---

## 4. Models

**File:** `internal/models/audit.go`

```go
package models

import (
    "time"

    "github.com/google/uuid"
)

type AuditLog struct {
    ID           uuid.UUID         `json:"id" db:"id"`
    OrgID        uuid.UUID         `json:"org_id" db:"org_id"`
    Event        string            `json:"event" db:"event"`
    ActorID      *uuid.UUID        `json:"actor_id,omitempty" db:"actor_id"`
    ActorType    string            `json:"actor_type" db:"actor_type"` // user, api_key, system
    ResourceType string            `json:"resource_type,omitempty" db:"resource_type"`
    ResourceID   *uuid.UUID        `json:"resource_id,omitempty" db:"resource_id"`
    IPAddress    string            `json:"ip_address,omitempty" db:"ip_address"`
    UserAgent    string            `json:"user_agent,omitempty" db:"user_agent"`
    Metadata     map[string]any    `json:"metadata,omitempty" db:"metadata"`
    CreatedAt    time.Time         `json:"created_at" db:"created_at"`
}

type Webhook struct {
    ID              uuid.UUID  `json:"id" db:"id"`
    OrgID           uuid.UUID  `json:"org_id" db:"org_id"`
    URL             string     `json:"url" db:"url"`
    Secret          string     `json:"-" db:"secret"`
    Events          []string   `json:"events" db:"events"`
    Enabled         bool       `json:"enabled" db:"enabled"`
    LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty" db:"last_triggered_at"`
    FailureCount    int        `json:"failure_count" db:"failure_count"`
    CreatedAt       time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type WebhookDelivery struct {
    ID         uuid.UUID  `json:"id" db:"id"`
    WebhookID  uuid.UUID  `json:"webhook_id" db:"webhook_id"`
    Event      string     `json:"event" db:"event"`
    Payload    []byte     `json:"-" db:"payload"`
    StatusCode int        `json:"status_code" db:"status_code"`
    Response   string     `json:"response,omitempty" db:"response"`
    Duration   int        `json:"duration_ms" db:"duration_ms"`
    Success    bool       `json:"success" db:"success"`
    CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}
```

---

## 5. Audit Service

**File:** `internal/service/audit_service.go`

```go
package service

import (
    "context"
    "time"

    "github.com/google/uuid"

    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
    "github.com/Bidon15/banhbaoring/control-plane/internal/repository"
)

type AuditService interface {
    Log(ctx context.Context, log *models.AuditLog) error
    Query(ctx context.Context, orgID uuid.UUID, filter AuditFilter) ([]*models.AuditLog, string, error)
    CleanupOldLogs(ctx context.Context) error
}

type AuditFilter struct {
    StartTime    *time.Time
    EndTime      *time.Time
    EventType    string
    ResourceType string
    ResourceID   *uuid.UUID
    ActorID      *uuid.UUID
    Limit        int
    Cursor       string
}

type auditService struct {
    auditRepo repository.AuditRepository
    orgRepo   repository.OrgRepository
}

func NewAuditService(
    auditRepo repository.AuditRepository,
    orgRepo repository.OrgRepository,
) AuditService {
    return &auditService{
        auditRepo: auditRepo,
        orgRepo:   orgRepo,
    }
}

func (s *auditService) Log(ctx context.Context, log *models.AuditLog) error {
    log.ID = uuid.New()
    log.CreatedAt = time.Now()
    return s.auditRepo.Create(ctx, log)
}

func (s *auditService) Query(ctx context.Context, orgID uuid.UUID, filter AuditFilter) ([]*models.AuditLog, string, error) {
    if filter.Limit == 0 || filter.Limit > 100 {
        filter.Limit = 100
    }

    logs, err := s.auditRepo.Query(ctx, orgID, filter)
    if err != nil {
        return nil, "", err
    }

    // Generate cursor for pagination
    var nextCursor string
    if len(logs) == filter.Limit {
        nextCursor = logs[len(logs)-1].ID.String()
    }

    return logs, nextCursor, nil
}

func (s *auditService) CleanupOldLogs(ctx context.Context) error {
    // Get all orgs and their retention periods
    orgs, err := s.orgRepo.ListAll(ctx)
    if err != nil {
        return err
    }

    for _, org := range orgs {
        limits := models.PlanLimitsMap[org.Plan]
        cutoff := time.Now().AddDate(0, 0, -limits.AuditRetentionDays)
        if err := s.auditRepo.DeleteBefore(ctx, org.ID, cutoff); err != nil {
            // Log error but continue
            continue
        }
    }

    return nil
}
```

---

## 6. Webhook Service

**File:** `internal/service/webhook_service.go`

```go
package service

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/google/uuid"

    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
    "github.com/Bidon15/banhbaoring/control-plane/internal/repository"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type WebhookService interface {
    Create(ctx context.Context, orgID uuid.UUID, req CreateWebhookRequest) (*models.Webhook, error)
    List(ctx context.Context, orgID uuid.UUID) ([]*models.Webhook, error)
    Update(ctx context.Context, orgID, webhookID uuid.UUID, req UpdateWebhookRequest) (*models.Webhook, error)
    Delete(ctx context.Context, orgID, webhookID uuid.UUID) error
    
    // Delivery
    Deliver(ctx context.Context, orgID uuid.UUID, event string, payload any) error
    GetDeliveries(ctx context.Context, webhookID uuid.UUID) ([]*models.WebhookDelivery, error)
    RetryDelivery(ctx context.Context, deliveryID uuid.UUID) error
}

type CreateWebhookRequest struct {
    URL    string   `json:"url" validate:"required,url"`
    Events []string `json:"events" validate:"required,min=1"`
}

type UpdateWebhookRequest struct {
    URL     *string   `json:"url,omitempty" validate:"omitempty,url"`
    Events  []string  `json:"events,omitempty"`
    Enabled *bool     `json:"enabled,omitempty"`
}

type webhookService struct {
    webhookRepo repository.WebhookRepository
    httpClient  *http.Client
}

func NewWebhookService(webhookRepo repository.WebhookRepository) WebhookService {
    return &webhookService{
        webhookRepo: webhookRepo,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (s *webhookService) Create(ctx context.Context, orgID uuid.UUID, req CreateWebhookRequest) (*models.Webhook, error) {
    // Validate events
    validEvents := map[string]bool{
        "key.created": true, "key.deleted": true, "key.signed": true,
        "signature.completed": true, "quota.warning": true, "quota.exceeded": true,
        "payment.succeeded": true, "payment.failed": true,
    }
    for _, event := range req.Events {
        if !validEvents[event] {
            return nil, apierrors.NewValidationError("events", fmt.Sprintf("invalid event: %s", event))
        }
    }

    // Generate secret
    secret := generateWebhookSecret()

    webhook := &models.Webhook{
        ID:      uuid.New(),
        OrgID:   orgID,
        URL:     req.URL,
        Secret:  secret,
        Events:  req.Events,
        Enabled: true,
    }

    if err := s.webhookRepo.Create(ctx, webhook); err != nil {
        return nil, err
    }

    return webhook, nil
}

func (s *webhookService) Deliver(ctx context.Context, orgID uuid.UUID, event string, payload any) error {
    // Get all webhooks for this org that subscribe to this event
    webhooks, err := s.webhookRepo.ListByOrgAndEvent(ctx, orgID, event)
    if err != nil {
        return err
    }

    for _, webhook := range webhooks {
        go s.deliverToWebhook(context.Background(), webhook, event, payload)
    }

    return nil
}

func (s *webhookService) deliverToWebhook(ctx context.Context, webhook *models.Webhook, event string, payload any) {
    // Build webhook payload
    webhookPayload := map[string]any{
        "id":         uuid.New().String(),
        "event":      event,
        "created_at": time.Now().UTC().Format(time.RFC3339),
        "data":       payload,
    }

    body, err := json.Marshal(webhookPayload)
    if err != nil {
        return
    }

    // Calculate signature
    signature := s.calculateSignature(webhook.Secret, body)

    // Create request
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(body))
    if err != nil {
        return
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Webhook-Signature", signature)
    req.Header.Set("X-Webhook-Event", event)
    req.Header.Set("User-Agent", "BanhBaoRing-Webhook/1.0")

    // Send request
    start := time.Now()
    resp, err := s.httpClient.Do(req)
    duration := time.Since(start)

    // Record delivery
    delivery := &models.WebhookDelivery{
        ID:        uuid.New(),
        WebhookID: webhook.ID,
        Event:     event,
        Payload:   body,
        Duration:  int(duration.Milliseconds()),
    }

    if err != nil {
        delivery.Success = false
        delivery.Response = err.Error()
        delivery.StatusCode = 0
    } else {
        defer resp.Body.Close()
        delivery.StatusCode = resp.StatusCode
        delivery.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
        
        // Read response (limited)
        respBody := make([]byte, 1024)
        n, _ := resp.Body.Read(respBody)
        delivery.Response = string(respBody[:n])
    }

    _ = s.webhookRepo.CreateDelivery(ctx, delivery)

    // Update webhook status
    if delivery.Success {
        _ = s.webhookRepo.ResetFailureCount(ctx, webhook.ID)
    } else {
        _ = s.webhookRepo.IncrementFailureCount(ctx, webhook.ID)
    }
}

func (s *webhookService) calculateSignature(secret string, body []byte) string {
    timestamp := time.Now().Unix()
    signedPayload := fmt.Sprintf("%d.%s", timestamp, string(body))
    
    h := hmac.New(sha256.New, []byte(secret))
    h.Write([]byte(signedPayload))
    signature := hex.EncodeToString(h.Sum(nil))
    
    return fmt.Sprintf("t=%d,v1=%s", timestamp, signature)
}

func generateWebhookSecret() string {
    b := make([]byte, 32)
    rand.Read(b)
    return fmt.Sprintf("whsec_%s", base64.URLEncoding.EncodeToString(b))
}

// ... implement remaining methods (List, Update, Delete, GetDeliveries, RetryDelivery)
```

---

## 7. Handlers

**File:** `internal/handler/audit_handler.go`

```go
package handler

import (
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"

    "github.com/Bidon15/banhbaoring/control-plane/internal/middleware"
    "github.com/Bidon15/banhbaoring/control-plane/internal/service"
    "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
)

type AuditHandler struct {
    auditService service.AuditService
}

func NewAuditHandler(auditService service.AuditService) *AuditHandler {
    return &AuditHandler{auditService: auditService}
}

func (h *AuditHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.With(middleware.RequireScope("audit:read")).Get("/logs", h.ListLogs)
    return r
}

func (h *AuditHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    filter := service.AuditFilter{
        EventType:    r.URL.Query().Get("event_type"),
        ResourceType: r.URL.Query().Get("resource_type"),
        Cursor:       r.URL.Query().Get("cursor"),
        Limit:        50,
    }

    if startStr := r.URL.Query().Get("start_time"); startStr != "" {
        if t, err := time.Parse(time.RFC3339, startStr); err == nil {
            filter.StartTime = &t
        }
    }
    if endStr := r.URL.Query().Get("end_time"); endStr != "" {
        if t, err := time.Parse(time.RFC3339, endStr); err == nil {
            filter.EndTime = &t
        }
    }

    logs, nextCursor, err := h.auditService.Query(r.Context(), orgID, filter)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.JSONWithMeta(w, http.StatusOK, logs, &response.Meta{
        NextCursor: nextCursor,
    })
}
```

---

## 8. Deliverables

| File | Description |
|------|-------------|
| `internal/models/audit.go` | Audit, Webhook, Delivery models |
| `internal/repository/audit_repo.go` | Audit DB operations |
| `internal/repository/webhook_repo.go` | Webhook DB operations |
| `internal/service/audit_service.go` | Audit business logic |
| `internal/service/webhook_service.go` | Webhook delivery logic |
| `internal/handler/audit_handler.go` | Audit HTTP handlers |
| `internal/handler/webhook_handler.go` | Webhook HTTP handlers |

---

## 9. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/audit/logs` | Query audit logs |
| POST | `/v1/webhooks` | Create webhook |
| GET | `/v1/webhooks` | List webhooks |
| PATCH | `/v1/webhooks/{id}` | Update webhook |
| DELETE | `/v1/webhooks/{id}` | Delete webhook |
| GET | `/v1/webhooks/{id}/deliveries` | List deliveries |
| POST | `/v1/webhooks/{id}/deliveries/{deliveryId}/retry` | Retry delivery |

---

## 10. Success Criteria

- [ ] Audit logs created for all key operations
- [ ] Audit log querying with filters works
- [ ] Audit retention by plan enforced
- [ ] Webhook CRUD works
- [ ] Webhook delivery with signature works
- [ ] Retry logic works
- [ ] Tests pass

---

## 11. Agent Prompt

```
You are Agent 10A - Audit & Webhooks. Implement audit logging and webhook delivery.

Read the spec: doc/implementation/IMPL_10A_AUDIT_WEBHOOKS.md

Deliverables:
1. Audit log models and repository
2. Audit service with query and cleanup
3. Webhook models and repository
4. Webhook service with HMAC signature delivery
5. Retry logic for failed deliveries
6. HTTP handlers
7. Tests

Dependencies: Agents 07, 08, 09 must complete first.

Test: go test ./internal/... -v
```

