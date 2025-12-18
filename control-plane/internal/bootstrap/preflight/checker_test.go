package preflight

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChecker(t *testing.T) {
	checker := NewChecker()
	assert.NotNil(t, checker)
	assert.Equal(t, DefaultTimeout, checker.timeout)
}

func TestChecker_WithTimeout(t *testing.T) {
	checker := NewChecker().WithTimeout(5 * time.Second)
	assert.Equal(t, 5*time.Second, checker.timeout)
}

func TestChecker_ValidateRequest(t *testing.T) {
	checker := NewChecker()

	tests := []struct {
		name    string
		req     *PreflightRequest
		wantErr string
	}{
		{
			name: "valid request",
			req: &PreflightRequest{
				L1RPC:           "https://eth-sepolia.example.com",
				L1ChainID:       11155111,
				DeployerAddress: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "",
		},
		{
			name: "missing l1_rpc",
			req: &PreflightRequest{
				L1ChainID:       11155111,
				DeployerAddress: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "l1_rpc is required",
		},
		{
			name: "missing l1_chain_id",
			req: &PreflightRequest{
				L1RPC:           "https://eth-sepolia.example.com",
				DeployerAddress: "0x1234567890123456789012345678901234567890",
			},
			wantErr: "l1_chain_id is required",
		},
		{
			name: "missing deployer_address",
			req: &PreflightRequest{
				L1RPC:     "https://eth-sepolia.example.com",
				L1ChainID: 11155111,
			},
			wantErr: "deployer_address is required",
		},
		{
			name: "invalid deployer_address",
			req: &PreflightRequest{
				L1RPC:           "https://eth-sepolia.example.com",
				L1ChainID:       11155111,
				DeployerAddress: "not-an-address",
			},
			wantErr: "deployer_address is not a valid Ethereum address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checker.validateRequest(tc.req)
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestChecker_GetRequiredFunding(t *testing.T) {
	checker := NewChecker()

	tests := []struct {
		name     string
		chainID  uint64
		expected *big.Int
	}{
		{
			name:     "mainnet requires 5 ETH",
			chainID:  1,
			expected: new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18)),
		},
		{
			name:     "sepolia requires 1 ETH",
			chainID:  11155111,
			expected: big.NewInt(1e18),
		},
		{
			name:     "holesky requires 1 ETH",
			chainID:  17000,
			expected: big.NewInt(1e18),
		},
		{
			name:     "unknown testnet requires 1 ETH",
			chainID:  999999,
			expected: big.NewInt(1e18),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.getRequiredFunding(tc.chainID)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestWeiToETHString(t *testing.T) {
	tests := []struct {
		name     string
		wei      *big.Int
		expected string
	}{
		{
			name:     "nil returns 0",
			wei:      nil,
			expected: "0",
		},
		{
			name:     "0 wei",
			wei:      big.NewInt(0),
			expected: "0.0000",
		},
		{
			name:     "1 ETH",
			wei:      big.NewInt(1e18),
			expected: "1.0000",
		},
		{
			name:     "0.5 ETH",
			wei:      big.NewInt(5e17),
			expected: "0.5000",
		},
		{
			name:     "5 ETH",
			wei:      new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18)),
			expected: "5.0000",
		},
		{
			name:     "0.1234 ETH",
			wei:      big.NewInt(1234e14),
			expected: "0.1234",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := weiToETHString(tc.wei)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetNetworkName(t *testing.T) {
	tests := []struct {
		chainID  uint64
		expected string
	}{
		{1, "Ethereum Mainnet"},
		{11155111, "Sepolia"},
		{17000, "Holesky"},
		{5, "Goerli (deprecated)"},
		{999999, "Chain 999999"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := GetNetworkName(tc.chainID)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCheckResult_Structure(t *testing.T) {
	result := CheckResult{
		Name:    CheckL1Reachable,
		Passed:  true,
		Message: "Connected successfully",
		Details: map[string]interface{}{
			"latency_ms": 50,
		},
	}

	assert.Equal(t, CheckL1Reachable, result.Name)
	assert.True(t, result.Passed)
	assert.Equal(t, "Connected successfully", result.Message)
	assert.Equal(t, 50, result.Details["latency_ms"])
}

func TestPreflightResponse_Structure(t *testing.T) {
	resp := PreflightResponse{
		OK: true,
		Checks: []CheckResult{
			{Name: CheckL1Reachable, Passed: true, Message: "OK"},
			{Name: CheckChainIDMatch, Passed: true, Message: "OK"},
			{Name: CheckDeployerBalance, Passed: true, Message: "OK"},
		},
		DeployerAddress:    "0x1234",
		RequiredFundingETH: "1.0000",
		CurrentBalanceETH:  "1.5000",
	}

	assert.True(t, resp.OK)
	assert.Len(t, resp.Checks, 3)
	assert.Equal(t, "0x1234", resp.DeployerAddress)
	assert.Equal(t, "1.0000", resp.RequiredFundingETH)
	assert.Equal(t, "1.5000", resp.CurrentBalanceETH)
}

func TestChecker_RunChecks_InvalidRequest(t *testing.T) {
	checker := NewChecker()
	ctx := context.Background()

	// Test with missing required field
	req := &PreflightRequest{
		L1RPC:     "https://eth-sepolia.example.com",
		L1ChainID: 11155111,
		// Missing DeployerAddress
	}

	resp, err := checker.RunChecks(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "deployer_address is required")
}

func TestChecker_RunChecks_InvalidRPC(t *testing.T) {
	checker := NewChecker().WithTimeout(2 * time.Second)
	ctx := context.Background()

	req := &PreflightRequest{
		L1RPC:           "http://localhost:99999", // Invalid port
		L1ChainID:       11155111,
		DeployerAddress: "0x1234567890123456789012345678901234567890",
	}

	resp, err := checker.RunChecks(ctx, req)
	require.NoError(t, err) // The method returns a response, not an error
	require.NotNil(t, resp)

	// Should have at least one check result
	assert.Len(t, resp.Checks, 1)
	assert.Equal(t, CheckL1Reachable, resp.Checks[0].Name)
	assert.False(t, resp.Checks[0].Passed)
	assert.False(t, resp.OK)
}

func TestCheckName_Constants(t *testing.T) {
	// Ensure check names are consistent strings
	assert.Equal(t, CheckName("l1_reachable"), CheckL1Reachable)
	assert.Equal(t, CheckName("chain_id_match"), CheckChainIDMatch)
	assert.Equal(t, CheckName("deployer_balance"), CheckDeployerBalance)
}

func TestDefaultTimeout(t *testing.T) {
	assert.Equal(t, 10*time.Second, DefaultTimeout)
}

