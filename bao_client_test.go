package popsigner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaoClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config with defaults",
			cfg: Config{
				BaoAddr:  "https://localhost:8200",
				BaoToken: "test-token",
			},
			wantErr: false,
		},
		{
			name: "valid config with all options",
			cfg: Config{
				BaoAddr:       "https://localhost:8200/",
				BaoToken:      "test-token",
				BaoNamespace:  "ns1",
				Secp256k1Path: "custom-path",
				HTTPTimeout:   60 * time.Second,
				SkipTLSVerify: true,
			},
			wantErr: false,
		},
		{
			name: "empty config uses defaults",
			cfg: Config{
				BaoAddr: "https://localhost:8200",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewBaoClient(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, client)

			// Verify defaults applied
			if tt.cfg.Secp256k1Path == "" {
				assert.Equal(t, DefaultSecp256k1Path, client.secp256k1Path)
			} else {
				assert.Equal(t, tt.cfg.Secp256k1Path, client.secp256k1Path)
			}

			// Verify trailing slash removed (only check if baseURL is not empty)
			if len(client.baseURL) > 0 {
				assert.False(t, client.baseURL[len(client.baseURL)-1] == '/')
			}
		})
	}
}

func TestBaoClient_CreateKey(t *testing.T) {
	tests := []struct {
		name       string
		keyName    string
		opts       KeyOptions
		serverResp func(w http.ResponseWriter, r *http.Request)
		wantErr    bool
		errCheck   func(error) bool
	}{
		{
			name:    "successful key creation",
			keyName: "test-key",
			opts:    KeyOptions{Exportable: true},
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Contains(t, r.URL.Path, "/v1/secp256k1/keys/test-key")
				assert.Equal(t, "test-token", r.Header.Get("X-Vault-Token"))

				var body map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&body)
				require.NoError(t, err)
				assert.True(t, body["exportable"].(bool))

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": KeyInfo{
						Name:       "test-key",
						PublicKey:  "02abcdef1234567890",
						Address:    "cosmos1abc123",
						Exportable: true,
					},
				})
			},
			wantErr: false,
		},
		{
			name:    "server returns error",
			keyName: "bad-key",
			opts:    KeyOptions{},
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []string{"permission denied"},
				})
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrBaoAuth)
			},
		},
		{
			name:    "invalid json response",
			keyName: "invalid-key",
			opts:    KeyOptions{},
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("not json"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(tt.serverResp))
			defer server.Close()

			client, err := NewBaoClient(Config{
				BaoAddr:       server.URL,
				BaoToken:      "test-token",
				SkipTLSVerify: true,
			})
			require.NoError(t, err)

			info, err := client.CreateKey(context.Background(), tt.keyName, tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errCheck != nil {
					assert.True(t, tt.errCheck(err), "error check failed: %v", err)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.keyName, info.Name)
		})
	}
}

func TestBaoClient_GetKey(t *testing.T) {
	tests := []struct {
		name       string
		keyName    string
		serverResp func(w http.ResponseWriter, r *http.Request)
		wantErr    bool
		errCheck   func(error) bool
	}{
		{
			name:    "successful get",
			keyName: "test-key",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": KeyInfo{
						Name:      "test-key",
						PublicKey: "02abcdef",
					},
				})
			},
			wantErr: false,
		},
		{
			name:    "key not found",
			keyName: "missing-key",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []string{"key not found"},
				})
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrKeyNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(tt.serverResp))
			defer server.Close()

			client, err := NewBaoClient(Config{
				BaoAddr:       server.URL,
				BaoToken:      "test-token",
				SkipTLSVerify: true,
			})
			require.NoError(t, err)

			info, err := client.GetKey(context.Background(), tt.keyName)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errCheck != nil {
					assert.True(t, tt.errCheck(err), "error check failed: %v", err)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.keyName, info.Name)
		})
	}
}

