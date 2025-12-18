package opstack

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/bootstrap/repository"
)

// L1Client defines the interface for L1 Ethereum operations.
type L1Client interface {
	ChainID(ctx context.Context) (*big.Int, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	Close()
}

// L1ClientFactory creates L1 clients from RPC URLs.
type L1ClientFactory interface {
	Dial(ctx context.Context, rpcURL string) (L1Client, error)
}

// ProgressCallback is called during deployment to report progress.
type ProgressCallback func(stage Stage, progress float64, message string)

// OrchestratorConfig contains configuration for the orchestrator.
type OrchestratorConfig struct {
	// Logger for structured logging
	Logger *slog.Logger

	// CacheDir for op-deployer artifacts
	CacheDir string

	// RetryAttempts for transient failures within a stage
	RetryAttempts int

	// RetryDelay between retry attempts
	RetryDelay time.Duration
}

// Orchestrator coordinates OP Stack chain deployments.
// It manages the deployment lifecycle through multiple stages,
// integrates with SignerFn for transaction signing and StateWriter
// for state persistence, enabling resumable deployments.
type Orchestrator struct {
	repo          repository.Repository
	signerFactory SignerFactory
	l1Factory     L1ClientFactory
	config        OrchestratorConfig
	logger        *slog.Logger
}

// SignerFactory creates POPSigner instances for deployments.
type SignerFactory interface {
	CreateSigner(endpoint, apiKey string, chainID *big.Int) *POPSigner
}

// DefaultSignerFactory implements SignerFactory.
type DefaultSignerFactory struct{}

// CreateSigner creates a new POPSigner with the given configuration.
func (f *DefaultSignerFactory) CreateSigner(endpoint, apiKey string, chainID *big.Int) *POPSigner {
	return NewPOPSigner(SignerConfig{
		Endpoint: endpoint,
		APIKey:   apiKey,
		ChainID:  chainID,
	})
}

// NewOrchestrator creates a new deployment orchestrator.
func NewOrchestrator(
	repo repository.Repository,
	signerFactory SignerFactory,
	l1Factory L1ClientFactory,
	config OrchestratorConfig,
) *Orchestrator {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if config.RetryAttempts <= 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 5 * time.Second
	}

	return &Orchestrator{
		repo:          repo,
		signerFactory: signerFactory,
		l1Factory:     l1Factory,
		config:        config,
		logger:        logger,
	}
}

// DeploymentContext holds runtime context for a deployment.
type DeploymentContext struct {
	DeploymentID uuid.UUID
	Config       *DeploymentConfig
	StateWriter  *StateWriter
	Signer       *POPSigner
	L1Client     L1Client
	OnProgress   ProgressCallback
}

