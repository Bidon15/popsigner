package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ListOrganizations returns all organizations the user has access to.
func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var resp orgsResponse
	if err := c.Get(ctx, "/v1/organizations", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetOrganization retrieves an organization by ID.
func (c *Client) GetOrganization(ctx context.Context, orgID uuid.UUID) (*Organization, error) {
	var resp orgResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/organizations/%s", orgID), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ListNamespaces returns all namespaces in an organization.
func (c *Client) ListNamespaces(ctx context.Context, orgID uuid.UUID) ([]Namespace, error) {
	var resp namespacesResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/organizations/%s/namespaces", orgID), &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetNamespace retrieves a namespace by ID.
func (c *Client) GetNamespace(ctx context.Context, orgID, namespaceID uuid.UUID) (*Namespace, error) {
	var resp namespaceResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/organizations/%s/namespaces/%s", orgID, namespaceID), &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CreateNamespace creates a new namespace.
func (c *Client) CreateNamespace(ctx context.Context, orgID uuid.UUID, req CreateNamespaceRequest) (*Namespace, error) {
	var resp namespaceResponse
	if err := c.Post(ctx, fmt.Sprintf("/v1/organizations/%s/namespaces", orgID), req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeleteNamespace deletes a namespace.
func (c *Client) DeleteNamespace(ctx context.Context, orgID, namespaceID uuid.UUID) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/organizations/%s/namespaces/%s", orgID, namespaceID))
}

