package opstack

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// extractGenesis retrieves the genesis.json from the database.
func (e *ArtifactExtractor) extractGenesis(ctx context.Context, deploymentID uuid.UUID) (json.RawMessage, error) {
	// Try new artifact name first (genesis.json)
	artifact, err := e.repo.GetArtifact(ctx, deploymentID, "genesis.json")
	if err != nil {
		return nil, fmt.Errorf("get genesis artifact: %w", err)
	}
	if artifact != nil && len(artifact.Content) > 0 {
		return artifact.Content, nil
	}

	// Fallback to old name (genesis) for backwards compatibility
	artifact, err = e.repo.GetArtifact(ctx, deploymentID, "genesis")
	if err != nil {
		return nil, fmt.Errorf("get genesis artifact (legacy): %w", err)
	}
	if artifact == nil {
		return nil, fmt.Errorf("genesis artifact not found")
	}
	return artifact.Content, nil
}

// extractRollupConfig retrieves the rollup.json from the database.
// It prefers the saved artifact from op-deployer (which uses inspect.GenesisAndRollup)
// and falls back to building from deployment config if not found.
func (e *ArtifactExtractor) extractRollupConfig(ctx context.Context, deploymentID uuid.UUID, cfg *DeploymentConfig) (json.RawMessage, error) {
	// Try new artifact name first (rollup.json)
	artifact, err := e.repo.GetArtifact(ctx, deploymentID, "rollup.json")
	if err != nil {
		return nil, fmt.Errorf("get rollup config artifact: %w", err)
	}
	if artifact != nil && len(artifact.Content) > 0 {
		return artifact.Content, nil
	}

	// Fallback to old name (rollup_config)
	artifact, err = e.repo.GetArtifact(ctx, deploymentID, "rollup_config")
	if err != nil {
		return nil, fmt.Errorf("get rollup config artifact (legacy): %w", err)
	}
	if artifact != nil && len(artifact.Content) > 0 {
		// Return the saved rollup config directly (it's already in the correct format)
		return artifact.Content, nil
	}

	// Fallback: build from deployment config (legacy path)
	rollup, err := e.buildRollupConfig(ctx, deploymentID, cfg)
	if err != nil {
		return nil, fmt.Errorf("build rollup config: %w", err)
	}
	return json.MarshalIndent(rollup, "", "  ")
}

// buildRollupConfig constructs the rollup.json from deployment state and config.
func (e *ArtifactExtractor) buildRollupConfig(
	ctx context.Context,
	deploymentID uuid.UUID,
	cfg *DeploymentConfig,
) (*RollupConfig, error) {
	// Get deployment state
	artifact, err := e.repo.GetArtifact(ctx, deploymentID, "deployment_state")
	if err != nil {
		return nil, fmt.Errorf("get state artifact: %w", err)
	}

	var state map[string]interface{}
	if artifact != nil {
		if err := json.Unmarshal(artifact.Content, &state); err != nil {
			return nil, fmt.Errorf("unmarshal state: %w", err)
		}
	}

	// Get rollup config artifact if already generated
	rollupArtifact, _ := e.repo.GetArtifact(ctx, deploymentID, "rollup_config")
	if rollupArtifact != nil {
		var rollup RollupConfig
		if err := json.Unmarshal(rollupArtifact.Content, &rollup); err == nil {
			return &rollup, nil
		}
	}

	// Extract chain_state for contract addresses (uses camelCase from Go struct serialization)
	chainState, _ := state["chain_state"].(map[string]interface{})
	if chainState == nil {
		chainState = state // Fallback to top level
	}

	// Build rollup config from deployment config
	// L2 genesis time should come from state, default to now
	l2Time := uint64(time.Now().Unix())
	if ts, ok := state["l2_genesis_time"].(float64); ok {
		l2Time = uint64(ts)
	}

	rollup := &RollupConfig{
		Genesis: RollupGenesisConfig{
			L1: GenesisBlockRef{
				Hash:   getStringFromState(state, "l1_genesis_hash", "0x0000000000000000000000000000000000000000000000000000000000000000"),
				Number: getUint64FromState(state, "l1_genesis_number", 0),
			},
			L2: GenesisBlockRef{
				Hash:   getStringFromState(state, "l2_genesis_hash", "0x0000000000000000000000000000000000000000000000000000000000000000"),
				Number: 0,
			},
			L2Time: l2Time,
			SystemConfig: SystemConfig{
				BatcherAddr: cfg.BatcherAddress,
				Overhead:    "0x0000000000000000000000000000000000000000000000000000000000000834",
				Scalar:      "0x00000000000000000000000000000000000000000000000000000000000f4240",
				GasLimit:    cfg.GasLimit,
			},
		},
		BlockTime:           cfg.BlockTime,
		MaxSequencerDrift:   cfg.MaxSequencerDrift,
		SequencerWindowSize: cfg.SequencerWindowSize,
		ChannelTimeout:      300,
		L1ChainID:           cfg.L1ChainID,
		L2ChainID:           cfg.ChainID,
		BatchInboxAddress:   calculateBatchInboxAddress(cfg.ChainID),
		DepositContractAddr: getAddressFromState(chainState, "OptimismPortalProxy"),
		L1SystemConfigAddr:  getAddressFromState(chainState, "SystemConfigProxy"),
	}

	// Add hardfork timestamps (set at genesis for new chains)
	zero := uint64(0)
	rollup.RegolithTime = &zero
	rollup.CanyonTime = &zero
	rollup.DeltaTime = &zero
	rollup.EcotoneTime = &zero
	rollup.FjordTime = &zero
	rollup.GraniteTime = &zero

	// Alt-DA configuration - always enabled for Celestia DA
	// POPKins exclusively uses Celestia as the DA layer
	rollup.AltDAEnabled = true

	return rollup, nil
}

// Helper functions for state extraction

// getStringFromState safely extracts a string value from state map.
func getStringFromState(state map[string]interface{}, key string, defaultVal string) string {
	if v, ok := state[key].(string); ok {
		return v
	}
	return defaultVal
}

// getUint64FromState safely extracts a uint64 value from state map.
func getUint64FromState(state map[string]interface{}, key string, defaultVal uint64) uint64 {
	if v, ok := state[key].(float64); ok {
		return uint64(v)
	}
	return defaultVal
}

// calculateBatchInboxAddress calculates the deterministic batch inbox address for a chain.
func calculateBatchInboxAddress(chainID uint64) string {
	// Standard batch inbox format: 0xff00...{chainID in 4 bytes}
	return fmt.Sprintf("0xff00000000000000000000000000000000%08x", chainID)
}
