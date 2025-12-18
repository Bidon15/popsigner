package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/bootstrap/bundle"
	"github.com/Bidon15/popsigner/control-plane/internal/bootstrap/preflight"
	"github.com/Bidon15/popsigner/control-plane/internal/bootstrap/repository"
	apierrors "github.com/Bidon15/popsigner/control-plane/internal/pkg/errors"
	"github.com/Bidon15/popsigner/control-plane/internal/pkg/response"
)

// Orchestrator defines the interface for starting deployments.
// This will be implemented by the stack-specific orchestrators.
type Orchestrator interface {
	StartDeployment(ctx context.Context, deploymentID uuid.UUID) error
}

// noopOrchestrator is a placeholder orchestrator that does nothing.
// Used when no orchestrator is configured.
type noopOrchestrator struct{}

func (n *noopOrchestrator) StartDeployment(_ context.Context, _ uuid.UUID) error {
	return nil
}

// DeploymentHandler handles deployment-related HTTP requests.
type DeploymentHandler struct {
	repo         repository.Repository
	orchestrator Orchestrator
	bundler      *bundle.Bundler
}

// NewDeploymentHandler creates a new deployment handler.
func NewDeploymentHandler(repo repository.Repository, orch Orchestrator) *DeploymentHandler {
	if orch == nil {
		orch = &noopOrchestrator{}
	}
	return &DeploymentHandler{
		repo:         repo,
		orchestrator: orch,
	}
}

// SetBundler sets the bundler for bundle generation.
func (h *DeploymentHandler) SetBundler(b *bundle.Bundler) {
	h.bundler = b
}

// Create handles POST /api/v1/deployments
func (h *DeploymentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("invalid request body"))
		return
	}

	// Validate required fields
	if req.ChainID == 0 {
		response.Error(w, apierrors.NewValidationError("chain_id", "chain_id is required"))
		return
	}

	if req.Stack == "" {
		response.Error(w, apierrors.NewValidationError("stack", "stack is required"))
		return
	}

	// Validate stack type
	stack := repository.Stack(req.Stack)
	if stack != repository.StackOPStack && stack != repository.StackNitro {
		response.Error(w, apierrors.NewValidationError("stack", "stack must be 'opstack' or 'nitro'"))
		return
	}

	if len(req.Config) == 0 {
		response.Error(w, apierrors.NewValidationError("config", "config is required"))
		return
	}

	// Check for existing deployment with same chain_id
	existing, err := h.repo.GetDeploymentByChainID(r.Context(), req.ChainID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		response.Error(w, apierrors.ErrInternal)
		return
	}
	if existing != nil {
		response.Error(w, apierrors.NewConflictError("deployment already exists for this chain_id"))
		return
	}

	// Create deployment
	deployment := &repository.Deployment{
		ID:      uuid.New(),
		ChainID: req.ChainID,
		Stack:   stack,
		Status:  repository.StatusPending,
		Config:  req.Config,
	}

	if err := h.repo.CreateDeployment(r.Context(), deployment); err != nil {
		response.Error(w, apierrors.ErrInternal.WithMessage("failed to create deployment"))
		return
	}

	response.Created(w, toDeploymentResponse(deployment))
}

// Get handles GET /api/v1/deployments/{id}
func (h *DeploymentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("invalid deployment ID"))
		return
	}

	deployment, err := h.repo.GetDeployment(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(w, apierrors.NewNotFoundError("deployment"))
			return
		}
		response.Error(w, apierrors.ErrInternal)
		return
	}

	response.OK(w, toDeploymentResponse(deployment))
}

