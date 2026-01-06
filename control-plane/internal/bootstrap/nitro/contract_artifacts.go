// Package nitro provides Nitro chain deployment infrastructure using pre-built contracts.
package nitro

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ArtifactVersion is the current version of the nitro-contracts artifacts.
// Update this when uploading new artifacts to S3.
const ArtifactVersion = "v3.2.0-beta.0"

// ArtifactBaseURL is the base URL for Nitro contract artifact storage.
const ArtifactBaseURL = "https://nitro-contracts.s3.nl-ams.scw.cloud"

// ContractArtifactURL is the URL to nitro-contracts v3.2.0-beta.0 artifacts.
// v3.2 includes CUSTOM_DA_MESSAGE_HEADER_FLAG (0x01) for External DA support.
var ContractArtifactURL = ArtifactBaseURL + "/" + ArtifactVersion + ".zip"

// ContractArtifact represents a compiled Solidity contract with ABI and bytecode.
type ContractArtifact struct {
	ABI              json.RawMessage `json:"abi"`
	Bytecode         Bytecode        `json:"bytecode"`
	DeployedBytecode Bytecode        `json:"deployedBytecode,omitempty"`
}

// Bytecode contains the contract bytecode.
type Bytecode struct {
	Object string `json:"object"`
}

// GetBytecode returns the bytecode as a hex string (with 0x prefix).
func (a *ContractArtifact) GetBytecode() string {
	return a.Bytecode.Object
}

// NitroArtifacts contains all compiled Nitro contract artifacts needed for deployment.
type NitroArtifacts struct {
	// Core deployment contracts
	RollupCreator *ContractArtifact
	BridgeCreator *ContractArtifact

	// Rollup infrastructure
	SequencerInbox   *ContractArtifact
	Bridge           *ContractArtifact
	Inbox            *ContractArtifact
	Outbox           *ContractArtifact
	RollupCore       *ContractArtifact
	RollupAdminLogic *ContractArtifact
	RollupUserLogic  *ContractArtifact

	// Challenge/Fraud proof contracts (BOLD protocol in v3.2+)
	// Note: EdgeChallengeManager replaces the old ChallengeManager in BOLD
	EdgeChallengeManager *ContractArtifact
	OneStepProofEntry    *ContractArtifact
	OneStepProver0       *ContractArtifact
	OneStepProverMemory  *ContractArtifact
	OneStepProverMath    *ContractArtifact
	OneStepProverHostIo  *ContractArtifact

	// Upgrade infrastructure
	UpgradeExecutor *ContractArtifact

	// Version metadata
	Version   string
	LoadedAt  time.Time
	SourceURL string
}

// ContractArtifactDownloader handles downloading and parsing Nitro contract artifacts from S3.
type ContractArtifactDownloader struct {
	cacheDir string
	mu       sync.Mutex
}

// NewContractArtifactDownloader creates a new artifact downloader.
func NewContractArtifactDownloader(cacheDir string) *ContractArtifactDownloader {
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}
	return &ContractArtifactDownloader{
		cacheDir: cacheDir,
	}
}

// Download downloads and parses Nitro contract artifacts from S3.
// Returns a NitroArtifacts struct with all contracts loaded.
func (d *ContractArtifactDownloader) Download(ctx context.Context, url string) (*NitroArtifacts, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Create cache directory if needed
	if err := os.MkdirAll(d.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	// Download the zip file
	zipPath := filepath.Join(d.cacheDir, fmt.Sprintf("nitro-contracts-%d.zip", time.Now().UnixNano()))
	if err := d.downloadFile(ctx, url, zipPath); err != nil {
		return nil, fmt.Errorf("download artifacts: %w", err)
	}
	defer os.Remove(zipPath)

	// Extract and parse artifacts
	artifacts, err := d.parseZip(zipPath)
	if err != nil {
		return nil, fmt.Errorf("parse artifacts: %w", err)
	}

	artifacts.Version = ArtifactVersion
	artifacts.LoadedAt = time.Now()
	artifacts.SourceURL = url

	return artifacts, nil
}

// DownloadDefault downloads artifacts from the default URL.
func (d *ContractArtifactDownloader) DownloadDefault(ctx context.Context) (*NitroArtifacts, error) {
	return d.Download(ctx, ContractArtifactURL)
}

// downloadFile downloads a file from URL to the given path.
func (d *ContractArtifactDownloader) downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d from %s", resp.StatusCode, url)
	}

	// Write to temp file first, then rename for atomicity
	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("write file: %w", err)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// parseZip extracts and parses contract artifacts from a zip file.
