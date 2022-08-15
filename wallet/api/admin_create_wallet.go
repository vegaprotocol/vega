package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type CreateWalletParams struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

type CreateWalletResult struct {
	Wallet CreatedWallet  `json:"wallet"`
	Key    FirstPublicKey `json:"key"`
}

type CreatedWallet struct {
	Name           string `json:"name"`
	Version        uint32 `json:"version"`
	RecoveryPhrase string `json:"recoveryPhrase"`
}

type FirstPublicKey struct {
	PublicKey string           `json:"publicKey"`
	Algorithm wallet.Algorithm `json:"algorithm"`
	Meta      []wallet.Meta    `json:"meta"`
}

type CreateWallet struct {
	walletStore WalletStore
}

// Handle creates a wallet and generates its first key.
func (h *CreateWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateCreateWalletParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("couldn't verify wallet existence: %w", err))
	} else if exist {
		return nil, invalidParams(ErrWalletAlreadyExists)
	}

	w, recoveryPhrase, err := wallet.NewHDWallet(params.Wallet)
	if err != nil {
		return nil, internalError(fmt.Errorf("couldn't create HD wallet: %w", err))
	}

	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		return nil, internalError(fmt.Errorf("couldn't generate first key-pair: %w", err))
	}

	if err := h.walletStore.SaveWallet(ctx, w, params.Passphrase); err != nil {
		return nil, internalError(fmt.Errorf("couldn't save wallet: %w", err))
	}

	return CreateWalletResult{
		Wallet: CreatedWallet{
			Name:           params.Wallet,
			Version:        w.Version(),
			RecoveryPhrase: recoveryPhrase,
		},
		Key: FirstPublicKey{
			PublicKey: kp.PublicKey(),
			Algorithm: wallet.Algorithm{
				Name:    kp.AlgorithmName(),
				Version: kp.AlgorithmVersion(),
			},
			Meta: kp.Meta(),
		},
	}, nil
}

func validateCreateWalletParams(rawParams jsonrpc.Params) (CreateWalletParams, error) {
	if rawParams == nil {
		return CreateWalletParams{}, ErrParamsRequired
	}

	params := CreateWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return CreateWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return CreateWalletParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return CreateWalletParams{}, ErrPassphraseIsRequired
	}

	return params, nil
}

func NewCreateWallet(
	walletStore WalletStore,
) *CreateWallet {
	return &CreateWallet{
		walletStore: walletStore,
	}
}
