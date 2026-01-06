// Package nitro provides Nitro chain deployment infrastructure.
package nitro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Default deployment parameters (matching TypeScript implementation)
const (
	DefaultConfirmPeriodBlocks      = 45818 // ~1 week on Ethereum
	DefaultExtraChallengeTimeBlocks = 0
	DefaultMaxDataSize              = 117964
)

// GoDataAvailabilityType represents the data availability mode for the chain.
type GoDataAvailabilityType string

const (
	GoDATypeCelestia GoDataAvailabilityType = "celestia"
	GoDATypeRollup   GoDataAvailabilityType = "rollup"
	GoDATypeAnytrust GoDataAvailabilityType = "anytrust"
)

// GoDeployConfig contains all configuration for deploying a Nitro rollup using Go.
type GoDeployConfig struct {
	// Chain configuration
	ChainID        int64  `json:"chainId"`
	ChainName      string `json:"chainName"`
	ParentChainID  int64  `json:"parentChainId"`
	ParentChainRPC string `json:"parentChainRpc"`

	// Ownership and operators
	Owner        common.Address   `json:"owner"`
	BatchPosters []common.Address `json:"batchPosters"`
	Validators   []common.Address `json:"validators"`

	// Staking
	StakeToken common.Address `json:"stakeToken"`
	BaseStake  *big.Int       `json:"baseStake"`

	// Optional parameters with defaults
	ConfirmPeriodBlocks      int64                  `json:"confirmPeriodBlocks,omitempty"`
	ExtraChallengeTimeBlocks int64                  `json:"extraChallengeTimeBlocks,omitempty"`
	MaxDataSize              int64                  `json:"maxDataSize,omitempty"`
	DataAvailability         GoDataAvailabilityType `json:"dataAvailability,omitempty"`
	NativeToken              common.Address         `json:"nativeToken,omitempty"`
	DeployFactoriesToL2      bool                   `json:"deployFactoriesToL2,omitempty"`
}

// GoCoreContracts contains addresses of all deployed core contracts (Go version).
type GoCoreContracts struct {
	Rollup                 common.Address `json:"rollup"`
	Inbox                  common.Address `json:"inbox"`
	Outbox                 common.Address `json:"outbox"`
	Bridge                 common.Address `json:"bridge"`
	SequencerInbox         common.Address `json:"sequencerInbox"`
	RollupEventInbox       common.Address `json:"rollupEventInbox"`
	ChallengeManager       common.Address `json:"challengeManager"`
	AdminProxy             common.Address `json:"adminProxy"`
	UpgradeExecutor        common.Address `json:"upgradeExecutor"`
	ValidatorWalletCreator common.Address `json:"validatorWalletCreator"`
	NativeToken            common.Address `json:"nativeToken"`
	DeployedAtBlockNumber  uint64         `json:"deployedAtBlockNumber"`
}

