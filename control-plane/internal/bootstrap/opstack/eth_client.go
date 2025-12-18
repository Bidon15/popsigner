package opstack

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
)

// EthClientFactory creates L1 clients using go-ethereum's ethclient.
type EthClientFactory struct{}

// ethClientWrapper wraps ethclient.Client to implement L1Client interface.
type ethClientWrapper struct {
	*ethclient.Client
}

// NewEthClientFactory creates a new EthClientFactory.
func NewEthClientFactory() *EthClientFactory {
	return &EthClientFactory{}
}

// Dial connects to an Ethereum RPC endpoint.
func (f *EthClientFactory) Dial(ctx context.Context, rpcURL string) (L1Client, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	return &ethClientWrapper{Client: client}, nil
}

