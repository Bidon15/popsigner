// Package nitro provides Nitro chain deployment infrastructure.
package nitro

import (
	"context"
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

// ensureWETHBalance wraps ETH to WETH if the signer doesn't have enough WETH.
// This automates the WETH wrapping so users don't have to do it manually.
func (d *RollupDeployer) ensureWETHBalance(
	ctx context.Context,
	client *ethclient.Client,
	wethAddress common.Address,
	requiredAmount *big.Int,
) error {
	signerAddr := d.signer.Address()

	// Check current WETH balance
	// WETH.balanceOf(address) -> uint256
	wethABI, err := abi.JSON(strings.NewReader(`[{
		"inputs": [{"name": "account", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "", "type": "uint256"}],
		"stateMutability": "view",
		"type": "function"
	}, {
		"inputs": [],
		"name": "deposit",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	}]`))
	if err != nil {
		return fmt.Errorf("parse WETH ABI: %w", err)
	}

	balanceData, err := wethABI.Pack("balanceOf", signerAddr)
	if err != nil {
		return fmt.Errorf("pack balanceOf: %w", err)
	}

	result, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &wethAddress,
		Data: balanceData,
	}, nil)
	if err != nil {
		return fmt.Errorf("call balanceOf: %w", err)
	}

	var currentBalance *big.Int
	if err := wethABI.UnpackIntoInterface(&currentBalance, "balanceOf", result); err != nil {
		return fmt.Errorf("unpack balanceOf: %w", err)
	}

	d.logger.Info("checked WETH balance",
		slog.String("address", signerAddr.Hex()),
		slog.String("current_weth", currentBalance.String()),
		slog.String("required_weth", requiredAmount.String()),
	)

	// If we have enough WETH, we're done
	if currentBalance.Cmp(requiredAmount) >= 0 {
		d.logger.Info("sufficient WETH balance, no wrapping needed")
		return nil
	}

	// Calculate how much more we need (with some buffer)
	needed := new(big.Int).Sub(requiredAmount, currentBalance)
	// Add 50% buffer to avoid running out
	wrapAmount := new(big.Int).Mul(needed, big.NewInt(150))
	wrapAmount = wrapAmount.Div(wrapAmount, big.NewInt(100))

	// Check ETH balance
	ethBalance, err := client.BalanceAt(ctx, signerAddr, nil)
	if err != nil {
		return fmt.Errorf("get ETH balance: %w", err)
	}

	// Need at least wrapAmount + gas costs
	gasBuffer := big.NewInt(100000000000000) // 0.0001 ETH for gas
	minRequired := new(big.Int).Add(wrapAmount, gasBuffer)

	if ethBalance.Cmp(minRequired) < 0 {
		return fmt.Errorf("insufficient ETH to wrap: have %s, need %s",
			ethBalance.String(), minRequired.String())
	}

	d.logger.Info("wrapping ETH to WETH for BOLD staking",
		slog.String("amount", wrapAmount.String()),
		slog.String("weth_contract", wethAddress.Hex()),
	)

	// Create deposit() transaction
	depositData, err := wethABI.Pack("deposit")
	if err != nil {
		return fmt.Errorf("pack deposit: %w", err)
	}

	nonce, err := client.PendingNonceAt(ctx, signerAddr)
	if err != nil {
		return fmt.Errorf("get nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("get gas price: %w", err)
	}

	tx := types.NewTransaction(
		nonce,
		wethAddress,
		wrapAmount, // Send ETH with the transaction
		100000,     // Gas limit for deposit is low
		gasPrice,
		depositData,
	)

	signedTx, err := d.signer.SignTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("sign deposit transaction: %w", err)
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("send deposit transaction: %w", err)
	}

	d.logger.Info("WETH deposit transaction submitted",
		slog.String("tx_hash", signedTx.Hash().Hex()),
		slog.String("amount", wrapAmount.String()),
	)

	// Wait for confirmation
	receipt, err := bind.WaitMined(ctx, client, signedTx)
	if err != nil {
		return fmt.Errorf("wait for deposit receipt: %w", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("WETH deposit transaction reverted")
	}

	d.logger.Info("ETH wrapped to WETH successfully",
		slog.String("tx_hash", signedTx.Hash().Hex()),
		slog.String("amount_wrapped", wrapAmount.String()),
	)

	return nil
}
