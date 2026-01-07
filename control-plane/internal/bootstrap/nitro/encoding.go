// Package nitro provides Nitro chain deployment infrastructure.
package nitro

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// BOLD protocol default values (from nitro-contracts v3.2)
const (
	// Default ArbOS version - ArbOS 51 from Nitro consensus-v51
	// Source: https://github.com/OffchainLabs/nitro/releases/tag/consensus-v51
	DefaultArbOSVersion = 51

	// Minimum assertion period in blocks (75 blocks ~= 15 minutes on Ethereum)
	DefaultMinimumAssertionPeriod = 75
	// Validator AFK timeout in blocks (201600 ~= 28 days)
	DefaultValidatorAfkBlocks = 201600
	// Layer zero heights for dispute game
	DefaultLayerZeroBlockEdgeHeight     = 1 << 25 // 2^25
	DefaultLayerZeroBigStepEdgeHeight   = 1 << 19 // 2^19
	DefaultLayerZeroSmallStepEdgeHeight = 1 << 23 // 2^23
	// Number of big step levels in dispute game
	DefaultNumBigStepLevel = 3
	// Challenge grace period in blocks (14400 ~= 2 days)
	DefaultChallengeGracePeriodBlocks = 14400
	// Data cost estimate (0 = no estimate)
	DefaultDataCostEstimate = 0
)

// DefaultWasmModuleRoot is the WASM module root hash for Nitro consensus-v51 (ArbOS 51)
// Source: https://github.com/OffchainLabs/nitro/releases/tag/consensus-v51
var DefaultWasmModuleRoot = common.HexToHash("0x8a7513bf7bb3e3db04b0d982d0e973bcf57bf8b88aef7c6d03dba3a81a56a499")

// RollupEncoder handles ABI encoding for createRollup calls.
type RollupEncoder struct {
	rollupCreatorABI abi.ABI
	logger           *slog.Logger
}

// NewRollupEncoder creates a new rollup encoder.
func NewRollupEncoder(rollupCreatorABI abi.ABI, logger *slog.Logger) *RollupEncoder {
	return &RollupEncoder{
		rollupCreatorABI: rollupCreatorABI,
		logger:           logger,
	}
}

