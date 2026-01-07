// Package nitro provides Nitro chain deployment infrastructure.
package nitro

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// BOLD protocol ABI types for createRollup encoding
// These must match the exact structure expected by nitro-contracts v3.2.0
// Note: abi tags must match the Solidity struct field names exactly (camelCase).

// GlobalState represents the global state in an assertion.
type GlobalState struct {
	Bytes32Vals [2][32]byte `abi:"bytes32Vals"`
	U64Vals     [2]uint64   `abi:"u64Vals"`
}

// AssertionState represents the state of an assertion.
type AssertionState struct {
	GlobalState    GlobalState `abi:"globalState"`
	MachineStatus  uint8       `abi:"machineStatus"`
	EndHistoryRoot [32]byte    `abi:"endHistoryRoot"`
}

// MaxTimeVariation represents sequencer inbox time bounds.
type MaxTimeVariation struct {
	DelayBlocks   *big.Int `abi:"delayBlocks"`
	FutureBlocks  *big.Int `abi:"futureBlocks"`
	DelaySeconds  *big.Int `abi:"delaySeconds"`
	FutureSeconds *big.Int `abi:"futureSeconds"`
}

// BufferConfig represents delay buffer configuration.
type BufferConfig struct {
	Threshold            uint64 `abi:"threshold"`
	Max                  uint64 `abi:"max"`
	ReplenishRateInBasis uint64 `abi:"replenishRateInBasis"`
}

// BOLDConfig represents the BOLD protocol chain configuration.
type BOLDConfig struct {
	ConfirmPeriodBlocks            uint64           `abi:"confirmPeriodBlocks"`
	StakeToken                     common.Address   `abi:"stakeToken"`
	BaseStake                      *big.Int         `abi:"baseStake"`
	WasmModuleRoot                 [32]byte         `abi:"wasmModuleRoot"`
	Owner                          common.Address   `abi:"owner"`
	LoserStakeEscrow               common.Address   `abi:"loserStakeEscrow"`
	ChainId                        *big.Int         `abi:"chainId"`
	ChainConfig                    string           `abi:"chainConfig"`
	MinimumAssertionPeriod         *big.Int         `abi:"minimumAssertionPeriod"`
	ValidatorAfkBlocks             uint64           `abi:"validatorAfkBlocks"`
	MiniStakeValues                []*big.Int       `abi:"miniStakeValues"`
	SequencerInboxMaxTimeVariation MaxTimeVariation `abi:"sequencerInboxMaxTimeVariation"`
	LayerZeroBlockEdgeHeight       *big.Int         `abi:"layerZeroBlockEdgeHeight"`
	LayerZeroBigStepEdgeHeight     *big.Int         `abi:"layerZeroBigStepEdgeHeight"`
	LayerZeroSmallStepEdgeHeight   *big.Int         `abi:"layerZeroSmallStepEdgeHeight"`
	GenesisAssertionState          AssertionState   `abi:"genesisAssertionState"`
	GenesisInboxCount              *big.Int         `abi:"genesisInboxCount"`
	AnyTrustFastConfirmer          common.Address   `abi:"anyTrustFastConfirmer"`
	NumBigStepLevel                uint8            `abi:"numBigStepLevel"`
	ChallengeGracePeriodBlocks     uint64           `abi:"challengeGracePeriodBlocks"`
	BufferConfig                   BufferConfig     `abi:"bufferConfig"`
	DataCostEstimate               *big.Int         `abi:"dataCostEstimate"`
}

// RollupDeploymentParams represents the full createRollup parameters.
type RollupDeploymentParams struct {
	Config                    BOLDConfig       `abi:"config"`
	Validators                []common.Address `abi:"validators"`
	MaxDataSize               *big.Int         `abi:"maxDataSize"`
	NativeToken               common.Address   `abi:"nativeToken"`
	DeployFactoriesToL2       bool             `abi:"deployFactoriesToL2"`
	MaxFeePerGasForRetryables *big.Int         `abi:"maxFeePerGasForRetryables"`
	BatchPosters              []common.Address `abi:"batchPosters"`
	BatchPosterManager        common.Address   `abi:"batchPosterManager"`
	FeeTokenPricer            common.Address   `abi:"feeTokenPricer"`
	CustomOsp                 common.Address   `abi:"customOsp"`
}
