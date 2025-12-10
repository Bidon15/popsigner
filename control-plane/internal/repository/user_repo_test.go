package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// MockUserRepository is a mock implementation of UserRepository for testing.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	args := m.Called(ctx, id, passwordHash)
	return args.Error(0)
}

func (m *MockUserRepository) SetEmailVerified(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) GetByOAuth(ctx context.Context, provider, providerID string) (*models.User, error) {
	args := m.Called(ctx, provider, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateOAuth(ctx context.Context, userID uuid.UUID, provider, providerID string) error {
	args := m.Called(ctx, userID, provider, providerID)
	return args.Error(0)
}

// Verify MockUserRepository implements UserRepository
var _ UserRepository = (*MockUserRepository)(nil)

func TestMockUserRepository_Create(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	name := "Test User"
	hash := "hashedpassword"
	user := &models.User{
		Email:        "test@example.com",
		PasswordHash: &hash,
		Name:         &name,
	}

	mockRepo.On("Create", ctx, user).Return(nil)

	err := mockRepo.Create(ctx, user)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockUserRepository_GetByID(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	userID := uuid.New()
	name := "Test User"
	expectedUser := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  &name,
	}

	mockRepo.On("GetByID", ctx, userID).Return(expectedUser, nil)

	user, err := mockRepo.GetByID(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)
}

func TestMockUserRepository_GetByID_NotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("GetByID", ctx, userID).Return(nil, nil)

	user, err := mockRepo.GetByID(ctx, userID)
	assert.NoError(t, err)
	assert.Nil(t, user)
	mockRepo.AssertExpectations(t)
}

func TestMockUserRepository_GetByEmail(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	email := "test@example.com"
	name := "Test User"
	expectedUser := &models.User{
		ID:    uuid.New(),
		Email: email,
		Name:  &name,
	}

	mockRepo.On("GetByEmail", ctx, email).Return(expectedUser, nil)

	user, err := mockRepo.GetByEmail(ctx, email)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)
}

func TestMockUserRepository_Update(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	name := "Updated Name"
	user := &models.User{
		ID:   uuid.New(),
		Name: &name,
	}

	mockRepo.On("Update", ctx, user).Return(nil)

	err := mockRepo.Update(ctx, user)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockUserRepository_UpdatePassword(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	userID := uuid.New()
	newHash := "newhash"

	mockRepo.On("UpdatePassword", ctx, userID, newHash).Return(nil)

	err := mockRepo.UpdatePassword(ctx, userID, newHash)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockUserRepository_SetEmailVerified(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("SetEmailVerified", ctx, userID).Return(nil)

	err := mockRepo.SetEmailVerified(ctx, userID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockUserRepository_UpdateLastLogin(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("UpdateLastLogin", ctx, userID).Return(nil)

	err := mockRepo.UpdateLastLogin(ctx, userID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockUserRepository_GetByOAuth(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	provider := "github"
	providerID := "12345"
	name := "Test User"
	expectedUser := &models.User{
		ID:              uuid.New(),
		Email:           "test@example.com",
		Name:            &name,
		OAuthProvider:   &provider,
		OAuthProviderID: &providerID,
	}

	mockRepo.On("GetByOAuth", ctx, provider, providerID).Return(expectedUser, nil)

	user, err := mockRepo.GetByOAuth(ctx, provider, providerID)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser, user)
	mockRepo.AssertExpectations(t)
}

func TestMockUserRepository_UpdateOAuth(t *testing.T) {
	mockRepo := new(MockUserRepository)
	ctx := context.Background()

	userID := uuid.New()
	provider := "github"
	providerID := "12345"

	mockRepo.On("UpdateOAuth", ctx, userID, provider, providerID).Return(nil)

	err := mockRepo.UpdateOAuth(ctx, userID, provider, providerID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

