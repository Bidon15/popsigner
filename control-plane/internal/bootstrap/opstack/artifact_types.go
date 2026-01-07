// Package opstack provides OP Stack deployment functionality.
package opstack

import (
	"encoding/json"
)

// OPStackArtifacts contains all artifacts from an OP Stack deployment.
type OPStackArtifacts struct {
	Genesis       json.RawMessage   `json:"genesis"`            // L2 genesis.json
	Rollup        json.RawMessage   `json:"rollup"`             // rollup.json configuration
	Addresses     ContractAddresses `json:"contract_addresses"` // Deployed contract addresses
	DeployConfig  json.RawMessage   `json:"deploy_config"`      // Original deployment config
	JWTSecret     string            `json:"jwt_secret"`         // Engine API JWT secret
	DockerCompose string            `json:"docker_compose"`     // Generated docker-compose.yml
	EnvExample    string            `json:"env_example"`        // .env.example template
	AltDAConfig   string            `json:"altda_config"`       // op-alt-da config.toml (Celestia)
	Readme        string            `json:"readme"`             // Bundle README.md
}

// ContractAddresses contains all deployed OP Stack contract addresses.
type ContractAddresses struct {
	// Superchain contracts
	SuperchainConfig string `json:"superchain_config,omitempty"`
	ProtocolVersions string `json:"protocol_versions,omitempty"`

	// Proxy contracts
	OptimismPortalProxy         string `json:"optimism_portal_proxy"`
	L1CrossDomainMessengerProxy string `json:"l1_cross_domain_messenger_proxy"`
	L1StandardBridgeProxy       string `json:"l1_standard_bridge_proxy"`
	L1ERC721BridgeProxy         string `json:"l1_erc721_bridge_proxy,omitempty"`
	SystemConfigProxy           string `json:"system_config_proxy"`
	DisputeGameFactoryProxy     string `json:"dispute_game_factory_proxy,omitempty"`
	AnchorStateRegistryProxy    string `json:"anchor_state_registry_proxy,omitempty"`
	DelayedWETHProxy            string `json:"delayed_weth_proxy,omitempty"`

	// Other contracts
	OptimismMintableERC20Factory string `json:"optimism_mintable_erc20_factory,omitempty"`
	AddressManager               string `json:"address_manager,omitempty"`
	BatchInbox                   string `json:"batch_inbox"`
	L2OutputOracle               string `json:"l2_output_oracle,omitempty"` // Legacy, pre-fault proofs
}

// RollupConfig represents the rollup.json configuration structure.
type RollupConfig struct {
	Genesis              RollupGenesisConfig `json:"genesis"`
	BlockTime            uint64              `json:"block_time"`
	MaxSequencerDrift    uint64              `json:"max_sequencer_drift"`
	SequencerWindowSize  uint64              `json:"sequencer_window_size"`
	ChannelTimeout       uint64              `json:"channel_timeout"`
	L1ChainID            uint64              `json:"l1_chain_id"`
	L2ChainID            uint64              `json:"l2_chain_id"`
	RegolithTime         *uint64             `json:"regolith_time,omitempty"`
	CanyonTime           *uint64             `json:"canyon_time,omitempty"`
	DeltaTime            *uint64             `json:"delta_time,omitempty"`
	EcotoneTime          *uint64             `json:"ecotone_time,omitempty"`
	FjordTime            *uint64             `json:"fjord_time,omitempty"`
	GraniteTime          *uint64             `json:"granite_time,omitempty"`
	HoloceneTime         *uint64             `json:"holocene_time,omitempty"`
	BatchInboxAddress    string              `json:"batch_inbox_address"`
	DepositContractAddr  string              `json:"deposit_contract_address"`
	L1SystemConfigAddr   string              `json:"l1_system_config_address"`
	ProtocolVersionsAddr string              `json:"protocol_versions_address,omitempty"`

	// Alt-DA configuration
	AltDAEnabled    bool   `json:"alt_da_enabled,omitempty"`
	DAChallengeAddr string `json:"da_challenge_address,omitempty"`
}

// RollupGenesisConfig represents the genesis portion of rollup.json.
type RollupGenesisConfig struct {
	L1           GenesisBlockRef `json:"l1"`
	L2           GenesisBlockRef `json:"l2"`
	L2Time       uint64          `json:"l2_time"`
	SystemConfig SystemConfig    `json:"system_config"`
}

// GenesisBlockRef represents a block reference in rollup genesis.
type GenesisBlockRef struct {
	Hash   string `json:"hash"`
	Number uint64 `json:"number"`
}

// SystemConfig represents the system configuration in rollup.json.
type SystemConfig struct {
	BatcherAddr       string `json:"batcherAddr"`
	Overhead          string `json:"overhead"`
	Scalar            string `json:"scalar"`
	GasLimit          uint64 `json:"gasLimit"`
	BaseFeeScalar     uint64 `json:"baseFeeScalar,omitempty"`
	BlobBaseFeeScalar uint64 `json:"blobBaseFeeScalar,omitempty"`
}
