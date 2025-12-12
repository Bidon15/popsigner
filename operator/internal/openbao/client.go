package openbao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for OpenBao API
type Client struct {
	addr      string
	token     string
	namespace string
	client    *http.Client
}

// NewClient creates a new OpenBao client
func NewClient(addr, token string) *Client {
	return &Client{
		addr:  addr,
		token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithNamespace returns a new client scoped to the given namespace
func (c *Client) WithNamespace(namespace string) *Client {
	return &Client{
		addr:      c.addr,
		token:     c.token,
		namespace: namespace,
		client:    c.client,
	}
}

// CreateNamespace creates a new namespace in OpenBao
func (c *Client) CreateNamespace(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/v1/sys/namespaces/%s", c.addr, name), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// 200 OK or 204 No Content are success
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to create namespace (status %d): %s", resp.StatusCode, body)
}

// DeleteNamespace deletes a namespace from OpenBao
func (c *Client) DeleteNamespace(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/v1/sys/namespaces/%s", c.addr, name), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to delete namespace (status %d): %s", resp.StatusCode, body)
}

// NamespaceExists checks if a namespace exists
func (c *Client) NamespaceExists(ctx context.Context, name string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/v1/sys/namespaces/%s", c.addr, name), nil)
	if err != nil {
		return false, fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("failed to check namespace (status %d): %s", resp.StatusCode, body)
}

// CreatePolicy creates or updates an ACL policy
func (c *Client) CreatePolicy(ctx context.Context, name, policy string) error {
	payload := map[string]string{"policy": policy}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling policy: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/v1/sys/policies/acl/%s", c.addr, name),
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to create policy (status %d): %s", resp.StatusCode, respBody)
}

// DeletePolicy deletes an ACL policy
func (c *Client) DeletePolicy(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/v1/sys/policies/acl/%s", c.addr, name), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to delete policy (status %d): %s", resp.StatusCode, body)
}

// EnableSecretsEngine enables a secrets engine at a given path
func (c *Client) EnableSecretsEngine(ctx context.Context, path, engineType string) error {
	payload := map[string]interface{}{
		"type": engineType,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/v1/sys/mounts/%s", c.addr, path),
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to enable secrets engine (status %d): %s", resp.StatusCode, respBody)
}

// HealthCheck checks if OpenBao is healthy and initialized
func (c *Client) HealthCheck(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/v1/sys/health", c.addr), nil)
	if err != nil {
		return false, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// 200 = initialized, unsealed, active
	// 429 = unsealed, standby
	// 472 = data recovery mode replication secondary
	// 473 = performance standby
	// 501 = not initialized
	// 503 = sealed
	switch resp.StatusCode {
	case http.StatusOK, 429, 473:
		return true, nil
	default:
		return false, nil
	}
}

// setHeaders sets the required headers for OpenBao API requests
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("X-Vault-Token", c.token)
	if c.namespace != "" {
		req.Header.Set("X-Vault-Namespace", c.namespace)
	}
}

// TokenInfo holds information about the current token
type TokenInfo struct {
	Accessor   string   `json:"accessor"`
	Policies   []string `json:"policies"`
	Renewable  bool     `json:"renewable"`
	TTL        int      `json:"ttl"`
	EntityID   string   `json:"entity_id"`
	ExpireTime string   `json:"expire_time"`
}

// LookupSelf returns information about the current token
func (c *Client) LookupSelf(ctx context.Context) (*TokenInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/v1/auth/token/lookup-self", c.addr), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to lookup token (status %d): %s", resp.StatusCode, body)
	}

	var result struct {
		Data TokenInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result.Data, nil
}
