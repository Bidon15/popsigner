package popdeployer

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Bidon15/popsigner/control-plane/internal/bootstrap/opstack"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/inspect"
)

// ConfigWriter handles generating all bundle configuration files.
type ConfigWriter struct {
	logger        *slog.Logger
	result        *opstack.DeployResult
	config        *DeploymentConfig
	celestiaKeyID string
}

// GenerateAll generates all configuration files and returns them as a map.
// Keys are artifact types (filenames), values are the file contents as bytes.
func (w *ConfigWriter) GenerateAll() (map[string][]byte, error) {
	artifacts := make(map[string][]byte)

	// Generate each config file
	generators := []struct {
		name string
		fn   func() ([]byte, error)
	}{
		{"genesis.json", w.generateGenesis},
		{"rollup.json", w.generateRollupConfig},
		{"addresses.json", w.generateAddresses},
		{"jwt.txt", w.generateJWT},
		{"config.toml", w.generateConfigToml},
		{"l1-chain-config.json", w.generateL1ChainConfig},
		{"docker-compose.yml", w.generateDockerCompose},
		{".env.example", w.generateEnvExample},
		{"README.md", w.generateREADME},
	}

	for _, gen := range generators {
		w.logger.Info("generating artifact", slog.String("type", gen.name))
		data, err := gen.fn()
		if err != nil {
			return nil, fmt.Errorf("generate %s: %w", gen.name, err)
		}
		artifacts[gen.name] = data
	}

	return artifacts, nil
}

// generateGenesis generates the L2 genesis.json file.
func (w *ConfigWriter) generateGenesis() ([]byte, error) {
	if len(w.result.ChainStates) == 0 {
		return nil, fmt.Errorf("no chain states in deployment result")
	}

	chainState := w.result.ChainStates[0]

	// Generate genesis using op-deployer's inspect package
	l2Genesis, _, err := inspect.GenesisAndRollup(w.result.State, chainState.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate genesis: %w", err)
	}
	if l2Genesis == nil {
		return nil, fmt.Errorf("genesis generation returned nil")
	}

	data, err := json.MarshalIndent(l2Genesis, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal genesis: %w", err)
	}

	w.logger.Info("genesis.json generated", slog.Int("size_mb", len(data)/(1024*1024)))
	return data, nil
}

// generateRollupConfig generates the rollup.json configuration file.
func (w *ConfigWriter) generateRollupConfig() ([]byte, error) {
	if len(w.result.ChainStates) == 0 {
		return nil, fmt.Errorf("no chain states in deployment result")
	}

	chainState := w.result.ChainStates[0]

	// Generate rollup config using op-deployer's inspect package
	_, rollupCfg, err := inspect.GenesisAndRollup(w.result.State, chainState.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rollup config: %w", err)
	}
	if rollupCfg == nil {
		return nil, fmt.Errorf("rollup config generation returned nil")
	}

	data, err := json.MarshalIndent(rollupCfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal rollup config: %w", err)
	}

	return data, nil
}

// generateAddresses generates the addresses.json file with contract addresses.
func (w *ConfigWriter) generateAddresses() ([]byte, error) {
	addresses := make(map[string]interface{})

	// Superchain contracts
	if w.result.SuperchainContracts != nil {
		addresses["superchain"] = w.result.SuperchainContracts
	}

	// Implementation contracts
	if w.result.ImplementationsContracts != nil {
		addresses["implementations"] = w.result.ImplementationsContracts
	}

	// Chain-specific contracts
	if len(w.result.ChainStates) > 0 {
		chainState := w.result.ChainStates[0]
		addresses["chain_state"] = chainState
	}

	// Deployment info
	addresses["deployment"] = map[string]interface{}{
		"create2_salt":          w.result.Create2Salt.Hex(),
		"infrastructure_reused": w.result.InfrastructureReused,
		"chain_id":              w.config.ChainID,
		"chain_name":            w.config.ChainName,
		"deployer_address":      w.config.DeployerAddress,
		"batcher_address":       w.config.BatcherAddress,
		"proposer_address":      w.config.ProposerAddress,
	}

	data, err := json.MarshalIndent(addresses, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal addresses: %w", err)
	}

	return data, nil
}

