package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminUpdatePassphraseParams struct {
	Wallet        string `json:"wallet"`
	Passphrase    string `json:"passphrase"`
	NewPassphrase string `json:"newPassphrase"`
}

type AdminUpdatePassphrase struct {
	walletStore WalletStore
}

func (h *AdminUpdatePassphrase) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateUpdatePassphraseParams(rawParams)
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

	if err := h.walletStore.UpdatePassphrase(ctx, params.Wallet, params.NewPassphrase); err != nil {
		return nil, internalError(fmt.Errorf("could not save the wallet with the new passphrase: %w", err))
	}

	return nil, nil
}

func validateUpdatePassphraseParams(rawParams jsonrpc.Params) (AdminUpdatePassphraseParams, error) {
	if rawParams == nil {
		return AdminUpdatePassphraseParams{}, ErrParamsRequired
	}

	params := AdminUpdatePassphraseParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminUpdatePassphraseParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminUpdatePassphraseParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AdminUpdatePassphraseParams{}, ErrPassphraseIsRequired
	}

	if params.NewPassphrase == "" {
		return AdminUpdatePassphraseParams{}, ErrNewPassphraseIsRequired
	}

	return params, nil
}

func NewAdminUpdatePassphrase(
	walletStore WalletStore,
) *AdminUpdatePassphrase {
	return &AdminUpdatePassphrase{
		walletStore: walletStore,
	}
}
