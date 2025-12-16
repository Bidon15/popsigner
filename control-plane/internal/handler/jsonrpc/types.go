// Package jsonrpc provides JSON-RPC 2.0 handler implementation.
package jsonrpc

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 Error Codes
const (
	// Standard JSON-RPC errors
	ParseError     = -32700 // Invalid JSON
	InvalidRequest = -32600 // Not a valid request object
	MethodNotFound = -32601 // Method does not exist
	InvalidParams  = -32602 // Invalid method parameters
	InternalError  = -32603 // Internal JSON-RPC error

	// Ethereum-specific errors (-32000 to -32099)
	ServerError       = -32000 // Generic server error
	ResourceNotFound  = -32001 // Requested resource not found
	ResourceUnavail   = -32002 // Resource temporarily unavailable
	TransactionError  = -32010 // Transaction-related error
	SigningError      = -32020 // Signing operation failed
	UnauthorizedError = -32021 // Not authorized for this operation
	RateLimitError    = -32029 // Rate limit exceeded
)

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Error represents a JSON-RPC 2.0 error.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("RPC error %d: %s (data: %v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// NewError creates a new JSON-RPC error.
func NewError(code int, message string, data interface{}) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// Standard error constructors

// ErrParseError creates a parse error.
func ErrParseError(data interface{}) *Error {
	return NewError(ParseError, "Parse error", data)
}

// ErrInvalidRequest creates an invalid request error.
func ErrInvalidRequest(data interface{}) *Error {
	return NewError(InvalidRequest, "Invalid Request", data)
}

// ErrMethodNotFound creates a method not found error.
func ErrMethodNotFound(method string) *Error {
	return NewError(MethodNotFound, "Method not found", method)
}

// ErrInvalidParams creates an invalid params error.
func ErrInvalidParams(data interface{}) *Error {
	return NewError(InvalidParams, "Invalid params", data)
}

// ErrInternal creates an internal error.
func ErrInternal(data interface{}) *Error {
	return NewError(InternalError, "Internal error", data)
}

// ErrUnauthorized creates an unauthorized error.
func ErrUnauthorized(data interface{}) *Error {
	return NewError(UnauthorizedError, "Unauthorized", data)
}

// ErrSigningFailed creates a signing failed error.
func ErrSigningFailed(data interface{}) *Error {
	return NewError(SigningError, "Signing failed", data)
}

// ErrRateLimit creates a rate limit error.
func ErrRateLimit(data interface{}) *Error {
	return NewError(RateLimitError, "Rate limit exceeded", data)
}

// ErrResourceNotFound creates a resource not found error.
func ErrResourceNotFound(resource string) *Error {
	return NewError(ResourceNotFound, fmt.Sprintf("%s not found", resource), nil)
}

// ErrTransactionError creates a transaction error.
func ErrTransactionError(data interface{}) *Error {
	return NewError(TransactionError, "Transaction error", data)
}

// BatchRequest is a slice of requests for batch processing.
type BatchRequest []Request

// BatchResponse is a slice of responses for batch processing.
type BatchResponse []Response

// Validate checks if the request is valid JSON-RPC 2.0.
func (r *Request) Validate() *Error {
	if r.JSONRPC != "2.0" {
		return ErrInvalidRequest("jsonrpc must be '2.0'")
	}
	if r.Method == "" {
		return ErrInvalidRequest("method is required")
	}
	// ID can be string, number, or null (for notifications)
	return nil
}

