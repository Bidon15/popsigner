package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_SingleRequest(t *testing.T) {
	h := NewHandler(nil)
	h.RegisterMethod("test_echo", func(ctx context.Context, params json.RawMessage) (interface{}, *Error) {
		var args []string
		json.Unmarshal(params, &args)
		return args, nil
	})

	t.Run("valid request", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","method":"test_echo","params":["hello"],"id":1}`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))
		httpReq.Header.Set("Content-Type", "application/json")

		h.ServeHTTP(rec, httpReq)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Equal(t, float64(1), resp.ID)
		assert.Nil(t, resp.Error)
		assert.Equal(t, []interface{}{"hello"}, resp.Result)
	})

	t.Run("method not found", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","method":"unknown","params":[],"id":1}`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		var resp Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, MethodNotFound, resp.Error.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		req := `{invalid json}`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		var resp Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, ParseError, resp.Error.Code)
	})

	t.Run("invalid jsonrpc version", func(t *testing.T) {
		req := `{"jsonrpc":"1.0","method":"test_echo","params":[],"id":1}`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		var resp Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, InvalidRequest, resp.Error.Code)
	})

	t.Run("empty method", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","method":"","params":[],"id":1}`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		var resp Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, InvalidRequest, resp.Error.Code)
	})

	t.Run("string id", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","method":"test_echo","params":["test"],"id":"abc123"}`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		var resp Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.Equal(t, "abc123", resp.ID)
		assert.Nil(t, resp.Error)
	})

	t.Run("notification no response", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","method":"test_echo","params":["test"]}`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Body.Bytes())
	})
}

func TestHandler_BatchRequest(t *testing.T) {
	h := NewHandler(nil)
	h.RegisterMethod("test_add", func(ctx context.Context, params json.RawMessage) (interface{}, *Error) {
		var args []int
		json.Unmarshal(params, &args)
		sum := 0
		for _, v := range args {
			sum += v
		}
		return sum, nil
	})

	t.Run("valid batch", func(t *testing.T) {
		req := `[
			{"jsonrpc":"2.0","method":"test_add","params":[1,2],"id":1},
			{"jsonrpc":"2.0","method":"test_add","params":[3,4],"id":2}
		]`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		var responses BatchResponse
		err := json.Unmarshal(rec.Body.Bytes(), &responses)
		require.NoError(t, err)
		assert.Len(t, responses, 2)
	})

	t.Run("notification in batch", func(t *testing.T) {
		req := `[
			{"jsonrpc":"2.0","method":"test_add","params":[1,2],"id":1},
			{"jsonrpc":"2.0","method":"test_add","params":[3,4]}
		]`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		var responses BatchResponse
		json.Unmarshal(rec.Body.Bytes(), &responses)
		// Only one response (the non-notification)
		assert.Len(t, responses, 1)
	})

	t.Run("all notifications", func(t *testing.T) {
		req := `[
			{"jsonrpc":"2.0","method":"test_add","params":[1,2]},
			{"jsonrpc":"2.0","method":"test_add","params":[3,4]}
		]`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("empty batch", func(t *testing.T) {
		req := `[]`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		var resp Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, InvalidRequest, resp.Error.Code)
	})

	t.Run("mixed success and error", func(t *testing.T) {
		req := `[
			{"jsonrpc":"2.0","method":"test_add","params":[1,2],"id":1},
			{"jsonrpc":"2.0","method":"unknown","params":[],"id":2}
		]`
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

		h.ServeHTTP(rec, httpReq)

		var responses BatchResponse
		err := json.Unmarshal(rec.Body.Bytes(), &responses)
		require.NoError(t, err)
		assert.Len(t, responses, 2)

		// Find the error response
		var hasError bool
		for _, resp := range responses {
			if resp.Error != nil {
				hasError = true
				assert.Equal(t, MethodNotFound, resp.Error.Code)
			}
		}
		assert.True(t, hasError)
	})
}

func TestHandler_HTTPMethods(t *testing.T) {
	h := NewHandler(nil)

	t.Run("GET not allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("GET", "/", nil)

		h.ServeHTTP(rec, httpReq)

		var resp Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, InvalidRequest, resp.Error.Code)
	})

	t.Run("empty body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(""))

		h.ServeHTTP(rec, httpReq)

		var resp Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, InvalidRequest, resp.Error.Code)
	})
}

func TestHandler_MethodReturnsError(t *testing.T) {
	h := NewHandler(nil)
	h.RegisterMethod("test_fail", func(ctx context.Context, params json.RawMessage) (interface{}, *Error) {
		return nil, ErrSigningFailed("test error")
	})

	req := `{"jsonrpc":"2.0","method":"test_fail","params":[],"id":1}`
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest("POST", "/", bytes.NewBufferString(req))

	h.ServeHTTP(rec, httpReq)

	var resp Response
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, SigningError, resp.Error.Code)
}

func TestHandler_HasMethod(t *testing.T) {
	h := NewHandler(nil)
	h.RegisterMethod("test_method", func(ctx context.Context, params json.RawMessage) (interface{}, *Error) {
		return nil, nil
	})

	assert.True(t, h.HasMethod("test_method"))
	assert.False(t, h.HasMethod("unknown_method"))
}

func TestError_ErrorInterface(t *testing.T) {
	t.Run("with data", func(t *testing.T) {
		err := NewError(ParseError, "Parse error", "invalid character")
		assert.Contains(t, err.Error(), "RPC error -32700")
		assert.Contains(t, err.Error(), "Parse error")
		assert.Contains(t, err.Error(), "invalid character")
	})

	t.Run("without data", func(t *testing.T) {
		err := NewError(MethodNotFound, "Method not found", nil)
		assert.Contains(t, err.Error(), "RPC error -32601")
		assert.Contains(t, err.Error(), "Method not found")
		assert.NotContains(t, err.Error(), "data:")
	})
}

func TestRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			Method:  "test",
			ID:      1,
		}
		assert.Nil(t, req.Validate())
	})

	t.Run("invalid version", func(t *testing.T) {
		req := &Request{
			JSONRPC: "1.0",
			Method:  "test",
			ID:      1,
		}
		err := req.Validate()
		assert.NotNil(t, err)
		assert.Equal(t, InvalidRequest, err.Code)
	})

	t.Run("empty method", func(t *testing.T) {
		req := &Request{
			JSONRPC: "2.0",
			Method:  "",
			ID:      1,
		}
		err := req.Validate()
		assert.NotNil(t, err)
		assert.Equal(t, InvalidRequest, err.Code)
	})
}