// generateJWT generates a random JWT secret.
func (w *ConfigWriter) generateJWT() ([]byte, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate random secret: %w", err)
	}

	// Return as hex string without 0x prefix
	jwtSecret := hex.EncodeToString(secret)
	return []byte(jwtSecret), nil
}

// generateConfigToml generates the config.toml file for Celestia DA.
func (w *ConfigWriter) generateConfigToml() ([]byte, error) {
	// Celestia-specific configuration
	config := fmt.Sprintf(`[celestia]
# Celestia configuration for DA (uses Localestia for local testing)
key = "%s"
namespace_id = "000000000000ffff"
localestia_url = "http://localestia:7980"
rpc_url = "http://localestia:26658"
auth_token = ""
`, w.celestiaKeyID)

	return []byte(config), nil
}

// generateL1ChainConfig generates the l1-chain-config.json file.
func (w *ConfigWriter) generateL1ChainConfig() ([]byte, error) {
	// For Anvil L1, use a simple config
	chainConfig := map[string]interface{}{
		"chainId":             w.config.L1ChainID,
		"homesteadBlock":      0,
		"eip150Block":         0,
		"eip155Block":         0,
		"eip158Block":         0,
		"byzantiumBlock":      0,
		"constantinopleBlock": 0,
		"petersburgBlock":     0,
		"istanbulBlock":       0,
		"berlinBlock":         0,
		"londonBlock":         0,
	}

	data, err := json.MarshalIndent(chainConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal l1 chain config: %w", err)
	}

	return data, nil
}

