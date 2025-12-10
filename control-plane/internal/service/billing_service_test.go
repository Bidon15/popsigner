package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v76"

	"github.com/Bidon15/banhbaoring/control-plane/internal/config"
	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// mockBillingOrgRepo is a mock implementation of OrgRepository for billing tests.
type mockBillingOrgRepo struct {
	orgs              map[uuid.UUID]*models.Organization
	members           map[uuid.UUID][]*models.OrgMember
	stripeCustomers   map[string]*models.Organization
	createError       error
	getError          error
	updateError       error
	lastUpdatedPlan   models.Plan
	lastUpdatedCustID string
	lastUpdatedSubID  string
	clearedSubOrgID   uuid.UUID
}

func newMockBillingOrgRepo() *mockBillingOrgRepo {
	return &mockBillingOrgRepo{
		orgs:            make(map[uuid.UUID]*models.Organization),
		members:         make(map[uuid.UUID][]*models.OrgMember),
		stripeCustomers: make(map[string]*models.Organization),
	}
}

func (m *mockBillingOrgRepo) Create(ctx context.Context, org *models.Organization, ownerID uuid.UUID) error {
	if m.createError != nil {
		return m.createError
	}
	m.orgs[org.ID] = org
	return nil
}

func (m *mockBillingOrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	return m.orgs[id], nil
}

func (m *mockBillingOrgRepo) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	return nil, nil
}

func (m *mockBillingOrgRepo) Update(ctx context.Context, org *models.Organization) error {
	return m.updateError
}

func (m *mockBillingOrgRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockBillingOrgRepo) AddMember(ctx context.Context, orgID, userID uuid.UUID, role models.Role, invitedBy *uuid.UUID) error {
	return nil
}

func (m *mockBillingOrgRepo) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	return nil
}

func (m *mockBillingOrgRepo) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role models.Role) error {
	return nil
}

func (m *mockBillingOrgRepo) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrgMember, error) {
	return nil, nil
}

func (m *mockBillingOrgRepo) ListMembers(ctx context.Context, orgID uuid.UUID) ([]*models.OrgMember, error) {
	return m.members[orgID], nil
}

func (m *mockBillingOrgRepo) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error) {
	return nil, nil
}

func (m *mockBillingOrgRepo) CountMembers(ctx context.Context, orgID uuid.UUID) (int, error) {
	return len(m.members[orgID]), nil
}

func (m *mockBillingOrgRepo) CreateNamespace(ctx context.Context, ns *models.Namespace) error {
	return nil
}

func (m *mockBillingOrgRepo) GetNamespace(ctx context.Context, id uuid.UUID) (*models.Namespace, error) {
	return nil, nil
}

func (m *mockBillingOrgRepo) GetNamespaceByName(ctx context.Context, orgID uuid.UUID, name string) (*models.Namespace, error) {
	return nil, nil
}

func (m *mockBillingOrgRepo) ListNamespaces(ctx context.Context, orgID uuid.UUID) ([]*models.Namespace, error) {
	return nil, nil
}

func (m *mockBillingOrgRepo) DeleteNamespace(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockBillingOrgRepo) CountNamespaces(ctx context.Context, orgID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockBillingOrgRepo) CreateInvitation(ctx context.Context, inv *models.Invitation) error {
	return nil
}

func (m *mockBillingOrgRepo) GetInvitationByToken(ctx context.Context, token string) (*models.Invitation, error) {
	return nil, nil
}

func (m *mockBillingOrgRepo) GetInvitationByEmail(ctx context.Context, orgID uuid.UUID, email string) (*models.Invitation, error) {
	return nil, nil
}

func (m *mockBillingOrgRepo) ListPendingInvitations(ctx context.Context, orgID uuid.UUID) ([]*models.Invitation, error) {
	return nil, nil
}

func (m *mockBillingOrgRepo) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	return nil
}

