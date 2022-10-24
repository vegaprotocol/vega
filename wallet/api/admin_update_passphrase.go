package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
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

// Handle renames the wallet.
func (h *AdminUpdatePassphrase) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateUpdatePassphraseParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet, params.Passphrase)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	if err := h.walletStore.SaveWallet(ctx, w, params.NewPassphrase); err != nil {
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
