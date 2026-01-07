package nitro

import (
	"fmt"
	"strings"
)

// ============================================================================
// Environment File Generators
// ============================================================================

// GenerateEnvExample creates a .env.example file.
func GenerateEnvExample(config *DeployConfig, result *DeployResult) string {
	// Determine parent chain name
	parentChainName := "Sepolia"
	beaconURL := "https://ethereum-sepolia-beacon-api.publicnode.com"
	rpcExample := "wss://sepolia.infura.io/ws/v3/YOUR_KEY"
	if config.ParentChainID == 1 {
		parentChainName = "Mainnet"
		beaconURL = "https://ethereum-mainnet-beacon-api.publicnode.com"
		rpcExample = "wss://mainnet.infura.io/ws/v3/YOUR_KEY"
	}

	// Get the batch poster address from config.BatchPosters
	batchPosterAddress := ""
	if len(config.BatchPosters) > 0 {
		batchPosterAddress = config.BatchPosters[0]
	}
	if batchPosterAddress == "" {
		batchPosterAddress = "REPLACE_WITH_YOUR_BATCH_POSTER_ADDRESS"
	}

	// Get staker address from config.Validators (NOT same as batch poster!)
	stakerAddress := ""
	if len(config.Validators) > 0 {
		stakerAddress = config.Validators[0]
	}
	if stakerAddress == "" {
		stakerAddress = "REPLACE_WITH_YOUR_STAKER_ADDRESS"
	}

	return fmt.Sprintf(`# =============================================================================
# %s Nitro + Celestia DA Environment Configuration
# Chain ID: %d | Parent Chain: %s (%d)
# =============================================================================

# =============================================================================
# PARENT CHAIN (%s) - L1 CONNECTIONS
# =============================================================================

# Main L1 RPC endpoint
# WSS is RECOMMENDED for sequencers (real-time updates, lower latency)
# Examples:
#   Infura WSS:  wss://%s.infura.io/ws/v3/YOUR_KEY
#   Alchemy WSS: wss://eth-%s.g.alchemy.com/v2/YOUR_KEY
#   Infura HTTP: https://%s.infura.io/v3/YOUR_KEY
#
L1_RPC_URL=%s

# Beacon Chain API for EIP-4844 blob data (always HTTP)
# Public options:
#   %s
#
L1_BEACON_URL=%s

# =============================================================================
# POPSIGNER FOR L1 (%s) - Batch Poster & Validator Signing
# Used for signing L1 transactions (batch submissions, staking)
# =============================================================================

# PopSigner mTLS endpoint for transaction signing
POPSIGNER_MTLS_URL=https://rpc-mtls.popsigner.com:8546

# Note: mTLS certificates are included in ./certs/
#   - ./certs/client.crt  (auto-generated during deployment)
#   - ./certs/client.key  (auto-generated during deployment)
#   - ./certs/ca.crt      (CA certificate for verification)

# =============================================================================
# EXTERNAL SIGNER ADDRESSES
# These are the Ethereum addresses that PopSigner will sign transactions for
# =============================================================================

# Batch Poster Address - the address used to submit batches to L1
# This should be the address associated with your PopSigner key
BATCH_POSTER_ADDRESS=%s

# Staker Address - the address used for staking/validation (if staker enabled)
# Can be same as batch poster or a different address
STAKER_ADDRESS=%s

# =============================================================================
# POPSIGNER FOR CELESTIA - Blob Submission Signing
# Used for signing Celestia blob transactions (SEPARATE from L1 signer!)
# =============================================================================

# PopSigner API key for Celestia
POPSIGNER_CELESTIA_API_KEY=REPLACE_WITH_YOUR_CELESTIA_API_KEY

# PopSigner Key ID for your Celestia signing key
POPSIGNER_CELESTIA_KEY_ID=REPLACE_WITH_YOUR_CELESTIA_KEY_ID

# =============================================================================
# NITRO IMAGE
# =============================================================================

# After running 'make docker' in nitro repo, use:
NITRO_IMAGE=nitro-node-dev:latest
NITRO_DAS_IMAGE=ghcr.io/celestiaorg/nitro-das-celestia:v0.7.0

# Alternative options:
#   nitro-node:local     (if you tagged it with :local)
#   nitro-node-slim      (minimal image)
#   ghcr.io/celestiaorg/nitro:v3.6.8  (Celestia fork, legacy)

# =============================================================================
# OPTIONAL: Additional Configuration
# =============================================================================

# Uncomment if you need custom Celestia namespace
# CELESTIA_NAMESPACE_ID=YOUR_NAMESPACE_HEX

# Uncomment for custom log level
# LOG_LEVEL=INFO
`, config.ChainName, config.ChainID, parentChainName, config.ParentChainID,
		parentChainName,
		strings.ToLower(parentChainName), strings.ToLower(parentChainName), strings.ToLower(parentChainName),
		rpcExample, beaconURL, beaconURL, parentChainName,
		batchPosterAddress, stakerAddress)
}

