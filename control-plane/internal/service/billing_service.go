// Package service provides business logic for the control plane API.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/invoice"
	"github.com/stripe/stripe-go/v76/paymentmethod"
	"github.com/stripe/stripe-go/v76/setupintent"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/Bidon15/banhbaoring/control-plane/internal/config"
	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
	apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
	"github.com/Bidon15/banhbaoring/control-plane/internal/repository"
)

// BillingService defines the interface for billing operations.
type BillingService interface {
	// Customer management
	CreateCustomer(ctx context.Context, orgID uuid.UUID, email, name string) error
	GetCustomer(ctx context.Context, orgID uuid.UUID) (*stripe.Customer, error)

	// Subscription management
	GetSubscription(ctx context.Context, orgID uuid.UUID) (*SubscriptionInfo, error)
	CreateSubscription(ctx context.Context, orgID uuid.UUID, priceID string) (*SubscriptionInfo, error)
	ChangePlan(ctx context.Context, orgID uuid.UUID, newPriceID string) (*SubscriptionInfo, error)
	CancelSubscription(ctx context.Context, orgID uuid.UUID) error
	ReactivateSubscription(ctx context.Context, orgID uuid.UUID) error

	// Payment methods
	CreateSetupIntent(ctx context.Context, orgID uuid.UUID) (string, error)
	ListPaymentMethods(ctx context.Context, orgID uuid.UUID) ([]*stripe.PaymentMethod, error)
	SetDefaultPaymentMethod(ctx context.Context, orgID uuid.UUID, paymentMethodID string) error

	// Invoices
	ListInvoices(ctx context.Context, orgID uuid.UUID) ([]*stripe.Invoice, error)

	// Usage
	GetCurrentUsage(ctx context.Context, orgID uuid.UUID) (*UsageInfo, error)
	ReportUsage(ctx context.Context, orgID uuid.UUID, metric string, quantity int64) error

	// Webhooks
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
}

// SubscriptionInfo represents subscription details returned by the API.
type SubscriptionInfo struct {
	ID                 string    `json:"id"`
	Plan               string    `json:"plan"`
	Status             string    `json:"status"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CancelAtPeriodEnd  bool      `json:"cancel_at_period_end"`
}

// UsageInfo represents current usage information for an organization.
type UsageInfo struct {
	Signatures      int64     `json:"signatures"`
	SignaturesLimit int64     `json:"signatures_limit"`
	Keys            int       `json:"keys"`
	KeysLimit       int       `json:"keys_limit"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
}

type billingService struct {
	orgRepo   repository.OrgRepository
	usageRepo repository.UsageRepository
	keyRepo   repository.KeyRepository
	config    *config.StripeConfig
}

// NewBillingService creates a new billing service.
func NewBillingService(
	orgRepo repository.OrgRepository,
	usageRepo repository.UsageRepository,
	keyRepo repository.KeyRepository,
	cfg *config.StripeConfig,
) BillingService {
	stripe.Key = cfg.SecretKey
	return &billingService{
		orgRepo:   orgRepo,
		usageRepo: usageRepo,
		keyRepo:   keyRepo,
		config:    cfg,
	}
}

// CreateCustomer creates a Stripe customer for an organization.
func (s *billingService) CreateCustomer(ctx context.Context, orgID uuid.UUID, email, name string) error {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}
	if org == nil {
		return apierrors.NewNotFoundError("Organization")
	}
	if org.StripeCustomerID != nil {
		return nil // Already has a customer
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

// GetCustomer retrieves the Stripe customer for an organization.
func (s *billingService) GetCustomer(ctx context.Context, orgID uuid.UUID) (*stripe.Customer, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, apierrors.NewNotFoundError("Organization")
	}
	if org.StripeCustomerID == nil {
		return nil, apierrors.NewNotFoundError("Customer")
	}

	cust, err := customer.Get(*org.StripeCustomerID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Stripe customer: %w", err)
	}

	return cust, nil
}

// GetSubscription retrieves subscription info for an organization.
func (s *billingService) GetSubscription(ctx context.Context, orgID uuid.UUID) (*SubscriptionInfo, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, apierrors.NewNotFoundError("Organization")
	}
	if org.StripeSubscriptionID == nil {
		// Return free plan info
		return &SubscriptionInfo{
			Plan:   string(models.PlanFree),
			Status: "active",
		}, nil
	}

	sub, err := subscription.Get(*org.StripeSubscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Stripe subscription: %w", err)
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

// CreateSubscription creates a new subscription for an organization.
func (s *billingService) CreateSubscription(ctx context.Context, orgID uuid.UUID, priceID string) (*SubscriptionInfo, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, apierrors.NewNotFoundError("Organization")
	}
	if org.StripeCustomerID == nil {
		return nil, apierrors.NewValidationError("customer", "Create payment method first")
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
	if err := s.orgRepo.UpdatePlan(ctx, orgID, models.Plan(plan)); err != nil {
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

// ChangePlan changes the subscription plan with proration.
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
		return nil, fmt.Errorf("failed to retrieve subscription: %w", err)
	}

	// Update subscription item with proration
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
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Update plan in database
	plan := s.priceIDToPlan(newPriceID)
	_ = s.orgRepo.UpdatePlan(ctx, orgID, models.Plan(plan))

	return &SubscriptionInfo{
		ID:                 updatedSub.ID,
		Plan:               plan,
		Status:             string(updatedSub.Status),
		CurrentPeriodStart: time.Unix(updatedSub.CurrentPeriodStart, 0),
		CurrentPeriodEnd:   time.Unix(updatedSub.CurrentPeriodEnd, 0),
		CancelAtPeriodEnd:  updatedSub.CancelAtPeriodEnd,
	}, nil
}

// CancelSubscription cancels a subscription at the end of the billing period.
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
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	return nil
}