// Start handles POST /api/v1/deployments/{id}/start
func (h *DeploymentHandler) Start(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("invalid deployment ID"))
		return
	}

	deployment, err := h.repo.GetDeployment(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(w, apierrors.NewNotFoundError("deployment"))
			return
		}
		response.Error(w, apierrors.ErrInternal)
		return
	}

	// Only pending or paused deployments can be started
	if deployment.Status != repository.StatusPending && deployment.Status != repository.StatusPaused {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("deployment cannot be started (status: "+string(deployment.Status)+")"))
		return
	}

	// Update status to running
	if err := h.repo.UpdateDeploymentStatus(r.Context(), id, repository.StatusRunning, nil); err != nil {
		response.Error(w, apierrors.ErrInternal.WithMessage("failed to update deployment status"))
		return
	}

	// Start deployment asynchronously
	go func() {
		if err := h.orchestrator.StartDeployment(context.Background(), id); err != nil {
			// Error is handled by the orchestrator (updates deployment status)
			_ = h.repo.SetDeploymentError(context.Background(), id, err.Error())
		}
	}()

	response.Accepted(w, &StartResponse{
		Status:  "started",
		Message: "Deployment started. Poll GET /api/v1/deployments/" + id.String() + " for status updates.",
	})
}

// GetArtifacts handles GET /api/v1/deployments/{id}/artifacts
func (h *DeploymentHandler) GetArtifacts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("invalid deployment ID"))
		return
	}

	// Verify deployment exists
	_, err = h.repo.GetDeployment(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(w, apierrors.NewNotFoundError("deployment"))
			return
		}
		response.Error(w, apierrors.ErrInternal)
		return
	}

	artifacts, err := h.repo.GetAllArtifacts(r.Context(), id)
	if err != nil {
		response.Error(w, apierrors.ErrInternal.WithMessage("failed to fetch artifacts"))
		return
	}

	artifactResponses := make([]ArtifactResponse, len(artifacts))
	for i, a := range artifacts {
		artifactResponses[i] = *toArtifactResponse(&a)
	}

	response.OK(w, &ArtifactListResponse{Artifacts: artifactResponses})
}

// GetArtifact handles GET /api/v1/deployments/{id}/artifacts/{type}
func (h *DeploymentHandler) GetArtifact(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("invalid deployment ID"))
		return
	}

	artifactType := chi.URLParam(r, "type")
	if artifactType == "" {
		response.Error(w, apierrors.NewValidationError("type", "artifact type is required"))
		return
	}

	artifact, err := h.repo.GetArtifact(r.Context(), id, artifactType)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(w, apierrors.NewNotFoundError("artifact"))
			return
		}
		response.Error(w, apierrors.ErrInternal)
		return
	}

	response.OK(w, toArtifactResponse(artifact))
}

// GetBundle handles GET /api/v1/deployments/{id}/bundle
// Returns a downloadable .tar.gz bundle containing all deployment artifacts.
func (h *DeploymentHandler) GetBundle(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("invalid deployment ID"))
		return
	}

	// Verify deployment exists
	deployment, err := h.repo.GetDeployment(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(w, apierrors.NewNotFoundError("deployment"))
			return
		}
		response.Error(w, apierrors.ErrInternal)
		return
	}

	// Check if bundler is configured
	if h.bundler == nil {
		response.Error(w, apierrors.ErrInternal.WithMessage("bundler not configured"))
		return
	}

	// Generate bundle
	bundleResult, err := h.bundler.CreateBundle(r.Context(), id)
	if err != nil {
		response.Error(w, apierrors.ErrInternal.WithMessage(fmt.Sprintf("failed to generate bundle: %v", err)))
		return
	}

	// Use filename from result or generate from deployment
	filename := bundleResult.Filename
	if filename == "" {
		chainName := h.extractChainName(deployment)
		filename = fmt.Sprintf("%s-%s-artifacts.tar.gz", chainName, deployment.Stack)
	}

	// Set headers for download
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(bundleResult.Data)))

	w.Write(bundleResult.Data)
}

// extractChainName gets the chain name from deployment config.
func (h *DeploymentHandler) extractChainName(d *repository.Deployment) string {
	if d.Config != nil {
		type config struct {
			ChainName string `json:"chain_name"`
			Name      string `json:"name"`
		}
		var cfg config
		if err := json.Unmarshal(d.Config, &cfg); err == nil {
			if cfg.ChainName != "" {
				return cfg.ChainName
			}
			if cfg.Name != "" {
				return cfg.Name
			}
		}
	}
	return fmt.Sprintf("chain-%d", d.ChainID)
}

