// Package preflight provides pre-deployment validation checks.
package preflight

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// DefaultTimeout is the default timeout for RPC calls.
const DefaultTimeout = 10 * time.Second

// CheckName identifies a specific pre-flight check.
type CheckName string

const (
	// CheckL1Reachable verifies the L1 RPC endpoint is reachable.
	CheckL1Reachable CheckName = "l1_reachable"
	// CheckChainIDMatch verifies the L1 chain ID matches the expected value.
	CheckChainIDMatch CheckName = "chain_id_match"
	// CheckDeployerBalance verifies the deployer has sufficient funds.
	CheckDeployerBalance CheckName = "deployer_balance"
)

// CheckResult represents the result of a single pre-flight check.
type CheckResult struct {
	Name    CheckName              `json:"name"`
	Passed  bool                   `json:"passed"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// PreflightRequest contains the parameters for pre-flight checks.
type PreflightRequest struct {
	L1RPC           string `json:"l1_rpc"`
	L1ChainID       uint64 `json:"l1_chain_id"`
	DeployerAddress string `json:"deployer_address"`
}

// PreflightResponse contains the results of all pre-flight checks.
type PreflightResponse struct {
	OK                 bool          `json:"ok"`
	Checks             []CheckResult `json:"checks"`
	DeployerAddress    string        `json:"deployer_address"`
	RequiredFundingETH string        `json:"required_funding_eth"`
	CurrentBalanceETH  string        `json:"current_balance_eth,omitempty"`
}

// Checker performs pre-flight validation checks.
type Checker struct {
	timeout time.Duration
}

// NewChecker creates a new pre-flight checker.
func NewChecker() *Checker {
	return &Checker{
		timeout: DefaultTimeout,
	}
}

// WithTimeout sets a custom timeout for RPC calls.
func (c *Checker) WithTimeout(timeout time.Duration) *Checker {
	c.timeout = timeout
	return c
}

// RunChecks performs all pre-flight checks and returns the results.
func (c *Checker) RunChecks(ctx context.Context, req *PreflightRequest) (*PreflightResponse, error) {
	if err := c.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Create timeout context for RPC calls
	rpcCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response := &PreflightResponse{
		OK:              true,
		Checks:          make([]CheckResult, 0, 3),
		DeployerAddress: req.DeployerAddress,
	}

	// Calculate required funding
	requiredWei := c.getRequiredFunding(req.L1ChainID)
	response.RequiredFundingETH = weiToETHString(requiredWei)

	// Check 1: L1 Reachable
	client, reachableResult := c.checkL1Reachable(rpcCtx, req.L1RPC)
	response.Checks = append(response.Checks, reachableResult)
	if !reachableResult.Passed {
		response.OK = false
		return response, nil // Can't continue without connection
	}
	defer client.Close()

	// Check 2: Chain ID Match
	chainIDResult := c.checkChainIDMatch(rpcCtx, client, req.L1ChainID)
	response.Checks = append(response.Checks, chainIDResult)
	if !chainIDResult.Passed {
		response.OK = false
	}

	// Check 3: Deployer Balance
	balanceResult := c.checkDeployerBalance(rpcCtx, client, req.DeployerAddress, requiredWei)
	response.Checks = append(response.Checks, balanceResult)
	if !balanceResult.Passed {
		response.OK = false
	}

	// Extract current balance for response
	if details := balanceResult.Details; details != nil {
		if haveETH, ok := details["have_eth"].(string); ok {
			response.CurrentBalanceETH = haveETH
		}
	}

	return response, nil
}

// validateRequest validates the pre-flight request parameters.
func (c *Checker) validateRequest(req *PreflightRequest) error {
	if req.L1RPC == "" {
		return fmt.Errorf("l1_rpc is required")
	}
	if req.L1ChainID == 0 {
		return fmt.Errorf("l1_chain_id is required")
	}
	if req.DeployerAddress == "" {
		return fmt.Errorf("deployer_address is required")
	}
	if !common.IsHexAddress(req.DeployerAddress) {
		return fmt.Errorf("deployer_address is not a valid Ethereum address")
	}
	return nil
}

// checkL1Reachable verifies the L1 RPC endpoint is reachable.
func (c *Checker) checkL1Reachable(ctx context.Context, rpcURL string) (*ethclient.Client, CheckResult) {
	result := CheckResult{
		Name: CheckL1Reachable,
	}

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Failed to connect to L1 RPC: %v", err)
		result.Details = map[string]interface{}{
			"error": err.Error(),
		}
		return nil, result
	}

	// Verify connection works by making a simple call
	_, err = client.ChainID(ctx)
	if err != nil {
		client.Close()
		result.Passed = false
		result.Message = fmt.Sprintf("L1 RPC connection failed: %v", err)
		result.Details = map[string]interface{}{
			"error": err.Error(),
		}
		return nil, result
	}

	result.Passed = true
	result.Message = "Connected to L1 RPC successfully"
	return client, result
}

// checkChainIDMatch verifies the L1 chain ID matches the expected value.
func (c *Checker) checkChainIDMatch(ctx context.Context, client *ethclient.Client, expectedChainID uint64) CheckResult {
	result := CheckResult{
		Name: CheckChainIDMatch,
	}

	actualChainID, err := client.ChainID(ctx)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Failed to get chain ID: %v", err)
		result.Details = map[string]interface{}{
			"error": err.Error(),
		}
		return result
	}

	expected := new(big.Int).SetUint64(expectedChainID)
	if actualChainID.Cmp(expected) != 0 {
		result.Passed = false
		result.Message = fmt.Sprintf("Chain ID mismatch: expected %d, got %d", expectedChainID, actualChainID.Uint64())
		result.Details = map[string]interface{}{
			"expected": expectedChainID,
			"actual":   actualChainID.Uint64(),
		}
		return result
	}

	result.Passed = true
	result.Message = fmt.Sprintf("Chain ID %d confirmed", expectedChainID)
	result.Details = map[string]interface{}{
		"chain_id": expectedChainID,
	}
	return result
}

// checkDeployerBalance verifies the deployer has sufficient funds.
func (c *Checker) checkDeployerBalance(ctx context.Context, client *ethclient.Client, deployerAddr string, requiredWei *big.Int) CheckResult {
	result := CheckResult{
		Name: CheckDeployerBalance,
	}

	addr := common.HexToAddress(deployerAddr)
	balance, err := client.BalanceAt(ctx, addr, nil)
	if err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("Failed to get deployer balance: %v", err)
		result.Details = map[string]interface{}{
			"error": err.Error(),
		}
		return result
	}

	haveETH := weiToETHString(balance)
	needETH := weiToETHString(requiredWei)

	result.Details = map[string]interface{}{
		"have_wei": balance.String(),
		"need_wei": requiredWei.String(),
		"have_eth": haveETH,
		"need_eth": needETH,
	}

	if balance.Cmp(requiredWei) < 0 {
		result.Passed = false
		result.Message = fmt.Sprintf("Insufficient deployer balance: have %s ETH, need %s ETH", haveETH, needETH)
		return result
	}

	result.Passed = true
	result.Message = fmt.Sprintf("Deployer has sufficient balance: %s ETH", haveETH)
	return result
}

// getRequiredFunding returns the required funding in wei based on the network.
func (c *Checker) getRequiredFunding(chainID uint64) *big.Int {
	switch chainID {
	case 1: // Ethereum Mainnet
		// 5 ETH
		return new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18))
	default:
		// Testnets: 1 ETH
		return big.NewInt(1e18)
	}
}

// weiToETHString converts wei to a human-readable ETH string.
func weiToETHString(wei *big.Int) string {
	if wei == nil {
		return "0"
	}

	// Convert to float for display
	weiFloat := new(big.Float).SetInt(wei)
	ethFloat := new(big.Float).Quo(weiFloat, big.NewFloat(1e18))

	// Format with up to 4 decimal places
	return ethFloat.Text('f', 4)
}

// GetNetworkName returns a human-readable name for a chain ID.
func GetNetworkName(chainID uint64) string {
	switch chainID {
	case 1:
		return "Ethereum Mainnet"
	case 11155111:
		return "Sepolia"
	case 17000:
		return "Holesky"
	case 5:
		return "Goerli (deprecated)"
	default:
		return fmt.Sprintf("Chain %d", chainID)
	}
}