func (m *mockBillingOrgRepo) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockBillingOrgRepo) GetByStripeCustomer(ctx context.Context, customerID string) (*models.Organization, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	return m.stripeCustomers[customerID], nil
}

func (m *mockBillingOrgRepo) UpdateStripeCustomer(ctx context.Context, orgID uuid.UUID, customerID string) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.lastUpdatedCustID = customerID
	if org, ok := m.orgs[orgID]; ok {
		org.StripeCustomerID = &customerID
		m.stripeCustomers[customerID] = org
	}
	return nil
}

func (m *mockBillingOrgRepo) UpdateStripeSubscription(ctx context.Context, orgID uuid.UUID, subscriptionID string) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.lastUpdatedSubID = subscriptionID
	if org, ok := m.orgs[orgID]; ok {
		org.StripeSubscriptionID = &subscriptionID
	}
	return nil
}

func (m *mockBillingOrgRepo) ClearStripeSubscription(ctx context.Context, orgID uuid.UUID) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.clearedSubOrgID = orgID
	if org, ok := m.orgs[orgID]; ok {
		org.StripeSubscriptionID = nil
	}
	return nil
}

func (m *mockBillingOrgRepo) UpdatePlan(ctx context.Context, orgID uuid.UUID, plan models.Plan) error {
	if m.updateError != nil {
		return m.updateError
	}
	m.lastUpdatedPlan = plan
	if org, ok := m.orgs[orgID]; ok {
		org.Plan = plan
	}
	return nil
}

// mockBillingUsageRepo is a mock implementation of UsageRepository for billing tests.
type mockBillingUsageRepo struct {
	metrics      map[string]int64
	incrementErr error
}

func newMockBillingUsageRepo() *mockBillingUsageRepo {
	return &mockBillingUsageRepo{
		metrics: make(map[string]int64),
	}
}

func (m *mockBillingUsageRepo) Increment(ctx context.Context, orgID uuid.UUID, metric string, value int64) error {
	if m.incrementErr != nil {
		return m.incrementErr
	}
	key := orgID.String() + ":" + metric
	m.metrics[key] += value
	return nil
}

func (m *mockBillingUsageRepo) GetCurrentPeriod(ctx context.Context, orgID uuid.UUID, metric string) (int64, error) {
	key := orgID.String() + ":" + metric
	return m.metrics[key], nil
}

func (m *mockBillingUsageRepo) GetMetric(ctx context.Context, orgID uuid.UUID, metric string, periodStart time.Time) (*models.UsageMetric, error) {
	return nil, nil
}

func (m *mockBillingUsageRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time) ([]*models.UsageMetric, error) {
	return nil, nil
}

func (m *mockBillingUsageRepo) GetSummary(ctx context.Context, orgID uuid.UUID, plan models.Plan) (*models.UsageSummary, error) {
	return nil, nil
}

// mockBillingKeyRepo is a mock implementation of KeyRepository for billing tests.
type mockBillingKeyRepo struct {
	keyCount int
}

func newMockBillingKeyRepo() *mockBillingKeyRepo {
	return &mockBillingKeyRepo{}
}

func (m *mockBillingKeyRepo) Create(ctx context.Context, key *models.Key) error {
	return nil
}

func (m *mockBillingKeyRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Key, error) {
	return nil, nil
}

func (m *mockBillingKeyRepo) GetByName(ctx context.Context, orgID, namespaceID uuid.UUID, name string) (*models.Key, error) {
	return nil, nil
}

func (m *mockBillingKeyRepo) GetByAddress(ctx context.Context, orgID uuid.UUID, address string) (*models.Key, error) {
	return nil, nil
}

func (m *mockBillingKeyRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.Key, error) {
	return nil, nil
}

func (m *mockBillingKeyRepo) ListByNamespace(ctx context.Context, namespaceID uuid.UUID) ([]*models.Key, error) {
	return nil, nil
}