// GetTransactions handles GET /api/v1/deployments/{id}/transactions
func (h *DeploymentHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("invalid deployment ID"))
		return
	}

	// Verify deployment exists
	_, err = h.repo.GetDeployment(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(w, apierrors.NewNotFoundError("deployment"))
			return
		}
		response.Error(w, apierrors.ErrInternal)
		return
	}

	transactions, err := h.repo.GetTransactionsByDeployment(r.Context(), id)
	if err != nil {
		response.Error(w, apierrors.ErrInternal.WithMessage("failed to fetch transactions"))
		return
	}

	txResponses := make([]*TransactionResponse, len(transactions))
	for i, tx := range transactions {
		txResponses[i] = toTransactionResponse(&tx)
	}

	response.OK(w, txResponses)
}

// List handles GET /api/v1/deployments
func (h *DeploymentHandler) List(w http.ResponseWriter, r *http.Request) {
	// Optional status filter
	statusFilter := r.URL.Query().Get("status")

	var deployments []*repository.Deployment
	var err error

	if statusFilter != "" {
		status := repository.Status(statusFilter)
		deployments, err = h.repo.ListDeploymentsByStatus(r.Context(), status)
	} else {
		// List all - fetch each status and combine
		// For now, just return pending ones as a simple implementation
		deployments, err = h.repo.ListDeploymentsByStatus(r.Context(), repository.StatusPending)
		if err == nil {
			running, _ := h.repo.ListDeploymentsByStatus(r.Context(), repository.StatusRunning)
			deployments = append(deployments, running...)
			completed, _ := h.repo.ListDeploymentsByStatus(r.Context(), repository.StatusCompleted)
			deployments = append(deployments, completed...)
			failed, _ := h.repo.ListDeploymentsByStatus(r.Context(), repository.StatusFailed)
			deployments = append(deployments, failed...)
			paused, _ := h.repo.ListDeploymentsByStatus(r.Context(), repository.StatusPaused)
			deployments = append(deployments, paused...)
		}
	}

	if err != nil {
		response.Error(w, apierrors.ErrInternal.WithMessage("failed to list deployments"))
		return
	}

	responses := make([]*DeploymentResponse, len(deployments))
	for i, d := range deployments {
		responses[i] = toDeploymentResponse(d)
	}

	response.OK(w, responses)
}

// PreflightRequest is the request body for pre-flight checks.
type PreflightRequest struct {
	L1RPC           string `json:"l1_rpc"`
	L1ChainID       uint64 `json:"l1_chain_id"`
	DeployerAddress string `json:"deployer_address"`
}

// Preflight handles POST /api/v1/deployments/preflight
// It performs pre-flight checks before starting a deployment.
func (h *DeploymentHandler) Preflight(w http.ResponseWriter, r *http.Request) {
	var req PreflightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage("invalid request body"))
		return
	}

	// Validate required fields
	if req.L1RPC == "" {
		response.Error(w, apierrors.NewValidationError("l1_rpc", "l1_rpc is required"))
		return
	}
	if req.L1ChainID == 0 {
		response.Error(w, apierrors.NewValidationError("l1_chain_id", "l1_chain_id is required"))
		return
	}
	if req.DeployerAddress == "" {
		response.Error(w, apierrors.NewValidationError("deployer_address", "deployer_address is required"))
		return
	}

	// Run pre-flight checks
	checker := preflight.NewChecker()
	preflightReq := &preflight.PreflightRequest{
		L1RPC:           req.L1RPC,
		L1ChainID:       req.L1ChainID,
		DeployerAddress: req.DeployerAddress,
	}

	result, err := checker.RunChecks(r.Context(), preflightReq)
	if err != nil {
		response.Error(w, apierrors.ErrBadRequest.WithMessage(err.Error()))
		return
	}

	response.OK(w, result)
}