// Deploy executes an OP Stack deployment.
// It loads the deployment configuration, determines the starting stage
// (for resumability), and executes each stage in order.
func (o *Orchestrator) Deploy(ctx context.Context, deploymentID uuid.UUID, onProgress ProgressCallback) error {
	o.logger.Info("starting OP Stack deployment",
		slog.String("deployment_id", deploymentID.String()),
	)

	// 1. Load deployment from database
	deployment, err := o.repo.GetDeployment(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("load deployment: %w", err)
	}
	if deployment == nil {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	// 2. Parse configuration
	cfg, err := ParseConfig(deployment.Config)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	// 3. Create state writer
	stateWriter := NewStateWriter(o.repo, deploymentID)
	if onProgress != nil {
		stateWriter.SetUpdateCallback(func(id uuid.UUID, stage string) {
			onProgress(Stage(stage), 0, fmt.Sprintf("Entering stage: %s", stage))
		})
	}

	// 4. Create signer
	signer := o.signerFactory.CreateSigner(
		cfg.POPSignerEndpoint,
		cfg.POPSignerAPIKey,
		cfg.L1ChainIDBig(),
	)

	// 5. Connect to L1
	l1Client, err := o.l1Factory.Dial(ctx, cfg.L1RPC)
	if err != nil {
		return fmt.Errorf("connect to L1: %w", err)
	}
	defer l1Client.Close()

	// 6. Build deployment context
	dctx := &DeploymentContext{
		DeploymentID: deploymentID,
		Config:       cfg,
		StateWriter:  stateWriter,
		Signer:       signer,
		L1Client:     l1Client,
		OnProgress:   onProgress,
	}

	// 7. Determine starting stage (for resumability)
	startStage, err := o.determineStartStage(ctx, stateWriter)
	if err != nil {
		return fmt.Errorf("determine start stage: %w", err)
	}

	o.logger.Info("deployment will start from stage",
		slog.String("deployment_id", deploymentID.String()),
		slog.String("start_stage", startStage.String()),
	)

	// 8. Execute stages
	if err := o.executeStages(ctx, dctx, startStage); err != nil {
		// Mark as failed with error
		if markErr := stateWriter.MarkFailed(ctx, err.Error()); markErr != nil {
			o.logger.Error("failed to mark deployment as failed",
				slog.String("error", markErr.Error()),
			)
		}
		return err
	}

	// 9. Mark complete
	if err := stateWriter.MarkComplete(ctx); err != nil {
		return fmt.Errorf("mark complete: %w", err)
	}

	o.logger.Info("OP Stack deployment completed successfully",
		slog.String("deployment_id", deploymentID.String()),
	)

	if onProgress != nil {
		onProgress(StageCompleted, 1.0, "Deployment completed successfully!")
	}

	return nil
}

// determineStartStage returns the stage to start from based on previous progress.
func (o *Orchestrator) determineStartStage(ctx context.Context, stateWriter *StateWriter) (Stage, error) {
	canResume, err := stateWriter.CanResume(ctx)
	if err != nil {
		return StageInit, err
	}

	if !canResume {
		return StageInit, nil
	}

	currentStage, err := stateWriter.GetCurrentStage(ctx)
	if err != nil {
		return StageInit, err
	}

	// If deployment was previously at a stage, resume from that stage
	// (it may have partially completed before failure)
	return currentStage, nil
}

// executeStages runs all deployment stages from startStage.
func (o *Orchestrator) executeStages(ctx context.Context, dctx *DeploymentContext, startStage Stage) error {
	startIdx := StageIndex(startStage)
	if startIdx < 0 {
		return fmt.Errorf("invalid start stage: %s", startStage)
	}

	totalStages := len(StageOrder)

	for i := startIdx; i < totalStages; i++ {
		stage := StageOrder[i]

		// Skip completed stage marker
		if stage == StageCompleted {
			continue
		}

		// Calculate and report progress
		progress := float64(i) / float64(totalStages-1)
		if dctx.OnProgress != nil {
			dctx.OnProgress(stage, progress, fmt.Sprintf("Executing stage: %s", stage))
		}

		// Update stage in state writer
		if err := dctx.StateWriter.UpdateStage(ctx, stage); err != nil {
			return fmt.Errorf("update stage %s: %w", stage, err)
		}

		o.logger.Info("executing stage",
			slog.String("deployment_id", dctx.DeploymentID.String()),
			slog.String("stage", stage.String()),
			slog.Float64("progress", progress),
		)

		// Execute the stage with retry logic
		if err := o.executeStageWithRetry(ctx, dctx, stage); err != nil {
			return fmt.Errorf("stage %s failed: %w", stage, err)
		}

		o.logger.Info("stage completed",
			slog.String("deployment_id", dctx.DeploymentID.String()),
			slog.String("stage", stage.String()),
		)
	}

	return nil
}

// executeStageWithRetry executes a single stage with retry logic for transient failures.
func (o *Orchestrator) executeStageWithRetry(ctx context.Context, dctx *DeploymentContext, stage Stage) error {
	var lastErr error

	for attempt := 0; attempt < o.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			o.logger.Info("retrying stage",
				slog.String("stage", stage.String()),
				slog.Int("attempt", attempt+1),
				slog.Int("max_attempts", o.config.RetryAttempts),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(o.config.RetryDelay):
			}
		}

		err := o.executeStage(ctx, dctx, stage)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}

		o.logger.Warn("stage failed with retryable error",
			slog.String("stage", stage.String()),
			slog.String("error", err.Error()),
		)
	}

	return fmt.Errorf("stage failed after %d attempts: %w", o.config.RetryAttempts, lastErr)
}