func (m *mockBillingKeyRepo) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	return m.keyCount, nil
}

func (m *mockBillingKeyRepo) Update(ctx context.Context, key *models.Key) error {
	return nil
}

func (m *mockBillingKeyRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockBillingKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestBillingService_GetSubscription_FreePlan(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	sub, err := svc.GetSubscription(context.Background(), orgID)
	assert.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, "free", sub.Plan)
	assert.Equal(t, "active", sub.Status)
}

func TestBillingService_GetSubscription_OrgNotFound(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	sub, err := svc.GetSubscription(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Nil(t, sub)
}

func TestBillingService_GetCurrentUsage(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}
	orgRepo.orgs[orgID] = org

	// Set up mock data
	usageKey := orgID.String() + ":signatures"
	usageRepo.metrics[usageKey] = 500
	keyRepo.keyCount = 2

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	usage, err := svc.GetCurrentUsage(context.Background(), orgID)
	assert.NoError(t, err)
	assert.NotNil(t, usage)
	assert.Equal(t, int64(500), usage.Signatures)
	assert.Equal(t, int64(10000), usage.SignaturesLimit) // Free plan limit
	assert.Equal(t, 2, usage.Keys)
	assert.Equal(t, 3, usage.KeysLimit) // Free plan limit
}

func TestBillingService_GetCurrentUsage_OrgNotFound(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	usage, err := svc.GetCurrentUsage(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Nil(t, usage)
}

func TestBillingService_ReportUsage(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	orgID := uuid.New()
	err := svc.ReportUsage(context.Background(), orgID, "signatures", 10)
	assert.NoError(t, err)

	// Verify usage was incremented
	usageKey := orgID.String() + ":signatures"
	assert.Equal(t, int64(10), usageRepo.metrics[usageKey])
}

func TestBillingService_PriceIDToPlan(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro_monthly",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg).(*billingService)

	// Test Pro plan
	plan := svc.priceIDToPlan("price_pro_monthly")
	assert.Equal(t, string(models.PlanPro), plan)

	// Test unknown price ID defaults to free
	plan = svc.priceIDToPlan("unknown_price")
	assert.Equal(t, string(models.PlanFree), plan)
}

func TestBillingService_HandleWebhook_InvalidSignature(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_test",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	// Invalid payload/signature should fail
	err := svc.HandleWebhook(context.Background(), []byte("invalid"), "invalid_signature")
	assert.Error(t, err)
}

func TestBillingService_GetSubscription_WithStripeSubscription(t *testing.T) {
	// This test verifies the code path when an org has a Stripe subscription ID
	// Note: We can't actually call Stripe in unit tests, so we just verify the org lookup
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	subID := "sub_test123"
	org := &models.Organization{
		ID:                   orgID,
		Name:                 "Test Org",
		Plan:                 models.PlanPro,
		StripeSubscriptionID: &subID,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	// This will fail at the Stripe API call, but verifies the org lookup path
	_, err := svc.GetSubscription(context.Background(), orgID)
	// We expect an error because we can't actually call Stripe
	assert.Error(t, err)
}

func TestBillingService_CreateSubscription_NoCustomer(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
		// No StripeCustomerID
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	sub, err := svc.CreateSubscription(context.Background(), orgID, "price_pro")
	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Contains(t, err.Error(), "Create payment method first")
}

func TestBillingService_CreateSubscription_AlreadyHasSubscription(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	custID := "cus_test123"
	subID := "sub_test123"
	org := &models.Organization{
		ID:                   orgID,
		Name:                 "Test Org",
		Plan:                 models.PlanPro,
		StripeCustomerID:     &custID,
		StripeSubscriptionID: &subID,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	sub, err := svc.CreateSubscription(context.Background(), orgID, "price_pro")
	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Contains(t, err.Error(), "Already has subscription")
}

func TestBillingService_CancelSubscription_NoSubscription(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	err := svc.CancelSubscription(context.Background(), orgID)
	assert.Error(t, err)
}

func TestBillingService_ChangePlan_NoSubscription(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	sub, err := svc.ChangePlan(context.Background(), orgID, "price_pro")
	assert.Error(t, err)
	assert.Nil(t, sub)
}

func TestBillingService_ListPaymentMethods_NoCustomer(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	methods, err := svc.ListPaymentMethods(context.Background(), orgID)
	assert.NoError(t, err)
	assert.Nil(t, methods)
}

func TestBillingService_ListInvoices_NoCustomer(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	invoices, err := svc.ListInvoices(context.Background(), orgID)
	assert.NoError(t, err)
	assert.Nil(t, invoices)
}

func TestBillingService_SetDefaultPaymentMethod_NoCustomer(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	err := svc.SetDefaultPaymentMethod(context.Background(), orgID, "pm_xxx")
	assert.Error(t, err)
}

func TestBillingService_ReactivateSubscription_NoSubscription(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	err := svc.ReactivateSubscription(context.Background(), orgID)
	assert.Error(t, err)
}

func TestBillingService_GetCustomer_NoCustomer(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	cust, err := svc.GetCustomer(context.Background(), orgID)
	assert.Error(t, err)
	assert.Nil(t, cust)
}

func TestBillingService_CreateCustomer_AlreadyExists(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	custID := "cus_existing"
	org := &models.Organization{
		ID:               orgID,
		Name:             "Test Org",
		Plan:             models.PlanFree,
		StripeCustomerID: &custID,
	}
	orgRepo.orgs[orgID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg)

	// Should return nil (no error) if customer already exists
	err := svc.CreateCustomer(context.Background(), orgID, "test@example.com", "Test Org")
	assert.NoError(t, err)
}

func TestBillingService_HandleSubscriptionDeleted(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	custID := "cus_test123"
	subID := "sub_test123"
	org := &models.Organization{
		ID:                   orgID,
		Name:                 "Test Org",
		Plan:                 models.PlanPro,
		StripeCustomerID:     &custID,
		StripeSubscriptionID: &subID,
	}
	orgRepo.orgs[orgID] = org
	orgRepo.stripeCustomers[custID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg).(*billingService)

	// Simulate subscription deletion via internal handler
	sub := &stripe.Subscription{
		Customer: &stripe.Customer{ID: custID},
	}
	err := svc.handleSubscriptionDeleted(context.Background(), sub)
	assert.NoError(t, err)

	// Verify plan was downgraded
	assert.Equal(t, models.PlanFree, orgRepo.lastUpdatedPlan)
	assert.Equal(t, orgID, orgRepo.clearedSubOrgID)
}

func TestBillingService_GetOrgIDFromCustomer(t *testing.T) {
	orgRepo := newMockBillingOrgRepo()
	usageRepo := newMockBillingUsageRepo()
	keyRepo := newMockBillingKeyRepo()

	orgID := uuid.New()
	custID := "cus_test123"
	org := &models.Organization{
		ID:               orgID,
		Name:             "Test Org",
		StripeCustomerID: &custID,
	}
	orgRepo.orgs[orgID] = org
	orgRepo.stripeCustomers[custID] = org

	cfg := &config.StripeConfig{
		SecretKey:     "sk_test_xxx",
		WebhookSecret: "whsec_xxx",
		PriceIDPro:    "price_pro",
	}

	svc := NewBillingService(orgRepo, usageRepo, keyRepo, cfg).(*billingService)

	// Test finding org by customer ID
	foundOrgID, err := svc.getOrgIDFromCustomer(context.Background(), custID)
	assert.NoError(t, err)
	assert.Equal(t, orgID, foundOrgID)

	// Test not finding org
	notFoundOrgID, err := svc.getOrgIDFromCustomer(context.Background(), "cus_nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, uuid.Nil, notFoundOrgID)
}