// generateDockerCompose generates the docker-compose.yml file.
func (w *ConfigWriter) generateDockerCompose() ([]byte, error) {
	// Note: Contract addresses are available in addresses.json
	// The docker-compose.yml references them via environment variables when needed

	compose := fmt.Sprintf(`version: '3.8'

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  anvil:
    image: ghcr.io/foundry-rs/foundry:latest
    entrypoint: anvil
    command:
      - --host=0.0.0.0
      - --port=8545
      - --chain-id=%d
      - --load-state=/data/anvil-state.json
      - --block-time=%d
      - --gas-limit=%d
    ports:
      - "8545:8545"
    volumes:
      - ./anvil-state.json:/data/anvil-state.json:ro
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "8545"]
      interval: 5s
      timeout: 3s
      retries: 10

  popsigner-lite:
    image: ghcr.io/bidon15/popsigner-lite:latest
    environment:
      - JSONRPC_PORT=8545
      - REST_API_PORT=3000
    ports:
      - "8555:8545"
      - "3000:3000"
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://localhost:3000/health"]
      interval: 5s
      timeout: 3s
      retries: 5

  localestia:
    image: ghcr.io/rollkit/local-celestia-devnet:latest
    environment:
      - CELESTIA_CUSTOM_NAMESPACE=000000000000ffff
    ports:
      - "26657:26657"
      - "26658:26658"
      - "7980:7980"
    depends_on:
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:7980/"]
      interval: 10s
      timeout: 5s
      retries: 10

  op-alt-da:
    image: ghcr.io/ethereum-optimism/op-alt-da:latest
    command:
      - --celestia.rpc=http://localestia:26658
      - --celestia.namespace-id=000000000000ffff
      - --celestia.auth-token=
      - --celestia.no-auth
      - --server.host=0.0.0.0
      - --server.port=3100
      - --log.level=info
    ports:
      - "3100:3100"
    depends_on:
      localestia:
        condition: service_healthy
    environment:
      - CELESTIA_CUSTOM_NAMESPACE=000000000000ffff
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://localhost:3100/health"]
      interval: 10s
      timeout: 5s
      retries: 10

  op-geth:
    image: us-docker.pkg.dev/oplabs-tools-artifacts/images/op-geth:latest
    command:
      - --datadir=/data
      - --http
      - --http.addr=0.0.0.0
      - --http.port=8545
      - --http.api=web3,debug,eth,net,engine
      - --http.corsdomain=*
      - --http.vhosts=*
      - --ws
      - --ws.addr=0.0.0.0
      - --ws.port=8546
      - --ws.api=web3,debug,eth,net,engine
      - --ws.origins=*
      - --authrpc.addr=0.0.0.0
      - --authrpc.port=8551
      - --authrpc.vhosts=*
      - --authrpc.jwtsecret=/config/jwt.txt
      - --syncmode=full
      - --gcmode=archive
      - --nodiscover
      - --maxpeers=0
      - --networkid=%d
      - --rollup.disabletxpoolgossip
      - --rollup.sequencerhttp=http://op-node:8545
    ports:
      - "9545:8545"
      - "9546:8546"
      - "8551:8551"
    volumes:
      - op_geth_data:/data
      - ./genesis.json:/config/genesis.json:ro
      - ./jwt.txt:/config/jwt.txt:ro
    depends_on:
      anvil:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "8545"]
      interval: 5s
      timeout: 3s
      retries: 10

  op-node:
    image: us-docker.pkg.dev/oplabs-tools-artifacts/images/op-node:latest
    command:
      - op-node
      - --l1=%s
      - --l2=http://op-geth:8551
      - --l2.jwt-secret=/config/jwt.txt
      - --rollup.config=/config/rollup.json
      - --rpc.addr=0.0.0.0
      - --rpc.port=8545
      - --p2p.disable
      - --rpc.enable-admin
      - --l1.trustrpc
      - --l1.rpckind=basic
    ports:
      - "7545:8545"
      - "7300:7300"
      - "6060:6060"
    volumes:
      - ./rollup.json:/config/rollup.json:ro
      - ./jwt.txt:/config/jwt.txt:ro
    depends_on:
      anvil:
        condition: service_healthy
      op-geth:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "8545"]
      interval: 5s
      timeout: 3s
      retries: 10

  op-batcher:
    image: us-docker.pkg.dev/oplabs-tools-artifacts/images/op-batcher:latest
    command:
      - op-batcher
      - --l1-eth-rpc=%s
      - --l2-eth-rpc=http://op-geth:8545
      - --rollup-rpc=http://op-node:8545
      - --poll-interval=1s
      - --sub-safety-margin=6
      - --num-confirmations=1
      - --safe-abort-nonce-too-low-count=3
      - --resubmission-timeout=30s
      - --rpc.addr=0.0.0.0
      - --rpc.port=8548
      - --rpc.enable-admin
      - --max-channel-duration=1
      - --target-num-frames=1
      - --txmgr.send-interval=1s
      - --pprof.enabled
      - --pprof.addr=0.0.0.0
      - --pprof.port=6060
      - --metrics.enabled
      - --metrics.addr=0.0.0.0
      - --metrics.port=7300
      - --plasma.enabled=true
      - --plasma.da-server=http://op-alt-da:3100
      - --signer.endpoint=http://popsigner-lite:8545
      - --signer.header=Authorization=Bearer psk_local_dev_00000000000000000000000000000000
      - --signer.address=%s
      - --batch-inbox-address=%s
    ports:
      - "8548:8548"
    depends_on:
      anvil:
        condition: service_healthy
      op-node:
        condition: service_healthy
      popsigner-lite:
        condition: service_healthy
      op-alt-da:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "8548"]
      interval: 5s
      timeout: 3s
      retries: 10

  op-proposer:
    image: us-docker.pkg.dev/oplabs-tools-artifacts/images/op-proposer:latest
    command:
      - op-proposer
      - --poll-interval=12s
      - --rpc.port=8560
      - --rollup-rpc=http://op-node:8545
      - --l2oo-address=%s
      - --l1-eth-rpc=%s
      - --signer.endpoint=http://popsigner-lite:8545
      - --signer.header=Authorization=Bearer psk_local_dev_00000000000000000000000000000000
      - --signer.address=%s
    ports:
      - "8560:8560"
    depends_on:
      anvil:
        condition: service_healthy
      op-node:
        condition: service_healthy
      popsigner-lite:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "8560"]
      interval: 5s
      timeout: 3s
      retries: 10

volumes:
  op_geth_data:
`,
		w.config.L1ChainID,
		w.config.BlockTime,
		w.config.GasLimit,
		w.config.ChainID,
		w.config.L1RPC,
		w.config.L1RPC,
		w.config.BatcherAddress,
		"0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000", // Batch inbox - see addresses.json
		"0xC0FFEEC0FFEEC0FFEEC0FFEEC0FFEEC0FFEE0000", // L2OutputOracle - see addresses.json
		w.config.L1RPC,
		w.config.ProposerAddress,
	)

	return []byte(compose), nil
}

