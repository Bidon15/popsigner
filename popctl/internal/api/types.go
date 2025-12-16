package api

import (
	"time"

	"github.com/google/uuid"
)

// Key represents a cryptographic key.
type Key struct {
	ID          uuid.UUID         `json:"id"`
	NamespaceID uuid.UUID         `json:"namespace_id"`
	Name        string            `json:"name"`
	PublicKey   string            `json:"public_key"`
	Address     string            `json:"address"`
	Algorithm   string            `json:"algorithm"`
	Exportable  bool              `json:"exportable"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Version     int               `json:"version"`
	CreatedAt   time.Time         `json:"created_at"`
}

// CreateKeyRequest is the request for creating a key.
type CreateKeyRequest struct {
	Name        string            `json:"name"`
	NamespaceID uuid.UUID         `json:"namespace_id"`
	Algorithm   string            `json:"algorithm,omitempty"`
	Exportable  bool              `json:"exportable,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CreateBatchRequest creates multiple keys at once.
type CreateBatchRequest struct {
	Prefix      string    `json:"prefix"`
	Count       int       `json:"count"`
	NamespaceID uuid.UUID `json:"namespace_id"`
	Exportable  bool      `json:"exportable,omitempty"`
}

// ImportKeyRequest is the request for importing a key.
type ImportKeyRequest struct {
	Name        string    `json:"name"`
	NamespaceID uuid.UUID `json:"namespace_id"`
	PrivateKey  string    `json:"private_key"`
	Exportable  bool      `json:"exportable,omitempty"`
}

// ExportKeyResponse is the response from exporting a key.
type ExportKeyResponse struct {
	PrivateKey string `json:"private_key"`
	Warning    string `json:"warning"`
}

// Organization represents an organization.
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Namespace represents a key namespace.
type Namespace struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateNamespaceRequest is the request for creating a namespace.
type CreateNamespaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SignRequest is a single sign request.
type SignRequest struct {
	KeyID     uuid.UUID `json:"key_id"`
	Data      string    `json:"data"` // base64-encoded
	Prehashed bool      `json:"prehashed,omitempty"`
}

// SignResponse is the response from a sign operation.
type SignResponse struct {
	KeyID      uuid.UUID `json:"key_id"`
	Signature  string    `json:"signature"` // base64-encoded
	PublicKey  string    `json:"public_key"`
	KeyVersion int       `json:"key_version"`
}

// BatchSignRequest is a batch sign request.
type BatchSignRequest struct {
	Requests []SignRequest `json:"requests"`
}

// BatchSignResult is a single result from batch signing.
type BatchSignResult struct {
	KeyID      uuid.UUID `json:"key_id"`
	Signature  string    `json:"signature,omitempty"`
	PublicKey  string    `json:"public_key,omitempty"`
	KeyVersion int       `json:"key_version,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// BatchSignResponse is the response from batch signing.
type BatchSignResponse struct {
	Signatures []BatchSignResult `json:"signatures"`
	Count      int               `json:"count"`
}

// API response wrappers

type keyResponse struct {
	Data Key `json:"data"`
}

type keysResponse struct {
	Data []Key `json:"data"`
}

type batchKeysResponse struct {
	Data struct {
		Keys  []Key `json:"keys"`
		Count int   `json:"count"`
	} `json:"data"`
}

type exportResponse struct {
	Data ExportKeyResponse `json:"data"`
}

type orgResponse struct {
	Data Organization `json:"data"`
}

type orgsResponse struct {
	Data []Organization `json:"data"`
}

type namespaceResponse struct {
	Data Namespace `json:"data"`
}

type namespacesResponse struct {
	Data []Namespace `json:"data"`
}

type signResponse struct {
	Data SignResponse `json:"data"`
}

type batchSignResponseWrapper struct {
	Data BatchSignResponse `json:"data"`
}

