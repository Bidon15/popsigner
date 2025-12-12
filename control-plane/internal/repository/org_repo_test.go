package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
)

// MockOrgRepository is a mock implementation of OrgRepository for testing.
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

// Verify MockOrgRepository implements OrgRepository
var _ OrgRepository = (*MockOrgRepository)(nil)

func TestMockOrgRepository_Create(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	ownerID := uuid.New()
	org := &models.Organization{
		Name: "Test Organization",
	}

	mockRepo.On("Create", ctx, org, ownerID).Return(nil)

	err := mockRepo.Create(ctx, org, ownerID)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, org.ID)
	assert.Equal(t, models.PlanFree, org.Plan)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_GetByID(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()
	expectedOrg := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
		Plan: models.PlanFree,
	}

	mockRepo.On("GetByID", ctx, orgID).Return(expectedOrg, nil)

	org, err := mockRepo.GetByID(ctx, orgID)
	assert.NoError(t, err)
	assert.Equal(t, expectedOrg, org)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_GetByID_NotFound(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()

	mockRepo.On("GetByID", ctx, orgID).Return(nil, nil)

	org, err := mockRepo.GetByID(ctx, orgID)
	assert.NoError(t, err)
	assert.Nil(t, org)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_GetBySlug(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	expectedOrg := &models.Organization{
		ID:   uuid.New(),
		Name: "Test Org",
		Slug: "test-org",
		Plan: models.PlanPro,
	}

	mockRepo.On("GetBySlug", ctx, "test-org").Return(expectedOrg, nil)

	org, err := mockRepo.GetBySlug(ctx, "test-org")
	assert.NoError(t, err)
	assert.Equal(t, expectedOrg, org)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_Update(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	org := &models.Organization{
		ID:   uuid.New(),
		Name: "Updated Org",
	}

	mockRepo.On("Update", ctx, org).Return(nil)

	err := mockRepo.Update(ctx, org)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_Delete(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()

	mockRepo.On("Delete", ctx, orgID).Return(nil)

	err := mockRepo.Delete(ctx, orgID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_AddMember(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()
	userID := uuid.New()
	inviterID := uuid.New()

	mockRepo.On("AddMember", ctx, orgID, userID, models.RoleViewer, &inviterID).Return(nil)

	err := mockRepo.AddMember(ctx, orgID, userID, models.RoleViewer, &inviterID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_RemoveMember(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()
	userID := uuid.New()

	mockRepo.On("RemoveMember", ctx, orgID, userID).Return(nil)

	err := mockRepo.RemoveMember(ctx, orgID, userID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_UpdateMemberRole(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()
	userID := uuid.New()

	mockRepo.On("UpdateMemberRole", ctx, orgID, userID, models.RoleAdmin).Return(nil)

	err := mockRepo.UpdateMemberRole(ctx, orgID, userID, models.RoleAdmin)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_GetMember(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()
	userID := uuid.New()

	expectedMember := &models.OrgMember{
		OrgID:    orgID,
		UserID:   userID,
		Role:     models.RoleAdmin,
		JoinedAt: time.Now(),
	}

	mockRepo.On("GetMember", ctx, orgID, userID).Return(expectedMember, nil)

	member, err := mockRepo.GetMember(ctx, orgID, userID)
	assert.NoError(t, err)
	assert.Equal(t, expectedMember, member)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_ListMembers(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()

	expectedMembers := []*models.OrgMember{
		{OrgID: orgID, UserID: uuid.New(), Role: models.RoleOwner},
		{OrgID: orgID, UserID: uuid.New(), Role: models.RoleAdmin},
	}

	mockRepo.On("ListMembers", ctx, orgID).Return(expectedMembers, nil)

	members, err := mockRepo.ListMembers(ctx, orgID)
	assert.NoError(t, err)
	assert.Len(t, members, 2)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_ListUserOrgs(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	userID := uuid.New()

	expectedOrgs := []*models.Organization{
		{ID: uuid.New(), Name: "Org 1", Slug: "org-1"},
		{ID: uuid.New(), Name: "Org 2", Slug: "org-2"},
	}

	mockRepo.On("ListUserOrgs", ctx, userID).Return(expectedOrgs, nil)

	orgs, err := mockRepo.ListUserOrgs(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, orgs, 2)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_CountMembers(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()

	mockRepo.On("CountMembers", ctx, orgID).Return(5, nil)

	count, err := mockRepo.CountMembers(ctx, orgID)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_CreateNamespace(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	desc := "Test namespace"
	ns := &models.Namespace{
		OrgID:       uuid.New(),
		Name:        "staging",
		Description: &desc,
	}

	mockRepo.On("CreateNamespace", ctx, ns).Return(nil)

	err := mockRepo.CreateNamespace(ctx, ns)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, ns.ID)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_GetNamespace(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	nsID := uuid.New()
	expectedNS := &models.Namespace{
		ID:    nsID,
		OrgID: uuid.New(),
		Name:  "production",
	}

	mockRepo.On("GetNamespace", ctx, nsID).Return(expectedNS, nil)

	ns, err := mockRepo.GetNamespace(ctx, nsID)
	assert.NoError(t, err)
	assert.Equal(t, expectedNS, ns)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_GetNamespaceByName(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()
	expectedNS := &models.Namespace{
		ID:    uuid.New(),
		OrgID: orgID,
		Name:  "production",
	}

	mockRepo.On("GetNamespaceByName", ctx, orgID, "production").Return(expectedNS, nil)

	ns, err := mockRepo.GetNamespaceByName(ctx, orgID, "production")
	assert.NoError(t, err)
	assert.Equal(t, expectedNS, ns)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_ListNamespaces(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()
	expectedNS := []*models.Namespace{
		{ID: uuid.New(), OrgID: orgID, Name: "production"},
		{ID: uuid.New(), OrgID: orgID, Name: "staging"},
	}

	mockRepo.On("ListNamespaces", ctx, orgID).Return(expectedNS, nil)

	namespaces, err := mockRepo.ListNamespaces(ctx, orgID)
	assert.NoError(t, err)
	assert.Len(t, namespaces, 2)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_DeleteNamespace(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	nsID := uuid.New()

	mockRepo.On("DeleteNamespace", ctx, nsID).Return(nil)

	err := mockRepo.DeleteNamespace(ctx, nsID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_CountNamespaces(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()

	mockRepo.On("CountNamespaces", ctx, orgID).Return(3, nil)

	count, err := mockRepo.CountNamespaces(ctx, orgID)
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_CreateInvitation(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	inv := &models.Invitation{
		OrgID:     uuid.New(),
		Email:     "test@example.com",
		Role:      models.RoleViewer,
		Token:     "test-token",
		InvitedBy: uuid.New(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	mockRepo.On("CreateInvitation", ctx, inv).Return(nil)

	err := mockRepo.CreateInvitation(ctx, inv)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, inv.ID)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_GetInvitationByToken(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	expectedInv := &models.Invitation{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Email:     "test@example.com",
		Role:      models.RoleViewer,
		Token:     "test-token",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	mockRepo.On("GetInvitationByToken", ctx, "test-token").Return(expectedInv, nil)

	inv, err := mockRepo.GetInvitationByToken(ctx, "test-token")
	assert.NoError(t, err)
	assert.Equal(t, expectedInv, inv)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_GetInvitationByEmail(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()
	expectedInv := &models.Invitation{
		ID:    uuid.New(),
		OrgID: orgID,
		Email: "test@example.com",
		Role:  models.RoleViewer,
	}

	mockRepo.On("GetInvitationByEmail", ctx, orgID, "test@example.com").Return(expectedInv, nil)

	inv, err := mockRepo.GetInvitationByEmail(ctx, orgID, "test@example.com")
	assert.NoError(t, err)
	assert.Equal(t, expectedInv, inv)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_ListPendingInvitations(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	orgID := uuid.New()
	expectedInvitations := []*models.Invitation{
		{ID: uuid.New(), OrgID: orgID, Email: "user1@example.com"},
		{ID: uuid.New(), OrgID: orgID, Email: "user2@example.com"},
	}

	mockRepo.On("ListPendingInvitations", ctx, orgID).Return(expectedInvitations, nil)

	invitations, err := mockRepo.ListPendingInvitations(ctx, orgID)
	assert.NoError(t, err)
	assert.Len(t, invitations, 2)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_AcceptInvitation(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("AcceptInvitation", ctx, "test-token", userID).Return(nil)

	err := mockRepo.AcceptInvitation(ctx, "test-token", userID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockOrgRepository_DeleteInvitation(t *testing.T) {
	mockRepo := new(MockOrgRepository)
	ctx := context.Background()

	invID := uuid.New()

	mockRepo.On("DeleteInvitation", ctx, invID).Return(nil)

	err := mockRepo.DeleteInvitation(ctx, invID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

