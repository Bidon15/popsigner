// Package banhbaoring provides a Cosmos SDK keyring implementation
// backed by OpenBao for secure secp256k1 signing.
package banhbaoring

import (
	"crypto/tls"
	"time"
)

// TODO(01A): Implement all types below

// Algorithm constants
const (
	AlgorithmSecp256k1   = "secp256k1"
	DefaultSecp256k1Path = "secp256k1"
	DefaultHTTPTimeout   = 30 * time.Second
	DefaultStoreVersion  = 1
)

// Source constants
const (
	SourceGenerated = "generated"
	SourceImported  = "imported"
	SourceSynced    = "synced"
)

// Config holds configuration for BaoKeyring.
type Config struct {
	BaoAddr       string
	BaoToken      string
	BaoNamespace  string
	Secp256k1Path string
	StorePath     string
	HTTPTimeout   time.Duration
	TLSConfig     *tls.Config
	SkipTLSVerify bool
}

// KeyMetadata contains locally stored key information.
type KeyMetadata struct {
	UID         string    `json:"uid"`
	Name        string    `json:"name"`
	PubKeyBytes []byte    `json:"pub_key"`
	PubKeyType  string    `json:"pub_key_type"`
	Address     string    `json:"address"`
	BaoKeyPath  string    `json:"bao_key_path"`
	Algorithm   string    `json:"algorithm"`
	Exportable  bool      `json:"exportable"`
	CreatedAt   time.Time `json:"created_at"`
	Source      string    `json:"source"`
}

// KeyInfo represents public key information from OpenBao.
type KeyInfo struct {
	Name       string    `json:"name"`
	PublicKey  string    `json:"public_key"`
	Address    string    `json:"address"`
	Exportable bool      `json:"exportable"`
	CreatedAt  time.Time `json:"created_at"`
}

// KeyOptions configures key creation.
type KeyOptions struct {
	Exportable bool
}

// SignRequest for OpenBao signing.
type SignRequest struct {
	Input        string `json:"input"`
	Prehashed    bool   `json:"prehashed"`
	HashAlgo     string `json:"hash_algorithm,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// SignResponse from OpenBao signing.
type SignResponse struct {
	Signature  string `json:"signature"`
	PublicKey  string `json:"public_key"`
	KeyVersion int    `json:"key_version"`
}

// StoreData is the persisted store format.
type StoreData struct {
	Version int                     `json:"version"`
	Keys    map[string]*KeyMetadata `json:"keys"`
}

