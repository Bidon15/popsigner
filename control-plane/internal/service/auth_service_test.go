package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
	apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	if args.Error(0) == nil {
		user.ID = uuid.New()
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
	}
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

// MockSessionRepository is a mock implementation of SessionRepository
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
	if args.Error(0) == nil {
		session.CreatedAt = time.Now()
	}
	return args.Error(0)
}

func (m *MockSessionRepository) Get(ctx context.Context, id string) (*models.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

func (m *MockSessionRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionRepository) CleanupExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func newTestAuthService(userRepo *MockUserRepository, sessionRepo *MockSessionRepository) AuthService {
	config := AuthServiceConfig{
		BCryptCost:    4, // Low cost for tests
		SessionExpiry: 7 * 24 * time.Hour,
	}
	return NewAuthService(userRepo, sessionRepo, config)
}

func TestAuthService_Register_Success(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	req := RegisterRequest{
		Email:    "newuser@example.com",
		Password: "password123",
		Name:     "New User",
	}

	// Email doesn't exist
	userRepo.On("GetByEmail", ctx, req.Email).Return(nil, nil)
	// Create succeeds
	userRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	user, err := svc.Register(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, req.Email, user.Email)
	assert.Equal(t, req.Name, *user.Name)
	userRepo.AssertExpectations(t)
}

func TestAuthService_Register_EmailExists(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	req := RegisterRequest{
		Email:    "existing@example.com",
		Password: "password123",
		Name:     "New User",
	}

	name := "Existing User"
	existingUser := &models.User{
		ID:    uuid.New(),
		Email: req.Email,
		Name:  &name,
	}

	userRepo.On("GetByEmail", ctx, req.Email).Return(existingUser, nil)

	user, err := svc.Register(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, user)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "conflict", apiErr.Code)
	userRepo.AssertExpectations(t)
}

func TestAuthService_Login_Success(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	email := "user@example.com"
	password := "password123"

	// Create a bcrypt hash for the password
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 4)
	hashStr := string(hash)
	name := "Test User"

	existingUser := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &hashStr,
		Name:         &name,
	}

	userRepo.On("GetByEmail", ctx, email).Return(existingUser, nil)
	sessionRepo.On("Create", ctx, mock.AnythingOfType("*models.Session")).Return(nil)
	userRepo.On("UpdateLastLogin", ctx, existingUser.ID).Return(nil)

	user, sessionID, err := svc.Login(ctx, email, password)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, sessionID)
	assert.Equal(t, existingUser.ID, user.ID)
	userRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidEmail(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	email := "nonexistent@example.com"
	password := "password123"

	userRepo.On("GetByEmail", ctx, email).Return(nil, nil)

	user, sessionID, err := svc.Login(ctx, email, password)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, sessionID)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "unauthorized", apiErr.Code)
	userRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	email := "user@example.com"
	correctPassword := "correctpassword"
	wrongPassword := "wrongpassword"

	hash, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), 4)
	hashStr := string(hash)
	name := "Test User"

	existingUser := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &hashStr,
		Name:         &name,
	}

	userRepo.On("GetByEmail", ctx, email).Return(existingUser, nil)

	user, sessionID, err := svc.Login(ctx, email, wrongPassword)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, sessionID)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "unauthorized", apiErr.Code)
	userRepo.AssertExpectations(t)
}

