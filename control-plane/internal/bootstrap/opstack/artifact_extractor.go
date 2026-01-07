package opstack

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/bootstrap/repository"
)

// ArtifactExtractor extracts and bundles deployment artifacts from OP Stack state.
type ArtifactExtractor struct {
	repo repository.Repository
}

// NewArtifactExtractor creates a new artifact extractor.
func NewArtifactExtractor(repo repository.Repository) *ArtifactExtractor {
	return &ArtifactExtractor{repo: repo}
}

// ExtractArtifacts extracts all deployment artifacts from the deployment state.
func (e *ArtifactExtractor) ExtractArtifacts(
	ctx context.Context,
	deploymentID uuid.UUID,
	cfg *DeploymentConfig,
) (*OPStackArtifacts, error) {
	artifacts := &OPStackArtifacts{}

	// 1. Extract genesis.json from saved artifact
	genesis, err := e.extractGenesis(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("extract genesis: %w", err)
	}
	artifacts.Genesis = genesis

	// 2. Extract rollup.json - prefer saved artifact from op-deployer, fallback to building from config
	rollupJSON, err := e.extractRollupConfig(ctx, deploymentID, cfg)
	if err != nil {
		return nil, fmt.Errorf("extract rollup config: %w", err)
	}
	artifacts.Rollup = rollupJSON

	// 3. Extract contract addresses from state
	addrs, err := e.extractContractAddresses(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("extract addresses: %w", err)
	}
	artifacts.Addresses = addrs

	// 4. Get original deployment config
	cfgJSON, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	artifacts.DeployConfig = cfgJSON

	// 5. Generate JWT secret for Engine API
	artifacts.JWTSecret = generateJWTSecret()

	// 6. Generate Docker Compose
	compose, err := GenerateDockerCompose(cfg, artifacts)
	if err != nil {
		return nil, fmt.Errorf("generate docker-compose: %w", err)
	}
	artifacts.DockerCompose = compose

	// 7. Generate .env.example
	artifacts.EnvExample = GenerateEnvExample(cfg, &addrs)

	// 8. Generate op-alt-da config.toml (Celestia DA - always enabled for POPKins)
	altDAConfig, err := GenerateAltDAConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("generate altda config: %w", err)
	}
	artifacts.AltDAConfig = altDAConfig

	// 9. Generate README (POPKins always uses Celestia DA)
	artifacts.Readme = GenerateBundleReadme(cfg.ChainName, true)

	// 10. Save all artifacts to database
	if err := e.saveAllArtifacts(ctx, deploymentID, artifacts); err != nil {
		return nil, fmt.Errorf("save artifacts: %w", err)
	}

	return artifacts, nil
}

// GetArtifact retrieves a specific artifact by type.
func (e *ArtifactExtractor) GetArtifact(ctx context.Context, deploymentID uuid.UUID, artifactType string) ([]byte, error) {
	artifact, err := e.repo.GetArtifact(ctx, deploymentID, artifactType)
	if err != nil {
		return nil, fmt.Errorf("get artifact: %w", err)
	}
	if artifact == nil {
		return nil, fmt.Errorf("artifact %s not found", artifactType)
	}
	return artifact.Content, nil
}

// ListArtifacts returns all available artifact types for a deployment.
func (e *ArtifactExtractor) ListArtifacts(ctx context.Context, deploymentID uuid.UUID) ([]string, error) {
	artifacts, err := e.repo.GetAllArtifacts(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("get artifacts: %w", err)
	}

	types := make([]string, 0, len(artifacts))
	for _, a := range artifacts {
		// Skip internal artifacts
		if a.ArtifactType == "deployment_state" {
			continue
		}
		types = append(types, a.ArtifactType)
	}
	return types, nil
}

