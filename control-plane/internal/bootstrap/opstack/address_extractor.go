package opstack

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// extractContractAddresses retrieves deployed contract addresses from state.
// The state structure from op-deployer is:
//
//	{
//	  "chain_state": {
//	    "OptimismPortalProxy": "0x...",      // camelCase from Go struct
//	    "L1CrossDomainMessengerProxy": "0x...",
//	    ...
//	  },
//	  "superchain_deployment": { ... },
//	  "implementations_deployment": { ... }
//	}
func (e *ArtifactExtractor) extractContractAddresses(ctx context.Context, deploymentID uuid.UUID) (ContractAddresses, error) {
	artifact, err := e.repo.GetArtifact(ctx, deploymentID, "deployment_state")
	if err != nil {
		return ContractAddresses{}, fmt.Errorf("get state artifact: %w", err)
	}

	addrs := ContractAddresses{}

	if artifact != nil {
		var state map[string]interface{}
		if err := json.Unmarshal(artifact.Content, &state); err != nil {
			return addrs, fmt.Errorf("unmarshal state: %w", err)
		}

		// The addresses are nested in chain_state (from op-deployer's ChainState struct)
		chainState, _ := state["chain_state"].(map[string]interface{})
		if chainState == nil {
			// Fallback: try top-level (older format)
			chainState = state
		}

		// Extract addresses using camelCase keys (Go struct JSON serialization)
		// OpChainCoreContracts
		addrs.OptimismPortalProxy = getAddressFromState(chainState, "OptimismPortalProxy")
		addrs.L1CrossDomainMessengerProxy = getAddressFromState(chainState, "L1CrossDomainMessengerProxy")
		addrs.L1StandardBridgeProxy = getAddressFromState(chainState, "L1StandardBridgeProxy")
		addrs.L1ERC721BridgeProxy = getAddressFromState(chainState, "L1Erc721BridgeProxy")
		addrs.SystemConfigProxy = getAddressFromState(chainState, "SystemConfigProxy")
		addrs.OptimismMintableERC20Factory = getAddressFromState(chainState, "OptimismMintableErc20FactoryProxy")
		addrs.AddressManager = getAddressFromState(chainState, "AddressManagerImpl")

		// OpChainFaultProofsContracts
		addrs.DisputeGameFactoryProxy = getAddressFromState(chainState, "DisputeGameFactoryProxy")
		addrs.AnchorStateRegistryProxy = getAddressFromState(chainState, "AnchorStateRegistryProxy")
		addrs.DelayedWETHProxy = getAddressFromState(chainState, "DelayedWethPermissionedGameProxy")

		// SuperchainContracts (from superchain_deployment)
		superchain, _ := state["superchain_deployment"].(map[string]interface{})
		if superchain != nil {
			addrs.SuperchainConfig = getAddressFromState(superchain, "SuperchainConfigProxy")
			addrs.ProtocolVersions = getAddressFromState(superchain, "ProtocolVersionsProxy")
		}
	}

	// Get batch inbox address from chain ID
	deployment, err := e.repo.GetDeployment(ctx, deploymentID)
	if err == nil && deployment != nil {
		addrs.BatchInbox = calculateBatchInboxAddress(uint64(deployment.ChainID))
	}

	return addrs, nil
}

// getAddressFromState extracts an Ethereum address from the state map.
// It handles both string addresses and common.Address types (which serialize as hex strings).
func getAddressFromState(state map[string]interface{}, key string) string {
	if state == nil {
		return ""
	}
	if v, ok := state[key].(string); ok {
		return v
	}
	return ""
}