// EncodeCreateRollup encodes the createRollup function call for BOLD protocol.
func (e *RollupEncoder) EncodeCreateRollup(cfg *RollupConfig, chainConfig map[string]interface{}) ([]byte, error) {
	// Encode chain config as JSON
	chainConfigJSON, err := json.Marshal(chainConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal chain config: %w", err)
	}

	baseStake := cfg.BaseStake
	if baseStake == nil {
		baseStake = big.NewInt(100000000000000000) // 0.1 ETH default
	}

	// Mini stake values (stake required at each challenge level)
	// EdgeChallengeManager requires numBigStepLevel + 2 stake amounts
	// For numBigStepLevel=3, we need 5 stake amounts
	miniStake := new(big.Int).Div(baseStake, big.NewInt(10))
	miniStakeValues := []*big.Int{miniStake, miniStake, miniStake, miniStake, miniStake}

	// BOLD protocol requires a stake token (ERC20), zero address is not allowed
	// Use WETH on Sepolia if no stake token is provided
	stakeToken := cfg.StakeToken
	if stakeToken == (common.Address{}) {
		// Default to WETH on Sepolia: 0x7b79995e5f793A07Bc00c21412e50Ecae098E7f9
		stakeToken = common.HexToAddress("0x7b79995e5f793A07Bc00c21412e50Ecae098E7f9")
		e.logger.Info("stake token was zero, defaulted to Sepolia WETH",
			slog.String("stake_token", stakeToken.Hex()),
		)
	}

	e.logger.Info("createRollup config",
		slog.String("stake_token", stakeToken.Hex()),
		slog.String("owner", cfg.Owner.Hex()),
		slog.Int64("chain_id", cfg.ChainID),
		slog.String("base_stake", baseStake.String()),
	)

	// Build the entire RollupDeploymentParams as a single struct
	// The go-ethereum ABI encoder requires all nested structs to be concrete types
	deployParams := RollupDeploymentParams{
		Config: BOLDConfig{
			ConfirmPeriodBlocks: uint64(cfg.ConfirmPeriodBlocks),
			StakeToken:          stakeToken,
			BaseStake:           baseStake,
			WasmModuleRoot:      DefaultWasmModuleRoot, // Latest Nitro WASM root
			Owner:               cfg.Owner,
			LoserStakeEscrow:    cfg.Owner,
			ChainId:             big.NewInt(cfg.ChainID),
			ChainConfig:         string(chainConfigJSON),
			MinimumAssertionPeriod: big.NewInt(DefaultMinimumAssertionPeriod),
			ValidatorAfkBlocks:    DefaultValidatorAfkBlocks,
			MiniStakeValues:       miniStakeValues,
			SequencerInboxMaxTimeVariation: MaxTimeVariation{
				DelayBlocks:   big.NewInt(5760),
				FutureBlocks:  big.NewInt(64),
				DelaySeconds:  big.NewInt(86400),
				FutureSeconds: big.NewInt(3600),
			},
			LayerZeroBlockEdgeHeight:     big.NewInt(DefaultLayerZeroBlockEdgeHeight),
			LayerZeroBigStepEdgeHeight:   big.NewInt(DefaultLayerZeroBigStepEdgeHeight),
			LayerZeroSmallStepEdgeHeight: big.NewInt(DefaultLayerZeroSmallStepEdgeHeight),
			GenesisAssertionState: AssertionState{
				GlobalState:    GlobalState{},
				MachineStatus:  1, // FINISHED
				EndHistoryRoot: [32]byte{},
			},
			GenesisInboxCount:          big.NewInt(1),
			AnyTrustFastConfirmer:      common.Address{},
			NumBigStepLevel:            DefaultNumBigStepLevel,
			ChallengeGracePeriodBlocks: DefaultChallengeGracePeriodBlocks,
			BufferConfig:               BufferConfig{},
			DataCostEstimate:           big.NewInt(DefaultDataCostEstimate),
		},
		Validators:                cfg.Validators,
		MaxDataSize:               big.NewInt(cfg.MaxDataSize),
		NativeToken:               cfg.NativeToken,
		DeployFactoriesToL2:       cfg.DeployFactoriesToL2,
		MaxFeePerGasForRetryables: big.NewInt(100000000), // 0.1 gwei
		BatchPosters:              cfg.BatchPosters,
		BatchPosterManager:        cfg.Owner,
		FeeTokenPricer:            common.Address{},
		CustomOsp:                 common.Address{},
	}

	return e.rollupCreatorABI.Pack("createRollup", deployParams)
}

// PrepareChainConfig prepares the chain configuration JSON.
func PrepareChainConfig(cfg *RollupConfig) map[string]interface{} {
	// Standard Ethereum hardfork configuration
	return map[string]interface{}{
		"homesteadBlock":      0,
		"daoForkBlock":        nil,
		"daoForkSupport":      true,
		"eip150Block":         0,
		"eip150Hash":          "0x0000000000000000000000000000000000000000000000000000000000000000",
		"eip155Block":         0,
		"eip158Block":         0,
		"byzantiumBlock":      0,
		"constantinopleBlock": 0,
		"petersburgBlock":     0,
		"istanbulBlock":       0,
		"muirGlacierBlock":    0,
		"berlinBlock":         0,
		"londonBlock":         0,
		"clique": map[string]interface{}{
			"period": 0,
			"epoch":  0,
		},
		"arbitrum": map[string]interface{}{
			"EnableArbOS":               true,
			"AllowDebugPrecompiles":     false,
			"DataAvailabilityCommittee": cfg.DataAvailability == DAModeAnytrust,
			"InitialArbOSVersion":       DefaultArbOSVersion,
			"GenesisBlockNum":           0,
			"MaxCodeSize":               24576,
			"MaxInitCodeSize":           49152,
			"InitialChainOwner":         cfg.Owner.Hex(),
		},
		"chainId": cfg.ChainID,
	}
}