// executeStage dispatches to the appropriate stage handler.
func (o *Orchestrator) executeStage(ctx context.Context, dctx *DeploymentContext, stage Stage) error {
	switch stage {
	case StageInit:
		return o.stageInit(ctx, dctx)
	case StageSuperchain:
		return o.stageSuperchain(ctx, dctx)
	case StageImplementations:
		return o.stageImplementations(ctx, dctx)
	case StageOPChain:
		return o.stageOPChain(ctx, dctx)
	case StageAltDA:
		return o.stageAltDA(ctx, dctx)
	case StageGenesis:
		return o.stageGenesis(ctx, dctx)
	case StageStartBlock:
		return o.stageStartBlock(ctx, dctx)
	default:
		return fmt.Errorf("unknown stage: %s", stage)
	}
}

// stageInit validates L1 connection and configuration.
func (o *Orchestrator) stageInit(ctx context.Context, dctx *DeploymentContext) error {
	// Validate L1 chain ID
	chainID, err := dctx.L1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("get L1 chain ID: %w", err)
	}

	expectedChainID := dctx.Config.L1ChainIDBig()
	if chainID.Cmp(expectedChainID) != 0 {
		return fmt.Errorf("L1 chain ID mismatch: expected %s, got %s", expectedChainID, chainID)
	}

	// Check deployer balance
	deployerAddr := common.HexToAddress(dctx.Config.DeployerAddress)
	balance, err := dctx.L1Client.BalanceAt(ctx, deployerAddr, nil)
	if err != nil {
		return fmt.Errorf("get deployer balance: %w", err)
	}

	requiredFunding := dctx.Config.RequiredFundingWei
	if balance.Cmp(requiredFunding) < 0 {
		return fmt.Errorf("insufficient deployer balance: have %s wei, need %s wei", balance, requiredFunding)
	}

	o.logger.Info("init stage completed",
		slog.String("l1_chain_id", chainID.String()),
		slog.String("deployer_balance", balance.String()),
	)

	// Save init state
	initState := map[string]interface{}{
		"l1_chain_id":      chainID.String(),
		"deployer_address": dctx.Config.DeployerAddress,
		"deployer_balance": balance.String(),
		"initialized_at":   time.Now().UTC().Format(time.RFC3339),
	}

	return dctx.StateWriter.WriteState(ctx, initState)
}

// stageSuperchain deploys superchain contracts.
// In a full implementation, this would call op-deployer's pipeline.DeploySuperchain.
func (o *Orchestrator) stageSuperchain(ctx context.Context, dctx *DeploymentContext) error {
	// Check if already completed (idempotency)
	complete, err := dctx.StateWriter.IsStageComplete(ctx, StageSuperchain)
	if err != nil {
		return err
	}
	if complete {
		o.logger.Info("skipping superchain stage (already complete)")
		return nil
	}

	// Read existing state to check for partial completion
	existingState, err := dctx.StateWriter.ReadState(ctx)
	if err != nil {
		return fmt.Errorf("read existing state: %w", err)
	}

	var state map[string]interface{}
	if existingState != nil {
		if err := json.Unmarshal(existingState, &state); err != nil {
			state = make(map[string]interface{})
		}
	} else {
		state = make(map[string]interface{})
	}

	// In production, this would call:
	// pipeline.DeploySuperchain(pEnv, intent, st)
	// For now, we record the stage as a placeholder
	state["superchain_deployed"] = true
	state["superchain_deployed_at"] = time.Now().UTC().Format(time.RFC3339)

	// Record a placeholder transaction
	txHash := fmt.Sprintf("0x%s_superchain", dctx.DeploymentID.String()[:8])
	if err := dctx.StateWriter.RecordTransaction(ctx, StageSuperchain, txHash, "Deploy SuperchainConfig"); err != nil {
		return fmt.Errorf("record transaction: %w", err)
	}

	return dctx.StateWriter.WriteState(ctx, state)
}

