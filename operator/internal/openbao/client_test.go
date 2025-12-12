package openbao

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8200", "test-token")

	if client.addr != "http://localhost:8200" {
		t.Errorf("expected addr to be http://localhost:8200, got %s", client.addr)
	}
	if client.token != "test-token" {
		t.Errorf("expected token to be test-token, got %s", client.token)
	}
	if client.client == nil {
		t.Error("expected HTTP client to be initialized")
	}
}

func TestWithNamespace(t *testing.T) {
	client := NewClient("http://localhost:8200", "test-token")
	nsClient := client.WithNamespace("tenant-test")

	if nsClient.namespace != "tenant-test" {
		t.Errorf("expected namespace to be tenant-test, got %s", nsClient.namespace)
	}
	if nsClient.addr != client.addr {
		t.Error("expected addr to be inherited from parent")
	}
	if nsClient.token != client.token {
		t.Error("expected token to be inherited from parent")
	}
}

func TestCreateNamespace(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"success OK", http.StatusOK, false},
		{"success NoContent", http.StatusNoContent, false},
		{"error BadRequest", http.StatusBadRequest, true},
		{"error Forbidden", http.StatusForbidden, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.Header.Get("X-Vault-Token") != "test-token" {
					t.Error("expected X-Vault-Token header")
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			err := client.CreateNamespace(context.Background(), "test-ns")

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateNamespace() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteNamespace(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"success OK", http.StatusOK, false},
		{"success NoContent", http.StatusNoContent, false},
		{"success NotFound", http.StatusNotFound, false},
		{"error Forbidden", http.StatusForbidden, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("expected DELETE, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			err := client.DeleteNamespace(context.Background(), "test-ns")

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteNamespace() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNamespaceExists(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
		wantErr    bool
	}{
		{"exists", http.StatusOK, true, false},
		{"not exists", http.StatusNotFound, false, false},
		{"error", http.StatusForbidden, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			got, err := client.NamespaceExists(context.Background(), "test-ns")

			if (err != nil) != tt.wantErr {
				t.Errorf("NamespaceExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NamespaceExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreatePolicy(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"success OK", http.StatusOK, false},
		{"success NoContent", http.StatusNoContent, false},
		{"error BadRequest", http.StatusBadRequest, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("expected PUT, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("expected Content-Type application/json")
				}

				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error("failed to decode request body")
				}
				if body["policy"] == "" {
					t.Error("expected policy in request body")
				}

				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			err := client.CreatePolicy(context.Background(), "test-policy", "path \"secret/*\" { capabilities = [\"read\"] }")

			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeletePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.DeletePolicy(context.Background(), "test-policy")

	if err != nil {
		t.Errorf("DeletePolicy() unexpected error: %v", err)
	}
}

func TestEnableSecretsEngine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error("failed to decode request body")
		}
		if body["type"] != "secp256k1" {
			t.Errorf("expected type secp256k1, got %v", body["type"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.EnableSecretsEngine(context.Background(), "keys", "secp256k1")

	if err != nil {
		t.Errorf("EnableSecretsEngine() unexpected error: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"healthy active", http.StatusOK, true},
		{"healthy standby", 429, true},
		{"healthy performance standby", 473, true},
		{"not initialized", 501, false},
		{"sealed", 503, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			got, err := client.HealthCheck(context.Background())

			if err != nil {
				t.Errorf("HealthCheck() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("HealthCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLookupSelf(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"accessor":    "test-accessor",
				"policies":    []string{"default", "admin"},
				"renewable":   true,
				"ttl":         3600,
				"entity_id":   "entity-123",
				"expire_time": "2024-12-31T23:59:59Z",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	info, err := client.LookupSelf(context.Background())

	if err != nil {
		t.Fatalf("LookupSelf() unexpected error: %v", err)
	}

	if info.Accessor != "test-accessor" {
		t.Errorf("expected accessor test-accessor, got %s", info.Accessor)
	}
	if len(info.Policies) != 2 {
		t.Errorf("expected 2 policies, got %d", len(info.Policies))
	}
	if !info.Renewable {
		t.Error("expected renewable to be true")
	}
}

func TestNamespaceHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ns := r.Header.Get("X-Vault-Namespace")
		if ns != "tenant-test" {
			t.Errorf("expected X-Vault-Namespace tenant-test, got %s", ns)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	nsClient := client.WithNamespace("tenant-test")

	err := nsClient.CreatePolicy(context.Background(), "test", "path \"*\" { capabilities = [\"read\"] }")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