func (d *ContractArtifactDownloader) parseZip(zipPath string) (*NitroArtifacts, error) {
	// Open the zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	// Map to store loaded artifacts by contract name
	loaded := make(map[string]*ContractArtifact)

	// List of contracts we need (BOLD protocol for v3.2+)
	requiredContracts := []string{
		"RollupCreator",
		"BridgeCreator",
		"SequencerInbox",
		"Bridge",
		"Inbox",
		"Outbox",
		"RollupCore",
		"RollupAdminLogic",
		"RollupUserLogic",
		"EdgeChallengeManager", // BOLD protocol (replaces old ChallengeManager)
		"OneStepProofEntry",
		"OneStepProver0",
		"OneStepProverMemory",
		"OneStepProverMath",
		"OneStepProverHostIo",
		"UpgradeExecutor",
	}

	// Build a set for quick lookup
	requiredSet := make(map[string]bool)
	for _, name := range requiredContracts {
		requiredSet[name] = true
	}

	// Extract and parse each JSON file
	for _, f := range r.File {
		// Skip directories
		if f.FileInfo().IsDir() {
			continue
		}

		// Only process JSON files
		if filepath.Ext(f.Name) != ".json" {
			continue
		}

		// Extract contract name from filename (e.g., "RollupCreator.json" -> "RollupCreator")
		baseName := filepath.Base(f.Name)
		contractName := baseName[:len(baseName)-5] // Remove .json

		// Only load contracts we need
		if !requiredSet[contractName] {
			continue
		}

		// Read and parse the file
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", f.Name, err)
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.Name, err)
		}

		var artifact ContractArtifact
		if err := json.Unmarshal(data, &artifact); err != nil {
			return nil, fmt.Errorf("parse %s: %w", f.Name, err)
		}

		loaded[contractName] = &artifact
	}

	// Verify we loaded all required contracts
	var missing []string
	for _, name := range requiredContracts {
		if loaded[name] == nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required contracts: %v", missing)
	}

	// Build the NitroArtifacts struct
	return &NitroArtifacts{
		RollupCreator:        loaded["RollupCreator"],
		BridgeCreator:        loaded["BridgeCreator"],
		SequencerInbox:       loaded["SequencerInbox"],
		Bridge:               loaded["Bridge"],
		Inbox:                loaded["Inbox"],
		Outbox:               loaded["Outbox"],
		RollupCore:           loaded["RollupCore"],
		RollupAdminLogic:     loaded["RollupAdminLogic"],
		RollupUserLogic:      loaded["RollupUserLogic"],
		EdgeChallengeManager: loaded["EdgeChallengeManager"],
		OneStepProofEntry:    loaded["OneStepProofEntry"],
		OneStepProver0:       loaded["OneStepProver0"],
		OneStepProverMemory:  loaded["OneStepProverMemory"],
		OneStepProverMath:    loaded["OneStepProverMath"],
		OneStepProverHostIo:  loaded["OneStepProverHostIo"],
		UpgradeExecutor:      loaded["UpgradeExecutor"],
	}, nil
}

