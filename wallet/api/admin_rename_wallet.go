package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminRenameWalletParams struct {
	Wallet  string `json:"wallet"`
	NewName string `json:"newName"`
}

type AdminRenameWallet struct {
	walletStore WalletStore
}

// Handle renames a wallet.
func (h *AdminRenameWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateRenameWalletParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.NewName); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if exist {
		return nil, invalidParams(ErrWalletAlreadyExists)
	}

	if err := h.walletStore.RenameWallet(ctx, params.Wallet, params.NewName); err != nil {
		return nil, internalError(fmt.Errorf("could not rename the wallet: %w", err))
	}

	return nil, nil
}

func validateRenameWalletParams(rawParams jsonrpc.Params) (AdminRenameWalletParams, error) {
	if rawParams == nil {
		return AdminRenameWalletParams{}, ErrParamsRequired
	}

	params := AdminRenameWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminRenameWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminRenameWalletParams{}, ErrWalletIsRequired
	}

	if params.NewName == "" {
		return AdminRenameWalletParams{}, ErrNewNameIsRequired
	}

	return params, nil
}

func NewAdminRenameWallet(
	walletStore WalletStore,
) *AdminRenameWallet {
	return &AdminRenameWallet{
		walletStore: walletStore,
	}
}
