package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminRemoveWalletParams struct {
	Wallet string `json:"wallet"`
}

type AdminRemoveWallet struct {
	walletStore WalletStore
}

// Handle removes a wallet from the computer.
func (h *AdminRemoveWallet) Handle(ctx context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRemoveWalletParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	if err := h.walletStore.DeleteWallet(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not remove the wallet: %w", err))
	}

	return nil, nil
}

func validateRemoveWalletParams(rawParams jsonrpc.Params) (AdminRemoveWalletParams, error) {
	if rawParams == nil {
		return AdminRemoveWalletParams{}, ErrParamsRequired
	}

	params := AdminRemoveWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminRemoveWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminRemoveWalletParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewAdminRemoveWallet(
	walletStore WalletStore,
) *AdminRemoveWallet {
	return &AdminRemoveWallet{
		walletStore: walletStore,
	}
}
