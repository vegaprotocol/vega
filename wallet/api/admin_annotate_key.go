package api

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminAnnotateKeyParams struct {
	Wallet     string            `json:"wallet"`
	PublicKey  string            `json:"publicKey"`
	Metadata   []wallet.Metadata `json:"metadata"`
	Passphrase string            `json:"passphrase"`
}

type AdminAnnotateKeyResult struct {
	Metadata []wallet.Metadata `json:"metadata"`
}

type AdminAnnotateKey struct {
	walletStore WalletStore
}

// Handle attaches metadata to the specified public key. It doesn't update in
// place. It overwrites. All existing metadata have to be specified to not
// lose them.
func (h *AdminAnnotateKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAnnotateKeyParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet existence: %w", err))
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

	updatedMeta, err := w.AnnotateKey(params.PublicKey, params.Metadata)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not annotate the key: %w", err))
	}

	if err := h.walletStore.UpdateWallet(ctx, w); err != nil {
		return nil, internalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return AdminAnnotateKeyResult{
		Metadata: updatedMeta,
	}, nil
}

func validateAnnotateKeyParams(rawParams jsonrpc.Params) (AdminAnnotateKeyParams, error) {
	if rawParams == nil {
		return AdminAnnotateKeyParams{}, ErrParamsRequired
	}

	params := AdminAnnotateKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminAnnotateKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminAnnotateKeyParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AdminAnnotateKeyParams{}, ErrPassphraseIsRequired
	}

	if params.PublicKey == "" {
		return AdminAnnotateKeyParams{}, ErrPublicKeyIsRequired
	}

	return params, nil
}

func NewAdminAnnotateKey(
	walletStore WalletStore,
) *AdminAnnotateKey {
	return &AdminAnnotateKey{
		walletStore: walletStore,
	}
}
