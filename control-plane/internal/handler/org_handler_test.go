package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	apierrors "github.com/Bidon15/popsigner/control-plane/internal/pkg/errors"
	"github.com/Bidon15/popsigner/control-plane/internal/service"
)

// MockOrgService is a mock implementation of OrgService
type MockOrgService struct {
	mock.Mock
}

func (m *MockOrgService) Create(ctx context.Context, name string, ownerID uuid.UUID) (*models.Organization, error) {
	args := m.Called(ctx, name, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrgService) Get(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrgService) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrgService) Update(ctx context.Context, id uuid.UUID, name string, actorID uuid.UUID) (*models.Organization, error) {
	args := m.Called(ctx, id, name, actorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrgService) Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	args := m.Called(ctx, id, actorID)
	return args.Error(0)
}

func (m *MockOrgService) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}

func (m *MockOrgService) InviteMember(ctx context.Context, orgID uuid.UUID, email string, role models.Role, inviterID uuid.UUID) (*models.Invitation, error) {
	args := m.Called(ctx, orgID, email, role, inviterID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Invitation), args.Error(1)
}

func (m *MockOrgService) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (*models.Organization, error) {
	args := m.Called(ctx, token, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrgService) RemoveMember(ctx context.Context, orgID, userID, actorID uuid.UUID) error {
	args := m.Called(ctx, orgID, userID, actorID)
	return args.Error(0)
}

func (m *MockOrgService) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role models.Role, actorID uuid.UUID) error {
	args := m.Called(ctx, orgID, userID, role, actorID)
	return args.Error(0)
}

func (m *MockOrgService) ListMembers(ctx context.Context, orgID uuid.UUID, actorID uuid.UUID) ([]*models.OrgMember, error) {
	args := m.Called(ctx, orgID, actorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.OrgMember), args.Error(1)
}

func (m *MockOrgService) ListPendingInvitations(ctx context.Context, orgID uuid.UUID, actorID uuid.UUID) ([]*models.Invitation, error) {
	args := m.Called(ctx, orgID, actorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Invitation), args.Error(1)
}

func (m *MockOrgService) CancelInvitation(ctx context.Context, orgID, invitationID uuid.UUID, actorID uuid.UUID) error {
	args := m.Called(ctx, orgID, invitationID, actorID)
	return args.Error(0)
}

func (m *MockOrgService) CheckAccess(ctx context.Context, orgID, userID uuid.UUID, requiredRole models.Role) error {
	args := m.Called(ctx, orgID, userID, requiredRole)
	return args.Error(0)
}

func (m *MockOrgService) GetMemberRole(ctx context.Context, orgID, userID uuid.UUID) (models.Role, error) {
	args := m.Called(ctx, orgID, userID)
	return args.Get(0).(models.Role), args.Error(1)
}

func (m *MockOrgService) GetLimits(ctx context.Context, orgID uuid.UUID) (*models.PlanLimits, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PlanLimits), args.Error(1)
}

func (m *MockOrgService) CreateNamespace(ctx context.Context, orgID uuid.UUID, name, description string, actorID uuid.UUID) (*models.Namespace, error) {
	args := m.Called(ctx, orgID, name, description, actorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Namespace), args.Error(1)
}

func (m *MockOrgService) GetNamespace(ctx context.Context, orgID, nsID uuid.UUID, actorID uuid.UUID) (*models.Namespace, error) {
	args := m.Called(ctx, orgID, nsID, actorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Namespace), args.Error(1)
}

func (m *MockOrgService) ListNamespaces(ctx context.Context, orgID uuid.UUID, actorID uuid.UUID) ([]*models.Namespace, error) {
	args := m.Called(ctx, orgID, actorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Namespace), args.Error(1)
}

func (m *MockOrgService) DeleteNamespace(ctx context.Context, orgID, nsID uuid.UUID, actorID uuid.UUID) error {
	args := m.Called(ctx, orgID, nsID, actorID)
	return args.Error(0)
}

// MockAuthServiceForOrg is a mock implementation of AuthService for org handler tests
type MockAuthServiceForOrg struct {
	mock.Mock
}

func (m *MockAuthServiceForOrg) Register(ctx context.Context, req service.RegisterRequest) (*models.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthServiceForOrg) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*models.User), args.String(1), args.Error(2)
}

func (m *MockAuthServiceForOrg) Logout(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockAuthServiceForOrg) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAuthServiceForOrg) ValidateSession(ctx context.Context, sessionID string) (*models.User, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthServiceForOrg) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthServiceForOrg) UpdateProfile(ctx context.Context, userID uuid.UUID, req service.UpdateProfileRequest) (*models.User, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthServiceForOrg) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	args := m.Called(ctx, userID, oldPassword, newPassword)
	return args.Error(0)
}

func (m *MockAuthServiceForOrg) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	args := m.Called(ctx, email)
	return args.String(0), args.Error(1)
}

