package api

import (
	"context"
	"fmt"
)

// CreateDeployment creates a new chain deployment.
func (c *Client) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) (*Deployment, error) {
	var resp deploymentResponse
	if err := c.Post(ctx, "/v1/deployments", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetDeployment retrieves a deployment by ID.
func (c *Client) GetDeployment(ctx context.Context, id string) (*Deployment, error) {
	var resp deploymentResponse
	if err := c.Get(ctx, "/v1/deployments/"+id, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ListDeployments lists all deployments, optionally filtered by status.
func (c *Client) ListDeployments(ctx context.Context, status string) ([]Deployment, error) {
	path := "/v1/deployments"
	if status != "" {
		path = fmt.Sprintf("/v1/deployments?status=%s", status)
	}
	var resp deploymentsResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// StartDeployment starts a pending or paused deployment.
func (c *Client) StartDeployment(ctx context.Context, id string) error {
	var resp startDeploymentResponse
	if err := c.Post(ctx, "/v1/deployments/"+id+"/start", nil, &resp); err != nil {
		return err
	}
	return nil
}

// GetArtifacts retrieves all artifacts for a deployment.
func (c *Client) GetArtifacts(ctx context.Context, deploymentID string) ([]Artifact, error) {
	var resp artifactsResponse
	if err := c.Get(ctx, "/v1/deployments/"+deploymentID+"/artifacts", &resp); err != nil {
		return nil, err
	}
	return resp.Data.Artifacts, nil
}

// GetArtifact retrieves a specific artifact by type.
func (c *Client) GetArtifact(ctx context.Context, deploymentID, artifactType string) (*Artifact, error) {
	var resp artifactResponse
	if err := c.Get(ctx, "/v1/deployments/"+deploymentID+"/artifacts/"+artifactType, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetTransactions retrieves all transactions for a deployment.
func (c *Client) GetTransactions(ctx context.Context, deploymentID string) ([]Transaction, error) {
	var resp transactionsResponse
	if err := c.Get(ctx, "/v1/deployments/"+deploymentID+"/transactions", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

