package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Bidon15/banhbaoring/control-plane/internal/models"
)

// MockSessionRepository is a mock implementation of SessionRepository for testing.
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *models.Session) error {
	args := m.Called(ctx, session)
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

// Verify MockSessionRepository implements SessionRepository
var _ SessionRepository = (*MockSessionRepository)(nil)

func TestMockSessionRepository_Create(t *testing.T) {
	mockRepo := new(MockSessionRepository)
	ctx := context.Background()

	session := &models.Session{
		ID:        "test-session-id",
		UserID:    uuid.New(),
		Data:      map[string]interface{}{"key": "value"},
		ExpiresAt: time.Now().Add(time.Hour),
	}

	mockRepo.On("Create", ctx, session).Return(nil)

	err := mockRepo.Create(ctx, session)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockSessionRepository_Get(t *testing.T) {
	mockRepo := new(MockSessionRepository)
	ctx := context.Background()

	sessionID := "test-session-id"
	expectedSession := &models.Session{
		ID:        sessionID,
		UserID:    uuid.New(),
		Data:      map[string]interface{}{"key": "value"},
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	mockRepo.On("Get", ctx, sessionID).Return(expectedSession, nil)

	session, err := mockRepo.Get(ctx, sessionID)
	assert.NoError(t, err)
	assert.Equal(t, expectedSession, session)
	mockRepo.AssertExpectations(t)
}

func TestMockSessionRepository_Get_NotFound(t *testing.T) {
	mockRepo := new(MockSessionRepository)
	ctx := context.Background()

	sessionID := "nonexistent-session"

	mockRepo.On("Get", ctx, sessionID).Return(nil, nil)

	session, err := mockRepo.Get(ctx, sessionID)
	assert.NoError(t, err)
	assert.Nil(t, session)
	mockRepo.AssertExpectations(t)
}

func TestMockSessionRepository_Delete(t *testing.T) {
	mockRepo := new(MockSessionRepository)
	ctx := context.Background()

	sessionID := "test-session-id"

	mockRepo.On("Delete", ctx, sessionID).Return(nil)

	err := mockRepo.Delete(ctx, sessionID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockSessionRepository_DeleteAllForUser(t *testing.T) {
	mockRepo := new(MockSessionRepository)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("DeleteAllForUser", ctx, userID).Return(nil)

	err := mockRepo.DeleteAllForUser(ctx, userID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockSessionRepository_CleanupExpired(t *testing.T) {
	mockRepo := new(MockSessionRepository)
	ctx := context.Background()

	mockRepo.On("CleanupExpired", ctx).Return(int64(5), nil)

	count, err := mockRepo.CleanupExpired(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	mockRepo.AssertExpectations(t)
}

