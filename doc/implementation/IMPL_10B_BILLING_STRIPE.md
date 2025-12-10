# Implementation: Billing - Stripe Integration

## Agent: 10B - Stripe Billing

> **Phase 5.3** - Can run in parallel with 10A after Phase 5.2 completes.

---

## 1. Overview

Implement subscription billing with Stripe including plans, payments, and usage tracking.

---

## 2. Scope

| Feature | Included |
|---------|----------|
| Stripe customer creation | ✅ |
| Subscription management | ✅ |
| Plan changes | ✅ |
| Usage-based billing | ✅ |
| Webhook handling | ✅ |
| Invoice retrieval | ✅ |
| Crypto payments | ❌ (out of scope) |

---

## 3. Plans

| Plan | Price | Stripe Price ID |
|------|-------|-----------------|
| Free | $0/mo | (no subscription) |
| Pro | $49/mo | `price_pro_monthly` |
| Enterprise | Custom | `price_enterprise_*` |

---

## 4. Service

**File:** `internal/service/billing_service.go`

```go
package service

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/stripe/stripe-go/v76"
    "github.com/stripe/stripe-go/v76/customer"
    "github.com/stripe/stripe-go/v76/subscription"
    "github.com/stripe/stripe-go/v76/invoice"
    "github.com/stripe/stripe-go/v76/paymentmethod"

    "github.com/Bidon15/banhbaoring/control-plane/internal/config"
    "github.com/Bidon15/banhbaoring/control-plane/internal/models"
    "github.com/Bidon15/banhbaoring/control-plane/internal/repository"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type BillingService interface {
    // Customer
    CreateCustomer(ctx context.Context, orgID uuid.UUID, email, name string) error
    GetCustomer(ctx context.Context, orgID uuid.UUID) (*stripe.Customer, error)
    
    // Subscription
    GetSubscription(ctx context.Context, orgID uuid.UUID) (*SubscriptionInfo, error)
    CreateSubscription(ctx context.Context, orgID uuid.UUID, priceID string) (*SubscriptionInfo, error)
    ChangePlan(ctx context.Context, orgID uuid.UUID, newPriceID string) (*SubscriptionInfo, error)
    CancelSubscription(ctx context.Context, orgID uuid.UUID) error
    
    // Payment methods
    CreateSetupIntent(ctx context.Context, orgID uuid.UUID) (string, error)
    ListPaymentMethods(ctx context.Context, orgID uuid.UUID) ([]*stripe.PaymentMethod, error)
    SetDefaultPaymentMethod(ctx context.Context, orgID uuid.UUID, pmID string) error
    
    // Invoices
    ListInvoices(ctx context.Context, orgID uuid.UUID) ([]*stripe.Invoice, error)
    
    // Usage
    GetCurrentUsage(ctx context.Context, orgID uuid.UUID) (*UsageInfo, error)
    ReportUsage(ctx context.Context, orgID uuid.UUID, metric string, quantity int64) error
    
    // Webhooks
    HandleWebhook(ctx context.Context, payload []byte, signature string) error
}

type SubscriptionInfo struct {
    ID                 string    `json:"id"`
    Plan               string    `json:"plan"`
    Status             string    `json:"status"`
    CurrentPeriodStart time.Time `json:"current_period_start"`
    CurrentPeriodEnd   time.Time `json:"current_period_end"`
    CancelAtPeriodEnd  bool      `json:"cancel_at_period_end"`
}

type UsageInfo struct {
    Signatures      int64 `json:"signatures"`
    SignaturesLimit int64 `json:"signatures_limit"`
    Keys            int   `json:"keys"`
    KeysLimit       int   `json:"keys_limit"`
    PeriodStart     time.Time `json:"period_start"`
    PeriodEnd       time.Time `json:"period_end"`
}

type billingService struct {
    orgRepo   repository.OrgRepository
    usageRepo repository.UsageRepository
    config    *config.StripeConfig
}

func NewBillingService(
    orgRepo repository.OrgRepository,
    usageRepo repository.UsageRepository,
    cfg *config.StripeConfig,
) BillingService {
    stripe.Key = cfg.SecretKey
    return &billingService{
        orgRepo:   orgRepo,
        usageRepo: usageRepo,
        config:    cfg,
    }
}

func (s *billingService) CreateCustomer(ctx context.Context, orgID uuid.UUID, email, name string) error {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return err
    }
    if org == nil {
        return apierrors.NewNotFoundError("Organization")
    }
    if org.StripeCustomerID != nil {
        return nil // Already has customer
    }

    params := &stripe.CustomerParams{
        Email: stripe.String(email),
        Name:  stripe.String(name),
        Metadata: map[string]string{
            "org_id": orgID.String(),
        },
    }

    cust, err := customer.New(params)
    if err != nil {
        return fmt.Errorf("stripe customer creation failed: %w", err)
    }

    return s.orgRepo.UpdateStripeCustomer(ctx, orgID, cust.ID)
}

func (s *billingService) GetSubscription(ctx context.Context, orgID uuid.UUID) (*SubscriptionInfo, error) {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return nil, err
    }
    if org == nil {
        return nil, apierrors.NewNotFoundError("Organization")
    }
    if org.StripeSubscriptionID == nil {
        // Free plan
        return &SubscriptionInfo{
            Plan:   "free",
            Status: "active",
        }, nil
    }

    sub, err := subscription.Get(*org.StripeSubscriptionID, nil)
    if err != nil {
        return nil, err
    }

    return &SubscriptionInfo{
        ID:                 sub.ID,
        Plan:               s.priceIDToPlan(sub.Items.Data[0].Price.ID),
        Status:             string(sub.Status),
        CurrentPeriodStart: time.Unix(sub.CurrentPeriodStart, 0),
        CurrentPeriodEnd:   time.Unix(sub.CurrentPeriodEnd, 0),
        CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
    }, nil
}

func (s *billingService) CreateSubscription(ctx context.Context, orgID uuid.UUID, priceID string) (*SubscriptionInfo, error) {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return nil, err
    }
    if org == nil {
        return nil, apierrors.NewNotFoundError("Organization")
    }
    if org.StripeCustomerID == nil {
        return nil, apierrors.NewValidationError("", "Create payment method first")
    }
    if org.StripeSubscriptionID != nil {
        return nil, apierrors.NewConflictError("Already has subscription")
    }

    params := &stripe.SubscriptionParams{
        Customer: org.StripeCustomerID,
        Items: []*stripe.SubscriptionItemsParams{
            {Price: stripe.String(priceID)},
        },
        PaymentBehavior: stripe.String("default_incomplete"),
        Expand:          []*string{stripe.String("latest_invoice.payment_intent")},
    }

    sub, err := subscription.New(params)
    if err != nil {
        return nil, fmt.Errorf("stripe subscription creation failed: %w", err)
    }

    // Update org with subscription ID
    if err := s.orgRepo.UpdateStripeSubscription(ctx, orgID, sub.ID); err != nil {
        return nil, err
    }

    // Update plan
    plan := s.priceIDToPlan(priceID)
    if err := s.orgRepo.UpdatePlan(ctx, orgID, plan); err != nil {
        return nil, err
    }

    return &SubscriptionInfo{
        ID:                 sub.ID,
        Plan:               plan,
        Status:             string(sub.Status),
        CurrentPeriodStart: time.Unix(sub.CurrentPeriodStart, 0),
        CurrentPeriodEnd:   time.Unix(sub.CurrentPeriodEnd, 0),
    }, nil
}

func (s *billingService) ChangePlan(ctx context.Context, orgID uuid.UUID, newPriceID string) (*SubscriptionInfo, error) {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return nil, err
    }
    if org == nil || org.StripeSubscriptionID == nil {
        return nil, apierrors.NewNotFoundError("Subscription")
    }

    // Get current subscription
    sub, err := subscription.Get(*org.StripeSubscriptionID, nil)
    if err != nil {
        return nil, err
    }

    // Update subscription item
    params := &stripe.SubscriptionParams{
        Items: []*stripe.SubscriptionItemsParams{
            {
                ID:    stripe.String(sub.Items.Data[0].ID),
                Price: stripe.String(newPriceID),
            },
        },
        ProrationBehavior: stripe.String("create_prorations"),
    }

    updatedSub, err := subscription.Update(*org.StripeSubscriptionID, params)
    if err != nil {
        return nil, err
    }

    // Update plan in database
    plan := s.priceIDToPlan(newPriceID)
    _ = s.orgRepo.UpdatePlan(ctx, orgID, plan)

    return &SubscriptionInfo{
        ID:                 updatedSub.ID,
        Plan:               plan,
        Status:             string(updatedSub.Status),
        CurrentPeriodStart: time.Unix(updatedSub.CurrentPeriodStart, 0),
        CurrentPeriodEnd:   time.Unix(updatedSub.CurrentPeriodEnd, 0),
    }, nil
}

func (s *billingService) CancelSubscription(ctx context.Context, orgID uuid.UUID) error {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return err
    }
    if org == nil || org.StripeSubscriptionID == nil {
        return apierrors.NewNotFoundError("Subscription")
    }

    params := &stripe.SubscriptionParams{
        CancelAtPeriodEnd: stripe.Bool(true),
    }

    _, err = subscription.Update(*org.StripeSubscriptionID, params)
    return err
}

func (s *billingService) CreateSetupIntent(ctx context.Context, orgID uuid.UUID) (string, error) {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return "", err
    }
    if org == nil {
        return "", apierrors.NewNotFoundError("Organization")
    }

    // Create customer if needed
    if org.StripeCustomerID == nil {
        // Get org owner email
        members, _ := s.orgRepo.ListMembers(ctx, orgID)
        var ownerEmail string
        for _, m := range members {
            if m.Role == models.RoleOwner && m.User != nil {
                ownerEmail = m.User.Email
                break
            }
        }
        if err := s.CreateCustomer(ctx, orgID, ownerEmail, org.Name); err != nil {
            return "", err
        }
        org, _ = s.orgRepo.GetByID(ctx, orgID)
    }

    params := &stripe.SetupIntentParams{
        Customer:           org.StripeCustomerID,
        PaymentMethodTypes: []*string{stripe.String("card")},
    }

    si, err := setupintent.New(params)
    if err != nil {
        return "", err
    }

    return si.ClientSecret, nil
}

func (s *billingService) ListInvoices(ctx context.Context, orgID uuid.UUID) ([]*stripe.Invoice, error) {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return nil, err
    }
    if org == nil || org.StripeCustomerID == nil {
        return nil, nil
    }

    params := &stripe.InvoiceListParams{
        Customer: org.StripeCustomerID,
    }
    params.Limit = stripe.Int64(20)

    var invoices []*stripe.Invoice
    i := invoice.List(params)
    for i.Next() {
        invoices = append(invoices, i.Invoice())
    }

    return invoices, i.Err()
}

func (s *billingService) GetCurrentUsage(ctx context.Context, orgID uuid.UUID) (*UsageInfo, error) {
    org, err := s.orgRepo.GetByID(ctx, orgID)
    if err != nil {
        return nil, err
    }
    if org == nil {
        return nil, apierrors.NewNotFoundError("Organization")
    }

    limits := models.PlanLimitsMap[org.Plan]
    
    // Get current period usage
    now := time.Now()
    periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
    periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

    signatures, _ := s.usageRepo.GetPeriodTotal(ctx, orgID, "signatures", periodStart)
    keyCount, _ := s.keyRepo.CountByOrg(ctx, orgID)

    return &UsageInfo{
        Signatures:      signatures,
        SignaturesLimit: limits.SignaturesPerMonth,
        Keys:            keyCount,
        KeysLimit:       limits.Keys,
        PeriodStart:     periodStart,
        PeriodEnd:       periodEnd,
    }, nil
}

func (s *billingService) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
    event, err := webhook.ConstructEvent(payload, signature, s.config.WebhookSecret)
    if err != nil {
        return err
    }

    switch event.Type {
    case "customer.subscription.updated":
        var sub stripe.Subscription
        if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
            return err
        }
        // Update org plan based on subscription status
        return s.handleSubscriptionUpdated(ctx, &sub)

    case "customer.subscription.deleted":
        var sub stripe.Subscription
        if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
            return err
        }
        // Downgrade to free plan
        return s.handleSubscriptionDeleted(ctx, &sub)

    case "invoice.payment_succeeded":
        // Record successful payment
        return nil

    case "invoice.payment_failed":
        // Handle failed payment
        return nil
    }

    return nil
}

func (s *billingService) priceIDToPlan(priceID string) string {
    switch priceID {
    case s.config.PriceIDPro:
        return models.PlanPro
    default:
        return models.PlanFree
    }
}

func (s *billingService) handleSubscriptionUpdated(ctx context.Context, sub *stripe.Subscription) error {
    orgID, err := s.getOrgIDFromCustomer(ctx, sub.Customer.ID)
    if err != nil {
        return err
    }

    plan := s.priceIDToPlan(sub.Items.Data[0].Price.ID)
    return s.orgRepo.UpdatePlan(ctx, orgID, plan)
}

func (s *billingService) handleSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription) error {
    orgID, err := s.getOrgIDFromCustomer(ctx, sub.Customer.ID)
    if err != nil {
        return err
    }

    // Downgrade to free
    _ = s.orgRepo.UpdatePlan(ctx, orgID, models.PlanFree)
    return s.orgRepo.ClearStripeSubscription(ctx, orgID)
}

func (s *billingService) getOrgIDFromCustomer(ctx context.Context, customerID string) (uuid.UUID, error) {
    org, err := s.orgRepo.GetByStripeCustomer(ctx, customerID)
    if err != nil || org == nil {
        return uuid.Nil, err
    }
    return org.ID, nil
}
```

