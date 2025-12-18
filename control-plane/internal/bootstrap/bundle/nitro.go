package bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"time"
)

// createNitroBundle creates the Nitro/Orbit specific bundle structure.
//
// Bundle structure:
//
//	{chain-name}-nitro-artifacts/
//	├── README.md
//	├── manifest.json
//	├── docker-compose.yml
//	├── .env.example
//	├── config/
//	│   ├── chain-info.json
//	│   ├── node-config.json
//	│   └── core-contracts.json
//	├── certs/                     # Nitro-specific (mTLS)
//	│   ├── client.crt
//	│   ├── client.key             # Mode 0600
//	│   └── ca.crt
//	└── scripts/
//	    ├── start.sh
//	    └── healthcheck.sh
func (b *Bundler) createNitroBundle(cfg *BundleConfig) (*BundleResult, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	tarW := newTarWriter(tw)

	baseDir := fmt.Sprintf("%s-nitro-artifacts", sanitizeName(cfg.ChainName))

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
		Description: "Docker Compose configuration for Nitro",
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

	// chain-info.json
	if chainInfo, ok := cfg.Artifacts["chain_info"]; ok {
		if err := tarW.addFile(baseDir+"/config/chain-info.json", chainInfo); err != nil {
			return nil, fmt.Errorf("add chain-info.json: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "config/chain-info.json",
			Description: "Chain metadata and configuration",
			Required:    true,
			SizeBytes:   int64(len(chainInfo)),
		})
	}

	// node-config.json
	if nodeConfig, ok := cfg.Artifacts["node_config"]; ok {
		if err := tarW.addFile(baseDir+"/config/node-config.json", nodeConfig); err != nil {
			return nil, fmt.Errorf("add node-config.json: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "config/node-config.json",
			Description: "Nitro node configuration",
			Required:    true,
			SizeBytes:   int64(len(nodeConfig)),
		})
	}

	// core-contracts.json
	if coreContracts, ok := cfg.Artifacts["core_contracts"]; ok {
		if err := tarW.addFile(baseDir+"/config/core-contracts.json", coreContracts); err != nil {
			return nil, fmt.Errorf("add core-contracts.json: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "config/core-contracts.json",
			Description: "Deployed contract addresses",
			Required:    true,
			SizeBytes:   int64(len(coreContracts)),
		})
	}

	// ===========================================
	// CERTS DIRECTORY (Nitro-specific mTLS!)
	// ===========================================

	certsIncluded := false

	// client.crt
	if cfg.ClientCert != nil && len(cfg.ClientCert) > 0 {
		if err := tarW.addFile(baseDir+"/certs/client.crt", cfg.ClientCert); err != nil {
			return nil, fmt.Errorf("add client.crt: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "certs/client.crt",
			Description: "mTLS client certificate for POPSigner",
			Required:    true,
			SizeBytes:   int64(len(cfg.ClientCert)),
		})
		certsIncluded = true
	}

	// client.key (with restricted permissions!)
	if cfg.ClientKey != nil && len(cfg.ClientKey) > 0 {
		if err := tarW.addFileWithMode(baseDir+"/certs/client.key", cfg.ClientKey, 0600); err != nil {
			return nil, fmt.Errorf("add client.key: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "certs/client.key",
			Description: "mTLS client private key (keep secure!)",
			Required:    true,
			SizeBytes:   int64(len(cfg.ClientKey)),
		})
	}

	// ca.crt (optional)
	if cfg.CACert != nil && len(cfg.CACert) > 0 {
		if err := tarW.addFile(baseDir+"/certs/ca.crt", cfg.CACert); err != nil {
			return nil, fmt.Errorf("add ca.crt: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "certs/ca.crt",
			Description: "POPSigner CA certificate",
			Required:    false,
			SizeBytes:   int64(len(cfg.CACert)),
		})
	}

	// If no certs provided, add placeholder files with instructions
	if !certsIncluded {
		placeholder := []byte("# Placeholder - replace with your POPSigner mTLS certificate\n# Download from: https://dashboard.popsigner.com/certificates\n")
		if err := tarW.addFile(baseDir+"/certs/.gitkeep", placeholder); err != nil {
			return nil, fmt.Errorf("add .gitkeep: %w", err)
		}
		files = append(files, FileEntry{
			Path:        "certs/.gitkeep",
			Description: "Placeholder - add your mTLS certificates here",
			Required:    false,
		})
	}

	// ===========================================
	// SCRIPTS DIRECTORY
	// ===========================================

	startScript := generateNitroStartScript()
	if err := tarW.addExecutable(baseDir+"/scripts/start.sh", []byte(startScript)); err != nil {
		return nil, fmt.Errorf("add start.sh: %w", err)
	}
	files = append(files, FileEntry{
		Path:        "scripts/start.sh",
		Description: "Helper script to start all services",
		Required:    false,
	})

	healthScript := generateNitroHealthScript()
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

	readme := generateNitroReadme(cfg, certsIncluded)
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
		Stack:       StackNitro,
		ChainID:     cfg.ChainID,
		ChainName:   cfg.ChainName,
		GeneratedAt: time.Now().UTC(),
		Files:       files,
		POPSignerInfo: POPSignerInfo{
			MTLSEndpoint:        cfg.POPSignerMTLSEndpoint,
			CertificateIncluded: certsIncluded,
			BatchPosterAddr:     cfg.BatcherAddress,
			ValidatorAddr:       cfg.ValidatorAddress,
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
		Filename:  fmt.Sprintf("%s-nitro-artifacts.tar.gz", sanitizeName(cfg.ChainName)),
		Manifest:  manifest,
		SizeBytes: int64(len(data)),
		Checksum:  calculateBundleChecksum(data),
	}, nil
}