func TestBaoClient_ListKeys(t *testing.T) {
	tests := []struct {
		name       string
		serverResp func(w http.ResponseWriter, r *http.Request)
		wantKeys   []string
		wantErr    bool
	}{
		{
			name: "successful list",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "LIST", r.Method)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"keys": []string{"key1", "key2", "key3"},
					},
				})
			},
			wantKeys: []string{"key1", "key2", "key3"},
			wantErr:  false,
		},
		{
			name: "empty list",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{
						"keys": []string{},
					},
				})
			},
			wantKeys: []string{},
			wantErr:  false,
		},
		{
			name: "server error",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []string{"internal error"},
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(tt.serverResp))
			defer server.Close()

			client, err := NewBaoClient(Config{
				BaoAddr:       server.URL,
				BaoToken:      "test-token",
				SkipTLSVerify: true,
			})
			require.NoError(t, err)

			keys, err := client.ListKeys(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantKeys, keys)
		})
	}
}

func TestBaoClient_DeleteKey(t *testing.T) {
	tests := []struct {
		name       string
		keyName    string
		serverResp func(w http.ResponseWriter, r *http.Request)
		wantErr    bool
	}{
		{
			name:    "successful delete",
			keyName: "test-key",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				// Handle both the config update and delete requests
				if r.Method == "POST" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				if r.Method == "DELETE" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			},
			wantErr: false,
		},
		{
			name:    "delete fails",
			keyName: "protected-key",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []string{"deletion not allowed"},
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(tt.serverResp))
			defer server.Close()

			client, err := NewBaoClient(Config{
				BaoAddr:       server.URL,
				BaoToken:      "test-token",
				SkipTLSVerify: true,
			})
			require.NoError(t, err)

			err = client.DeleteKey(context.Background(), tt.keyName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestBaoClient_Sign(t *testing.T) {
	validSig := make([]byte, 64)
	for i := range validSig {
		validSig[i] = byte(i)
	}

	tests := []struct {
		name       string
		keyName    string
		data       []byte
		prehashed  bool
		serverResp func(w http.ResponseWriter, r *http.Request)
		wantErr    bool
		errCheck   func(error) bool
	}{
		{
			name:      "successful sign prehashed",
			keyName:   "test-key",
			data:      make([]byte, 32),
			prehashed: true,
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				var body map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&body)
				require.NoError(t, err)
				assert.True(t, body["prehashed"].(bool))
				assert.Equal(t, "cosmos", body["output_format"])
				assert.Nil(t, body["hash_algorithm"])

				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": SignResponse{
						Signature: base64.StdEncoding.EncodeToString(validSig),
						PublicKey: "02abcdef",
					},
				})
			},
			wantErr: false,
		},
		{
			name:      "successful sign not prehashed",
			keyName:   "test-key",
			data:      []byte("test message"),
			prehashed: false,
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&body)
				require.NoError(t, err)
				assert.False(t, body["prehashed"].(bool))
				assert.Equal(t, "sha256", body["hash_algorithm"])

				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": SignResponse{
						Signature: base64.StdEncoding.EncodeToString(validSig),
					},
				})
			},
			wantErr: false,
		},
		{
			name:      "invalid signature length",
			keyName:   "test-key",
			data:      []byte("test"),
			prehashed: true,
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				// Return signature that's not 64 bytes
				shortSig := make([]byte, 32)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": SignResponse{
						Signature: base64.StdEncoding.EncodeToString(shortSig),
					},
				})
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrInvalidSignature)
			},
		},
		{
			name:      "invalid base64 signature",
			keyName:   "test-key",
			data:      []byte("test"),
			prehashed: true,
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": SignResponse{
						Signature: "not-valid-base64!!!",
					},
				})
			},
			wantErr: true,
		},
		{
			name:      "server error",
			keyName:   "test-key",
			data:      []byte("test"),
			prehashed: true,
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []string{"signing denied"},
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(tt.serverResp))
			defer server.Close()

			client, err := NewBaoClient(Config{
				BaoAddr:       server.URL,
				BaoToken:      "test-token",
				SkipTLSVerify: true,
			})
			require.NoError(t, err)

			sig, err := client.Sign(context.Background(), tt.keyName, tt.data, tt.prehashed)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errCheck != nil {
					assert.True(t, tt.errCheck(err), "error check failed: %v", err)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, sig, 64)
		})
	}
}

