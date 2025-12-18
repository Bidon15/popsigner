package opstack

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// DeploymentConfig contains configuration for an OP Stack deployment.
type DeploymentConfig struct {
	// Chain identification
	ChainID   uint64 `json:"chain_id"`
	ChainName string `json:"chain_name"`

	// L1 connection
	L1ChainID uint64 `json:"l1_chain_id"`
	L1RPC     string `json:"l1_rpc"`

	// POPSigner configuration
	POPSignerEndpoint string `json:"popsigner_endpoint"`
	POPSignerAPIKey   string `json:"popsigner_api_key"`

	// Deployer address (derived from POPSigner key)
	DeployerAddress string `json:"deployer_address"`

	// Chain parameters
	BlockTime           uint64 `json:"block_time"`            // seconds (default: 2)
	MaxSequencerDrift   uint64 `json:"max_sequencer_drift"`   // seconds (default: 600)
	SequencerWindowSize uint64 `json:"sequencer_window_size"` // blocks (default: 3600)
	GasLimit            uint64 `json:"gas_limit"`             // (default: 30000000)

	// Data Availability
	DAType        string `json:"da_type"`         // "calldata", "celestia", or "alt-da"
	CelestiaRPC   string `json:"celestia_rpc,omitempty"`
	CelestiaToken string `json:"celestia_token,omitempty"`

	// Alt-DA (Generic Commitment)
	UseAltDA         bool   `json:"use_alt_da,omitempty"`
	DACommitmentType string `json:"da_commitment_type,omitempty"` // "GenericCommitment" or "KeccakCommitment"

	// Fee recipients
	BaseFeeVaultRecipient      string `json:"base_fee_vault_recipient,omitempty"`
	L1FeeVaultRecipient        string `json:"l1_fee_vault_recipient,omitempty"`
	SequencerFeeVaultRecipient string `json:"sequencer_fee_vault_recipient,omitempty"`

	// Role addresses (optional - defaults to deployer if not set)
	BatcherAddress   string `json:"batcher_address,omitempty"`
	ProposerAddress  string `json:"proposer_address,omitempty"`
	SequencerAddress string `json:"sequencer_address,omitempty"`
	ChallengerAddress string `json:"challenger_address,omitempty"`

	// Funding (optional - for funding check)
	RequiredFundingWei *big.Int `json:"-"` // Not serialized, set programmatically
}

// Validate checks that required fields are set and values are valid.
func (c *DeploymentConfig) Validate() error {
	if c.ChainID == 0 {
		return fmt.Errorf("chain_id is required")
	}
	if c.ChainName == "" {
		return fmt.Errorf("chain_name is required")
	}
	if c.L1ChainID == 0 {
		return fmt.Errorf("l1_chain_id is required")
	}
	if c.L1RPC == "" {
		return fmt.Errorf("l1_rpc is required")
	}
	if c.POPSignerEndpoint == "" {
		return fmt.Errorf("popsigner_endpoint is required")
	}
	if c.POPSignerAPIKey == "" {
		return fmt.Errorf("popsigner_api_key is required")
	}
	if c.DeployerAddress == "" {
		return fmt.Errorf("deployer_address is required")
	}

	// Validate DA type
	switch c.DAType {
	case "", "calldata", "celestia", "alt-da":
		// Valid
	default:
		return fmt.Errorf("invalid da_type: %s (must be calldata, celestia, or alt-da)", c.DAType)
	}

	// If using Celestia, require RPC and token
	if c.DAType == "celestia" {
		if c.CelestiaRPC == "" {
			return fmt.Errorf("celestia_rpc is required when da_type is celestia")
		}
	}

	return nil
}

// ApplyDefaults sets default values for optional fields.
func (c *DeploymentConfig) ApplyDefaults() {
	if c.BlockTime == 0 {
		c.BlockTime = 2 // 2 seconds
	}
	if c.MaxSequencerDrift == 0 {
		c.MaxSequencerDrift = 600 // 10 minutes
	}
	if c.SequencerWindowSize == 0 {
		c.SequencerWindowSize = 3600 // 1 hour of blocks
	}
	if c.GasLimit == 0 {
		c.GasLimit = 30000000 // 30M gas
	}
	if c.DAType == "" {
		c.DAType = "calldata" // Default to calldata
	}
	if c.DACommitmentType == "" && c.UseAltDA {
		c.DACommitmentType = "GenericCommitment"
	}

	// Default fee recipients to deployer
	if c.BaseFeeVaultRecipient == "" {
		c.BaseFeeVaultRecipient = c.DeployerAddress
	}
	if c.L1FeeVaultRecipient == "" {
		c.L1FeeVaultRecipient = c.DeployerAddress
	}
	if c.SequencerFeeVaultRecipient == "" {
		c.SequencerFeeVaultRecipient = c.DeployerAddress
	}

	// Default role addresses to deployer
	if c.BatcherAddress == "" {
		c.BatcherAddress = c.DeployerAddress
	}
	if c.ProposerAddress == "" {
		c.ProposerAddress = c.DeployerAddress
	}
	if c.SequencerAddress == "" {
		c.SequencerAddress = c.DeployerAddress
	}
	if c.ChallengerAddress == "" {
		c.ChallengerAddress = c.DeployerAddress
	}

	// Default required funding based on network
	if c.RequiredFundingWei == nil {
		if c.L1ChainID == 1 {
			// Mainnet: 5 ETH
			c.RequiredFundingWei = new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18))
		} else {
			// Testnet: 1 ETH
			c.RequiredFundingWei = big.NewInt(1e18)
		}
	}
}

// ParseConfig parses and validates a deployment configuration from JSON.
func ParseConfig(raw json.RawMessage) (*DeploymentConfig, error) {
	var cfg DeploymentConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.ApplyDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// L1ChainIDBig returns L1ChainID as *big.Int.
func (c *DeploymentConfig) L1ChainIDBig() *big.Int {
	return new(big.Int).SetUint64(c.L1ChainID)
}

// ChainIDBig returns ChainID as *big.Int.
func (c *DeploymentConfig) ChainIDBig() *big.Int {
	return new(big.Int).SetUint64(c.ChainID)
}

// IsTestnet returns true if the L1 chain is a testnet.
func (c *DeploymentConfig) IsTestnet() bool {
	switch c.L1ChainID {
	case 1: // Ethereum Mainnet
		return false
	default:
		return true
	}
}

