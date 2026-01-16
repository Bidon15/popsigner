package popdeployer

// DeploymentConfig holds configuration for a POPKins devnet bundle deployment.
type DeploymentConfig struct {
	// User-configurable parameters
	ChainID   uint64 `json:"chain_id"`
	ChainName string `json:"chain_name"`

	// Hardcoded parameters (populated by orchestrator)
	L1ChainID         uint64 `json:"l1_chain_id"`          // 31337 (Anvil)
	L1RPC             string `json:"l1_rpc"`               // http://localhost:8545
	DeployerAddress   string `json:"deployer_address"`     // anvil-0
	BatcherAddress    string `json:"batcher_address"`      // anvil-1
	ProposerAddress   string `json:"proposer_address"`     // anvil-2
	BlockTime         uint64 `json:"block_time"`           // 2 seconds
	GasLimit          uint64 `json:"gas_limit"`            // 30000000
	PopSignerRPC      string `json:"popsigner_rpc"`        // http://localhost:8555
	PopSignerAPIKey   string `json:"popsigner_api_key"`    // psk_local_dev_...
}