// GenerateEnv creates a ready-to-use .env file with actual values.
// This removes the need for users to manually copy/edit .env.example.
func GenerateEnv(config *DeployConfig, result *DeployResult) string {
	// Determine parent chain name and beacon URL
	parentChainName := "Sepolia"
	beaconURL := "https://ethereum-sepolia-beacon-api.publicnode.com"
	if config.ParentChainID == 1 {
		parentChainName = "Mainnet"
		beaconURL = "https://ethereum-mainnet-beacon-api.publicnode.com"
	}

	// Use actual RPC URL from config, or default to public endpoint
	rpcURL := config.ParentChainRpc
	if rpcURL == "" {
		if config.ParentChainID == 1 {
			rpcURL = "https://ethereum-rpc.publicnode.com"
		} else {
			rpcURL = "https://ethereum-sepolia-rpc.publicnode.com"
		}
	}

	// Get the batch poster address from config.BatchPosters
	batchPosterAddress := ""
	if len(config.BatchPosters) > 0 {
		batchPosterAddress = config.BatchPosters[0]
	}
	if batchPosterAddress == "" {
		batchPosterAddress = "REPLACE_WITH_YOUR_BATCH_POSTER_ADDRESS"
	}

	// Get staker address from config.Validators (NOT same as batch poster!)
	stakerAddress := ""
	if len(config.Validators) > 0 {
		stakerAddress = config.Validators[0]
	}
	if stakerAddress == "" {
		stakerAddress = "REPLACE_WITH_YOUR_STAKER_ADDRESS"
	}

	// Celestia config - these would come from the deployment config if set
	celestiaAPIKey := "REPLACE_WITH_YOUR_CELESTIA_API_KEY"
	celestiaKeyID := "REPLACE_WITH_YOUR_CELESTIA_KEY_ID"

	return fmt.Sprintf(`# =============================================================================
# %s Nitro + Celestia DA Environment Configuration
# Chain ID: %d | Parent Chain: %s (%d)
# Generated by PopSigner - Ready to use!
# =============================================================================

# =============================================================================
# PARENT CHAIN (%s) - L1 CONNECTIONS
# =============================================================================

# Main L1 RPC endpoint (configured during deployment)
L1_RPC_URL=%s

# Beacon Chain API for EIP-4844 blob data
L1_BEACON_URL=%s

# =============================================================================
# POPSIGNER FOR L1 (%s) - Batch Poster & Validator Signing
# =============================================================================

# PopSigner mTLS endpoint for transaction signing
POPSIGNER_MTLS_URL=https://rpc-mtls.popsigner.com:8546

# =============================================================================
# EXTERNAL SIGNER ADDRESSES (configured during deployment)
# =============================================================================

# Batch Poster Address - the address used to submit batches to L1
BATCH_POSTER_ADDRESS=%s

# Staker Address - the address used for staking/validation
STAKER_ADDRESS=%s

# =============================================================================
# POPSIGNER FOR CELESTIA - Blob Submission Signing
# =============================================================================

# PopSigner API key for Celestia (get from PopSigner dashboard)
POPSIGNER_CELESTIA_API_KEY=%s

# PopSigner Key ID for your Celestia signing key
POPSIGNER_CELESTIA_KEY_ID=%s

# =============================================================================
# NITRO IMAGE
# =============================================================================

NITRO_IMAGE=nitro-node-dev:latest
NITRO_DAS_IMAGE=ghcr.io/celestiaorg/nitro-das-celestia:v0.7.0
`, config.ChainName, config.ChainID, parentChainName, config.ParentChainID,
		parentChainName, rpcURL, beaconURL, parentChainName,
		batchPosterAddress, stakerAddress,
		celestiaAPIKey, celestiaKeyID)
}
