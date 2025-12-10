package banhbaoring

import "context"

// TODO(01C): Implement BaoClient

// BaoClient handles HTTP communication with OpenBao.
type BaoClient struct {
	// TODO: Add fields
}

// NewBaoClient creates a new client.
func NewBaoClient(cfg Config) (*BaoClient, error) {
	panic("TODO(01C): implement NewBaoClient")
}

// CreateKey creates a new secp256k1 key.
func (c *BaoClient) CreateKey(ctx context.Context, name string, opts KeyOptions) (*KeyInfo, error) {
	panic("TODO(01C): implement CreateKey")
}

// GetKey retrieves key info.
func (c *BaoClient) GetKey(ctx context.Context, name string) (*KeyInfo, error) {
	panic("TODO(01C): implement GetKey")
}

// ListKeys lists all keys.
func (c *BaoClient) ListKeys(ctx context.Context) ([]string, error) {
	panic("TODO(01C): implement ListKeys")
}

// DeleteKey deletes a key.
func (c *BaoClient) DeleteKey(ctx context.Context, name string) error {
	panic("TODO(01C): implement DeleteKey")
}

// Sign signs data.
func (c *BaoClient) Sign(ctx context.Context, keyName string, data []byte, prehashed bool) ([]byte, error) {
	panic("TODO(01C): implement Sign")
}

// Health checks OpenBao status.
func (c *BaoClient) Health(ctx context.Context) error {
	panic("TODO(01C): implement Health")
}

