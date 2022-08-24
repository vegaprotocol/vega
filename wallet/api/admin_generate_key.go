package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type GenerateKeyParams struct {
	Wallet     string        `json:"wallet"`
	Metadata   []wallet.Meta `json:"metadata"`
	Passphrase string        `json:"passphrase"`
}

type GenerateKeyResult struct {
	PublicKey string           `json:"publicKey"`
	Algorithm wallet.Algorithm `json:"algorithm"`
	Metadata  []wallet.Meta    `json:"metadata"`
}

type GenerateKey struct {
	walletStore WalletStore
}

// Handle generates a key of the specified wallet.
func (h *GenerateKey) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateGenerateKeyParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("couldn't verify wallet existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrWalletDoesNotExist)
	}

	w, err := h.walletStore.GetWallet(ctx, params.Wallet, params.Passphrase)
	if err != nil {
		return nil, internalError(fmt.Errorf("couldn't retrieve wallet: %w", err))
	}

	kp, err := w.GenerateKeyPair(params.Metadata)
	if err != nil {
		return nil, internalError(fmt.Errorf("couldn't generate a new key-pair: %w", err))
	}

	if err := h.walletStore.SaveWallet(ctx, w, params.Passphrase); err != nil {
		return nil, internalError(fmt.Errorf("couldn't save wallet: %w", err))
	}

	return GenerateKeyResult{
		PublicKey: kp.PublicKey(),
		Algorithm: wallet.Algorithm{
			Name:    kp.AlgorithmName(),
			Version: kp.AlgorithmVersion(),
		},
		Metadata: kp.Meta(),
	}, nil
}

func validateGenerateKeyParams(rawParams jsonrpc.Params) (GenerateKeyParams, error) {
	if rawParams == nil {
		return GenerateKeyParams{}, ErrParamsRequired
	}

	params := GenerateKeyParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return GenerateKeyParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return GenerateKeyParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return GenerateKeyParams{}, ErrPassphraseIsRequired
	}

	return params, nil
}

func NewGenerateKey(
	walletStore WalletStore,
) *GenerateKey {
	return &GenerateKey{
		walletStore: walletStore,
	}
}
