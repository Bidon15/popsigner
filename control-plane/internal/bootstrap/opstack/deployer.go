// Package opstack provides OP Stack chain deployment infrastructure.
package opstack

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	openv "github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
)

// OPDeployer wraps the op-deployer library for OP Stack contract deployment.
// It manages the deployment pipeline stages and integrates with POPSigner for
// transaction signing.
type OPDeployer struct {
	logger   *slog.Logger
	cacheDir string
}

// OPDeployerConfig contains configuration for the OPDeployer.
type OPDeployerConfig struct {
	Logger   *slog.Logger
	CacheDir string // Directory for caching downloaded artifacts
}

// NewOPDeployer creates a new OPDeployer instance.
func NewOPDeployer(cfg OPDeployerConfig) *OPDeployer {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	cacheDir := cfg.CacheDir
	if cacheDir == "" {
		cacheDir = os.TempDir()
	}

	return &OPDeployer{
		logger:   logger,
		cacheDir: cacheDir,
	}
}

// DeployResult contains the result of an OP Stack deployment.
type DeployResult struct {
	// State is the complete deployment state from op-deployer
	State *state.State

	// SuperchainContracts contains addresses of superchain contracts
	SuperchainContracts *addresses.SuperchainContracts

	// ImplementationsContracts contains addresses of implementation contracts
	ImplementationsContracts *addresses.ImplementationsContracts

	// ChainStates contains state for each deployed chain
	ChainStates []*state.ChainState
}