// generateNitroStartScript creates the start.sh helper script.
func generateNitroStartScript() string {
	return `#!/bin/bash
set -e

echo "Starting Nitro rollup..."

# Check for .env file
if [ ! -f .env ]; then
    echo "ERROR: .env file not found."
    echo "Copy .env.example to .env and configure it:"
    echo "  cp .env.example .env"
    echo "  # Edit .env with your L1 RPC URL"
    exit 1
fi

# Check for mTLS certificates
if [ ! -f certs/client.crt ] || [ ! -f certs/client.key ]; then
    echo "ERROR: mTLS certificates not found in certs/ directory."
    echo ""
    echo "Required files:"
    echo "  certs/client.crt  - Client certificate"
    echo "  certs/client.key  - Client private key"
    echo ""
    echo "Download your certificates from: https://dashboard.popsigner.com/certificates"
    exit 1
fi

# Verify certificate permissions
if [ "$(stat -f %Lp certs/client.key 2>/dev/null || stat -c %a certs/client.key 2>/dev/null)" != "600" ]; then
    echo "Warning: Setting proper permissions on client.key..."
    chmod 600 certs/client.key
fi

# Load environment
source .env

# Verify required variables
if [ -z "$L1_RPC_URL" ] || [ "$L1_RPC_URL" = "https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY" ]; then
    echo "ERROR: L1_RPC_URL not configured in .env"
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
echo "  L2 JSON-RPC: http://localhost:8547"
echo "  L2 WebSocket: ws://localhost:8548"
echo "  Metrics: http://localhost:9642"
`
}

