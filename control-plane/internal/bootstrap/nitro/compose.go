package nitro

import "fmt"

// ============================================================================
// Docker Compose Generator
// ============================================================================

// GenerateDockerCompose creates a docker-compose.yaml for Nitro + Celestia DA.
func GenerateDockerCompose(config *DeployConfig, result *DeployResult) string {
	return fmt.Sprintf(`version: '3.8'

# =============================================================================
# Nitro + Celestia DA Docker Compose
# Chain: %s (ID: %d) on Parent Chain %d
# Uses ClientTX: Direct connection to Celestia infrastructure (no local node)
# =============================================================================

services:
  # ===========================================================================
  # Celestia DAS Server
  # Translates between Nitro DA provider protocol and Celestia
  # Uses ClientTX with remote signer (popsigner) for blob submission
  # ===========================================================================
  celestia-das-server:
    image: ${NITRO_DAS_IMAGE:-ghcr.io/celestiaorg/nitro-das-celestia:v0.7.0}
    container_name: celestia-das-server
    restart: unless-stopped
    command:
      - --config
      - /config/celestia-config.toml
    ports:
      - "9876:9876" # DA provider RPC (Nitro connects here)
      - "6060:6060" # Metrics (optional)
    volumes:
      - ./config/celestia-config.toml:/config/celestia-config.toml:ro
    environment:
      # Remote signer (popsigner) credentials
      - POPSIGNER_API_KEY=${POPSIGNER_CELESTIA_API_KEY}
      # Parent chain RPC for Blobstream validation (fraud proofs)
      - ETH_RPC_URL=${L1_RPC_URL}
    # Healthcheck ensures celestia-das-server is ready before nitro-sequencer starts
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:9876/health"]
      interval: 5s
      timeout: 3s
      retries: 5
      start_period: 10s
    networks:
      - nitro-network

  # ===========================================================================
  # Nitro Sequencer Node (Batch Poster + Validator)
  # Uses official Offchain Labs image with DA API interface (PR #3949, #3237)
  #
  # Image options:
  # 1. Use latest release with DA support: offchainlabs/nitro-node:v3.9.0 (when released)
  # 2. Build from your local source (v3.9.0-rc.1): see README
  # 3. Use Celestia fork (legacy): ghcr.io/celestiaorg/nitro:v3.6.8
  # ===========================================================================
  nitro-sequencer:
    # TODO: Update to official release once v3.9.0 is published on Docker Hub
    # Your local repo is at v3.9.0-rc.1 - you may need to build this image locally
    # docker build -t nitro-node:local --target nitro-node .
    image: ${NITRO_IMAGE:-offchainlabs/nitro-node:v3.5.4-8de7ff5}
    container_name: nitro-sequencer
    restart: unless-stopped
    depends_on:
      celestia-das-server:
        condition: service_healthy  # Wait for healthcheck to pass, not just container start
    ports:
      - "8547:8547" # HTTP RPC
      - "8548:8548" # WebSocket RPC
      - "9642:9642" # Metrics
      - "9644:9644" # Feed
    volumes:
      - ./config:/config:ro
      - ./certs:/certs:ro
      - nitro-data:/home/user/.arbitrum
      - nitro-keystore:/home/user/l1keystore
    environment:
      # L1 Connections - WSS is recommended for sequencers for real-time updates
      - L1_RPC_URL=${L1_RPC_URL} # Can be HTTP or WSS (e.g., wss://sepolia.infura.io/ws/v3/KEY)
      - L1_BEACON_URL=${L1_BEACON_URL} # Beacon Chain API (HTTP)
      - POPSIGNER_MTLS_URL=${POPSIGNER_MTLS_URL}
      # External signer addresses (managed by PopSigner)
      - BATCH_POSTER_ADDRESS=${BATCH_POSTER_ADDRESS}
      - STAKER_ADDRESS=${STAKER_ADDRESS}
    command:
      # -------------------------------------------------------------------------
      # Core Chain Configuration
      # -------------------------------------------------------------------------
      - --chain.id=%d
      - --chain.name=%s
      - --chain.info-files=/config/chain-info.json
      # -------------------------------------------------------------------------
      # Parent Chain (L1) Connection
      # URL can be HTTP or WSS - WSS recommended for sequencers:
      #   HTTP: https://sepolia.infura.io/v3/YOUR_KEY
      #   WSS:  wss://sepolia.infura.io/ws/v3/YOUR_KEY
      # -------------------------------------------------------------------------
      - --parent-chain.connection.url=${L1_RPC_URL}
      # Beacon Chain API for EIP-4844 blob data (always HTTP)
      - --parent-chain.blob-client.beacon-url=${L1_BEACON_URL}
      # -------------------------------------------------------------------------
      # HTTP/WS RPC Configuration
      # -------------------------------------------------------------------------
      - --http.addr=0.0.0.0
      - --http.port=8547
      - --http.api=eth,net,web3,arb,debug
      - --http.vhosts=*
      - --http.corsdomain=*
      - --ws.addr=0.0.0.0
      - --ws.port=8548
      - --ws.api=eth,net,web3
      - --ws.origins=*
      # -------------------------------------------------------------------------
      # Sequencer Configuration
      # For single-sequencer setup, disable coordinator requirement
      # -------------------------------------------------------------------------
      - --node.sequencer=true
      - --execution.sequencer.enable=true
      - --node.delayed-sequencer.enable=true
      - --node.dangerous.no-sequencer-coordinator=true
      # -------------------------------------------------------------------------
      # Celestia DA Provider Configuration (PR #3949)
      # External DA provider connects to celestia-das-server
      # -------------------------------------------------------------------------
      - --node.da.external-provider.enable=true
      - --node.da.external-provider.with-writer=true
      - --node.da.external-provider.rpc.url=http://celestia-das-server:9876
      # -------------------------------------------------------------------------
      # Batch Poster Configuration (uses PopSigner for L1 tx signing)
      # -------------------------------------------------------------------------
      - --node.batch-poster.enable=true
      - --node.batch-poster.data-poster.external-signer.url=${POPSIGNER_MTLS_URL}
      - --node.batch-poster.data-poster.external-signer.address=${BATCH_POSTER_ADDRESS}
      - --node.batch-poster.data-poster.external-signer.method=eth_signTransaction
      - --node.batch-poster.data-poster.external-signer.client-cert=/certs/client.crt
      - --node.batch-poster.data-poster.external-signer.client-private-key=/certs/client.key
      - --node.batch-poster.data-poster.external-signer.root-ca=/certs/ca.crt
      # -------------------------------------------------------------------------
      # Staker/Validator Configuration (BOLD Protocol)
      # IMPORTANT: BOLD requires WETH (not native ETH) for staking!
      # Before starting, ensure your STAKER_ADDRESS has:
      #   1. WETH tokens (wrap ETH via WETH.deposit())
      #   2. Approved ChallengeManager to spend WETH
      # Stake amount: ~0.1 ETH equivalent per assertion level
      # -------------------------------------------------------------------------
      - --node.staker.enable=true
      - --node.staker.strategy=MakeNodes
      - --node.staker.data-poster.external-signer.url=${POPSIGNER_MTLS_URL}
      - --node.staker.data-poster.external-signer.address=${STAKER_ADDRESS}
      - --node.staker.data-poster.external-signer.method=eth_signTransaction
      - --node.staker.data-poster.external-signer.client-cert=/certs/client.crt
      - --node.staker.data-poster.external-signer.client-private-key=/certs/client.key
      - --node.staker.data-poster.external-signer.root-ca=/certs/ca.crt
      # -------------------------------------------------------------------------
      # Feed Output (for full nodes to subscribe)
      # -------------------------------------------------------------------------
      - --node.feed.output.enable=true
      - --node.feed.output.addr=0.0.0.0
      - --node.feed.output.port=9644
      # -------------------------------------------------------------------------
      # Metrics
      # -------------------------------------------------------------------------
      - --metrics
      - --metrics-server.addr=0.0.0.0
      - --metrics-server.port=9642
    networks:
      - nitro-network

networks:
  nitro-network:
    driver: bridge

volumes:
  nitro-data:
  nitro-keystore:
`, config.ChainName, config.ChainID, config.ParentChainID, config.ChainID, config.ChainName)
}

