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

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
	apierrors "github.com/Bidon15/banhbaoring/control-plane/internal/pkg/errors"
	"github.com/Bidon15/banhbaoring/control-plane/internal/service"
)

// MockAuthService is a mock implementation of AuthService for testing.
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, req service.RegisterRequest) (*models.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*models.User), args.String(1), args.Error(2)
}

func (m *MockAuthService) Logout(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockAuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAuthService) ValidateSession(ctx context.Context, sessionID string) (*models.User, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, req service.UpdateProfileRequest) (*models.User, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	args := m.Called(ctx, userID, oldPassword, newPassword)
	return args.Error(0)
}

func (m *MockAuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	args := m.Called(ctx, email)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}

func (m *MockAuthService) VerifyEmail(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

// Verify MockAuthService implements AuthService
var _ service.AuthService = (*MockAuthService)(nil)

func newTestHandler(mockService *MockAuthService) *AuthHandler {
	cfg := AuthHandlerConfig{
		SessionExpiry: 7 * 24 * time.Hour,
		SecureCookie:  false,
	}
	return NewAuthHandler(mockService, cfg)
}

func newTestRouter(handler *AuthHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Mount("/v1/auth", handler.Routes())
	return r
}

func TestAuthHandler_Register_Success(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	name := "Test User"
	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Name:      &name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockService.On("Register", mock.Anything, mock.MatchedBy(func(req service.RegisterRequest) bool {
		return req.Email == "test@example.com" && req.Password == "password123" && req.Name == "Test User"
	})).Return(user, nil)

	body := `{"email":"test@example.com","password":"password123","name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	// Missing required fields
	body := `{"email":"invalid-email"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_EmailExists(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	mockService.On("Register", mock.Anything, mock.Anything).Return(nil, apierrors.NewConflictError("Email already registered"))

	body := `{"email":"existing@example.com","password":"password123","name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	name := "Test User"
	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Name:      &name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	sessionID := "test-session-id"

	mockService.On("Login", mock.Anything, "test@example.com", "password123").Return(user, sessionID, nil)

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check that session cookie is set
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)
	assert.Equal(t, sessionID, sessionCookie.Value)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	mockService.On("Login", mock.Anything, "test@example.com", "wrongpassword").Return(nil, "", apierrors.ErrUnauthorized.WithMessage("Invalid email or password"))

	body := `{"email":"test@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	sessionID := "test-session-id"
	mockService.On("Logout", mock.Anything, sessionID).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Check that session cookie is cleared
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)
	assert.Equal(t, "", sessionCookie.Value)
	assert.True(t, sessionCookie.MaxAge < 0)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Logout_NoSession(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthHandler_Me_Success(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	name := "Test User"
	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Name:      &name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	sessionID := "test-session-id"

	mockService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["data"])
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Me_NoSession(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Me_InvalidSession(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	sessionID := "invalid-session-id"
	mockService.On("ValidateSession", mock.Anything, sessionID).Return(nil, apierrors.ErrUnauthorized.WithMessage("Invalid session"))

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_UpdateProfile_Success(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	sessionID := "test-session-id"
	userID := uuid.New()
	oldName := "Old Name"
	newName := "New Name"

	currentUser := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  &oldName,
	}

	updatedUser := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  &newName,
	}

	mockService.On("ValidateSession", mock.Anything, sessionID).Return(currentUser, nil)
	mockService.On("UpdateProfile", mock.Anything, userID, mock.MatchedBy(func(req service.UpdateProfileRequest) bool {
		return req.Name != nil && *req.Name == newName
	})).Return(updatedUser, nil)

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/auth/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	sessionID := "test-session-id"
	userID := uuid.New()
	name := "Test User"

	currentUser := &models.User{
		ID:    userID,
		Email: "test@example.com",
		Name:  &name,
	}

	mockService.On("ValidateSession", mock.Anything, sessionID).Return(currentUser, nil)
	mockService.On("ChangePassword", mock.Anything, userID, "oldpassword", "newpassword123").Return(nil)

	body := `{"old_password":"oldpassword","new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/password/change", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_ForgotPassword_Success(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	mockService.On("RequestPasswordReset", mock.Anything, "test@example.com").Return("reset-token", nil)

	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/password/forgot", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["message"], "password reset link")
	mockService.AssertExpectations(t)
}

func TestAuthHandler_Me_WithBearerToken(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	name := "Test User"
	user := &models.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Name:      &name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	sessionID := "test-session-id"

	mockService.On("ValidateSession", mock.Anything, sessionID).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestAuthHandler_InvalidJSON(t *testing.T) {
	mockService := new(MockAuthService)
	handler := newTestHandler(mockService)
	router := newTestRouter(handler)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

