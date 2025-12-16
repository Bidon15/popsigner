package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractAddressFromRPCRequest(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "eth_signTransaction extracts from address",
			body:     `{"jsonrpc":"2.0","method":"eth_signTransaction","params":[{"from":"0x742d35Cc6634C0532925a3b844Bc454e4438f44e","to":"0x123","gas":"0x5208"}],"id":1}`,
			expected: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		},
		{
			name:     "eth_sign extracts first param",
			body:     `{"jsonrpc":"2.0","method":"eth_sign","params":["0x742d35Cc6634C0532925a3b844Bc454e4438f44e","0x68656c6c6f"],"id":1}`,
			expected: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		},
		{
			name:     "personal_sign extracts second param",
			body:     `{"jsonrpc":"2.0","method":"personal_sign","params":["0x68656c6c6f","0x742d35Cc6634C0532925a3b844Bc454e4438f44e"],"id":1}`,
			expected: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e",
		},
		{
			name:     "eth_accounts returns empty",
			body:     `{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}`,
			expected: "",
		},
		{
			name:     "unknown method returns empty",
			body:     `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`,
			expected: "",
		},
		{
			name:     "invalid JSON returns empty",
			body:     `invalid json`,
			expected: "",
		},
		{
			name:     "batch request extracts from first request",
			body:     `[{"jsonrpc":"2.0","method":"eth_sign","params":["0xABC123","0x68656c6c6f"],"id":1}]`,
			expected: "0xABC123",
		},
		{
			name:     "eth_signTransaction with missing from returns empty",
			body:     `{"jsonrpc":"2.0","method":"eth_signTransaction","params":[{"to":"0x123","gas":"0x5208"}],"id":1}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAddressFromRPCRequest([]byte(tt.body))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractAddressFromSingleRequest(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "extracts from eth_signTransaction",
			body:     `{"method":"eth_signTransaction","params":[{"from":"0xABC"}]}`,
			expected: "0xABC",
		},
		{
			name:     "extracts from eth_sign",
			body:     `{"method":"eth_sign","params":["0xDEF","0x123"]}`,
			expected: "0xDEF",
		},
		{
			name:     "extracts from personal_sign",
			body:     `{"method":"personal_sign","params":["0x123","0xGHI"]}`,
			expected: "0xGHI",
		},
		{
			name:     "returns empty for eth_accounts",
			body:     `{"method":"eth_accounts","params":[]}`,
			expected: "",
		},
		{
			name:     "returns empty for empty params in eth_sign",
			body:     `{"method":"eth_sign","params":[]}`,
			expected: "",
		},
		{
			name:     "returns empty for single param in personal_sign",
			body:     `{"method":"personal_sign","params":["0x123"]}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAddressFromSingleRequest([]byte(tt.body))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriteRPCError(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		message        string
		expectedStatus int
	}{
		{
			name:           "rate limit error returns 429",
			code:           -32029,
			message:        "Rate limit exceeded",
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:           "parse error returns 400",
			code:           -32700,
			message:        "Parse error",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "generic error returns 400",
			code:           -32600,
			message:        "Invalid request",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeRPCError(w, tt.code, tt.message)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var resp struct {
				JSONRPC string `json:"jsonrpc"`
				Error   struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
				ID interface{} `json:"id"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, "2.0", resp.JSONRPC)
			assert.Equal(t, tt.code, resp.Error.Code)
			assert.Equal(t, tt.message, resp.Error.Message)
			assert.Nil(t, resp.ID)
		})
	}
}

func TestRPCRateLimit_EmptyBody(t *testing.T) {
	// Create a handler that tracks if it was called
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware (with nil Redis, will fail open)
	middleware := RPCRateLimit(nil, RPCRateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
	})

	handler := middleware(next)

	// Test with empty body
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(""))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.True(t, called, "Next handler should be called for empty body")
}

func TestRPCRateLimit_NoAddressAllowsRequest(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := RPCRateLimit(nil, RPCRateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
	})

	handler := middleware(next)

	// Test with eth_accounts which has no address
	body := `{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.True(t, called, "Next handler should be called when no address is found")
}

func TestRPCRateLimit_BodyPreserved(t *testing.T) {
	var receivedBody []byte
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	middleware := RPCRateLimit(nil, RPCRateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
	})

	handler := middleware(next)

	originalBody := `{"jsonrpc":"2.0","method":"eth_sign","params":["0xABC","0x123"],"id":1}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(originalBody))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, originalBody, string(receivedBody), "Body should be preserved for downstream handler")
}

