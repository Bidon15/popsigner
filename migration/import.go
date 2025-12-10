package migration

import "context"

// TODO(05A): Implement Import functions

// Import migrates a key from local to OpenBao.
func Import(ctx context.Context, cfg ImportConfig) (*ImportResult, error) {
	panic("TODO(05A): implement Import")
}

// BatchImport imports multiple keys.
func BatchImport(ctx context.Context, cfg BatchImportConfig) (*BatchImportResult, error) {
	panic("TODO(05A): implement BatchImport")
}

