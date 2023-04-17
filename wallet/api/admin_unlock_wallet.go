package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminUnlockWalletParams struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

type AdminUnlockWallet struct {
	walletStore WalletStore
}

func (h *AdminUnlockWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateUnlockWalletParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	if err := h.walletStore.UnlockWallet(ctx, params.Wallet, params.Passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return nil, invalidParams(err)
		}
		return nil, internalError(fmt.Errorf("could not unlock the wallet: %w", err))
	}
	return nil, nil
}

func validateUnlockWalletParams(rawParams jsonrpc.Params) (AdminUnlockWalletParams, error) {
	if rawParams == nil {
		return AdminUnlockWalletParams{}, ErrParamsRequired
	}

	params := AdminUnlockWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminUnlockWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminUnlockWalletParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AdminUnlockWalletParams{}, ErrPassphraseIsRequired
	}

	return params, nil
}

func NewAdminUnlockWallet(
	walletStore WalletStore,
) *AdminUnlockWallet {
	return &AdminUnlockWallet{
		walletStore: walletStore,
	}
}
