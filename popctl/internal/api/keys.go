package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ListKeys returns all keys, optionally filtered by namespace.
func (c *Client) ListKeys(ctx context.Context, namespaceID *uuid.UUID) ([]Key, error) {
	path := "/v1/keys"
	if namespaceID != nil {
		path = fmt.Sprintf("/v1/keys?namespace_id=%s", namespaceID)
	}

	var resp keysResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetKey retrieves a key by ID.
func (c *Client) GetKey(ctx context.Context, keyID uuid.UUID) (*Key, error) {
	var resp keyResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/keys/%s", keyID), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CreateKey creates a new key.
func (c *Client) CreateKey(ctx context.Context, req CreateKeyRequest) (*Key, error) {
	body := map[string]interface{}{
		"name":         req.Name,
		"namespace_id": req.NamespaceID.String(),
		"exportable":   req.Exportable,
	}
	if req.Algorithm != "" {
		body["algorithm"] = req.Algorithm
	}
	if len(req.Metadata) > 0 {
		body["metadata"] = req.Metadata
	}

	var resp keyResponse
	if err := c.Post(ctx, "/v1/keys", body, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CreateKeysBatch creates multiple keys in parallel.
func (c *Client) CreateKeysBatch(ctx context.Context, req CreateBatchRequest) ([]Key, error) {
	body := map[string]interface{}{
		"prefix":       req.Prefix,
		"count":        req.Count,
		"namespace_id": req.NamespaceID.String(),
		"exportable":   req.Exportable,
	}

	var resp batchKeysResponse
	if err := c.Post(ctx, "/v1/keys/batch", body, &resp); err != nil {
		return nil, err
	}
	return resp.Data.Keys, nil
}

// DeleteKey deletes a key.
func (c *Client) DeleteKey(ctx context.Context, keyID uuid.UUID) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/keys/%s", keyID))
}

// ImportKey imports a private key.
func (c *Client) ImportKey(ctx context.Context, req ImportKeyRequest) (*Key, error) {
	body := map[string]interface{}{
		"name":         req.Name,
		"namespace_id": req.NamespaceID.String(),
		"private_key":  req.PrivateKey,
		"exportable":   req.Exportable,
	}

	var resp keyResponse
	if err := c.Post(ctx, "/v1/keys/import", body, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ExportKey exports a key's private key.
func (c *Client) ExportKey(ctx context.Context, keyID uuid.UUID) (*ExportKeyResponse, error) {
	var resp exportResponse
	if err := c.Post(ctx, fmt.Sprintf("/v1/keys/%s/export", keyID), nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