// generateNitroHealthScript creates the healthcheck.sh helper script.
func generateNitroHealthScript() string {
	return `#!/bin/bash

echo "Checking Nitro services..."
echo ""

# Check nitro node
printf "nitro:        "
if curl -sf http://localhost:8547 > /dev/null 2>&1; then
    echo "✓ healthy"
else
    echo "✗ not responding"
fi

# Check batch-poster
printf "batch-poster: "
if docker compose ps batch-poster 2>/dev/null | grep -q "Up"; then
    echo "✓ running"
else
    echo "✗ not running"
fi

# Check validator
printf "validator:    "
if docker compose ps validator 2>/dev/null | grep -q "Up"; then
    echo "✓ running"
else
    echo "✗ not running"
fi

# Check metrics endpoint
printf "metrics:      "
if curl -sf http://localhost:9642/metrics > /dev/null 2>&1; then
    echo "✓ available"
else
    echo "○ not available"
fi

echo ""
echo "Container Status:"
docker compose ps
`
}

// generateNitroReadme creates the README.md documentation.
func generateNitroReadme(cfg *BundleConfig, certsIncluded bool) string {
	certNote := ""
	if !certsIncluded {
		certNote = `
> ⚠️ **Important:** mTLS certificates are not included in this bundle.
> Download your certificates from: https://dashboard.popsigner.com/certificates
> Place client.crt and client.key in the certs/ directory.
`
	}

	const readmeTemplate = `# %s - Nitro (Orbit) Rollup

This bundle contains everything needed to run your Arbitrum Nitro rollup.
%s
## Quick Start

### 1. Configure Environment

%scp .env.example .env%s

Edit .env and configure:
- L1_RPC_URL - Your L1 (parent chain) RPC endpoint

### 2. Verify Certificates

%sls -la certs/
# Should contain: client.crt, client.key%s

Ensure proper permissions:
%schmod 600 certs/client.key%s

### 3. Start the Rollup

%s./scripts/start.sh
# OR
docker compose up -d%s

### 4. Verify Health

%s./scripts/healthcheck.sh%s

## Chain Information

| Property | Value |
|----------|-------|
| Chain ID | %d |
| Chain Name | %s |
| Stack | Arbitrum Nitro (Orbit) |

## Endpoints (after startup)

| Service | URL |
|---------|-----|
| L2 JSON-RPC | http://localhost:8547 |
| L2 WebSocket | ws://localhost:8548 |
| Metrics | http://localhost:9642 |

## POPSigner Integration

This rollup uses **POPSigner** for secure key management. The batch-poster and 
validator connect to POPSigner using **mTLS authentication**.

**Security Notes:**
- Keep your private key (certs/client.key) secure
- Never commit certificates to version control
- The certificates are bound to your POPSigner account
- Set proper permissions: chmod 600 certs/client.key

## Bundle Contents

| File | Description |
|------|-------------|
| docker-compose.yml | Service definitions |
| .env.example | Environment template |
| config/chain-info.json | Chain metadata |
| config/node-config.json | Node configuration |
| config/core-contracts.json | Contract addresses |
| certs/client.crt | mTLS client certificate |
| certs/client.key | mTLS private key (keep secure!) |
| scripts/start.sh | Startup helper |
| scripts/healthcheck.sh | Health verification |
| manifest.json | Bundle metadata |

## Troubleshooting

### Services not starting?

1. Check Docker is running
2. Check logs: docker compose logs -f [service]
3. Verify .env configuration

### mTLS connection issues?

1. Verify certificate files exist in certs/
2. Check certificate permissions
3. Ensure certificates haven't expired
4. Verify network connectivity to mtls.popsigner.com

### Certificate permissions error?

%schmod 600 certs/client.key%s

## Documentation

- POPSigner Docs: https://docs.popsigner.com
- Arbitrum Docs: https://docs.arbitrum.io

## Support

For assistance, contact support@popsigner.com or visit https://popsigner.com
`
	codeStart := "```bash\n"
	codeEnd := "\n```"
	return fmt.Sprintf(readmeTemplate,
		cfg.ChainName, certNote,
		codeStart, codeEnd,
		codeStart, codeEnd,
		codeStart, codeEnd,
		codeStart, codeEnd,
		codeStart, codeEnd,
		cfg.ChainID, cfg.ChainName,
		codeStart, codeEnd)
}

