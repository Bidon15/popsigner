// Package opstack provides OP Stack chain deployment infrastructure.
package opstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
)

// POPSignerAdapter adapts our existing POPSigner to the op-deployer opcrypto.SignerFn interface.
// This allows using POPSigner for transaction signing in op-deployer pipeline stages.
type POPSignerAdapter struct {
	endpoint   string
	apiKey     string
	chainID    *big.Int
	httpClient *http.Client
}

// NewPOPSignerAdapter creates a new adapter for op-deployer integration.
func NewPOPSignerAdapter(endpoint, apiKey string, chainID *big.Int) *POPSignerAdapter {
	return &POPSignerAdapter{
		endpoint: endpoint,
		apiKey:   apiKey,
		chainID:  chainID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SignerFn returns an opcrypto.SignerFn compatible with op-deployer.
// This is the main entry point for transaction signing in the op-deployer pipeline.
func (a *POPSignerAdapter) SignerFn() opcrypto.SignerFn {
	return func(ctx context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
		return a.signTransaction(ctx, addr, tx)
	}
}

// signTransaction signs a transaction via the POPSigner JSON-RPC API.
func (a *POPSignerAdapter) signTransaction(ctx context.Context, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	// Build transaction args for JSON-RPC eth_signTransaction
	txArgs := a.buildTransactionArgs(addr, tx)

	// Create JSON-RPC request
	rpcReq := rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_signTransaction",
		Params:  []interface{}{txArgs},
		ID:      1,
	}

	// Marshal request body
	reqBody, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request to POPSigner RPC endpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", a.apiKey)

	// Execute request
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("signing request failed: %d %s", resp.StatusCode, string(body))
	}

	// Parse JSON-RPC response
	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Check for JSON-RPC error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// Extract result (signed transaction hex)
	var signedTxHex string
	if err := json.Unmarshal(rpcResp.Result, &signedTxHex); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	// Decode the signed transaction
	txBytes, err := hexutil.Decode(signedTxHex)
	if err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}

	var signedTx types.Transaction
	if err := signedTx.UnmarshalBinary(txBytes); err != nil {
		return nil, fmt.Errorf("unmarshal transaction: %w", err)
	}

	return &signedTx, nil
}

// buildTransactionArgs converts a go-ethereum transaction to JSON-RPC args.
func (a *POPSignerAdapter) buildTransactionArgs(addr common.Address, tx *types.Transaction) txArgs {
	args := txArgs{
		From:    addr.Hex(),
		Gas:     hexutil.EncodeUint64(tx.Gas()),
		Value:   hexutil.EncodeBig(tx.Value()),
		Nonce:   hexutil.EncodeUint64(tx.Nonce()),
		ChainID: hexutil.EncodeBig(a.chainID),
	}

	// Set recipient (nil for contract creation)
	if tx.To() != nil {
		to := tx.To().Hex()
		args.To = &to
	}

	// Set data/input
	if len(tx.Data()) > 0 {
		args.Data = hexutil.Encode(tx.Data())
	}

	// Handle gas pricing based on transaction type
	switch tx.Type() {
	case types.DynamicFeeTxType:
		// EIP-1559 transaction
		maxFee := hexutil.EncodeBig(tx.GasFeeCap())
		maxTip := hexutil.EncodeBig(tx.GasTipCap())
		args.MaxFeePerGas = &maxFee
		args.MaxPriorityFeePerGas = &maxTip
	default:
		// Legacy transaction
		gasPrice := hexutil.EncodeBig(tx.GasPrice())
		args.GasPrice = &gasPrice
	}

	return args
}

// rpcRequest represents a JSON-RPC 2.0 request.
type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// rpcResponse represents a JSON-RPC 2.0 response.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// rpcError represents a JSON-RPC 2.0 error.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// txArgs represents Ethereum transaction arguments for JSON-RPC.
type txArgs struct {
	From                 string  `json:"from"`
	To                   *string `json:"to,omitempty"`
	Gas                  string  `json:"gas"`
	GasPrice             *string `json:"gasPrice,omitempty"`
	MaxFeePerGas         *string `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas *string `json:"maxPriorityFeePerGas,omitempty"`
	Value                string  `json:"value"`
	Nonce                string  `json:"nonce"`
	Data                 string  `json:"data,omitempty"`
	ChainID              string  `json:"chainId"`
}

