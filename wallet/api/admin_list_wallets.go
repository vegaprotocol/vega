package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
)

type AdminListWalletsResult struct {
	Wallets []string `json:"wallets"`
}

type AdminListWallets struct {
	walletStore WalletStore
}

// Handle list all the wallets present on the computer.
func (h *AdminListWallets) Handle(ctx context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	wallets, err := h.walletStore.ListWallets(ctx)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not list the wallets: %w", err))
	}

	return AdminListWalletsResult{
		Wallets: wallets,
	}, nil
}

func NewAdminListWallets(
	walletStore WalletStore,
) *AdminListWallets {
	return &AdminListWallets{
		walletStore: walletStore,
	}
}
