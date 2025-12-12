package migration

import (
	popsigner "github.com/Bidon15/popsigner"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
)

// TODO(05A): Implement migration types

// ImportConfig configures key import.
type ImportConfig struct {
	SourceKeyring     keyring.Keyring
	DestKeyring       *popsigner.BaoKeyring
	KeyName           string
	NewKeyName        string
	DeleteAfterImport bool
	Exportable        bool
	VerifyAfterImport bool
}

// ImportResult contains import result.
type ImportResult struct {
	KeyName    string
	Address    string
	PubKey     []byte
	BaoKeyPath string
	Verified   bool
}

// ExportConfig configures key export.
type ExportConfig struct {
	SourceKeyring     *popsigner.BaoKeyring
	DestKeyring       keyring.Keyring
	KeyName           string
	NewKeyName        string
	DeleteAfterExport bool
	VerifyAfterExport bool
	Confirmed         bool
}

// ExportResult contains export result.
type ExportResult struct {
	KeyName  string
	Address  string
	DestPath string
	Verified bool
}

// BatchImportConfig for multiple keys.
type BatchImportConfig struct {
	SourceKeyring     keyring.Keyring
	DestKeyring       *popsigner.BaoKeyring
	KeyNames          []string
	DeleteAfterImport bool
	Exportable        bool
	VerifyAfterImport bool
}

// BatchImportResult contains batch results.
type BatchImportResult struct {
	Successful []ImportResult
	Failed     []ImportError
}

// ImportError for failed imports.
type ImportError struct {
	KeyName string
	Error   error
}