// stageImplementations deploys implementation contracts.
func (o *Orchestrator) stageImplementations(ctx context.Context, dctx *DeploymentContext) error {
	complete, err := dctx.StateWriter.IsStageComplete(ctx, StageImplementations)
	if err != nil {
		return err
	}
	if complete {
		o.logger.Info("skipping implementations stage (already complete)")
		return nil
	}

	existingState, _ := dctx.StateWriter.ReadState(ctx)
	var state map[string]interface{}
	if existingState != nil {
		json.Unmarshal(existingState, &state)
	}
	if state == nil {
		state = make(map[string]interface{})
	}

	// In production, this would call:
	// pipeline.DeployImplementations(pEnv, intent, st)
	state["implementations_deployed"] = true
	state["implementations_deployed_at"] = time.Now().UTC().Format(time.RFC3339)

	txHash := fmt.Sprintf("0x%s_implementations", dctx.DeploymentID.String()[:8])
	if err := dctx.StateWriter.RecordTransaction(ctx, StageImplementations, txHash, "Deploy Implementations"); err != nil {
		return fmt.Errorf("record transaction: %w", err)
	}

	return dctx.StateWriter.WriteState(ctx, state)
}

// stageOPChain deploys the OP chain contracts.
func (o *Orchestrator) stageOPChain(ctx context.Context, dctx *DeploymentContext) error {
	complete, err := dctx.StateWriter.IsStageComplete(ctx, StageOPChain)
	if err != nil {
		return err
	}
	if complete {
		o.logger.Info("skipping opchain stage (already complete)")
		return nil
	}

	existingState, _ := dctx.StateWriter.ReadState(ctx)
	var state map[string]interface{}
	if existingState != nil {
		json.Unmarshal(existingState, &state)
	}
	if state == nil {
		state = make(map[string]interface{})
	}

	// In production, this would call:
	// pipeline.DeployOPChain(pEnv, intent, st, chainID)
	state["opchain_deployed"] = true
	state["opchain_deployed_at"] = time.Now().UTC().Format(time.RFC3339)
	state["chain_id"] = dctx.Config.ChainID

	txHash := fmt.Sprintf("0x%s_opchain", dctx.DeploymentID.String()[:8])
	if err := dctx.StateWriter.RecordTransaction(ctx, StageOPChain, txHash, "Deploy OPChain"); err != nil {
		return fmt.Errorf("record transaction: %w", err)
	}

	return dctx.StateWriter.WriteState(ctx, state)
}

// stageAltDA deploys Alt-DA contracts if enabled.
func (o *Orchestrator) stageAltDA(ctx context.Context, dctx *DeploymentContext) error {
	// Skip if Alt-DA not enabled
	if !dctx.Config.UseAltDA {
		o.logger.Info("skipping alt-da stage (not enabled)")
		return nil
	}

	complete, err := dctx.StateWriter.IsStageComplete(ctx, StageAltDA)
	if err != nil {
		return err
	}
	if complete {
		o.logger.Info("skipping alt-da stage (already complete)")
		return nil
	}

	existingState, _ := dctx.StateWriter.ReadState(ctx)
	var state map[string]interface{}
	if existingState != nil {
		json.Unmarshal(existingState, &state)
	}
	if state == nil {
		state = make(map[string]interface{})
	}

	// In production, this would call:
	// pipeline.DeployAltDA(pEnv, intent, st, chainID)
	// Note: GenericCommitment requires 0 transactions
	state["alt_da_deployed"] = true
	state["alt_da_deployed_at"] = time.Now().UTC().Format(time.RFC3339)
	state["da_commitment_type"] = dctx.Config.DACommitmentType

	return dctx.StateWriter.WriteState(ctx, state)
}

