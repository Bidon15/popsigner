package migration

import "context"

// TODO(05B): Implement Export functions

// Export migrates a key from OpenBao to local.
func Export(ctx context.Context, cfg ExportConfig) (*ExportResult, error) {
	panic("TODO(05B): implement Export")
}

// SecurityWarning returns export warning text.
func SecurityWarning(keyName, address, destPath string) string {
	panic("TODO(05B): implement SecurityWarning")
}