func TestBaoClient_Health(t *testing.T) {
	tests := []struct {
		name       string
		serverResp func(w http.ResponseWriter, r *http.Request)
		wantErr    error
	}{
		{
			name: "healthy",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/sys/health", r.URL.Path)
				w.WriteHeader(http.StatusOK)
			},
			wantErr: nil,
		},
		{
			name: "sealed",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr: ErrBaoSealed,
		},
		{
			name: "unavailable",
			serverResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: ErrBaoUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(tt.serverResp))
			defer server.Close()

			client, err := NewBaoClient(Config{
				BaoAddr:       server.URL,
				BaoToken:      "test-token",
				SkipTLSVerify: true,
			})
			require.NoError(t, err)

			err = client.Health(context.Background())
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestBaoClient_HealthConnectionError(t *testing.T) {
	client, err := NewBaoClient(Config{
		BaoAddr:       "https://localhost:9999",
		BaoToken:      "test-token",
		HTTPTimeout:   1 * time.Second,
		SkipTLSVerify: true,
	})
	require.NoError(t, err)

	err = client.Health(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBaoConnection))
}

func TestBaoClient_NamespaceHeader(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-namespace", r.Header.Get("X-Vault-Namespace"))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": KeyInfo{Name: "test"},
		})
	}))
	defer server.Close()

	client, err := NewBaoClient(Config{
		BaoAddr:       server.URL,
		BaoToken:      "test-token",
		BaoNamespace:  "test-namespace",
		SkipTLSVerify: true,
	})
	require.NoError(t, err)

	_, err = client.GetKey(context.Background(), "test")
	require.NoError(t, err)
}

func TestBaoClient_ContextCancellation(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(5 * time.Second)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": KeyInfo{Name: "test"},
		})
	}))
	defer server.Close()

	client, err := NewBaoClient(Config{
		BaoAddr:       server.URL,
		BaoToken:      "test-token",
		SkipTLSVerify: true,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = client.GetKey(ctx, "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBaoConnection))
}

func TestBaoClient_ErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errCheck   func(error) bool
	}{
		{
			name:       "403 maps to ErrBaoAuth",
			statusCode: 403,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrBaoAuth)
			},
		},
		{
			name:       "404 maps to ErrKeyNotFound",
			statusCode: 404,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrKeyNotFound)
			},
		},
		{
			name:       "503 maps to ErrBaoSealed",
			statusCode: 503,
			errCheck: func(err error) bool {
				return errors.Is(err, ErrBaoSealed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []string{"error message"},
				})
			}))
			defer server.Close()

			client, err := NewBaoClient(Config{
				BaoAddr:       server.URL,
				BaoToken:      "test-token",
				SkipTLSVerify: true,
			})
			require.NoError(t, err)

			_, err = client.GetKey(context.Background(), "test")
			require.Error(t, err)
			assert.True(t, tt.errCheck(err), "error check failed for status %d: %v", tt.statusCode, err)
		})
	}
}

func TestBaoClient_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	client, err := NewBaoClient(Config{
		BaoAddr:       server.URL,
		BaoToken:      "test-token",
		SkipTLSVerify: true,
	})
	require.NoError(t, err)

	// Test GetKey with invalid JSON
	_, err = client.GetKey(context.Background(), "test")
	require.Error(t, err)

	// Test ListKeys with invalid JSON
	_, err = client.ListKeys(context.Background())
	require.Error(t, err)
}

func TestBaoClient_SignResponseUnmarshalError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid"))
	}))
	defer server.Close()

	client, err := NewBaoClient(Config{
		BaoAddr:       server.URL,
		BaoToken:      "test-token",
		SkipTLSVerify: true,
	})
	require.NoError(t, err)

	_, err = client.Sign(context.Background(), "key", []byte("data"), true)
	require.Error(t, err)
}