// stageGenesis generates the L2 genesis file.
func (o *Orchestrator) stageGenesis(ctx context.Context, dctx *DeploymentContext) error {
	complete, err := dctx.StateWriter.IsStageComplete(ctx, StageGenesis)
	if err != nil {
		return err
	}
	if complete {
		o.logger.Info("skipping genesis stage (already complete)")
		return nil
	}

	existingState, _ := dctx.StateWriter.ReadState(ctx)
	var state map[string]interface{}
	if existingState != nil {
		json.Unmarshal(existingState, &state)
	}
	if state == nil {
		state = make(map[string]interface{})
	}

	// In production, this would call:
	// pipeline.GenerateL2Genesis(pEnv, intent, bundle, st, chainID)
	// This is a local computation, no on-chain transactions
	state["genesis_generated"] = true
	state["genesis_generated_at"] = time.Now().UTC().Format(time.RFC3339)

	// Save genesis as artifact
	genesisData := json.RawMessage(`{"placeholder": "genesis data would be here"}`)
	if err := dctx.StateWriter.SaveArtifact(ctx, "genesis", genesisData); err != nil {
		return fmt.Errorf("save genesis artifact: %w", err)
	}

	return dctx.StateWriter.WriteState(ctx, state)
}

// stageStartBlock sets the L2 start block.
func (o *Orchestrator) stageStartBlock(ctx context.Context, dctx *DeploymentContext) error {
	complete, err := dctx.StateWriter.IsStageComplete(ctx, StageStartBlock)
	if err != nil {
		return err
	}
	if complete {
		o.logger.Info("skipping start-block stage (already complete)")
		return nil
	}

	existingState, _ := dctx.StateWriter.ReadState(ctx)
	var state map[string]interface{}
	if existingState != nil {
		json.Unmarshal(existingState, &state)
	}
	if state == nil {
		state = make(map[string]interface{})
	}

	// In production, this would call:
	// pipeline.SetStartBlockLiveStrategy(ctx, intent, pEnv, st, chainID)
	// This reads the current L1 block and sets it as the anchor
	state["start_block_set"] = true
	state["start_block_set_at"] = time.Now().UTC().Format(time.RFC3339)

	// Save rollup config as artifact
	rollupConfig := json.RawMessage(`{"placeholder": "rollup config would be here"}`)
	if err := dctx.StateWriter.SaveArtifact(ctx, "rollup_config", rollupConfig); err != nil {
		return fmt.Errorf("save rollup config artifact: %w", err)
	}

	return dctx.StateWriter.WriteState(ctx, state)
}

// Resume attempts to resume a paused or failed deployment.
func (o *Orchestrator) Resume(ctx context.Context, deploymentID uuid.UUID, onProgress ProgressCallback) error {
	o.logger.Info("resuming OP Stack deployment",
		slog.String("deployment_id", deploymentID.String()),
	)

	stateWriter := NewStateWriter(o.repo, deploymentID)

	canResume, err := stateWriter.CanResume(ctx)
	if err != nil {
		return fmt.Errorf("check resume capability: %w", err)
	}
	if !canResume {
		return fmt.Errorf("deployment cannot be resumed (status is not paused, running, or failed)")
	}

	// Delegate to Deploy - it will determine the start stage
	return o.Deploy(ctx, deploymentID, onProgress)
}

// Pause marks a running deployment as paused.
func (o *Orchestrator) Pause(ctx context.Context, deploymentID uuid.UUID) error {
	stateWriter := NewStateWriter(o.repo, deploymentID)
	return stateWriter.MarkPaused(ctx)
}

// GetDeploymentStatus returns the current status of a deployment.
func (o *Orchestrator) GetDeploymentStatus(ctx context.Context, deploymentID uuid.UUID) (*DeploymentStatus, error) {
	deployment, err := o.repo.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("get deployment: %w", err)
	}
	if deployment == nil {
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	stateWriter := NewStateWriter(o.repo, deploymentID)
	transactions, err := stateWriter.GetTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get transactions: %w", err)
	}

	currentStage, _ := stateWriter.GetCurrentStage(ctx)

	return &DeploymentStatus{
		DeploymentID:     deploymentID,
		Status:           deployment.Status,
		CurrentStage:     currentStage,
		TransactionCount: len(transactions),
		Error:            deployment.ErrorMessage,
	}, nil
}

// DeploymentStatus represents the current state of a deployment.
type DeploymentStatus struct {
	DeploymentID     uuid.UUID
	Status           repository.Status
	CurrentStage     Stage
	TransactionCount int
	Error            *string
}