func TestAuthService_Logout(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	sessionID := "test-session-id"

	sessionRepo.On("Delete", ctx, sessionID).Return(nil)

	err := svc.Logout(ctx, sessionID)

	assert.NoError(t, err)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_LogoutAll(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	userID := uuid.New()

	sessionRepo.On("DeleteAllForUser", ctx, userID).Return(nil)

	err := svc.LogoutAll(ctx, userID)

	assert.NoError(t, err)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_ValidateSession_Valid(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	sessionID := "valid-session"
	userID := uuid.New()

	session := &models.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	name := "Test User"
	user := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  &name,
	}

	sessionRepo.On("Get", ctx, sessionID).Return(session, nil)
	userRepo.On("GetByID", ctx, userID).Return(user, nil)

	result, err := svc.ValidateSession(ctx, sessionID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, user.ID, result.ID)
	sessionRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestAuthService_ValidateSession_Expired(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	sessionID := "expired-session"
	userID := uuid.New()

	session := &models.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(-time.Hour), // Expired
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	sessionRepo.On("Get", ctx, sessionID).Return(session, nil)
	sessionRepo.On("Delete", ctx, sessionID).Return(nil)

	result, err := svc.ValidateSession(ctx, sessionID)

	assert.Error(t, err)
	assert.Nil(t, result)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "unauthorized", apiErr.Code)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_ValidateSession_NotFound(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	sessionID := "nonexistent-session"

	sessionRepo.On("Get", ctx, sessionID).Return(nil, nil)

	result, err := svc.ValidateSession(ctx, sessionID)

	assert.Error(t, err)
	assert.Nil(t, result)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "unauthorized", apiErr.Code)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_GetUserByID_Found(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	userID := uuid.New()
	name := "Test User"

	user := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  &name,
	}

	userRepo.On("GetByID", ctx, userID).Return(user, nil)

	result, err := svc.GetUserByID(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.ID)
	userRepo.AssertExpectations(t)
}

func TestAuthService_GetUserByID_NotFound(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	userID := uuid.New()

	userRepo.On("GetByID", ctx, userID).Return(nil, nil)

	result, err := svc.GetUserByID(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, result)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "not_found", apiErr.Code)
	userRepo.AssertExpectations(t)
}

func TestAuthService_UpdateProfile(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	userID := uuid.New()
	oldName := "Old Name"
	newName := "New Name"

	existingUser := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  &oldName,
	}

	req := UpdateProfileRequest{
		Name: &newName,
	}

	userRepo.On("GetByID", ctx, userID).Return(existingUser, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*models.User")).Return(nil)

	result, err := svc.UpdateProfile(ctx, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, newName, *result.Name)
	userRepo.AssertExpectations(t)
}

func TestAuthService_ChangePassword_Success(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	userID := uuid.New()
	oldPassword := "oldpassword"
	newPassword := "newpassword"

	oldHash, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), 4)
	oldHashStr := string(oldHash)
	name := "Test User"

	existingUser := &models.User{
		ID:           userID,
		Email:        "test@example.com",
		PasswordHash: &oldHashStr,
		Name:         &name,
	}

	userRepo.On("GetByID", ctx, userID).Return(existingUser, nil)
	userRepo.On("UpdatePassword", ctx, userID, mock.AnythingOfType("string")).Return(nil)
	sessionRepo.On("DeleteAllForUser", ctx, userID).Return(nil)

	err := svc.ChangePassword(ctx, userID, oldPassword, newPassword)

	assert.NoError(t, err)
	userRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestAuthService_ChangePassword_WrongOldPassword(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	userID := uuid.New()
	correctOldPassword := "correctold"
	wrongOldPassword := "wrongold"
	newPassword := "newpassword"

	correctHash, _ := bcrypt.GenerateFromPassword([]byte(correctOldPassword), 4)
	hashStr := string(correctHash)
	name := "Test User"

	existingUser := &models.User{
		ID:           userID,
		Email:        "test@example.com",
		PasswordHash: &hashStr,
		Name:         &name,
	}

	userRepo.On("GetByID", ctx, userID).Return(existingUser, nil)

	err := svc.ChangePassword(ctx, userID, wrongOldPassword, newPassword)

	assert.Error(t, err)

	apiErr, ok := err.(*apierrors.APIError)
	assert.True(t, ok)
	assert.Equal(t, "unauthorized", apiErr.Code)
	userRepo.AssertExpectations(t)
}

func TestAuthService_RequestPasswordReset_UserExists(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	email := "test@example.com"
	name := "Test User"

	user := &models.User{
		ID:    uuid.New(),
		Email: email,
		Name:  &name,
	}

	userRepo.On("GetByEmail", ctx, email).Return(user, nil)

	token, err := svc.RequestPasswordReset(ctx, email)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	userRepo.AssertExpectations(t)
}

func TestAuthService_RequestPasswordReset_UserNotExists(t *testing.T) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionRepository)
	svc := newTestAuthService(userRepo, sessionRepo)

	ctx := context.Background()
	email := "nonexistent@example.com"

	userRepo.On("GetByEmail", ctx, email).Return(nil, nil)

	token, err := svc.RequestPasswordReset(ctx, email)

	// Should not error to prevent email enumeration
	assert.NoError(t, err)
	assert.Empty(t, token)
	userRepo.AssertExpectations(t)
}

