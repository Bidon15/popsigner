package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
	apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

// MockOrgRepository is a mock implementation of OrgRepository
type MockOrgRepository struct {
	mock.Mock
}

func (m *MockOrgRepository) Create(ctx context.Context, org *models.Organization, ownerID uuid.UUID) error {
	args := m.Called(ctx, org, ownerID)
	if args.Error(0) == nil {
		org.ID = uuid.New()
		org.Slug = "test-org"
		org.Plan = models.PlanFree
		org.CreatedAt = time.Now()
		org.UpdatedAt = time.Now()
	}
	return args.Error(0)
}

func (m *MockOrgRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrgRepository) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrgRepository) Update(ctx context.Context, org *models.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *MockOrgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOrgRepository) AddMember(ctx context.Context, orgID, userID uuid.UUID, role models.Role, invitedBy *uuid.UUID) error {
	args := m.Called(ctx, orgID, userID, role, invitedBy)
	return args.Error(0)
}

func (m *MockOrgRepository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	args := m.Called(ctx, orgID, userID)
	return args.Error(0)
}

func (m *MockOrgRepository) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role models.Role) error {
	args := m.Called(ctx, orgID, userID, role)
	return args.Error(0)
}

func (m *MockOrgRepository) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*models.OrgMember, error) {
	args := m.Called(ctx, orgID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrgMember), args.Error(1)
}

func (m *MockOrgRepository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]*models.OrgMember, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.OrgMember), args.Error(1)
}

func (m *MockOrgRepository) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}

func (m *MockOrgRepository) CountMembers(ctx context.Context, orgID uuid.UUID) (int, error) {
	args := m.Called(ctx, orgID)
	return args.Int(0), args.Error(1)
}

func (m *MockOrgRepository) CreateNamespace(ctx context.Context, ns *models.Namespace) error {
	args := m.Called(ctx, ns)
	if args.Error(0) == nil {
		ns.ID = uuid.New()
		ns.CreatedAt = time.Now()
	}
	return args.Error(0)
}

func (m *MockOrgRepository) GetNamespace(ctx context.Context, id uuid.UUID) (*models.Namespace, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Namespace), args.Error(1)
}

func (m *MockOrgRepository) GetNamespaceByName(ctx context.Context, orgID uuid.UUID, name string) (*models.Namespace, error) {
	args := m.Called(ctx, orgID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Namespace), args.Error(1)
}

func (m *MockOrgRepository) ListNamespaces(ctx context.Context, orgID uuid.UUID) ([]*models.Namespace, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Namespace), args.Error(1)
}

func (m *MockOrgRepository) DeleteNamespace(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOrgRepository) CountNamespaces(ctx context.Context, orgID uuid.UUID) (int, error) {
	args := m.Called(ctx, orgID)
	return args.Int(0), args.Error(1)
}

func (m *MockOrgRepository) CreateInvitation(ctx context.Context, inv *models.Invitation) error {
	args := m.Called(ctx, inv)
	if args.Error(0) == nil {
		inv.ID = uuid.New()
		inv.CreatedAt = time.Now()
	}
	return args.Error(0)
}

func (m *MockOrgRepository) GetInvitationByToken(ctx context.Context, token string) (*models.Invitation, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invitation), args.Error(1)
}

func (m *MockOrgRepository) GetInvitationByEmail(ctx context.Context, orgID uuid.UUID, email string) (*models.Invitation, error) {
	args := m.Called(ctx, orgID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invitation), args.Error(1)
}

func (m *MockOrgRepository) ListPendingInvitations(ctx context.Context, orgID uuid.UUID) ([]*models.Invitation, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invitation), args.Error(1)
}

func (m *MockOrgRepository) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	args := m.Called(ctx, token, userID)
	return args.Error(0)
}

