package jsonrpc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Bidon15/popsigner/control-plane/internal/middleware"
	"github.com/Bidon15/popsigner/control-plane/internal/models"
)

// MockKeyRepository is a mock implementation of KeyRepository for testing.
type MockKeyRepository struct {
	mock.Mock
}

func (m *MockKeyRepository) Create(ctx context.Context, key *models.Key) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Key, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Key), args.Error(1)
}

func (m *MockKeyRepository) GetByName(ctx context.Context, orgID, namespaceID uuid.UUID, name string) (*models.Key, error) {
	args := m.Called(ctx, orgID, namespaceID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Key), args.Error(1)
}

func (m *MockKeyRepository) GetByAddress(ctx context.Context, orgID uuid.UUID, address string) (*models.Key, error) {
	args := m.Called(ctx, orgID, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Key), args.Error(1)
}

func (m *MockKeyRepository) GetByEthAddress(ctx context.Context, orgID uuid.UUID, ethAddress string) (*models.Key, error) {
	args := m.Called(ctx, orgID, ethAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Key), args.Error(1)
}

func (m *MockKeyRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.Key, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Key), args.Error(1)
}

func (m *MockKeyRepository) ListByNamespace(ctx context.Context, namespaceID uuid.UUID) ([]*models.Key, error) {
	args := m.Called(ctx, namespaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Key), args.Error(1)
}

func (m *MockKeyRepository) ListByEthAddresses(ctx context.Context, orgID uuid.UUID, ethAddresses []string) (map[string]*models.Key, error) {
	args := m.Called(ctx, orgID, ethAddresses)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.Key), args.Error(1)
}

func (m *MockKeyRepository) ListEthAddresses(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockKeyRepository) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	args := m.Called(ctx, orgID)
	return args.Int(0), args.Error(1)
}

func (m *MockKeyRepository) Update(ctx context.Context, key *models.Key) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockKeyRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Helper to create context with org ID
func contextWithOrgID(orgID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.OrgIDKey, orgID.String())
}

func TestEthAccountsHandler_Handle(t *testing.T) {
	t.Run("returns addresses for org", func(t *testing.T) {
		mockRepo := new(MockKeyRepository)
		handler := NewEthAccountsHandler(mockRepo)

		orgID := uuid.New()
		expectedAddrs := []string{
			"0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
			"0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		}

		mockRepo.On("ListEthAddresses", mock.Anything, orgID).Return(expectedAddrs, nil)

		ctx := contextWithOrgID(orgID)
		result, err := handler.Handle(ctx, json.RawMessage(`[]`))

		require.Nil(t, err)
		addrs, ok := result.([]string)
		require.True(t, ok)
		assert.Len(t, addrs, 2)
		assert.Equal(t, expectedAddrs, addrs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty array when no addresses", func(t *testing.T) {
		mockRepo := new(MockKeyRepository)
		handler := NewEthAccountsHandler(mockRepo)

		orgID := uuid.New()

		mockRepo.On("ListEthAddresses", mock.Anything, orgID).Return([]string{}, nil)

		ctx := contextWithOrgID(orgID)
		result, err := handler.Handle(ctx, json.RawMessage(`[]`))

		require.Nil(t, err)
		addrs, ok := result.([]string)
		require.True(t, ok)
		assert.Empty(t, addrs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns empty array when nil addresses", func(t *testing.T) {
		mockRepo := new(MockKeyRepository)
		handler := NewEthAccountsHandler(mockRepo)

		orgID := uuid.New()

		var nilAddrs []string
		mockRepo.On("ListEthAddresses", mock.Anything, orgID).Return(nilAddrs, nil)

		ctx := contextWithOrgID(orgID)
		result, err := handler.Handle(ctx, json.RawMessage(`[]`))

		require.Nil(t, err)
		addrs, ok := result.([]string)
		require.True(t, ok)
		assert.Empty(t, addrs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("returns error without org context", func(t *testing.T) {
		mockRepo := new(MockKeyRepository)
		handler := NewEthAccountsHandler(mockRepo)

		ctx := context.Background()
		_, err := handler.Handle(ctx, json.RawMessage(`[]`))

		require.NotNil(t, err)
		assert.Equal(t, UnauthorizedError, err.Code)
	})
}