---

## 5. Handler

**File:** `internal/handler/billing_handler.go`

```go
package handler

import (
    "encoding/json"
    "io"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"

    "github.com/Bidon15/banhbaoring/control-plane/internal/middleware"
    "github.com/Bidon15/banhbaoring/control-plane/internal/service"
    "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
    apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

type BillingHandler struct {
    billingService service.BillingService
}

func NewBillingHandler(billingService service.BillingService) *BillingHandler {
    return &BillingHandler{billingService: billingService}
}

func (h *BillingHandler) Routes() chi.Router {
    r := chi.NewRouter()

    r.With(middleware.RequireScope("billing:read")).Get("/subscription", h.GetSubscription)
    r.With(middleware.RequireScope("billing:write")).Post("/subscription", h.CreateSubscription)
    r.With(middleware.RequireScope("billing:write")).Patch("/subscription", h.ChangePlan)
    r.With(middleware.RequireScope("billing:write")).Delete("/subscription", h.CancelSubscription)
    
    r.With(middleware.RequireScope("billing:read")).Get("/usage", h.GetUsage)
    r.With(middleware.RequireScope("billing:read")).Get("/invoices", h.ListInvoices)
    
    r.With(middleware.RequireScope("billing:write")).Post("/setup-intent", h.CreateSetupIntent)
    r.With(middleware.RequireScope("billing:read")).Get("/payment-methods", h.ListPaymentMethods)

    return r
}

func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    sub, err := h.billingService.GetSubscription(r.Context(), orgID)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.OK(w, sub)
}

func (h *BillingHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    var req struct {
        PriceID string `json:"price_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, apierrors.ErrBadRequest)
        return
    }

    sub, err := h.billingService.CreateSubscription(r.Context(), orgID, req.PriceID)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.Created(w, sub)
}