// ReactivateSubscription reactivates a subscription that was set to cancel at period end.
func (s *billingService) ReactivateSubscription(ctx context.Context, orgID uuid.UUID) error {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}
	if org == nil || org.StripeSubscriptionID == nil {
		return apierrors.NewNotFoundError("Subscription")
	}

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}

	_, err = subscription.Update(*org.StripeSubscriptionID, params)
	if err != nil {
		return fmt.Errorf("failed to reactivate subscription: %w", err)
	}

	return nil
}

// CreateSetupIntent creates a Stripe SetupIntent for collecting payment methods.
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
		return "", fmt.Errorf("failed to create setup intent: %w", err)
	}

	return si.ClientSecret, nil
}

// ListPaymentMethods lists payment methods for an organization.
func (s *billingService) ListPaymentMethods(ctx context.Context, orgID uuid.UUID) ([]*stripe.PaymentMethod, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil || org.StripeCustomerID == nil {
		return nil, nil
	}

	params := &stripe.PaymentMethodListParams{
		Customer: org.StripeCustomerID,
		Type:     stripe.String("card"),
	}

	var paymentMethods []*stripe.PaymentMethod
	i := paymentmethod.List(params)
	for i.Next() {
		paymentMethods = append(paymentMethods, i.PaymentMethod())
	}

	return paymentMethods, i.Err()
}

// SetDefaultPaymentMethod sets the default payment method for a customer.
func (s *billingService) SetDefaultPaymentMethod(ctx context.Context, orgID uuid.UUID, paymentMethodID string) error {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}
	if org == nil || org.StripeCustomerID == nil {
		return apierrors.NewNotFoundError("Customer")
	}

	params := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethodID),
		},
	}

	_, err = customer.Update(*org.StripeCustomerID, params)
	if err != nil {
		return fmt.Errorf("failed to set default payment method: %w", err)
	}

	return nil
}

// ListInvoices lists invoices for an organization.
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

// GetCurrentUsage retrieves current usage information for an organization.
func (s *billingService) GetCurrentUsage(ctx context.Context, orgID uuid.UUID) (*UsageInfo, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, apierrors.NewNotFoundError("Organization")
	}

	limits := models.GetPlanLimits(org.Plan)

	// Get current period
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	// Get signature usage
	signatures, _ := s.usageRepo.GetCurrentPeriod(ctx, orgID, string(models.MetricTypeSignatures))

	// Get key count
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

// ReportUsage reports usage for a metric.
func (s *billingService) ReportUsage(ctx context.Context, orgID uuid.UUID, metric string, quantity int64) error {
	return s.usageRepo.Increment(ctx, orgID, metric, quantity)
}

// HandleWebhook processes incoming Stripe webhook events.
func (s *billingService) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	event, err := webhook.ConstructEvent(payload, signature, s.config.WebhookSecret)
	if err != nil {
		return fmt.Errorf("webhook signature verification failed: %w", err)
	}

	switch event.Type {
	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return fmt.Errorf("failed to unmarshal subscription: %w", err)
		}
		return s.handleSubscriptionUpdated(ctx, &sub)

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return fmt.Errorf("failed to unmarshal subscription: %w", err)
		}
		return s.handleSubscriptionDeleted(ctx, &sub)

	case "invoice.payment_succeeded":
		// Record successful payment - could log or trigger events
		return nil

	case "invoice.payment_failed":
		// Handle failed payment - could send notifications
		return nil
	}

	return nil
}

// priceIDToPlan converts a Stripe price ID to a plan name.
func (s *billingService) priceIDToPlan(priceID string) string {
	switch priceID {
	case s.config.PriceIDPro:
		return string(models.PlanPro)
	default:
		return string(models.PlanFree)
	}
}

// handleSubscriptionUpdated handles subscription update events.
func (s *billingService) handleSubscriptionUpdated(ctx context.Context, sub *stripe.Subscription) error {
	orgID, err := s.getOrgIDFromCustomer(ctx, sub.Customer.ID)
	if err != nil {
		return err
	}
	if orgID == uuid.Nil {
		return nil // Organization not found, ignore
	}

	plan := s.priceIDToPlan(sub.Items.Data[0].Price.ID)
	return s.orgRepo.UpdatePlan(ctx, orgID, models.Plan(plan))
}

// handleSubscriptionDeleted handles subscription deletion events.
func (s *billingService) handleSubscriptionDeleted(ctx context.Context, sub *stripe.Subscription) error {
	orgID, err := s.getOrgIDFromCustomer(ctx, sub.Customer.ID)
	if err != nil {
		return err
	}
	if orgID == uuid.Nil {
		return nil // Organization not found, ignore
	}

	// Downgrade to free plan
	_ = s.orgRepo.UpdatePlan(ctx, orgID, models.PlanFree)
	return s.orgRepo.ClearStripeSubscription(ctx, orgID)
}

// getOrgIDFromCustomer retrieves the org ID from a Stripe customer ID.
func (s *billingService) getOrgIDFromCustomer(ctx context.Context, customerID string) (uuid.UUID, error) {
	org, err := s.orgRepo.GetByStripeCustomer(ctx, customerID)
	if err != nil || org == nil {
		return uuid.Nil, err
	}
	return org.ID, nil
}

// Compile-time check to ensure billingService implements BillingService.
var _ BillingService = (*billingService)(nil)

