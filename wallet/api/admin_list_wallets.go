package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
)

type ListWalletsResult struct {
	Wallets []string `json:"wallets"`
}

type ListWallets struct {
	walletStore WalletStore
}

// Handle list all the wallets present on the computer.
func (h *ListWallets) Handle(ctx context.Context, _ jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	wallets, err := h.walletStore.ListWallets(ctx)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not list the wallets: %w", err))
	}

	return ListWalletsResult{
		Wallets: wallets,
	}, nil
}

func NewListWallets(
	walletStore WalletStore,
) *ListWallets {
	return &ListWallets{
		walletStore: walletStore,
	}
}
