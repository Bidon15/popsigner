package nitro

import "fmt"

// ============================================================================
// README Generator
// ============================================================================

// GenerateReadme creates a README.md with deployment instructions.
func GenerateReadme(config *DeployConfig, result *DeployResult) string {
	contracts := result.CoreContracts
	if contracts == nil {
		contracts = &CoreContracts{}
	}

	// Determine parent chain name
	parentChainName := "Sepolia"
	celestiaNetwork := "Mocha Testnet"
	if config.ParentChainID == 1 {
		parentChainName = "Mainnet"
		celestiaNetwork = "Mainnet"
	}

	return fmt.Sprintf(`# %s Nitro + Celestia DA Deployment

This bundle deploys an Arbitrum Orbit chain (%s) with Celestia DA on %s.

## Architecture

`+"`"+``+"`"+``+"`"+`
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Docker Compose                                      │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌─────────────────┐              ┌─────────────────────┐                       │
│  │  Nitro Sequencer │  ─────────► │  Celestia DAS Server │                       │
│  │  (offchainlabs/  │  daprovider │  (nitro-das-celestia)│                       │
│  │   nitro-node)    │  RPC        │  ClientTX            │                       │
│  └────────┬─────────┘              └──────────┬──────────┘                       │
│           │                                   │                                  │
└───────────┼───────────────────────────────────┼──────────────────────────────────┘
            │                                   │
            │ L1 RPC                            │ gRPC (write) + JSON-RPC (read)
            ▼                                   ▼
    ┌───────────────┐               ┌───────────────────────────┐
    │   %s          │               │   Celestia %s             │
    │   (L1)        │◄──────────────│   - Consensus (gRPC): writes  │
    │               │  Blobstream   │   - DA Bridge (RPC): reads    │
    │   Blobstream X│               └───────────────────────────┘
    │   Contract    │
    └───────────────┘
`+"`"+``+"`"+``+"`"+`

**Key Design: No Local Celestia Node Required!**

Uses ClientTX architecture:

- **Writes**: Direct gRPC to Celestia consensus nodes
- **Reads**: JSON-RPC to Celestia DA Bridge nodes
- **Signing**: Remote signing via PopSigner

## Chain Configuration

| Parameter       | Value                                      |
| --------------- | ------------------------------------------ |
| Chain ID        | %d                                         |
| Chain Name      | %s                                         |
| Parent Chain    | %s (%d)                                    |
| Rollup Contract | %s |
| Sequencer Inbox | %s |
| Bridge          | %s |

## Prerequisites

1. **Docker & Docker Compose** installed
2. **PopSigner credentials** (two sets):
   - **L1 (%s)**: For batch poster and validator transactions
   - **Celestia**: For blob submission transactions
3. **%s RPC** endpoint
4. **%s Beacon RPC** endpoint
5. **TIA tokens** on Celestia %s (for your PopSigner Celestia key)

## Directory Structure

`+"`"+``+"`"+``+"`"+`
%s-nitro-bundle/
├── config/
│   ├── chain-info.json          # Chain configuration (from SDK)
│   ├── core-contracts.json      # Deployed contract addresses
│   ├── node-config.json         # Original node config (reference only)
│   └── celestia-config.toml     # Celestia DAS server config
├── certs/                       # PopSigner mTLS certificates (for L1)
│   ├── client.crt
│   └── client.key
├── docker-compose.yaml          # Main compose file
├── .env.example                 # Environment template
└── README.md
`+"`"+``+"`"+``+"`"+`

## Quick Start

### 1. Set up environment variables

`+"`"+``+"`"+``+"`"+`bash
cp .env.example .env
# Edit .env with your values
`+"`"+``+"`"+``+"`"+`

### 2. Add PopSigner mTLS certificates (for L1 signing)

`+"`"+``+"`"+``+"`"+`bash
mkdir -p certs
cp /path/to/client.crt certs/
cp /path/to/client.key certs/
`+"`"+``+"`"+``+"`"+`

### 3. Configure Celestia namespace

Edit `+"`"+`config/celestia-config.toml`+"`"+`:

`+"`"+``+"`"+``+"`"+`toml
[celestia]
namespace_id = "YOUR_UNIQUE_NAMESPACE_HEX"
`+"`"+``+"`"+``+"`"+`

Generate a unique namespace: https://docs.celestia.org/tutorials/node-tutorial#namespaces

### 4. Fund your Celestia key with TIA

Your PopSigner Celestia key needs TIA for gas:

- Get testnet TIA from: https://faucet.celestia-mocha.com/

### 5. Start the stack

`+"`"+``+"`"+``+"`"+`bash
docker compose up -d
docker compose logs -f
`+"`"+``+"`"+``+"`"+`

## Blobstream Configuration (Fraud Proofs)

**This is critical for validators!**

Blobstream is a bridge that relays Celestia data root attestations to Ethereum. It enables:

- Fraud proofs for batches posted to Celestia
- Verification that batch data was actually available on Celestia

### How Blobstream Works

1. **Batch Poster** posts batch data to Celestia → gets `+"`"+`BlobPointer`+"`"+`
2. **Batch Poster** posts batch commitment to Sequencer Inbox
3. **Blobstream** relays Celestia block data roots to L1
4. **Validator** (during fraud proof) calls `+"`"+`GetProof`+"`"+` on celestia-das-server
5. **celestia-das-server** queries Blobstream contract for attestation
6. **celestia-das-server** returns proof data for on-chain verification

## Ports

| Service             | Port | Description           |
| ------------------- | ---- | --------------------- |
| Nitro Sequencer     | 8547 | HTTP RPC              |
| Nitro Sequencer     | 8548 | WebSocket RPC         |
| Nitro Sequencer     | 9642 | Metrics               |
| Nitro Sequencer     | 9644 | Feed (for full nodes) |
| Celestia DAS Server | 9876 | DA Provider RPC       |
| Celestia DAS Server | 6060 | Metrics               |

## Two PopSigner Keys Explained

This setup uses **two separate PopSigner keys**:

### 1. L1 (%s) PopSigner

- **Used by**: Nitro batch poster and validator
- **For**: Signing L1 transactions (batch submissions, staking)
- **Config**: `+"`"+`POPSIGNER_MTLS_URL`+"`"+` + mTLS certificates
- **Funds needed**: ETH on %s

### 2. Celestia PopSigner

- **Used by**: Celestia DAS server
- **For**: Signing Celestia blob transactions
- **Config**: `+"`"+`POPSIGNER_CELESTIA_API_KEY`+"`"+` + `+"`"+`POPSIGNER_CELESTIA_KEY_ID`+"`"+`
- **Funds needed**: TIA on Celestia %s

## Nitro Image Options

### Option 1: Build from Local Source (Recommended)

`+"`"+``+"`"+``+"`"+`bash
cd /path/to/nitro

# Build the Docker image
make docker

# Or build specific target
docker build -t nitro-node:local --target nitro-node .
`+"`"+``+"`"+``+"`"+`

Then update your `+"`"+`.env`+"`"+`:

`+"`"+``+"`"+``+"`"+`bash
echo "NITRO_IMAGE=nitro-node:local" >> .env
`+"`"+``+"`"+``+"`"+`

### Option 2: Use Official Release

When `+"`"+`offchainlabs/nitro-node:v3.9.0`+"`"+` is published on Docker Hub, update:

`+"`"+``+"`"+``+"`"+`bash
echo "NITRO_IMAGE=offchainlabs/nitro-node:v3.9.0" >> .env
`+"`"+``+"`"+``+"`"+`

## Troubleshooting

### Check service health

`+"`"+``+"`"+``+"`"+`bash
docker compose ps
docker compose logs celestia-das-server
docker compose logs nitro-sequencer
`+"`"+``+"`"+``+"`"+`

### Test Celestia DAS server

`+"`"+``+"`"+``+"`"+`bash
curl -X POST http://localhost:9876 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"daprovider_getSupportedHeaderBytes","params":[],"id":1}'
`+"`"+``+"`"+``+"`"+`

### Batch poster not submitting to Celestia

1. Check if Celestia PopSigner key is funded with TIA
2. Check celestia-das-server logs: `+"`"+`docker compose logs -f celestia-das-server`+"`"+`
3. Verify Celestia endpoint connectivity

### Blobstream proof failures

1. Verify `+"`"+`blobstream_addr`+"`"+` is correct for your parent chain
2. Check that `+"`"+`eth_rpc`+"`"+` can reach the parent chain
3. Blobstream may need time to relay attestations (~1 hour)

## Resources

- [Arbitrum Orbit Docs](https://docs.arbitrum.io/launch-orbit-chain/orbit-gentle-introduction)
- [Celestia DA Docs](https://docs.celestia.org/)
- [nitro-das-celestia](https://github.com/celestiaorg/nitro-das-celestia)
- [Blobstream Docs](https://docs.celestia.org/how-to-guides/blobstream)
- [Celestia Mocha Faucet](https://faucet.celestia-mocha.com/)
- [PopSigner](https://github.com/Bidon15/popsigner)
`, config.ChainName, config.ChainName, parentChainName,
		parentChainName, celestiaNetwork,
		config.ChainID, config.ChainName, parentChainName, config.ParentChainID,
		contracts.Rollup, contracts.SequencerInbox, contracts.Bridge,
		parentChainName, parentChainName, parentChainName, celestiaNetwork,
		config.ChainName,
		parentChainName, parentChainName, celestiaNetwork)
}
