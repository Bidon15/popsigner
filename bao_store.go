package banhbaoring

// TODO(02A, 02B): Implement BaoStore

// BaoStore manages local key metadata.
type BaoStore struct {
	// TODO: Add fields
}

// NewBaoStore creates or opens a store.
func NewBaoStore(path string) (*BaoStore, error) {
	panic("TODO(02B): implement NewBaoStore")
}

// Save stores key metadata.
func (s *BaoStore) Save(meta *KeyMetadata) error {
	panic("TODO(02A): implement Save")
}

// Get retrieves metadata by UID.
func (s *BaoStore) Get(uid string) (*KeyMetadata, error) {
	panic("TODO(02A): implement Get")
}

// GetByAddress retrieves metadata by address.
func (s *BaoStore) GetByAddress(address string) (*KeyMetadata, error) {
	panic("TODO(02A): implement GetByAddress")
}

// List returns all metadata.
func (s *BaoStore) List() ([]*KeyMetadata, error) {
	panic("TODO(02A): implement List")
}

// Delete removes metadata.
func (s *BaoStore) Delete(uid string) error {
	panic("TODO(02A): implement Delete")
}

// Rename changes the UID.
func (s *BaoStore) Rename(oldUID, newUID string) error {
	panic("TODO(02A): implement Rename")
}

// Has checks existence.
func (s *BaoStore) Has(uid string) bool {
	panic("TODO(02A): implement Has")
}

// Count returns key count.
func (s *BaoStore) Count() int {
	panic("TODO(02A): implement Count")
}

// Sync flushes to disk.
func (s *BaoStore) Sync() error {
	panic("TODO(02B): implement Sync")
}

// Close releases resources.
func (s *BaoStore) Close() error {
	panic("TODO(02B): implement Close")
}

