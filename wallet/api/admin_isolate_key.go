package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminIsolateKeyParams struct {
	Wallet                   string `json:"wallet"`
	PublicKey                string `json:"publicKey"`
	Passphrase               string `json:"passphrase"`
	IsolatedWalletPassphrase string `json:"isolatedWalletPassphrase"`
}

type AdminIsolateKeyResult struct {
	Wallet string `json:"wallet"`
}

type AdminIsolateKey struct {
	walletStore WalletStore
}

// Handle isolates a key in a specific wallet.
func (h *AdminIsolateKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminIsolateKeyParams(rawParams)
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

	w, err := h.walletStore.GetWallet(ctx, params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	if !w.HasPublicKey(params.PublicKey) {
		return nil, invalidParams(ErrPublicKeyDoesNotExist)
	}

	isolatedWallet, err := w.IsolateWithKey(params.PublicKey)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not isolate the key: %w", err))
	}

	if err := h.walletStore.CreateWallet(ctx, isolatedWallet, params.IsolatedWalletPassphrase); err != nil {
		return nil, internalError(fmt.Errorf("could not save the wallet with isolated key: %w", err))
	}

	return AdminIsolateKeyResult{
		Wallet: isolatedWallet.Name(),
	}, nil
}

func validateAdminIsolateKeyParams(rawParams jsonrpc.Params) (AdminIsolateKeyParams, error) {
	if rawParams == nil {
		return AdminIsolateKeyParams{}, ErrParamsRequired
	}

	params := AdminIsolateKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminIsolateKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminIsolateKeyParams{}, ErrWalletIsRequired
	}

	if params.PublicKey == "" {
		return AdminIsolateKeyParams{}, ErrPublicKeyIsRequired
	}

	if params.Passphrase == "" {
		return AdminIsolateKeyParams{}, ErrPassphraseIsRequired
	}

	if params.IsolatedWalletPassphrase == "" {
		return AdminIsolateKeyParams{}, ErrIsolatedWalletPassphraseIsRequired
	}

	return params, nil
}

func NewAdminIsolateKey(
	walletStore WalletStore,
) *AdminIsolateKey {
	return &AdminIsolateKey{
		walletStore: walletStore,
	}
}