func (m *MockOrgRepository) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOrgRepository) GetByStripeCustomer(ctx context.Context, customerID string) (*models.Organization, error) {
	args := m.Called(ctx, customerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrgRepository) UpdateStripeCustomer(ctx context.Context, orgID uuid.UUID, customerID string) error {
	args := m.Called(ctx, orgID, customerID)
	return args.Error(0)
}

func (m *MockOrgRepository) UpdateStripeSubscription(ctx context.Context, orgID uuid.UUID, subscriptionID string) error {
	args := m.Called(ctx, orgID, subscriptionID)
	return args.Error(0)
}

func (m *MockOrgRepository) ClearStripeSubscription(ctx context.Context, orgID uuid.UUID) error {
	args := m.Called(ctx, orgID)
	return args.Error(0)
}

func (m *MockOrgRepository) UpdatePlan(ctx context.Context, orgID uuid.UUID, plan models.Plan) error {
	args := m.Called(ctx, orgID, plan)
	return args.Error(0)
}

func newTestOrgService(orgRepo *MockOrgRepository, userRepo *MockUserRepository) OrgService {
	config := OrgServiceConfig{
		InvitationExpiry: 7 * 24 * time.Hour,
	}
	return NewOrgService(orgRepo, userRepo, config)
}

func TestOrgService_Create_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	ownerID := uuid.New()

	orgRepo.On("Create", ctx, mock.AnythingOfType("*models.Organization"), ownerID).Return(nil)

	org, err := svc.Create(ctx, "Test Org", ownerID)

	assert.NoError(t, err)
	assert.NotNil(t, org)
	assert.Equal(t, "Test Org", org.Name)
	assert.Equal(t, models.PlanFree, org.Plan)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_Create_EmptyName(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	ownerID := uuid.New()

	org, err := svc.Create(ctx, "", ownerID)

	assert.Error(t, err)
	assert.Nil(t, org)
}

func TestOrgService_Get_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	expectedOrg := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
		Plan: models.PlanFree,
	}

	orgRepo.On("GetByID", ctx, orgID).Return(expectedOrg, nil)

	org, err := svc.Get(ctx, orgID)

	assert.NoError(t, err)
	assert.Equal(t, expectedOrg, org)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_Get_NotFound(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()

	orgRepo.On("GetByID", ctx, orgID).Return(nil, nil)

	org, err := svc.Get(ctx, orgID)

	assert.Error(t, err)
	assert.Nil(t, org)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "not_found", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_Update_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()

	existingOrg := &models.Organization{
		ID:   orgID,
		Name: "Old Name",
		Slug: "old-name",
		Plan: models.PlanFree,
	}

	member := &models.OrgMember{
		OrgID:  orgID,
		UserID: actorID,
		Role:   models.RoleAdmin,
	}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(member, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(existingOrg, nil)
	orgRepo.On("Update", ctx, mock.AnythingOfType("*models.Organization")).Return(nil)

	org, err := svc.Update(ctx, orgID, "New Name", actorID)

	assert.NoError(t, err)
	assert.NotNil(t, org)
	assert.Equal(t, "New Name", org.Name)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_Update_Forbidden(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()

	member := &models.OrgMember{
		OrgID:  orgID,
		UserID: actorID,
		Role:   models.RoleViewer,
	}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(member, nil)

	org, err := svc.Update(ctx, orgID, "New Name", actorID)

	assert.Error(t, err)
	assert.Nil(t, org)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "forbidden", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_Delete_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	ownerID := uuid.New()

	existingOrg := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Plan: models.PlanFree,
	}

	member := &models.OrgMember{
		OrgID:  orgID,
		UserID: ownerID,
		Role:   models.RoleOwner,
	}

	orgRepo.On("GetMember", ctx, orgID, ownerID).Return(member, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(existingOrg, nil)
	orgRepo.On("Delete", ctx, orgID).Return(nil)

	err := svc.Delete(ctx, orgID, ownerID)

	assert.NoError(t, err)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_Delete_NotOwner(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	adminID := uuid.New()

	member := &models.OrgMember{
		OrgID:  orgID,
		UserID: adminID,
		Role:   models.RoleAdmin,
	}

	orgRepo.On("GetMember", ctx, orgID, adminID).Return(member, nil)

	err := svc.Delete(ctx, orgID, adminID)

	assert.Error(t, err)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "forbidden", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_CheckAccess_Owner(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()

	member := &models.OrgMember{
		OrgID:  orgID,
		UserID: userID,
		Role:   models.RoleOwner,
	}

	orgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)

	// Owner should have access to all roles
	err := svc.CheckAccess(ctx, orgID, userID, models.RoleViewer)
	assert.NoError(t, err)

	err = svc.CheckAccess(ctx, orgID, userID, models.RoleOperator)
	assert.NoError(t, err)

	err = svc.CheckAccess(ctx, orgID, userID, models.RoleAdmin)
	assert.NoError(t, err)

	err = svc.CheckAccess(ctx, orgID, userID, models.RoleOwner)
	assert.NoError(t, err)

	orgRepo.AssertExpectations(t)
}

func TestOrgService_CheckAccess_Viewer_Denied(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()

	member := &models.OrgMember{
		OrgID:  orgID,
		UserID: userID,
		Role:   models.RoleViewer,
	}

	orgRepo.On("GetMember", ctx, orgID, userID).Return(member, nil)

	// Viewer should only have viewer access
	err := svc.CheckAccess(ctx, orgID, userID, models.RoleViewer)
	assert.NoError(t, err)

	err = svc.CheckAccess(ctx, orgID, userID, models.RoleOperator)
	assert.Error(t, err)

	err = svc.CheckAccess(ctx, orgID, userID, models.RoleAdmin)
	assert.Error(t, err)

	orgRepo.AssertExpectations(t)
}

func TestOrgService_CheckAccess_NotMember(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	userID := uuid.New()

	orgRepo.On("GetMember", ctx, orgID, userID).Return(nil, nil)

	err := svc.CheckAccess(ctx, orgID, userID, models.RoleViewer)

	assert.Error(t, err)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "forbidden", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_GetLimits_Free(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()

	org := &models.Organization{
		ID:   orgID,
		Plan: models.PlanFree,
	}

	orgRepo.On("GetByID", ctx, orgID).Return(org, nil)

	limits, err := svc.GetLimits(ctx, orgID)

	assert.NoError(t, err)
	assert.NotNil(t, limits)
	assert.Equal(t, 3, limits.Keys)
	assert.Equal(t, 1, limits.TeamMembers)
	assert.Equal(t, 1, limits.Namespaces)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_GetLimits_Enterprise(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()

	org := &models.Organization{
		ID:   orgID,
		Plan: models.PlanEnterprise,
	}

	orgRepo.On("GetByID", ctx, orgID).Return(org, nil)

	limits, err := svc.GetLimits(ctx, orgID)

	assert.NoError(t, err)
	assert.NotNil(t, limits)
	assert.Equal(t, -1, limits.Keys) // unlimited
	assert.Equal(t, -1, limits.TeamMembers)
	assert.Equal(t, -1, limits.Namespaces)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_CreateNamespace_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()

	org := &models.Organization{ID: orgID, Plan: models.PlanPro}
	member := &models.OrgMember{OrgID: orgID, UserID: actorID, Role: models.RoleOperator}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(member, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(org, nil)
	orgRepo.On("CountNamespaces", ctx, orgID).Return(1, nil)
	orgRepo.On("GetNamespaceByName", ctx, orgID, "staging").Return(nil, nil)
	orgRepo.On("CreateNamespace", ctx, mock.AnythingOfType("*models.Namespace")).Return(nil)

	ns, err := svc.CreateNamespace(ctx, orgID, "staging", "Staging environment", actorID)

	assert.NoError(t, err)
	assert.NotNil(t, ns)
	assert.Equal(t, "staging", ns.Name)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_CreateNamespace_QuotaExceeded(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()

	org := &models.Organization{ID: orgID, Plan: models.PlanFree} // Free = 1 namespace
	member := &models.OrgMember{OrgID: orgID, UserID: actorID, Role: models.RoleOperator}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(member, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(org, nil)
	orgRepo.On("CountNamespaces", ctx, orgID).Return(1, nil) // Already at limit

	ns, err := svc.CreateNamespace(ctx, orgID, "staging", "Staging", actorID)

	assert.Error(t, err)
	assert.Nil(t, ns)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "quota_exceeded", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_CreateNamespace_Duplicate(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()

	org := &models.Organization{ID: orgID, Plan: models.PlanPro}
	member := &models.OrgMember{OrgID: orgID, UserID: actorID, Role: models.RoleOperator}
	existingNS := &models.Namespace{ID: uuid.New(), OrgID: orgID, Name: "staging"}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(member, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(org, nil)
	orgRepo.On("CountNamespaces", ctx, orgID).Return(1, nil)
	orgRepo.On("GetNamespaceByName", ctx, orgID, "staging").Return(existingNS, nil)

	ns, err := svc.CreateNamespace(ctx, orgID, "staging", "Staging", actorID)

	assert.Error(t, err)
	assert.Nil(t, ns)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "conflict", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_DeleteNamespace_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	nsID := uuid.New()
	actorID := uuid.New()

	member := &models.OrgMember{OrgID: orgID, UserID: actorID, Role: models.RoleAdmin}
	ns := &models.Namespace{ID: nsID, OrgID: orgID, Name: "staging"}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(member, nil)
	orgRepo.On("GetNamespace", ctx, nsID).Return(ns, nil)
	orgRepo.On("CountNamespaces", ctx, orgID).Return(2, nil) // More than 1
	orgRepo.On("DeleteNamespace", ctx, nsID).Return(nil)

	err := svc.DeleteNamespace(ctx, orgID, nsID, actorID)

	assert.NoError(t, err)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_DeleteNamespace_LastNamespace(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	nsID := uuid.New()
	actorID := uuid.New()

	member := &models.OrgMember{OrgID: orgID, UserID: actorID, Role: models.RoleAdmin}
	ns := &models.Namespace{ID: nsID, OrgID: orgID, Name: "production"}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(member, nil)
	orgRepo.On("GetNamespace", ctx, nsID).Return(ns, nil)
	orgRepo.On("CountNamespaces", ctx, orgID).Return(1, nil) // Only 1 namespace

	err := svc.DeleteNamespace(ctx, orgID, nsID, actorID)

	assert.Error(t, err)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "bad_request", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_InviteMember_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	inviterID := uuid.New()

	org := &models.Organization{ID: orgID, Plan: models.PlanPro} // Pro = 10 team members
	member := &models.OrgMember{OrgID: orgID, UserID: inviterID, Role: models.RoleAdmin}

	orgRepo.On("GetMember", ctx, orgID, inviterID).Return(member, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(org, nil)
	orgRepo.On("CountMembers", ctx, orgID).Return(2, nil)
	orgRepo.On("ListPendingInvitations", ctx, orgID).Return([]*models.Invitation{}, nil)
	userRepo.On("GetByEmail", ctx, "newuser@example.com").Return(nil, nil)
	orgRepo.On("GetInvitationByEmail", ctx, orgID, "newuser@example.com").Return(nil, nil)
	orgRepo.On("CreateInvitation", ctx, mock.AnythingOfType("*models.Invitation")).Return(nil)

	inv, err := svc.InviteMember(ctx, orgID, "newuser@example.com", models.RoleViewer, inviterID)

	assert.NoError(t, err)
	assert.NotNil(t, inv)
	assert.Equal(t, "newuser@example.com", inv.Email)
	assert.Equal(t, models.RoleViewer, inv.Role)
	orgRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestOrgService_InviteMember_AlreadyMember(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	inviterID := uuid.New()
	existingUserID := uuid.New()

	org := &models.Organization{ID: orgID, Plan: models.PlanPro}
	inviterMember := &models.OrgMember{OrgID: orgID, UserID: inviterID, Role: models.RoleAdmin}
	existingUser := &models.User{ID: existingUserID, Email: "existing@example.com"}
	existingMember := &models.OrgMember{OrgID: orgID, UserID: existingUserID, Role: models.RoleViewer}

	orgRepo.On("GetMember", ctx, orgID, inviterID).Return(inviterMember, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(org, nil)
	orgRepo.On("CountMembers", ctx, orgID).Return(2, nil)
	orgRepo.On("ListPendingInvitations", ctx, orgID).Return([]*models.Invitation{}, nil)
	userRepo.On("GetByEmail", ctx, "existing@example.com").Return(existingUser, nil)
	orgRepo.On("GetMember", ctx, orgID, existingUserID).Return(existingMember, nil)

	inv, err := svc.InviteMember(ctx, orgID, "existing@example.com", models.RoleViewer, inviterID)

	assert.Error(t, err)
	assert.Nil(t, inv)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "conflict", apiErr.Code)
	orgRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestOrgService_InviteMember_QuotaExceeded(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	inviterID := uuid.New()

	org := &models.Organization{ID: orgID, Plan: models.PlanFree} // Free = 1 team member
	member := &models.OrgMember{OrgID: orgID, UserID: inviterID, Role: models.RoleAdmin}

	orgRepo.On("GetMember", ctx, orgID, inviterID).Return(member, nil)
	orgRepo.On("GetByID", ctx, orgID).Return(org, nil)
	orgRepo.On("CountMembers", ctx, orgID).Return(1, nil) // Already at limit
	orgRepo.On("ListPendingInvitations", ctx, orgID).Return([]*models.Invitation{}, nil)

	inv, err := svc.InviteMember(ctx, orgID, "newuser@example.com", models.RoleViewer, inviterID)

	assert.Error(t, err)
	assert.Nil(t, inv)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "quota_exceeded", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_InviteMember_CannotInviteAsOwner(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	inviterID := uuid.New()

	member := &models.OrgMember{OrgID: orgID, UserID: inviterID, Role: models.RoleAdmin}

	orgRepo.On("GetMember", ctx, orgID, inviterID).Return(member, nil)

	inv, err := svc.InviteMember(ctx, orgID, "newuser@example.com", models.RoleOwner, inviterID)

	assert.Error(t, err)
	assert.Nil(t, inv)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_RemoveMember_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()

	actorMember := &models.OrgMember{OrgID: orgID, UserID: actorID, Role: models.RoleAdmin}
	targetMember := &models.OrgMember{OrgID: orgID, UserID: targetID, Role: models.RoleViewer}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(actorMember, nil)
	orgRepo.On("GetMember", ctx, orgID, targetID).Return(targetMember, nil)
	orgRepo.On("RemoveMember", ctx, orgID, targetID).Return(nil)

	err := svc.RemoveMember(ctx, orgID, targetID, actorID)

	assert.NoError(t, err)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_RemoveMember_CannotRemoveOwner(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()
	ownerID := uuid.New()

	actorMember := &models.OrgMember{OrgID: orgID, UserID: actorID, Role: models.RoleAdmin}
	ownerMember := &models.OrgMember{OrgID: orgID, UserID: ownerID, Role: models.RoleOwner}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(actorMember, nil)
	orgRepo.On("GetMember", ctx, orgID, ownerID).Return(ownerMember, nil)

	err := svc.RemoveMember(ctx, orgID, ownerID, actorID)

	assert.Error(t, err)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "forbidden", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_UpdateMemberRole_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()
	targetID := uuid.New()

	actorMember := &models.OrgMember{OrgID: orgID, UserID: actorID, Role: models.RoleOwner}
	targetMember := &models.OrgMember{OrgID: orgID, UserID: targetID, Role: models.RoleViewer}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(actorMember, nil)
	orgRepo.On("GetMember", ctx, orgID, targetID).Return(targetMember, nil)
	orgRepo.On("UpdateMemberRole", ctx, orgID, targetID, models.RoleOperator).Return(nil)

	err := svc.UpdateMemberRole(ctx, orgID, targetID, models.RoleOperator, actorID)

	assert.NoError(t, err)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_UpdateMemberRole_CannotChangeOwnerRole(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	orgID := uuid.New()
	actorID := uuid.New()
	ownerID := uuid.New()

	actorMember := &models.OrgMember{OrgID: orgID, UserID: actorID, Role: models.RoleOwner}
	ownerMember := &models.OrgMember{OrgID: orgID, UserID: ownerID, Role: models.RoleOwner}

	orgRepo.On("GetMember", ctx, orgID, actorID).Return(actorMember, nil)
	orgRepo.On("GetMember", ctx, orgID, ownerID).Return(ownerMember, nil)

	err := svc.UpdateMemberRole(ctx, orgID, ownerID, models.RoleAdmin, actorID)

	assert.Error(t, err)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "forbidden", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_AcceptInvitation_Success(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	userID := uuid.New()
	orgID := uuid.New()

	org := &models.Organization{ID: orgID, Name: "Test Org", Slug: "test-org"}
	user := &models.User{ID: userID, Email: "invited@example.com"}
	inv := &models.Invitation{
		ID:           uuid.New(),
		OrgID:        orgID,
		Email:        "invited@example.com",
		Role:         models.RoleViewer,
		Token:        "test-token",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		Organization: org,
	}

	orgRepo.On("GetInvitationByToken", ctx, "test-token").Return(inv, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)
	orgRepo.On("AcceptInvitation", ctx, "test-token", userID).Return(nil)

	resultOrg, err := svc.AcceptInvitation(ctx, "test-token", userID)

	assert.NoError(t, err)
	assert.Equal(t, org, resultOrg)
	orgRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestOrgService_AcceptInvitation_Expired(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	userID := uuid.New()

	inv := &models.Invitation{
		ID:        uuid.New(),
		Email:     "invited@example.com",
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired
	}

	orgRepo.On("GetInvitationByToken", ctx, "expired-token").Return(inv, nil)

	org, err := svc.AcceptInvitation(ctx, "expired-token", userID)

	assert.Error(t, err)
	assert.Nil(t, org)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "bad_request", apiErr.Code)
	orgRepo.AssertExpectations(t)
}

func TestOrgService_AcceptInvitation_WrongEmail(t *testing.T) {
	orgRepo := new(MockOrgRepository)
	userRepo := new(MockUserRepository)
	svc := newTestOrgService(orgRepo, userRepo)

	ctx := context.Background()
	userID := uuid.New()

	org := &models.Organization{ID: uuid.New(), Name: "Test Org"}
	user := &models.User{ID: userID, Email: "different@example.com"}
	inv := &models.Invitation{
		ID:           uuid.New(),
		Email:        "invited@example.com",
		Token:        "test-token",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		Organization: org,
	}

	orgRepo.On("GetInvitationByToken", ctx, "test-token").Return(inv, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)

	resultOrg, err := svc.AcceptInvitation(ctx, "test-token", userID)

	assert.Error(t, err)
	assert.Nil(t, resultOrg)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "forbidden", apiErr.Code)
	orgRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