// ============================================================================
// Celestia Config Generator
// ============================================================================

// GenerateCelestiaConfig creates a celestia-config.toml for the DAS server.
func GenerateCelestiaConfig(config *DeployConfig, result *DeployResult) string {
	// Determine network and blobstream address based on parent chain
	celestiaNetwork := "mocha-4"
	blobstreamAddr := "0xF0c6429ebAB2e7DC6e05DaFB61128bE21f13cb1e" // Sepolia Blobstream SP1
	if config.ParentChainID == 1 {
		celestiaNetwork = "celestia"
		blobstreamAddr = "0x7Cf3876F681Dbb6EdA8f6FfC45D66B996Df08fAe" // Mainnet Blobstream
	}

	return fmt.Sprintf(`# =============================================================================
# Celestia DAS Server Configuration
# Chain: %s (ID: %d) on Parent Chain %d
#
# Uses ClientTX architecture:
# - Reader: Connects to Celestia DA Bridge node (JSON-RPC) for blob reads
# - Writer: Connects to Celestia Core node (gRPC) for blob submission
# - No local Celestia node required!
# =============================================================================

[server]
rpc_addr = "0.0.0.0"
rpc_port = 9876
rpc_body_limit = 0
read_timeout = "30s"
read_header_timeout = "10s"
write_timeout = "30s"
idle_timeout = "120s"

[celestia]
# Namespace ID for blob operations (hex string)
# IMPORTANT: Use a unique namespace for your chain!
# Generate one at: https://docs.celestia.org/tutorials/node-tutorial#namespaces
namespace_id = "YOUR_UNIQUE_NAMESPACE_HEX"

# Gas settings for blob transactions
gas_price = 0.01
gas_multiplier = 1.01

# Celestia network: "celestia" for mainnet, "mocha-4" for testnet
network = "%s"

# Enable blob submission (writer mode) - required for batch poster
with_writer = true
noop_writer = false

# Cache cleanup interval
cache_time = "30m"

# -----------------------------------------------------------------------------
# Reader configuration
# Connects to Celestia DA Bridge node (JSON-RPC) for reading blobs
# Uses public Celestia infrastructure - no local node needed!
# -----------------------------------------------------------------------------
[celestia.reader]
# Public Mocha testnet DA Bridge node
# You can also use providers like QuickNode: https://www.quicknode.com/docs/celestia
rpc = "https://YOUR_CELESTIA_RPC_ENDPOINT"
# Auth token (if using a provider like QuickNode)
auth_token = ""
enable_tls = true

# -----------------------------------------------------------------------------
# Writer configuration
# Connects to Celestia Core node (gRPC) for blob submission
# Uses ClientTX - direct gRPC connection to consensus node
# -----------------------------------------------------------------------------
[celestia.writer]
# Public Mocha testnet consensus node gRPC
core_grpc = "YOUR_CELESTIA_GRPC_ENDPOINT:9090"
core_token = ""
enable_tls = true

# -----------------------------------------------------------------------------
# Signer configuration
# Uses remote signing via PopSigner for Celestia transaction signing
# This is separate from the L1 PopSigner used by Nitro!
# -----------------------------------------------------------------------------
[celestia.signer]
type = "remote"

[celestia.signer.remote]
# PopSigner API key for Celestia signing (from environment variable)
api_key = "${POPSIGNER_CELESTIA_API_KEY}"
# PopSigner Key ID for your Celestia key
key_id = "${POPSIGNER_CELESTIA_KEY_ID}"
# Custom PopSigner endpoint (optional, leave empty for default)
base_url = ""

# Alternative: Local signer (if not using PopSigner for Celestia)
# Uncomment below and comment out the remote section above
# [celestia.signer]
# type = "local"
# [celestia.signer.local]
# key_name = "%s-celestia-key"
# key_path = ""  # Uses default: ~/.celestia-light-mocha-4/keys
# backend = "test"

# -----------------------------------------------------------------------------
# Retry configuration for failed blob operations
# -----------------------------------------------------------------------------
[celestia.retry]
max_retries = 5
initial_backoff = "10s"
max_backoff = "120s"
backoff_factor = 2.0

# -----------------------------------------------------------------------------
# Validator configuration for Blobstream proof validation (FRAUD PROOFS)
# This is CRITICAL for validators - enables fraud proofs with Celestia DA
# PR #3237: Custom DA Complete Fraud Proof Support
# -----------------------------------------------------------------------------
[celestia.validator]
# Parent chain RPC endpoint
# This is used to query Blobstream contract for data root attestations
eth_rpc = "${ETH_RPC_URL}"

# Blobstream X contract address
# Check latest address: https://docs.celestia.org/how-to-guides/blobstream#deployed-contracts
blobstream_addr = "%s"

# Seconds between Blobstream event polling (for catching up proofs)
sleep_time = 3600

# -----------------------------------------------------------------------------
# Fallback to Arbitrum AnyTrust DAS (optional)
# Enable if you want to fall back to AnyTrust when Celestia fails
# -----------------------------------------------------------------------------
[fallback]
enabled = false
das_rpc = ""

# -----------------------------------------------------------------------------
# Logging configuration
# -----------------------------------------------------------------------------
[logging]
level = "INFO"
type = "plaintext"

# -----------------------------------------------------------------------------
# Metrics and profiling configuration
# -----------------------------------------------------------------------------
[metrics]
enabled = true
addr = "0.0.0.0"
port = 6060
pprof = false
pprof_addr = "127.0.0.1"
pprof_port = 6061
`, config.ChainName, config.ChainID, config.ParentChainID, celestiaNetwork, config.ChainName, blobstreamAddr)
}
