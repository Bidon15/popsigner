package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"time"
)

// createOPStackBundle creates the OP Stack specific bundle structure.
//
// Bundle structure:
//
//	{chain-name}-opstack-artifacts/
//	├── README.md
//	├── manifest.json
//	├── docker-compose.yml
//	├── .env.example
//	├── config/
//	│   ├── rollup.json
//	│   ├── addresses.json
//	│   └── deploy-config.json
//	├── genesis/
//	│   └── genesis.json          # ~50MB (large!)
//	├── secrets/
//	│   └── jwt.txt                # Auto-generated
//	└── scripts/
//	    ├── start.sh
//	    └── healthcheck.sh
func (b *Bundler) createOPStackBundle(cfg *BundleConfig) (*BundleResult, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	tarW := newTarWriter(tw)

	baseDir := fmt.Sprintf("%s-opstack-artifacts", sanitizeName(cfg.ChainName))

	// Track files for manifest
	var files []FileEntry

	// ===========================================
	// ROOT FILES
	// ===========================================

	// docker-compose.yml
	if err := tarW.addFile(baseDir+"/docker-compose.yml", []byte(cfg.DockerCompose)); err != nil {
		return nil, fmt.Errorf("add docker-compose.yml: %w", err)
	}
	files = append(files, FileEntry{
		Path:        "docker-compose.yml",
		Description: "Docker Compose configuration for OP Stack",
		Required:    true,
		SizeBytes:   int64(len(cfg.DockerCompose)),
	})

	// .env.example
	if err := tarW.addFile(baseDir+"/.env.example", []byte(cfg.EnvExample)); err != nil {
		return nil, fmt.Errorf("add .env.example: %w", err)
	}
	files = append(files, FileEntry{
		Path:        ".env.example",
		Description: "Environment variables template (copy to .env and fill in)",
		Required:    true,
		SizeBytes:   int64(len(cfg.EnvExample)),
	})

	// ===========================================
	// CONFIG DIRECTORY
	// ===========================================

	// rollup.json
	if rollupConfig, ok := cfg.Artifacts["rollup_config"]; ok {
		if err := tarW.addFile(baseDir+"/config/rollup.json", rollupConfig); err != nil {
			return nil, fmt.Errorf("add rollup.json: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "config/rollup.json",
			Description: "Rollup configuration",
			Required:    true,
			SizeBytes:   int64(len(rollupConfig)),
		})
	}

	// addresses.json (deployed contracts)
	if addresses, ok := cfg.Artifacts["addresses"]; ok {
		if err := tarW.addFile(baseDir+"/config/addresses.json", addresses); err != nil {
			return nil, fmt.Errorf("add addresses.json: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "config/addresses.json",
			Description: "Deployed contract addresses",
			Required:    true,
			SizeBytes:   int64(len(addresses)),
		})
	}

	// deploy-config.json (original deployment config)
	if deployConfig, ok := cfg.Artifacts["deploy_config"]; ok {
		if err := tarW.addFile(baseDir+"/config/deploy-config.json", deployConfig); err != nil {
			return nil, fmt.Errorf("add deploy-config.json: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "config/deploy-config.json",
			Description: "Original deployment configuration",
			Required:    false,
			SizeBytes:   int64(len(deployConfig)),
		})
	}

	// ===========================================
	// GENESIS DIRECTORY
	// ===========================================

	// genesis.json (large file!)
	if genesis, ok := cfg.Artifacts["genesis"]; ok {
		if err := tarW.addFile(baseDir+"/genesis/genesis.json", genesis); err != nil {
			return nil, fmt.Errorf("add genesis.json: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "genesis/genesis.json",
			Description: "L2 genesis state (required for first boot)",
			Required:    true,
			SizeBytes:   int64(len(genesis)),
		})
	}

	// ===========================================
	// SECRETS DIRECTORY
	// ===========================================

	// jwt.txt (auto-generated for op-node <-> op-geth auth)
	jwt := generateJWT()
	if err := tarW.addFileWithMode(baseDir+"/secrets/jwt.txt", []byte(jwt), 0600); err != nil {
		return nil, fmt.Errorf("add jwt.txt: %w", err)
	}
	files = append(files, FileEntry{
		Path:        "secrets/jwt.txt",
		Description: "JWT secret for op-node/op-geth authentication",
		Required:    true,
	})

	// ===========================================
	// SCRIPTS DIRECTORY
	// ===========================================

	startScript := generateOPStackStartScript()
	if err := tarW.addExecutable(baseDir+"/scripts/start.sh", []byte(startScript)); err != nil {
		return nil, fmt.Errorf("add start.sh: %w", err)
	}
	files = append(files, FileEntry{
		Path:        "scripts/start.sh",
		Description: "Helper script to start all services",
		Required:    false,
	})

	healthScript := generateOPStackHealthScript()
	if err := tarW.addExecutable(baseDir+"/scripts/healthcheck.sh", []byte(healthScript)); err != nil {
		return nil, fmt.Errorf("add healthcheck.sh: %w", err)
	}
	files = append(files, FileEntry{
		Path:        "scripts/healthcheck.sh",
		Description: "Health check script for all services",
		Required:    false,
	})

	// ===========================================
	// README
	// ===========================================

	readme := generateOPStackReadme(cfg)
	if err := tarW.addFile(baseDir+"/README.md", []byte(readme)); err != nil {
		return nil, fmt.Errorf("add README.md: %w", err)
	}
	files = append(files, FileEntry{
		Path:        "README.md",
		Description: "Quick start guide and documentation",
		Required:    false,
	})

	// ===========================================
	// MANIFEST
	// ===========================================

	manifest := &BundleManifest{
		Version:     "1.0",
		Stack:       StackOPStack,
		ChainID:     cfg.ChainID,
		ChainName:   cfg.ChainName,
		GeneratedAt: time.Now().UTC(),
		Files:       files,
		POPSignerInfo: POPSignerInfo{
			Endpoint:         cfg.POPSignerEndpoint,
			APIKeyConfigured: true,
			BatcherAddr:      cfg.BatcherAddress,
			ProposerAddr:     cfg.ProposerAddress,
		},
		Checksums: tarW.checksums,
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	if err := tarW.addFile(baseDir+"/manifest.json", manifestBytes); err != nil {
		return nil, fmt.Errorf("add manifest.json: %w", err)
	}

	// Finalize
	data, err := finalizeTarGz(tw, gw, &buf)
	if err != nil {
		return nil, err
	}

	return &BundleResult{
		Data:      data,
		Filename:  fmt.Sprintf("%s-opstack-artifacts.tar.gz", sanitizeName(cfg.ChainName)),
		Manifest:  manifest,
		SizeBytes: int64(len(data)),
		Checksum:  calculateBundleChecksum(data),
	}, nil
}

// generateOPStackStartScript creates the start.sh helper script.
func generateOPStackStartScript() string {
	return `#!/bin/bash
set -e

echo "Starting OP Stack rollup..."

# Check for .env file
if [ ! -f .env ]; then
    echo "ERROR: .env file not found."
    echo "Copy .env.example to .env and configure it:"
    echo "  cp .env.example .env"
    echo "  # Edit .env with your L1 RPC URL and POPSigner API key"
    exit 1
fi

# Load environment
source .env

# Verify required variables
if [ -z "$L1_RPC_URL" ] || [ "$L1_RPC_URL" = "https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY" ]; then
    echo "ERROR: L1_RPC_URL not configured in .env"
    exit 1
fi

if [ -z "$POPSIGNER_API_KEY" ] || [ "$POPSIGNER_API_KEY" = "bbr_live_xxxxxxxxxxxxxxxxxxxxx" ]; then
    echo "ERROR: POPSIGNER_API_KEY not configured in .env"
    exit 1
fi

# Start services
echo "Starting Docker Compose services..."
docker compose up -d

echo ""
echo "Services started successfully!"
echo "Run './scripts/healthcheck.sh' to verify all services are healthy."
echo ""
echo "Endpoints:"
echo "  L2 JSON-RPC: http://localhost:8545"
echo "  L2 WebSocket: ws://localhost:8546"
echo "  OP Node RPC: http://localhost:9545"
`
}

// generateOPStackHealthScript creates the healthcheck.sh helper script.
func generateOPStackHealthScript() string {
	return `#!/bin/bash

echo "Checking OP Stack services..."
echo ""

# Check op-geth
printf "op-geth:     "
if curl -sf http://localhost:8545 > /dev/null 2>&1; then
    echo "✓ healthy"
else
    echo "✗ not responding"
fi

# Check op-node
printf "op-node:     "
if curl -sf http://localhost:9545/healthz > /dev/null 2>&1; then
    echo "✓ healthy"
else
    echo "✗ not responding"
fi

# Check op-batcher (logs only - no health endpoint)
printf "op-batcher:  "
if docker compose ps op-batcher 2>/dev/null | grep -q "Up"; then
    echo "✓ running"
else
    echo "✗ not running"
fi

# Check op-proposer (logs only - no health endpoint)
printf "op-proposer: "
if docker compose ps op-proposer 2>/dev/null | grep -q "Up"; then
    echo "✓ running"
else
    echo "✗ not running"
fi

echo ""
echo "Container Status:"
docker compose ps
`
}

// generateOPStackReadme creates the README.md documentation.
func generateOPStackReadme(cfg *BundleConfig) string {
	const readmeTemplate = `# %s - OP Stack Rollup

This bundle contains everything needed to run your OP Stack rollup.

## Quick Start

### 1. Configure Environment

%scp .env.example .env%s

Edit .env and configure:
- L1_RPC_URL - Your L1 (Ethereum) RPC endpoint
- POPSIGNER_API_KEY - Your POPSigner API key (from dashboard)

### 2. Start the Rollup

%s./scripts/start.sh
# OR
docker compose up -d%s

### 3. Verify Health

%s./scripts/healthcheck.sh%s

## Chain Information

| Property | Value |
|----------|-------|
| Chain ID | %d |
| Chain Name | %s |
| Stack | OP Stack |

## Endpoints (after startup)

| Service | URL |
|---------|-----|
| L2 JSON-RPC | http://localhost:8545 |
| L2 WebSocket | ws://localhost:8546 |
| OP Node RPC | http://localhost:9545 |

## POPSigner Integration

This rollup uses **POPSigner** for secure key management. The batcher and proposer
connect to POPSigner using API key authentication.

**Security Notes:**
- Never expose your POPSigner API key
- Keep your .env file secure and out of version control
- The batcher and proposer keys are managed by POPSigner

## Bundle Contents

| File | Description |
|------|-------------|
| docker-compose.yml | Service definitions |
| .env.example | Environment template |
| config/rollup.json | Rollup configuration |
| config/addresses.json | Deployed contract addresses |
| genesis/genesis.json | L2 genesis state |
| secrets/jwt.txt | JWT for op-node/op-geth auth |
| scripts/start.sh | Startup helper |
| scripts/healthcheck.sh | Health verification |
| manifest.json | Bundle metadata |

## Troubleshooting

### Services not starting?

1. Check Docker is running
2. Check logs: docker compose logs -f [service]
3. Verify .env configuration

### POPSigner connection issues?

1. Verify API key is correct
2. Check network connectivity to rpc.popsigner.com
3. Ensure batcher/proposer addresses match your POPSigner keys

## Documentation

- POPSigner Docs: https://docs.popsigner.com
- OP Stack Docs: https://docs.optimism.io

## Support

For assistance, contact support@popsigner.com or visit https://popsigner.com
`
	codeStart := "```bash\n"
	codeEnd := "\n```"
	return fmt.Sprintf(readmeTemplate,
		cfg.ChainName,
		codeStart, codeEnd,
		codeStart, codeEnd,
		codeStart, codeEnd,
		cfg.ChainID, cfg.ChainName)
}