// LoadFromDirectory loads artifacts from a local directory (for testing or offline use).
func LoadFromDirectory(dir string) (*NitroArtifacts, error) {
	// List of contracts we need
	contracts := map[string]**ContractArtifact{}

	artifacts := &NitroArtifacts{}
	contracts["RollupCreator"] = &artifacts.RollupCreator
	contracts["BridgeCreator"] = &artifacts.BridgeCreator
	contracts["SequencerInbox"] = &artifacts.SequencerInbox
	contracts["Bridge"] = &artifacts.Bridge
	contracts["Inbox"] = &artifacts.Inbox
	contracts["Outbox"] = &artifacts.Outbox
	contracts["RollupCore"] = &artifacts.RollupCore
	contracts["RollupAdminLogic"] = &artifacts.RollupAdminLogic
	contracts["RollupUserLogic"] = &artifacts.RollupUserLogic
	contracts["EdgeChallengeManager"] = &artifacts.EdgeChallengeManager
	contracts["OneStepProofEntry"] = &artifacts.OneStepProofEntry
	contracts["OneStepProver0"] = &artifacts.OneStepProver0
	contracts["OneStepProverMemory"] = &artifacts.OneStepProverMemory
	contracts["OneStepProverMath"] = &artifacts.OneStepProverMath
	contracts["OneStepProverHostIo"] = &artifacts.OneStepProverHostIo
	contracts["UpgradeExecutor"] = &artifacts.UpgradeExecutor

	var missing []string
	for name, ptr := range contracts {
		path := filepath.Join(dir, name+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			missing = append(missing, name)
			continue
		}

		var artifact ContractArtifact
		if err := json.Unmarshal(data, &artifact); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		*ptr = &artifact
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required contracts in %s: %v", dir, missing)
	}

	artifacts.Version = "local"
	artifacts.LoadedAt = time.Now()
	artifacts.SourceURL = "file://" + dir

	return artifacts, nil
}

// EncodeABI encodes a function call using the contract's ABI.
// This is a helper for encoding constructor arguments or function calls.
func EncodeABI(artifact *ContractArtifact, method string, args ...interface{}) ([]byte, error) {
	// Parse ABI
	abiInterface, err := parseABI(artifact.ABI)
	if err != nil {
		return nil, fmt.Errorf("parse ABI: %w", err)
	}

	// Find the method
	if method == "" {
		// Constructor
		return abiInterface.Pack("", args...)
	}

	return abiInterface.Pack(method, args...)
}

// parseABI is a placeholder - we'll use go-ethereum's ABI package
func parseABI(abiJSON json.RawMessage) (*abiEncoder, error) {
	return &abiEncoder{raw: abiJSON}, nil
}

// abiEncoder is a placeholder for go-ethereum ABI encoding
type abiEncoder struct {
	raw json.RawMessage
}

func (a *abiEncoder) Pack(method string, args ...interface{}) ([]byte, error) {
	// This will be replaced with go-ethereum ABI encoding in signer.go
	// For now, just return nil - actual implementation uses go-ethereum
	return nil, fmt.Errorf("ABI encoding requires go-ethereum import")
}

// GetBytecodeBytes returns the bytecode as a byte slice.
func (a *ContractArtifact) GetBytecodeBytes() ([]byte, error) {
	bytecode := a.Bytecode.Object
	if len(bytecode) == 0 {
		return nil, fmt.Errorf("empty bytecode")
	}

	// Remove 0x prefix if present
	if len(bytecode) >= 2 && bytecode[:2] == "0x" {
		bytecode = bytecode[2:]
	}

	return hexDecode(bytecode)
}

// hexDecode decodes a hex string to bytes.
func hexDecode(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("odd length hex string")
	}

	result := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var b byte
		_, err := fmt.Sscanf(s[i:i+2], "%02x", &b)
		if err != nil {
			return nil, fmt.Errorf("invalid hex at position %d: %w", i, err)
		}
		result[i/2] = b
	}
	return result, nil
}

// Ensure unused imports are used
var _ = bytes.Buffer{}
