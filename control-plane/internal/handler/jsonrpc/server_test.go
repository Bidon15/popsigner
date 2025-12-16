package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bidon15/popsigner/control-plane/internal/middleware"
	"github.com/Bidon15/popsigner/control-plane/internal/models"
	"github.com/Bidon15/popsigner/control-plane/internal/openbao"
)

// mockKeyRepoForServer implements repository.KeyRepository for server tests.
type mockKeyRepoForServer struct {
	addresses []string
}

func (m *mockKeyRepoForServer) Create(ctx context.Context, key *models.Key) error {
	return nil
}

func (m *mockKeyRepoForServer) GetByID(ctx context.Context, id uuid.UUID) (*models.Key, error) {
	return nil, nil
}

func (m *mockKeyRepoForServer) GetByName(ctx context.Context, orgID, namespaceID uuid.UUID, name string) (*models.Key, error) {
	return nil, nil
}

func (m *mockKeyRepoForServer) GetByAddress(ctx context.Context, orgID uuid.UUID, address string) (*models.Key, error) {
	return nil, nil
}

func (m *mockKeyRepoForServer) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.Key, error) {
	return nil, nil
}

func (m *mockKeyRepoForServer) ListByNamespace(ctx context.Context, namespaceID uuid.UUID) ([]*models.Key, error) {
	return nil, nil
}

func (m *mockKeyRepoForServer) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockKeyRepoForServer) Update(ctx context.Context, key *models.Key) error {
	return nil
}

func (m *mockKeyRepoForServer) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockKeyRepoForServer) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockKeyRepoForServer) GetByEthAddress(ctx context.Context, orgID uuid.UUID, ethAddress string) (*models.Key, error) {
	return nil, nil
}

func (m *mockKeyRepoForServer) ListByEthAddresses(ctx context.Context, orgID uuid.UUID, ethAddresses []string) (map[string]*models.Key, error) {
	return nil, nil
}

func (m *mockKeyRepoForServer) ListEthAddresses(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	return m.addresses, nil
}

func TestServer_NewServer(t *testing.T) {
	keyRepo := &mockKeyRepoForServer{addresses: []string{}}

	server := NewServer(ServerConfig{
		KeyRepo:   keyRepo,
		BaoClient: nil,
		Logger:    nil,
	})

	require.NotNil(t, server)
	require.NotNil(t, server.Handler())

	// Verify all methods are registered
	methods := server.RegisteredMethods()
	assert.Contains(t, methods, "eth_accounts")
	assert.Contains(t, methods, "eth_signTransaction")
	assert.Contains(t, methods, "eth_sign")
	assert.Contains(t, methods, "personal_sign")
}

func TestServer_ServeHTTP(t *testing.T) {
	keyRepo := &mockKeyRepoForServer{addresses: []string{"0x742d35Cc6634C0532925a3b844Bc454e4438f44e"}}
	server := NewServer(ServerConfig{
		KeyRepo:   keyRepo,
		BaoClient: nil,
		Logger:    nil,
	})

	// Create a context with org_id
	orgID := uuid.New()

	t.Run("handles eth_accounts request", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}`
		req := httptest.NewRequest("POST", "/", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Add org_id to context using the middleware's exported SetOrgIDInContext helper
		ctx := middleware.SetOrgIDInContext(req.Context(), orgID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Nil(t, resp.Error)
		assert.NotNil(t, resp.Result)
	})

	t.Run("returns error for unknown method", func(t *testing.T) {
		reqBody := `{"jsonrpc":"2.0","method":"eth_unknownMethod","params":[],"id":1}`
		req := httptest.NewRequest("POST", "/", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")

		ctx := middleware.SetOrgIDInContext(req.Context(), orgID)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, MethodNotFound, resp.Error.Code)
	})
}

func TestServer_RegisterMethod(t *testing.T) {
	keyRepo := &mockKeyRepoForServer{}
	server := NewServer(ServerConfig{
		KeyRepo:   keyRepo,
		BaoClient: nil,
		Logger:    nil,
	})

	// Register a custom method
	server.RegisterMethod("custom_method", func(ctx context.Context, params json.RawMessage) (interface{}, *Error) {
		return "custom_result", nil
	})

	methods := server.RegisteredMethods()
	assert.Contains(t, methods, "custom_method")
}

func TestServer_IntegrationWithBaoClient(t *testing.T) {
	// Skip if not running integration tests
	t.Skip("Integration test - requires OpenBao server")

	// This test would be enabled for full integration testing
	baoClient := openbao.NewClient(nil)
	keyRepo := &mockKeyRepoForServer{addresses: []string{}}

	server := NewServer(ServerConfig{
		KeyRepo:   keyRepo,
		BaoClient: baoClient,
		Logger:    nil,
	})

	require.NotNil(t, server)
}