// GoDeployResult contains the result of a deployment operation (Go version).
type GoDeployResult struct {
	Success         bool                   `json:"success"`
	CoreContracts   *GoCoreContracts       `json:"coreContracts,omitempty"`
	TransactionHash common.Hash            `json:"transactionHash,omitempty"`
	BlockNumber     uint64                 `json:"blockNumber,omitempty"`
	ChainConfig     map[string]interface{} `json:"chainConfig,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

// RollupDeployer handles deployment of Nitro rollups using RollupCreator.
type RollupDeployer struct {
	artifacts *NitroArtifacts
	signer    *NitroSigner
	logger    *slog.Logger

	// Cached ABIs
	rollupCreatorABI abi.ABI
	sequencerInboxABI abi.ABI
	upgradeExecutorABI abi.ABI
}

// NewRollupDeployer creates a new rollup deployer.
func NewRollupDeployer(
	artifacts *NitroArtifacts,
	signer *NitroSigner,
	logger *slog.Logger,
) (*RollupDeployer, error) {
	// Parse ABIs
	rollupCreatorABI, err := ParseContractABI(artifacts.RollupCreator.ABI)
	if err != nil {
		return nil, fmt.Errorf("parse RollupCreator ABI: %w", err)
	}

	sequencerInboxABI, err := ParseContractABI(artifacts.SequencerInbox.ABI)
	if err != nil {
		return nil, fmt.Errorf("parse SequencerInbox ABI: %w", err)
	}

	// UpgradeExecutor ABI - we only need executeCall
	upgradeExecutorABI, err := abi.JSON(strings.NewReader(`[{
		"inputs": [
			{"name": "upgrade", "type": "address"},
			{"name": "upgradeCallData", "type": "bytes"}
		],
		"name": "executeCall",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	}]`))
	if err != nil {
		return nil, fmt.Errorf("parse UpgradeExecutor ABI: %w", err)
	}

	return &RollupDeployer{
		artifacts:          artifacts,
		signer:             signer,
		logger:             logger,
		rollupCreatorABI:   rollupCreatorABI,
		sequencerInboxABI:  sequencerInboxABI,
		upgradeExecutorABI: upgradeExecutorABI,
	}, nil
}

// Deploy deploys a new Nitro rollup using the RollupCreator contract.
func (d *RollupDeployer) Deploy(
	ctx context.Context,
	cfg *GoDeployConfig,
	rollupCreatorAddr common.Address,
) (*GoDeployResult, error) {
	d.logger.Info("starting Nitro rollup deployment",
		slog.Int64("chain_id", cfg.ChainID),
		slog.String("chain_name", cfg.ChainName),
		slog.Int64("parent_chain_id", cfg.ParentChainID),
	)

	// Apply defaults
	d.applyDefaults(cfg)

	// Connect to parent chain
	client, err := ethclient.DialContext(ctx, cfg.ParentChainRPC)
	if err != nil {
		return d.errorResult(fmt.Errorf("connect to parent chain: %w", err))
	}
	defer client.Close()

	// Verify chain ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return d.errorResult(fmt.Errorf("get chain ID: %w", err))
	}
	if chainID.Int64() != cfg.ParentChainID {
		return d.errorResult(fmt.Errorf("chain ID mismatch: expected %d, got %d", cfg.ParentChainID, chainID.Int64()))
	}

	// Check deployer balance
	balance, err := client.BalanceAt(ctx, d.signer.Address(), nil)
	if err != nil {
		return d.errorResult(fmt.Errorf("get balance: %w", err))
	}
	d.logger.Info("deployer balance",
		slog.String("address", d.signer.Address().Hex()),
		slog.String("balance_wei", balance.String()),
	)

	if balance.Sign() == 0 {
		return d.errorResult(fmt.Errorf("deployer address has no ETH balance"))
	}

	// Prepare chain config
	chainConfig := d.prepareChainConfig(cfg)
	d.logger.Info("chain config prepared", slog.Any("config", chainConfig))

	// Encode createRollup call data
	callData, err := d.encodeCreateRollup(cfg, chainConfig)
	if err != nil {
		return d.errorResult(fmt.Errorf("encode createRollup: %w", err))
	}

	// Get nonce
	nonce, err := client.PendingNonceAt(ctx, d.signer.Address())
	if err != nil {
		return d.errorResult(fmt.Errorf("get nonce: %w", err))
	}

	// Get gas price with boost
	gasPrice, err := d.getGasPrice(ctx, client)
	if err != nil {
		return d.errorResult(fmt.Errorf("get gas price: %w", err))
	}

	// Estimate gas
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:     d.signer.Address(),
		To:       &rollupCreatorAddr,
		Gas:      0,
		GasPrice: gasPrice,
		Value:    big.NewInt(0),
		Data:     callData,
	})
	if err != nil {
		// Use a high default for rollup creation
		gasLimit = 15_000_000
		d.logger.Warn("gas estimation failed, using default",
			slog.Uint64("gas_limit", gasLimit),
			slog.String("error", err.Error()),
		)
	}
	// Add 20% buffer
	gasLimit = gasLimit * 120 / 100

	d.logger.Info("sending createRollup transaction",
		slog.String("rollup_creator", rollupCreatorAddr.Hex()),
		slog.Uint64("gas_limit", gasLimit),
		slog.String("gas_price", gasPrice.String()),
	)

	// Create transaction
	tx := types.NewTransaction(
		nonce,
		rollupCreatorAddr,
		big.NewInt(0), // Value
		gasLimit,
		gasPrice,
		callData,
	)

	// Sign and send
	signedTx, err := d.signer.SignTransaction(ctx, tx)
	if err != nil {
		return d.errorResult(fmt.Errorf("sign transaction: %w", err))
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return d.errorResult(fmt.Errorf("send transaction: %w", err))
	}

	d.logger.Info("transaction submitted, waiting for confirmation",
		slog.String("tx_hash", signedTx.Hash().Hex()),
	)

	// Wait for receipt
	receipt, err := bind.WaitMined(ctx, client, signedTx)
	if err != nil {
		return &GoDeployResult{
			Success:         false,
			TransactionHash: signedTx.Hash(),
			Error:           fmt.Sprintf("wait for receipt: %v", err),
		}, nil
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return &GoDeployResult{
			Success:         false,
			TransactionHash: signedTx.Hash(),
			BlockNumber:     receipt.BlockNumber.Uint64(),
			Error:           "transaction reverted",
		}, nil
	}

	d.logger.Info("transaction confirmed",
		slog.Uint64("block_number", receipt.BlockNumber.Uint64()),
	)

	// Parse contract addresses from logs
	coreContracts, err := d.parseDeploymentLogs(receipt)
	if err != nil {
		return d.errorResult(fmt.Errorf("parse deployment logs: %w", err))
	}
	coreContracts.DeployedAtBlockNumber = receipt.BlockNumber.Uint64()

	d.logger.Info("rollup deployed successfully",
		slog.String("rollup", coreContracts.Rollup.Hex()),
		slog.String("sequencer_inbox", coreContracts.SequencerInbox.Hex()),
	)

	// Whitelist batch posters
	if len(cfg.BatchPosters) > 0 {
		if err := d.whitelistBatchPosters(ctx, client, coreContracts, cfg.BatchPosters); err != nil {
			d.logger.Warn("failed to whitelist batch posters",
				slog.String("error", err.Error()),
			)
			// Don't fail the deployment, just log the warning
		}
	}

	return &GoDeployResult{
		Success:         true,
		CoreContracts:   coreContracts,
		TransactionHash: signedTx.Hash(),
		BlockNumber:     receipt.BlockNumber.Uint64(),
		ChainConfig:     chainConfig,
	}, nil
}

// applyDefaults applies default values to config.
func (d *RollupDeployer) applyDefaults(cfg *GoDeployConfig) {
	if cfg.ConfirmPeriodBlocks == 0 {
		cfg.ConfirmPeriodBlocks = DefaultConfirmPeriodBlocks
	}
	if cfg.MaxDataSize == 0 {
		cfg.MaxDataSize = DefaultMaxDataSize
	}
	if cfg.DataAvailability == "" {
		cfg.DataAvailability = GoDATypeCelestia
	}
}

// prepareChainConfig prepares the chain configuration JSON.
func (d *RollupDeployer) prepareChainConfig(cfg *GoDeployConfig) map[string]interface{} {
	// Standard Ethereum hardfork configuration
	return map[string]interface{}{
		"homesteadBlock":       0,
		"daoForkBlock":         nil,
		"daoForkSupport":       true,
		"eip150Block":          0,
		"eip150Hash":           "0x0000000000000000000000000000000000000000000000000000000000000000",
		"eip155Block":          0,
		"eip158Block":          0,
		"byzantiumBlock":       0,
		"constantinopleBlock":  0,
		"petersburgBlock":      0,
		"istanbulBlock":        0,
		"muirGlacierBlock":     0,
		"berlinBlock":          0,
		"londonBlock":          0,
		"clique": map[string]interface{}{
			"period": 0,
			"epoch":  0,
		},
		"arbitrum": map[string]interface{}{
			"EnableArbOS":               true,
			"AllowDebugPrecompiles":     false,
			"DataAvailabilityCommittee": cfg.DataAvailability == GoDATypeAnytrust,
			"InitialArbOSVersion":       32,
			"GenesisBlockNum":           0,
			"MaxCodeSize":               24576,
			"MaxInitCodeSize":           49152,
			"InitialChainOwner":         cfg.Owner.Hex(),
		},
		"chainId": cfg.ChainID,
	}
}

// encodeCreateRollup encodes the createRollup function call.
func (d *RollupDeployer) encodeCreateRollup(cfg *GoDeployConfig, chainConfig map[string]interface{}) ([]byte, error) {
	// Encode chain config as JSON
	chainConfigJSON, err := json.Marshal(chainConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal chain config: %w", err)
	}

	// Build RollupDeploymentParams struct
	// This matches the orbit-sdk's createRollupPrepareTransactionRequest
	deploymentParams := struct {
		Config struct {
			ConfirmPeriodBlocks      uint64
			ExtraChallengeTimeBlocks uint64
			StakeToken               common.Address
			BaseStake                *big.Int
			WasmModuleRoot           [32]byte
			Owner                    common.Address
			LoserStakeEscrow         common.Address
			ChainId                  *big.Int
			ChainConfig              string
			GenesisBlockNum          uint64
			SequencerInboxMaxTimeVariation struct {
				DelayBlocks   *big.Int
				FutureBlocks  *big.Int
				DelaySeconds  *big.Int
				FutureSeconds *big.Int
			}
		}
		BatchPosters        []common.Address
		Validators          []common.Address
		MaxDataSize         *big.Int
		NativeToken         common.Address
		DeployFactoriesToL2 bool
		MaxFeePerGasForRetryables *big.Int
	}{}

	deploymentParams.Config.ConfirmPeriodBlocks = uint64(cfg.ConfirmPeriodBlocks)
	deploymentParams.Config.ExtraChallengeTimeBlocks = uint64(cfg.ExtraChallengeTimeBlocks)
	deploymentParams.Config.StakeToken = cfg.StakeToken
	deploymentParams.Config.BaseStake = cfg.BaseStake
	// WasmModuleRoot - default empty for now
	deploymentParams.Config.Owner = cfg.Owner
	deploymentParams.Config.ChainId = big.NewInt(cfg.ChainID)
	deploymentParams.Config.ChainConfig = string(chainConfigJSON)
	
	// Default time variation values
	deploymentParams.Config.SequencerInboxMaxTimeVariation.DelayBlocks = big.NewInt(5760)
	deploymentParams.Config.SequencerInboxMaxTimeVariation.FutureBlocks = big.NewInt(64)
	deploymentParams.Config.SequencerInboxMaxTimeVariation.DelaySeconds = big.NewInt(86400)
	deploymentParams.Config.SequencerInboxMaxTimeVariation.FutureSeconds = big.NewInt(3600)

	deploymentParams.BatchPosters = cfg.BatchPosters
	deploymentParams.Validators = cfg.Validators
	deploymentParams.MaxDataSize = big.NewInt(cfg.MaxDataSize)
	deploymentParams.NativeToken = cfg.NativeToken
	deploymentParams.DeployFactoriesToL2 = cfg.DeployFactoriesToL2
	deploymentParams.MaxFeePerGasForRetryables = big.NewInt(100000000) // 0.1 gwei

	// Encode the function call
	return d.rollupCreatorABI.Pack("createRollup", deploymentParams)
}

// getGasPrice returns a boosted gas price for faster inclusion.
func (d *RollupDeployer) getGasPrice(ctx context.Context, client *ethclient.Client) (*big.Int, error) {
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	// Boost by 50%
	boosted := new(big.Int).Mul(gasPrice, big.NewInt(150))
	boosted = boosted.Div(boosted, big.NewInt(100))

	// Ensure at least 2 Gwei
	minGwei := big.NewInt(2_000_000_000)
	if boosted.Cmp(minGwei) < 0 {
		boosted = minGwei
	}

	return boosted, nil
}

// parseDeploymentLogs parses the RollupCreated event from transaction logs.
func (d *RollupDeployer) parseDeploymentLogs(receipt *types.Receipt) (*GoCoreContracts, error) {
	// RollupCreated event signature
	// event RollupCreated(
	//   address indexed rollupAddress,
	//   address indexed nativeToken,
	//   address inboxAddress,
	//   address outbox,
	//   address rollupEventInbox,
	//   address challengeManager,
	//   address adminProxy,
	//   address sequencerInbox,
	//   address bridge,
	//   address upgradeExecutor,
	//   address validatorWalletCreator
	// )
	
	// Find the RollupCreated event by looking for logs with the right data size
	for _, log := range receipt.Logs {
		if len(log.Topics) < 3 {
			continue
		}
		
		// For now, use a heuristic: look for logs with multiple address topics
		// The RollupCreated event has indexed rollupAddress and nativeToken
		
		// This is a simplified implementation - in production, we'd properly
		// decode the event using the ABI
		if len(log.Data) >= 32*9 { // 9 non-indexed address parameters
			contracts := &GoCoreContracts{}
			
			// First two topics are indexed addresses
			if len(log.Topics) >= 2 {
				contracts.Rollup = common.BytesToAddress(log.Topics[1].Bytes())
			}
			if len(log.Topics) >= 3 {
				contracts.NativeToken = common.BytesToAddress(log.Topics[2].Bytes())
			}
			
			// Remaining addresses are in data
			offset := 0
			contracts.Inbox = common.BytesToAddress(log.Data[offset : offset+32])
			offset += 32
			contracts.Outbox = common.BytesToAddress(log.Data[offset : offset+32])
			offset += 32
			contracts.RollupEventInbox = common.BytesToAddress(log.Data[offset : offset+32])
			offset += 32
			contracts.ChallengeManager = common.BytesToAddress(log.Data[offset : offset+32])
			offset += 32
			contracts.AdminProxy = common.BytesToAddress(log.Data[offset : offset+32])
			offset += 32
			contracts.SequencerInbox = common.BytesToAddress(log.Data[offset : offset+32])
			offset += 32
			contracts.Bridge = common.BytesToAddress(log.Data[offset : offset+32])
			offset += 32
			contracts.UpgradeExecutor = common.BytesToAddress(log.Data[offset : offset+32])
			offset += 32
			contracts.ValidatorWalletCreator = common.BytesToAddress(log.Data[offset : offset+32])
			
			return contracts, nil
		}
	}

	return nil, fmt.Errorf("RollupCreated event not found in logs")
}

// whitelistBatchPosters whitelists batch posters on the SequencerInbox via UpgradeExecutor.
func (d *RollupDeployer) whitelistBatchPosters(
	ctx context.Context,
	client *ethclient.Client,
	contracts *GoCoreContracts,
	batchPosters []common.Address,
) error {
	d.logger.Info("whitelisting batch posters via UpgradeExecutor",
		slog.Int("count", len(batchPosters)),
		slog.String("upgrade_executor", contracts.UpgradeExecutor.Hex()),
		slog.String("sequencer_inbox", contracts.SequencerInbox.Hex()),
	)

	for _, batchPoster := range batchPosters {
		// Check if already whitelisted
		isWhitelisted, err := d.isBatchPoster(ctx, client, contracts.SequencerInbox, batchPoster)
		if err != nil {
			d.logger.Warn("failed to check batch poster status",
				slog.String("batch_poster", batchPoster.Hex()),
				slog.String("error", err.Error()),
			)
			continue
		}

		if isWhitelisted {
			d.logger.Info("batch poster already whitelisted",
				slog.String("batch_poster", batchPoster.Hex()),
			)
			continue
		}

		// Encode setIsBatchPoster(batchPoster, true)
		innerCallData, err := d.sequencerInboxABI.Pack("setIsBatchPoster", batchPoster, true)
		if err != nil {
			return fmt.Errorf("encode setIsBatchPoster: %w", err)
		}

		// Encode executeCall(sequencerInbox, innerCallData)
		outerCallData, err := d.upgradeExecutorABI.Pack("executeCall", contracts.SequencerInbox, innerCallData)
		if err != nil {
			return fmt.Errorf("encode executeCall: %w", err)
		}

		// Get nonce
		nonce, err := client.PendingNonceAt(ctx, d.signer.Address())
		if err != nil {
			return fmt.Errorf("get nonce: %w", err)
		}

		// Get gas price
		gasPrice, err := d.getGasPrice(ctx, client)
		if err != nil {
			return fmt.Errorf("get gas price: %w", err)
		}

		// Estimate gas
		gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
			From:     d.signer.Address(),
			To:       &contracts.UpgradeExecutor,
			Gas:      0,
			GasPrice: gasPrice,
			Value:    big.NewInt(0),
			Data:     outerCallData,
		})
		if err != nil {
			gasLimit = 500_000 // Default
			d.logger.Warn("gas estimation failed for setIsBatchPoster",
				slog.String("error", err.Error()),
			)
		}
		gasLimit = gasLimit * 120 / 100 // 20% buffer

		// Create transaction
		tx := types.NewTransaction(
			nonce,
			contracts.UpgradeExecutor,
			big.NewInt(0),
			gasLimit,
			gasPrice,
			outerCallData,
		)

		// Sign and send
		signedTx, err := d.signer.SignTransaction(ctx, tx)
		if err != nil {
			return fmt.Errorf("sign transaction: %w", err)
		}

		if err := client.SendTransaction(ctx, signedTx); err != nil {
			return fmt.Errorf("send transaction: %w", err)
		}

		d.logger.Info("setIsBatchPoster transaction submitted",
			slog.String("batch_poster", batchPoster.Hex()),
			slog.String("tx_hash", signedTx.Hash().Hex()),
		)

		// Wait for confirmation
		receipt, err := bind.WaitMined(ctx, client, signedTx)
		if err != nil {
			return fmt.Errorf("wait for receipt: %w", err)
		}

		if receipt.Status != types.ReceiptStatusSuccessful {
			return fmt.Errorf("setIsBatchPoster reverted for %s", batchPoster.Hex())
		}

		// Verify
		isNowWhitelisted, err := d.isBatchPoster(ctx, client, contracts.SequencerInbox, batchPoster)
		if err != nil {
			d.logger.Warn("failed to verify batch poster whitelisting",
				slog.String("error", err.Error()),
			)
		} else if !isNowWhitelisted {
			return fmt.Errorf("batch poster %s not whitelisted after transaction", batchPoster.Hex())
		}

		d.logger.Info("batch poster whitelisted successfully",
			slog.String("batch_poster", batchPoster.Hex()),
		)
	}

	return nil
}

// isBatchPoster checks if an address is whitelisted as a batch poster.
func (d *RollupDeployer) isBatchPoster(
	ctx context.Context,
	client *ethclient.Client,
	sequencerInbox common.Address,
	addr common.Address,
) (bool, error) {
	callData, err := d.sequencerInboxABI.Pack("isBatchPoster", addr)
	if err != nil {
		return false, err
	}

	result, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &sequencerInbox,
		Data: callData,
	}, nil)
	if err != nil {
		return false, err
	}

	var isWhitelisted bool
	if err := d.sequencerInboxABI.UnpackIntoInterface(&isWhitelisted, "isBatchPoster", result); err != nil {
		return false, err
	}

	return isWhitelisted, nil
}

// errorResult creates an error result.
func (d *RollupDeployer) errorResult(err error) (*GoDeployResult, error) {
	return &GoDeployResult{
		Success: false,
		Error:   err.Error(),
	}, nil
}

// Suppress unused warning for bytes package
var _ = bytes.Buffer{}