func (h *BillingHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
    orgID := r.Context().Value(middleware.OrgIDContextKey).(uuid.UUID)

    usage, err := h.billingService.GetCurrentUsage(r.Context(), orgID)
    if err != nil {
        response.Error(w, err)
        return
    }

    response.OK(w, usage)
}

// Webhook handler (no auth - uses Stripe signature)
func (h *BillingHandler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
    payload, err := io.ReadAll(r.Body)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    signature := r.Header.Get("Stripe-Signature")

    if err := h.billingService.HandleWebhook(r.Context(), payload, signature); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusOK)
}

// ... implement remaining handlers
```

---

## 6. Deliverables

| File | Description |
|------|-------------|
| `internal/service/billing_service.go` | Stripe integration |
| `internal/handler/billing_handler.go` | Billing HTTP handlers |
| `internal/repository/usage_repo.go` | Usage tracking |

---

## 7. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/billing/subscription` | Get subscription |
| POST | `/v1/billing/subscription` | Create subscription |
| PATCH | `/v1/billing/subscription` | Change plan |
| DELETE | `/v1/billing/subscription` | Cancel subscription |
| GET | `/v1/billing/usage` | Get current usage |
| GET | `/v1/billing/invoices` | List invoices |
| POST | `/v1/billing/setup-intent` | Create setup intent |
| GET | `/v1/billing/payment-methods` | List payment methods |
| POST | `/v1/webhooks/stripe` | Stripe webhook (no auth) |

---

## 8. Success Criteria

- [ ] Stripe customer creation works
- [ ] Subscription creation works
- [ ] Plan changes work with proration
- [ ] Cancellation works
- [ ] Usage tracking works
- [ ] Webhook handling works
- [ ] Tests pass

---

## 9. Agent Prompt

```
You are Agent 10B - Stripe Billing. Implement subscription billing.

Read the spec: doc/implementation/IMPL_10B_BILLING_STRIPE.md

Deliverables:
1. Stripe customer management
2. Subscription CRUD
3. Plan changes with proration
4. Usage tracking
5. Stripe webhook handling
6. HTTP handlers
7. Tests

Dependencies: Agents 07, 08, 09 must complete first.

Test: go test ./internal/... -v
```

