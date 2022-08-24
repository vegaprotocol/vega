package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type RemoveWalletParams struct {
	Wallet string `json:"wallet"`
}

type RemoveWallet struct {
	walletStore WalletStore
}

// Handle removes a wallet from the computer.
func (h *RemoveWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRemoveWalletParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("couldn't verify wallet existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	if err := h.walletStore.DeleteWallet(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("couldn't remove wallet: %w", err))
	}

	return nil, nil
}

func validateRemoveWalletParams(rawParams jsonrpc.Params) (RemoveWalletParams, error) {
	if rawParams == nil {
		return RemoveWalletParams{}, ErrParamsRequired
	}

	params := RemoveWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return RemoveWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return RemoveWalletParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewRemoveWallet(
	walletStore WalletStore,
) *RemoveWallet {
	return &RemoveWallet{
		walletStore: walletStore,
	}
}