// CreateBundle packages all artifacts into a ZIP bundle.
func (e *ArtifactExtractor) CreateBundle(ctx context.Context, deploymentID uuid.UUID, chainName string) ([]byte, error) {
	artifacts, err := e.repo.GetAllArtifacts(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("get artifacts: %w", err)
	}

	if len(artifacts) == 0 {
		return nil, fmt.Errorf("no artifacts found for deployment %s", deploymentID)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Create bundle directory structure
	bundlePrefix := fmt.Sprintf("%s-opstack-bundle/", sanitizeChainName(chainName))

	// Organize artifacts into the bundle
	for _, a := range artifacts {
		var path string
		isPlainText := false // Non-JSON files stored as JSON strings need unwrapping
		switch a.ArtifactType {
		case "genesis.json":
			path = bundlePrefix + "genesis.json"
		case "rollup.json":
			path = bundlePrefix + "rollup.json"
		case "addresses.json":
			path = bundlePrefix + "addresses.json"
		case "deploy-config.json":
			path = bundlePrefix + "deploy-config.json"
		case "docker-compose.yml":
			path = bundlePrefix + "docker-compose.yml"
			isPlainText = true
		case ".env.example":
			path = bundlePrefix + ".env.example"
			isPlainText = true
		case "jwt.txt":
			path = bundlePrefix + "jwt.txt"
			isPlainText = true
		case "config.toml":
			path = bundlePrefix + "config.toml"
			isPlainText = true
		case "README.md":
			path = bundlePrefix + "README.md"
			isPlainText = true
		default:
			// Skip internal artifacts like deployment_state
			continue
		}

		// Get content, unwrapping JSON string if necessary
		content := a.Content
		if isPlainText {
			content = unwrapJSONString(a.Content)
		}

		if err := addToZip(zw, path, content); err != nil {
			return nil, fmt.Errorf("add %s to zip: %w", a.ArtifactType, err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// saveAllArtifacts saves all artifacts to the database.
func (e *ArtifactExtractor) saveAllArtifacts(ctx context.Context, deploymentID uuid.UUID, arts *OPStackArtifacts) error {
	// Save genesis.json (already saved during deployment, but update if needed)
	if len(arts.Genesis) > 0 {
		if err := e.saveArtifact(ctx, deploymentID, "genesis.json", arts.Genesis); err != nil {
			return err
		}
	}

	// Save rollup.json
	if len(arts.Rollup) > 0 {
		if err := e.saveArtifact(ctx, deploymentID, "rollup.json", arts.Rollup); err != nil {
			return err
		}
	}

	// Save addresses.json
	addrsBytes, err := json.MarshalIndent(arts.Addresses, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal addresses: %w", err)
	}
	if err := e.saveArtifact(ctx, deploymentID, "addresses.json", addrsBytes); err != nil {
		return err
	}

	// Save deploy-config.json
	if len(arts.DeployConfig) > 0 {
		if err := e.saveArtifact(ctx, deploymentID, "deploy-config.json", arts.DeployConfig); err != nil {
			return err
		}
	}

	// Save docker-compose.yml
	if arts.DockerCompose != "" {
		if err := e.saveArtifact(ctx, deploymentID, "docker-compose.yml", []byte(arts.DockerCompose)); err != nil {
			return err
		}
	}

	// Save .env.example
	if arts.EnvExample != "" {
		if err := e.saveArtifact(ctx, deploymentID, ".env.example", []byte(arts.EnvExample)); err != nil {
			return err
		}
	}

	// Save JWT secret
	if arts.JWTSecret != "" {
		if err := e.saveArtifact(ctx, deploymentID, "jwt.txt", []byte(arts.JWTSecret)); err != nil {
			return err
		}
	}

	// Save op-alt-da config.toml (Celestia)
	if arts.AltDAConfig != "" {
		if err := e.saveArtifact(ctx, deploymentID, "config.toml", []byte(arts.AltDAConfig)); err != nil {
			return err
		}
	}

	// Save README.md
	if arts.Readme != "" {
		if err := e.saveArtifact(ctx, deploymentID, "README.md", []byte(arts.Readme)); err != nil {
			return err
		}
	}

	return nil
}

// saveArtifact saves a single artifact to the database.
// For non-JSON content (like docker-compose.yml, jwt.txt), wraps as base64 in a JSON object.
// This avoids PostgreSQL JSONB normalization issues with escape sequences.
func (e *ArtifactExtractor) saveArtifact(ctx context.Context, deploymentID uuid.UUID, name string, content []byte) error {
	var jsonContent json.RawMessage

	// Check if content is already valid JSON
	if json.Valid(content) {
		jsonContent = content
	} else {
		// Wrap non-JSON content as base64 in a JSON object.
		// This avoids PostgreSQL JSONB escape sequence normalization issues.
		wrapper := struct {
			Type string `json:"_type"`
			Data string `json:"data"`
		}{
			Type: "base64",
			Data: base64.StdEncoding.EncodeToString(content),
		}
		encoded, err := json.Marshal(wrapper)
		if err != nil {
			return fmt.Errorf("marshal non-JSON content for %s: %w", name, err)
		}
		jsonContent = encoded
	}

	artifact := &repository.Artifact{
		ID:           uuid.New(),
		DeploymentID: deploymentID,
		ArtifactType: name,
		Content:      jsonContent,
		CreatedAt:    time.Now(),
	}
	return e.repo.SaveArtifact(ctx, artifact)
}

// Helper functions

// generateJWTSecret generates a random JWT secret for Engine API authentication.
func generateJWTSecret() string {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		// Fallback to a static secret (should not happen in practice)
		return "0x" + "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	}
	return "0x" + hex.EncodeToString(secret)
}

// addToZip adds a file to the ZIP archive.
func addToZip(zw *zip.Writer, name string, content []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}