// generateEnvExample generates the .env.example file.
func (w *ConfigWriter) generateEnvExample() ([]byte, error) {
	env := fmt.Sprintf(`# L1 Configuration (Anvil)
L1_CHAIN_ID=%d
L1_RPC_URL=http://anvil:8545

# L2 Configuration
L2_CHAIN_ID=%d
L2_CHAIN_NAME=%s

# POPSigner Configuration
POPSIGNER_RPC=http://popsigner-lite:8545
POPSIGNER_API_KEY=psk_local_dev_00000000000000000000000000000000

# Deployer Addresses
DEPLOYER_ADDRESS=%s
BATCHER_ADDRESS=%s
PROPOSER_ADDRESS=%s

# Celestia Configuration
CELESTIA_KEY_ID=%s
CELESTIA_NAMESPACE_ID=000000000000ffff
`,
		w.config.L1ChainID,
		w.config.ChainID,
		w.config.ChainName,
		w.config.DeployerAddress,
		w.config.BatcherAddress,
		w.config.ProposerAddress,
		w.celestiaKeyID,
	)

	return []byte(env), nil
}

// generateREADME generates the README.md file.
func (w *ConfigWriter) generateREADME() ([]byte, error) {
	readme := fmt.Sprintf(`# %s - POPKins Devnet Bundle

This bundle contains a complete, pre-deployed OP Stack + Celestia DA local devnet.

## What's Included

- **Anvil L1**: Ethereum L1 with pre-deployed OP Stack contracts
- **POPSigner-Lite**: Transaction signing service
- **Localestia**: Mock Celestia DA network
- **OP-ALT-DA**: Celestia DA server
- **OP-Geth**: L2 execution layer
- **OP-Node**: L2 consensus layer
- **OP-Batcher**: Batch submitter
- **OP-Proposer**: State root proposer

## Quick Start

1. Start the devnet:
   ` + "```bash\n   docker compose up -d\n   ```" + `

2. Wait for services to be healthy (~30-60 seconds):
   ` + "```bash\n   docker compose ps\n   ```" + `

3. Test L2 RPC:
   ` + "```bash\n   curl -X POST http://localhost:9545 \\\n     -H \"Content-Type: application/json\" \\\n     -d '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}'\n   ```" + `

## Configuration

- **L1 RPC**: http://localhost:8545
- **L2 RPC**: http://localhost:9545
- **Chain ID**: %d
- **Block Time**: %d seconds

## Contract Addresses

See ` + "`addresses.json`" + ` for all deployed contract addresses.

## Troubleshooting

View logs for any service:
` + "```bash\ndocker compose logs -f [service-name]\n```" + `

Stop the devnet:
` + "```bash\ndocker compose down\n```" + `

## Generated by POPSigner

This bundle was generated using POPSigner's POPKins Bundle Builder.
`,
		w.config.ChainName,
		w.config.ChainID,
		w.config.BlockTime,
	)

	return []byte(readme), nil
}
