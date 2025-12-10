// Package handler provides HTTP handlers for the control plane API.
package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Bidon15/banhbaoring/control-plane/internal/middleware"
	apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
	"github.com/Bidon15/banhbaoring/control-plane/internal/pkg/response"
	"github.com/Bidon15/banhbaoring/control-plane/internal/service"
)

// BillingHandler handles billing-related HTTP requests.
type BillingHandler struct {
	billingService service.BillingService
}

// NewBillingHandler creates a new billing handler.
func NewBillingHandler(billingService service.BillingService) *BillingHandler {
	return &BillingHandler{billingService: billingService}
}

// Routes returns a chi router with billing routes.
func (h *BillingHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Subscription management
	r.With(middleware.RequireScope("billing:read")).Get("/subscription", h.GetSubscription)
	r.With(middleware.RequireScope("billing:write")).Post("/subscription", h.CreateSubscription)
	r.With(middleware.RequireScope("billing:write")).Patch("/subscription", h.ChangePlan)
	r.With(middleware.RequireScope("billing:write")).Delete("/subscription", h.CancelSubscription)
	r.With(middleware.RequireScope("billing:write")).Post("/subscription/reactivate", h.ReactivateSubscription)

	// Usage and invoices
	r.With(middleware.RequireScope("billing:read")).Get("/usage", h.GetUsage)
	r.With(middleware.RequireScope("billing:read")).Get("/invoices", h.ListInvoices)

	// Payment methods
	r.With(middleware.RequireScope("billing:write")).Post("/setup-intent", h.CreateSetupIntent)
	r.With(middleware.RequireScope("billing:read")).Get("/payment-methods", h.ListPaymentMethods)
	r.With(middleware.RequireScope("billing:write")).Post("/payment-methods/default", h.SetDefaultPaymentMethod)

	return r
}

// GetSubscription handles GET /v1/billing/subscription
func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	sub, err := h.billingService.GetSubscription(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, sub)
}

// CreateSubscriptionRequest is the request body for creating a subscription.
type CreateSubscriptionRequest struct {
	PriceID string `json:"price_id"`
}

// CreateSubscription handles POST /v1/billing/subscription
func (h *BillingHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	var req CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid request body"))
		return
	}

	if req.PriceID == "" {
		response.Error(w, apierrors.NewValidationError("price_id", "price_id is required"))
		return
	}

	sub, err := h.billingService.CreateSubscription(r.Context(), orgID, req.PriceID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, sub)
}

// ChangePlanRequest is the request body for changing a plan.
type ChangePlanRequest struct {
	PriceID string `json:"price_id"`
}

// ChangePlan handles PATCH /v1/billing/subscription
func (h *BillingHandler) ChangePlan(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	var req ChangePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid request body"))
		return
	}

	if req.PriceID == "" {
		response.Error(w, apierrors.NewValidationError("price_id", "price_id is required"))
		return
	}

	sub, err := h.billingService.ChangePlan(r.Context(), orgID, req.PriceID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, sub)
}

// CancelSubscription handles DELETE /v1/billing/subscription
func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	if err := h.billingService.CancelSubscription(r.Context(), orgID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ReactivateSubscription handles POST /v1/billing/subscription/reactivate
func (h *BillingHandler) ReactivateSubscription(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	if err := h.billingService.ReactivateSubscription(r.Context(), orgID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// GetUsage handles GET /v1/billing/usage
func (h *BillingHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	usage, err := h.billingService.GetCurrentUsage(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, usage)
}

// ListInvoices handles GET /v1/billing/invoices
func (h *BillingHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	invoices, err := h.billingService.ListInvoices(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Convert to a simplified response format
	invoiceResponses := make([]InvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		invoiceResponses = append(invoiceResponses, InvoiceResponse{
			ID:          inv.ID,
			Number:      inv.Number,
			Status:      string(inv.Status),
			AmountDue:   inv.AmountDue,
			AmountPaid:  inv.AmountPaid,
			Currency:    string(inv.Currency),
			PeriodStart: inv.PeriodStart,
			PeriodEnd:   inv.PeriodEnd,
			Created:     inv.Created,
			InvoicePDF:  inv.InvoicePDF,
			HostedURL:   inv.HostedInvoiceURL,
		})
	}

	response.OK(w, invoiceResponses)
}

// InvoiceResponse is a simplified invoice response.
type InvoiceResponse struct {
	ID          string `json:"id"`
	Number      string `json:"number"`
	Status      string `json:"status"`
	AmountDue   int64  `json:"amount_due"`
	AmountPaid  int64  `json:"amount_paid"`
	Currency    string `json:"currency"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	Created     int64  `json:"created"`
	InvoicePDF  string `json:"invoice_pdf,omitempty"`
	HostedURL   string `json:"hosted_url,omitempty"`
}

// CreateSetupIntent handles POST /v1/billing/setup-intent
func (h *BillingHandler) CreateSetupIntent(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	clientSecret, err := h.billingService.CreateSetupIntent(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.OK(w, map[string]string{"client_secret": clientSecret})
}

// ListPaymentMethods handles GET /v1/billing/payment-methods
func (h *BillingHandler) ListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	methods, err := h.billingService.ListPaymentMethods(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Convert to a simplified response format
	methodResponses := make([]PaymentMethodResponse, 0, len(methods))
	for _, m := range methods {
		pm := PaymentMethodResponse{
			ID:   m.ID,
			Type: string(m.Type),
		}
		if m.Card != nil {
			pm.Card = &CardInfo{
				Brand:    string(m.Card.Brand),
				Last4:    m.Card.Last4,
				ExpMonth: int(m.Card.ExpMonth),
				ExpYear:  int(m.Card.ExpYear),
			}
		}
		methodResponses = append(methodResponses, pm)
	}

	response.OK(w, methodResponses)
}

// PaymentMethodResponse is a simplified payment method response.
type PaymentMethodResponse struct {
	ID   string    `json:"id"`
	Type string    `json:"type"`
	Card *CardInfo `json:"card,omitempty"`
}

// CardInfo contains card details.
type CardInfo struct {
	Brand    string `json:"brand"`
	Last4    string `json:"last4"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
}

// SetDefaultPaymentMethodRequest is the request body for setting default payment method.
type SetDefaultPaymentMethodRequest struct {
	PaymentMethodID string `json:"payment_method_id"`
}

// SetDefaultPaymentMethod handles POST /v1/billing/payment-methods/default
func (h *BillingHandler) SetDefaultPaymentMethod(w http.ResponseWriter, r *http.Request) {
	orgID := middleware.GetOrgIDFromContext(r.Context())
	if orgID == uuid.Nil {
		response.Error(w, apierrors.ErrUnauthorized)
		return
	}

	var req SetDefaultPaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("Invalid request body"))
		return
	}

	if req.PaymentMethodID == "" {
		response.Error(w, apierrors.NewValidationError("payment_method_id", "payment_method_id is required"))
		return
	}

	if err := h.billingService.SetDefaultPaymentMethod(r.Context(), orgID, req.PaymentMethodID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// WebhookHandler handles POST /v1/webhooks/stripe
// This endpoint does not require authentication - it uses Stripe signature verification.
func (h *BillingHandler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = 65536
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.billingService.HandleWebhook(r.Context(), payload, signature); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

