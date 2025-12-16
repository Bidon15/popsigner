package jsonrpc

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

// ContextKey is a type for context keys to avoid collisions.
type ContextKey string

const (
	// RPCRequestIDKey is the context key for the RPC request ID.
	RPCRequestIDKey ContextKey = "rpc_request_id"
)

// MethodHandler is the function signature for JSON-RPC method handlers.
type MethodHandler func(ctx context.Context, params json.RawMessage) (interface{}, *Error)

// Handler handles JSON-RPC 2.0 requests.
type Handler struct {
	methods map[string]MethodHandler
	mu      sync.RWMutex
	logger  *slog.Logger
}

// NewHandler creates a new JSON-RPC handler.
func NewHandler(logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		methods: make(map[string]MethodHandler),
		logger:  logger,
	}
}

// RegisterMethod registers a handler for a JSON-RPC method.
func (h *Handler) RegisterMethod(name string, handler MethodHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.methods[name] = handler
}

// HasMethod checks if a method is registered.
func (h *Handler) HasMethod(name string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.methods[name]
	return exists
}

// RegisteredMethods returns a list of all registered method names.
func (h *Handler) RegisteredMethods() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	methods := make([]string, 0, len(h.methods))
	for name := range h.methods {
		methods = append(methods, name)
	}
	return methods
}

// ServeHTTP implements http.Handler for JSON-RPC.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		h.writeError(w, nil, NewError(InvalidRequest, "Method not allowed", "use POST"))
		return
	}

	// Check content type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" && contentType != "" {
		h.writeError(w, nil, NewError(InvalidRequest, "Invalid content type", "use application/json"))
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, nil, ErrParseError("failed to read request body"))
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		h.writeError(w, nil, ErrInvalidRequest("empty request body"))
		return
	}

	// Detect batch vs single request
	if body[0] == '[' {
		h.handleBatch(w, r, body)
	} else {
		h.handleSingle(w, r, body)
	}
}

// handleSingle processes a single JSON-RPC request.
func (h *Handler) handleSingle(w http.ResponseWriter, r *http.Request, body []byte) {
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, nil, ErrParseError(err.Error()))
		return
	}

	resp := h.processRequest(r.Context(), &req)

	// Notifications (id=null) don't get responses
	if req.ID == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h.writeResponse(w, resp)
}

// handleBatch processes a batch of JSON-RPC requests.
func (h *Handler) handleBatch(w http.ResponseWriter, r *http.Request, body []byte) {
	var requests BatchRequest
	if err := json.Unmarshal(body, &requests); err != nil {
		h.writeError(w, nil, ErrParseError(err.Error()))
		return
	}

	if len(requests) == 0 {
		h.writeError(w, nil, ErrInvalidRequest("empty batch"))
		return
	}

	// Process requests concurrently
	responses := make(BatchResponse, 0, len(requests))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, req := range requests {
		wg.Add(1)
		go func(req Request) {
			defer wg.Done()
			resp := h.processRequest(r.Context(), &req)

			// Only include responses for non-notifications
			if req.ID != nil {
				mu.Lock()
				responses = append(responses, *resp)
				mu.Unlock()
			}
		}(req)
	}
	wg.Wait()

	// If all requests were notifications, return nothing
	if len(responses) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h.writeBatchResponse(w, responses)
}

// processRequest processes a single request and returns the response.
func (h *Handler) processRequest(ctx context.Context, req *Request) *Response {
	// Create response with same ID
	resp := &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	// Validate request
	if err := req.Validate(); err != nil {
		resp.Error = err
		return resp
	}

	// Find handler
	h.mu.RLock()
	handler, exists := h.methods[req.Method]
	h.mu.RUnlock()

	if !exists {
		resp.Error = ErrMethodNotFound(req.Method)
		return resp
	}

	// Add request ID to context for logging
	reqID := uuid.New().String()
	ctx = context.WithValue(ctx, RPCRequestIDKey, reqID)

	// Execute handler
	h.logger.Debug("Processing RPC request",
		slog.String("method", req.Method),
		slog.String("request_id", reqID),
	)

	result, err := handler(ctx, req.Params)
	if err != nil {
		h.logger.Warn("RPC request failed",
			slog.String("method", req.Method),
			slog.String("request_id", reqID),
			slog.Int("error_code", err.Code),
			slog.String("error_message", err.Message),
		)
		resp.Error = err
		return resp
	}

	resp.Result = result
	return resp
}

// writeResponse writes a single JSON-RPC response.
func (h *Handler) writeResponse(w http.ResponseWriter, resp *Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// writeBatchResponse writes a batch of JSON-RPC responses.
func (h *Handler) writeBatchResponse(w http.ResponseWriter, responses BatchResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responses)
}

// writeError writes an error response with null ID.
func (h *Handler) writeError(w http.ResponseWriter, id interface{}, err *Error) {
	resp := &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   err,
	}
	h.writeResponse(w, resp)
}

