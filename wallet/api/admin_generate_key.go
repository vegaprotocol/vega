package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminGenerateKeyParams struct {
	Wallet   string            `json:"wallet"`
	Metadata []wallet.Metadata `json:"metadata"`
}

type AdminGenerateKeyResult struct {
	PublicKey string            `json:"publicKey"`
	Algorithm wallet.Algorithm  `json:"algorithm"`
	Metadata  []wallet.Metadata `json:"metadata"`
}

type AdminGenerateKey struct {
	walletStore WalletStore
}

// Handle generates a key on the specified wallet.
func (h *AdminGenerateKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateGenerateKeyParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	alreadyUnlocked, err := h.walletStore.IsWalletAlreadyUnlocked(ctx, params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not verify whether the wallet is already unlock or not: %w", err))
	}
	if !alreadyUnlocked {
		return nil, requestNotPermittedError(ErrWalletIsLocked)
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the wallet: %w", err))
	}

	kp, err := w.GenerateKeyPair(params.Metadata)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not generate a new key: %w", err))
	}

	if err := h.walletStore.UpdateWallet(ctx, w); err != nil {
		return nil, internalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return AdminGenerateKeyResult{
		PublicKey: kp.PublicKey(),
		Algorithm: wallet.Algorithm{
			Name:    kp.AlgorithmName(),
			Version: kp.AlgorithmVersion(),
		},
		Metadata: kp.Metadata(),
	}, nil
}

func validateGenerateKeyParams(rawParams jsonrpc.Params) (AdminGenerateKeyParams, error) {
	if rawParams == nil {
		return AdminGenerateKeyParams{}, ErrParamsRequired
	}

	params := AdminGenerateKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminGenerateKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminGenerateKeyParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewAdminGenerateKey(
	walletStore WalletStore,
) *AdminGenerateKey {
	return &AdminGenerateKey{
		walletStore: walletStore,
	}
}
