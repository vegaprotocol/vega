package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AnnotateKeyParams struct {
	Wallet     string            `json:"wallet"`
	PublicKey  string            `json:"publicKey"`
	Metadata   []wallet.Metadata `json:"metadata"`
	Passphrase string            `json:"passphrase"`
}

type AnnotateKeyResult struct {
	Metadata []wallet.Metadata `json:"metadata"`
}

type AnnotateKey struct {
	walletStore WalletStore
}

// Handle creates a wallet and generates its first key.
func (h *AnnotateKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAnnotateKeyParams(rawParams)
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

	if !w.HasPublicKey(params.PublicKey) {
		return nil, invalidParams(ErrPublicKeyDoesNotExist)
	}

	updatedMeta, err := w.AnnotateKey(params.PublicKey, params.Metadata)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not annotate the key: %w", err))
	}

	if err := h.walletStore.SaveWallet(ctx, w, params.Passphrase); err != nil {
		return nil, internalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return AnnotateKeyResult{
		Metadata: updatedMeta,
	}, nil
}

func validateAnnotateKeyParams(rawParams jsonrpc.Params) (AnnotateKeyParams, error) {
	if rawParams == nil {
		return AnnotateKeyParams{}, ErrParamsRequired
	}

	params := AnnotateKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AnnotateKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AnnotateKeyParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AnnotateKeyParams{}, ErrPassphraseIsRequired
	}

	if params.PublicKey == "" {
		return AnnotateKeyParams{}, ErrPublicKeyIsRequired
	}

	return params, nil
}

func NewAnnotateKey(
	walletStore WalletStore,
) *AnnotateKey {
	return &AnnotateKey{
		walletStore: walletStore,
	}
}