// Deploy executes a full OP Stack deployment using the op-deployer pipeline.
// It runs all pipeline stages: Init, DeploySuperchain, DeployImplementations,
// DeployOPChain, and GenerateL2Genesis.
func (d *OPDeployer) Deploy(ctx context.Context, cfg *DeploymentConfig, signerAdapter *POPSignerAdapter) (*DeployResult, error) {
	d.logger.Info("starting OP Stack deployment via op-deployer",
		slog.String("chain_name", cfg.ChainName),
		slog.Uint64("chain_id", cfg.ChainID),
		slog.Uint64("l1_chain_id", cfg.L1ChainID),
	)

	// 1. Build the Intent from our config
	intent, err := BuildIntent(cfg)
	if err != nil {
		return nil, fmt.Errorf("build intent: %w", err)
	}

	// 2. Initialize state
	st := &state.State{
		Version: 1,
	}

	// 3. Connect to L1
	rpcClient, err := rpc.DialContext(ctx, cfg.L1RPC)
	if err != nil {
		return nil, fmt.Errorf("dial L1 RPC: %w", err)
	}
	defer rpcClient.Close()

	l1Client := ethclient.NewClient(rpcClient)

	// 4. Download and extract artifacts ourselves to avoid op-deployer's
	// finicky directory structure expectations
	d.logger.Info("downloading contract artifacts",
		slog.String("url", ContractArtifactURL),
	)

	// Clean any cached artifacts from op-deployer's cache to force fresh download
	// This ensures we always use the latest artifacts from S3
	if err := d.cleanArtifactCache(); err != nil {
		d.logger.Warn("failed to clean artifact cache", slog.String("error", err.Error()))
	}

	artifactDownloader := NewContractArtifactDownloader(d.cacheDir)
	artifactDir, err := artifactDownloader.Download(ctx, ContractArtifactURL)
	if err != nil {
		return nil, fmt.Errorf("download artifacts: %w", err)
	}

	d.logger.Info("artifacts downloaded and extracted",
		slog.String("path", artifactDir),
	)

	// Create file:// locator pointing to our extracted artifacts
	// op-deployer's file handler correctly looks for forge-artifacts/ subdirectory
	fileLocator, err := artifacts.NewFileLocator(artifactDir)
	if err != nil {
		return nil, fmt.Errorf("create file locator: %w", err)
	}

	// Update intent to use our local artifacts
	intent.L1ContractsLocator = fileLocator
	intent.L2ContractsLocator = fileLocator

	// Now use op-deployer's Download which will just use os.DirFS for file:// locators
	l1Artifacts, err := artifacts.Download(ctx, intent.L1ContractsLocator, nil, d.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("load L1 artifacts: %w", err)
	}

	// L2 uses same artifacts
	l2Artifacts := l1Artifacts

	bundle := pipeline.ArtifactsBundle{
		L1: l1Artifacts,
		L2: l2Artifacts,
	}

	d.logger.Info("artifacts downloaded successfully")

	// Debug: Log bytecode sizes from our downloaded artifacts
	d.logger.Info("checking bytecode sizes from downloaded artifacts", slog.String("path", artifactDir))
	d.logBytecodeSizes(artifactDir)

	// 5. Create broadcaster with our signer
	deployerAddr := common.HexToAddress(cfg.DeployerAddress)
	chainID := new(big.Int).SetUint64(cfg.L1ChainID)

	bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  d.gethLogger(),
		ChainID: chainID,
		Client:  l1Client,
		Signer:  signerAdapter.SignerFn(),
		From:    deployerAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("create broadcaster: %w", err)
	}

	// 6. Create script host for L1
	l1Host, err := openv.DefaultForkedScriptHost(
		ctx,
		bcaster,
		d.gethLogger(),
		deployerAddr,
		l1Artifacts,
		rpcClient,
	)
	if err != nil {
		return nil, fmt.Errorf("create L1 script host: %w", err)
	}

	// 7. Load all deployment scripts
	scripts, err := opcm.NewScripts(l1Host)
	if err != nil {
		return nil, fmt.Errorf("load deployment scripts: %w", err)
	}

	// 8. Create pipeline environment
	env := &pipeline.Env{
		StateWriter:  d.stateWriter(st),
		L1ScriptHost: l1Host,
		L1Client:     l1Client,
		Broadcaster:  bcaster,
		Deployer:     deployerAddr,
		Logger:       d.gethLogger(),
		Scripts:      scripts,
	}

	// 9. Run pipeline stages
	d.logger.Info("running pipeline: InitLiveStrategy")
	if err := pipeline.InitLiveStrategy(ctx, env, intent, st); err != nil {
		return nil, fmt.Errorf("init live strategy: %w", err)
	}

	// Broadcast any queued transactions and check for errors
	if _, err := bcaster.Broadcast(ctx); err != nil {
		return nil, fmt.Errorf("broadcast init transactions: %w", err)
	}

	d.logger.Info("running pipeline: DeploySuperchain")
	if err := pipeline.DeploySuperchain(env, intent, st); err != nil {
		return nil, fmt.Errorf("deploy superchain: %w", err)
	}
	if _, err := bcaster.Broadcast(ctx); err != nil {
		return nil, fmt.Errorf("broadcast superchain transactions: %w", err)
	}

	d.logger.Info("running pipeline: DeployImplementations")
	if err := pipeline.DeployImplementations(env, intent, st); err != nil {
		return nil, fmt.Errorf("deploy implementations: %w", err)
	}
	if _, err := bcaster.Broadcast(ctx); err != nil {
		return nil, fmt.Errorf("broadcast implementations transactions: %w", err)
	}

	// Deploy each chain
	for _, chainIntent := range intent.Chains {
		d.logger.Info("running pipeline: DeployOPChain", slog.String("chain_id", chainIntent.ID.Hex()))
		if err := pipeline.DeployOPChain(env, intent, st, chainIntent.ID); err != nil {
			return nil, fmt.Errorf("deploy OP chain %s: %w", chainIntent.ID.Hex(), err)
		}
		if _, err := bcaster.Broadcast(ctx); err != nil {
			return nil, fmt.Errorf("broadcast OP chain transactions: %w", err)
		}

		d.logger.Info("running pipeline: GenerateL2Genesis", slog.String("chain_id", chainIntent.ID.Hex()))
		if err := pipeline.GenerateL2Genesis(env, intent, bundle, st, chainIntent.ID); err != nil {
			return nil, fmt.Errorf("generate L2 genesis %s: %w", chainIntent.ID.Hex(), err)
		}
	}

	// 10. Store the applied intent
	st.AppliedIntent = intent

	d.logger.Info("OP Stack deployment completed successfully",
		slog.Int("chains_deployed", len(st.Chains)),
	)

	return &DeployResult{
		State:                    st,
		SuperchainContracts:      st.SuperchainDeployment,
		ImplementationsContracts: st.ImplementationsDeployment,
		ChainStates:              st.Chains,
	}, nil
}