func (m *MockAuthServiceForOrg) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}

func (m *MockAuthServiceForOrg) VerifyEmail(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func setupOrgTestHandler() (*OrgHandler, *MockOrgService, *MockAuthServiceForOrg) {
	orgService := new(MockOrgService)
	authService := new(MockAuthServiceForOrg)
	handler := NewOrgHandler(orgService, authService)
	return handler, orgService, authService
}

func createOrgTestRequest(method, path string, body interface{}, sessionID string) *http.Request {
	var reqBody bytes.Buffer
	if body != nil {
		json.NewEncoder(&reqBody).Encode(body)
	}
	req := httptest.NewRequest(method, path, &reqBody)
	req.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	}
	return req
}

func TestOrgHandler_CreateOrg_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	org := &models.Organization{
		ID:        uuid.New(),
		Name:      "Test Org",
		Slug:      "test-org",
		Plan:      models.PlanFree,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("Create", mock.Anything, "Test Org", userID).Return(org, nil)

	req := createOrgTestRequest("POST", "/v1/organizations", CreateOrgRequest{Name: "Test Org"}, sessionID)
	w := httptest.NewRecorder()

	handler.CreateOrg(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Test Org", data["name"])

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_CreateOrg_ValidationError(t *testing.T) {
	handler, _, authService := setupOrgTestHandler()

	userID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)

	req := createOrgTestRequest("POST", "/v1/organizations", CreateOrgRequest{Name: "A"}, sessionID)
	w := httptest.NewRecorder()

	handler.CreateOrg(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	authService.AssertExpectations(t)
}

func TestOrgHandler_CreateOrg_Unauthorized(t *testing.T) {
	handler, _, authService := setupOrgTestHandler()

	sessionID := "invalid-session"

	authService.On("ValidateSession", mock.Anything, sessionID).Return(nil, apierrors.ErrUnauthorized)

	req := createOrgTestRequest("POST", "/v1/organizations", CreateOrgRequest{Name: "Test Org"}, sessionID)
	w := httptest.NewRecorder()

	handler.CreateOrg(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	authService.AssertExpectations(t)
}

func TestOrgHandler_ListOrgs_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	orgs := []*models.Organization{
		{ID: uuid.New(), Name: "Org 1", Slug: "org-1"},
		{ID: uuid.New(), Name: "Org 2", Slug: "org-2"},
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("ListUserOrgs", mock.Anything, userID).Return(orgs, nil)

	req := createOrgTestRequest("GET", "/v1/organizations", nil, sessionID)
	w := httptest.NewRecorder()

	handler.ListOrgs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].([]interface{})
	assert.Len(t, data, 2)

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_GetOrg_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	org := &models.Organization{
		ID:   orgID,
		Name: "Test Org",
		Slug: "test-org",
		Plan: models.PlanFree,
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("CheckAccess", mock.Anything, orgID, userID, models.RoleViewer).Return(nil)
	orgService.On("Get", mock.Anything, orgID).Return(org, nil)

	// Create router to handle URL params
	r := chi.NewRouter()
	r.Get("/{orgId}", handler.GetOrg)

	req := createOrgTestRequest("GET", "/"+orgID.String(), nil, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_GetOrg_NotFound(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("CheckAccess", mock.Anything, orgID, userID, models.RoleViewer).Return(nil)
	orgService.On("Get", mock.Anything, orgID).Return(nil, apierrors.NewNotFoundError("Organization"))

	r := chi.NewRouter()
	r.Get("/{orgId}", handler.GetOrg)

	req := createOrgTestRequest("GET", "/"+orgID.String(), nil, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_UpdateOrg_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	updatedOrg := &models.Organization{
		ID:   orgID,
		Name: "New Name",
		Slug: "new-name",
		Plan: models.PlanFree,
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("Update", mock.Anything, orgID, "New Name", userID).Return(updatedOrg, nil)

	r := chi.NewRouter()
	r.Patch("/{orgId}", handler.UpdateOrg)

	req := createOrgTestRequest("PATCH", "/"+orgID.String(), UpdateOrgRequest{Name: "New Name"}, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "New Name", data["name"])

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_DeleteOrg_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("Delete", mock.Anything, orgID, userID).Return(nil)

	r := chi.NewRouter()
	r.Delete("/{orgId}", handler.DeleteOrg)

	req := createOrgTestRequest("DELETE", "/"+orgID.String(), nil, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_ListMembers_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	members := []*models.OrgMember{
		{OrgID: orgID, UserID: userID, Role: models.RoleOwner},
		{OrgID: orgID, UserID: uuid.New(), Role: models.RoleAdmin},
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("ListMembers", mock.Anything, orgID, userID).Return(members, nil)

	r := chi.NewRouter()
	r.Get("/{orgId}/members", handler.ListMembers)

	req := createOrgTestRequest("GET", "/"+orgID.String()+"/members", nil, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].([]interface{})
	assert.Len(t, data, 2)

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_InviteMember_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	inv := &models.Invitation{
		ID:        uuid.New(),
		OrgID:     orgID,
		Email:     "newuser@example.com",
		Role:      models.RoleViewer,
		InvitedBy: userID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("InviteMember", mock.Anything, orgID, "newuser@example.com", models.RoleViewer, userID).Return(inv, nil)

	r := chi.NewRouter()
	r.Post("/{orgId}/members", handler.InviteMember)

	req := createOrgTestRequest("POST", "/"+orgID.String()+"/members", InviteMemberRequest{
		Email: "newuser@example.com",
		Role:  models.RoleViewer,
	}, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "newuser@example.com", data["email"])

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_RemoveMember_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	targetID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("RemoveMember", mock.Anything, orgID, targetID, userID).Return(nil)

	r := chi.NewRouter()
	r.Delete("/{orgId}/members/{userId}", handler.RemoveMember)

	req := createOrgTestRequest("DELETE", "/"+orgID.String()+"/members/"+targetID.String(), nil, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_UpdateMemberRole_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	targetID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("UpdateMemberRole", mock.Anything, orgID, targetID, models.RoleOperator, userID).Return(nil)

	r := chi.NewRouter()
	r.Patch("/{orgId}/members/{userId}", handler.UpdateMemberRole)

	req := createOrgTestRequest("PATCH", "/"+orgID.String()+"/members/"+targetID.String(), UpdateMemberRoleRequest{
		Role: models.RoleOperator,
	}, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_ListNamespaces_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	namespaces := []*models.Namespace{
		{ID: uuid.New(), OrgID: orgID, Name: "production"},
		{ID: uuid.New(), OrgID: orgID, Name: "staging"},
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("ListNamespaces", mock.Anything, orgID, userID).Return(namespaces, nil)

	r := chi.NewRouter()
	r.Get("/{orgId}/namespaces", handler.ListNamespaces)

	req := createOrgTestRequest("GET", "/"+orgID.String()+"/namespaces", nil, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].([]interface{})
	assert.Len(t, data, 2)

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_CreateNamespace_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	desc := "Staging environment"
	ns := &models.Namespace{
		ID:          uuid.New(),
		OrgID:       orgID,
		Name:        "staging",
		Description: &desc,
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("CreateNamespace", mock.Anything, orgID, "staging", "Staging environment", userID).Return(ns, nil)

	r := chi.NewRouter()
	r.Post("/{orgId}/namespaces", handler.CreateNamespace)

	req := createOrgTestRequest("POST", "/"+orgID.String()+"/namespaces", CreateNamespaceRequest{
		Name:        "staging",
		Description: "Staging environment",
	}, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "staging", data["name"])

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_DeleteNamespace_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	nsID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("DeleteNamespace", mock.Anything, orgID, nsID, userID).Return(nil)

	r := chi.NewRouter()
	r.Delete("/{orgId}/namespaces/{namespaceId}", handler.DeleteNamespace)

	req := createOrgTestRequest("DELETE", "/"+orgID.String()+"/namespaces/"+nsID.String(), nil, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_GetLimits_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	orgID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	limits := &models.PlanLimits{
		Keys:               25,
		SignaturesPerMonth: 500000,
		Namespaces:         5,
		TeamMembers:        10,
		AuditRetentionDays: 90,
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("CheckAccess", mock.Anything, orgID, userID, models.RoleViewer).Return(nil)
	orgService.On("GetLimits", mock.Anything, orgID).Return(limits, nil)

	r := chi.NewRouter()
	r.Get("/{orgId}/limits", handler.GetLimits)

	req := createOrgTestRequest("GET", "/"+orgID.String()+"/limits", nil, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(25), data["keys"])
	assert.Equal(t, float64(10), data["team_members"])

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_AcceptInvitation_Success(t *testing.T) {
	handler, orgService, authService := setupOrgTestHandler()

	userID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	org := &models.Organization{
		ID:   uuid.New(),
		Name: "Test Org",
		Slug: "test-org",
	}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)
	orgService.On("AcceptInvitation", mock.Anything, "test-token", userID).Return(org, nil)

	req := createOrgTestRequest("POST", "/accept", AcceptInvitationRequest{Token: "test-token"}, sessionID)
	w := httptest.NewRecorder()

	handler.AcceptInvitation(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Test Org", data["name"])

	orgService.AssertExpectations(t)
	authService.AssertExpectations(t)
}

func TestOrgHandler_InvalidOrgID(t *testing.T) {
	handler, _, authService := setupOrgTestHandler()

	userID := uuid.New()
	sessionID := "test-session"
	user := &models.User{ID: userID, Email: "test@example.com"}

	authService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)

	r := chi.NewRouter()
	r.Get("/{orgId}", handler.GetOrg)

	req := createOrgTestRequest("GET", "/invalid-uuid", nil, sessionID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	authService.AssertExpectations(t)
}