// stateWriter returns a pipeline.StateWriter that updates the given state.
func (d *OPDeployer) stateWriter(st *state.State) pipeline.StateWriter {
	return stateWriterFunc(func(newState *state.State) error {
		*st = *newState
		return nil
	})
}

// stateWriterFunc is a function adapter for pipeline.StateWriter.
type stateWriterFunc func(st *state.State) error

func (f stateWriterFunc) WriteState(st *state.State) error {
	return f(st)
}

// gethLogger creates a go-ethereum compatible logger from slog.
func (d *OPDeployer) gethLogger() log.Logger {
	return log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true))
}

// cleanArtifactCache removes all cached artifacts to force fresh downloads.
// This ensures we always use the latest artifacts from S3.
func (d *OPDeployer) cleanArtifactCache() error {
	// Clean our custom download cache (artifacts-* directories and *.tzst files)
	entries, err := os.ReadDir(d.cacheDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		// Remove artifact directories and tzst files
		if strings.HasPrefix(name, "artifacts-") || strings.HasSuffix(name, ".tzst") {
			path := filepath.Join(d.cacheDir, name)
			d.logger.Info("cleaning cached artifact", slog.String("path", path))
			if err := os.RemoveAll(path); err != nil {
				d.logger.Warn("failed to remove cached artifact",
					slog.String("path", path),
					slog.String("error", err.Error()))
			}
		}
	}
	return nil
}

// logBytecodeSizes logs the bytecode sizes of key contracts for debugging.
// This helps identify which contracts might exceed the 24KB EIP-170 limit.
func (d *OPDeployer) logBytecodeSizes(artifactDir string) {
	contracts := []string{
		"OPContractsManager",
		"OPContractsManagerInterop",
		"OPContractsManagerStandardValidator",
		"OPContractsManagerGameTypeAdder",
		"OPContractsManagerDeployer",
		"OPContractsManagerUpgrader",
		"OptimismPortal2",
		"SystemConfig",
		"L1CrossDomainMessenger",
		"MIPS",
		// FaultDisputeGame contracts - often cause issues
		"FaultDisputeGame",
		"PermissionedDisputeGame",
		"FaultDisputeGameV2",
		"PermissionedDisputeGameV2",
		"PreimageOracle",
	}

	forgeDir := artifactDir + "/forge-artifacts"

	for _, name := range contracts {
		path := fmt.Sprintf("%s/%s.sol/%s.json", forgeDir, name, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip if not found
		}

		// Parse bytecode from JSON
		type artifact struct {
			Bytecode struct {
				Object string `json:"object"`
			} `json:"bytecode"`
			DeployedBytecode struct {
				Object string `json:"object"`
			} `json:"deployedBytecode"`
		}

		var a artifact
		if err := json.Unmarshal(data, &a); err != nil {
			continue
		}

		// Calculate sizes (hex string, so /2 for bytes, -2 for "0x" prefix)
		initSize := 0
		if len(a.Bytecode.Object) > 2 {
			initSize = (len(a.Bytecode.Object) - 2) / 2
		}
		deployedSize := 0
		if len(a.DeployedBytecode.Object) > 2 {
			deployedSize = (len(a.DeployedBytecode.Object) - 2) / 2
		}

		// Flag contracts that might cause issues
		status := "✓"
		if initSize > 24576 {
			status = "⚠️ INIT CODE EXCEEDS 24KB"
		} else if deployedSize > 24576 {
			status = "⚠️ DEPLOYED CODE EXCEEDS 24KB"
		} else if initSize > 20000 || deployedSize > 20000 {
			status = "⚠ CLOSE TO LIMIT"
		}

		d.logger.Info("contract bytecode size",
			slog.String("contract", name),
			slog.Int("init_bytes", initSize),
			slog.Int("deployed_bytes", deployedSize),
			slog.String("status", status),
		)
	}
}
