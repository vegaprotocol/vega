package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminCreateWalletParams struct {
	Wallet     string `json:"wallet"`
	Passphrase string `json:"passphrase"`
}

type AdminCreateWalletResult struct {
	Wallet AdminCreatedWallet  `json:"wallet"`
	Key    AdminFirstPublicKey `json:"key"`
}

type AdminCreatedWallet struct {
	Name                 string `json:"name"`
	KeyDerivationVersion uint32 `json:"keyDerivationVersion"`
	RecoveryPhrase       string `json:"recoveryPhrase"`
}

type AdminFirstPublicKey struct {
	PublicKey string            `json:"publicKey"`
	Algorithm wallet.Algorithm  `json:"algorithm"`
	Meta      []wallet.Metadata `json:"metadata"`
}

type AdminCreateWallet struct {
	walletStore WalletStore
}

// Handle creates a wallet and generates its first key.
func (h *AdminCreateWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateCreateWalletParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if exist {
		return nil, InvalidParams(ErrWalletAlreadyExists)
	}

	w, recoveryPhrase, err := wallet.NewHDWallet(params.Wallet)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not create the HD wallet: %w", err))
	}

	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not generate the first key: %w", err))
	}

	if err := h.walletStore.CreateWallet(ctx, w, params.Passphrase); err != nil {
		return nil, InternalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return AdminCreateWalletResult{
		Wallet: AdminCreatedWallet{
			Name:                 w.Name(),
			KeyDerivationVersion: w.KeyDerivationVersion(),
			RecoveryPhrase:       recoveryPhrase,
		},
		Key: AdminFirstPublicKey{
			PublicKey: kp.PublicKey(),
			Algorithm: wallet.Algorithm{
				Name:    kp.AlgorithmName(),
				Version: kp.AlgorithmVersion(),
			},
			Meta: kp.Metadata(),
		},
	}, nil
}

func validateCreateWalletParams(rawParams jsonrpc.Params) (AdminCreateWalletParams, error) {
	if rawParams == nil {
		return AdminCreateWalletParams{}, ErrParamsRequired
	}

	params := AdminCreateWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminCreateWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminCreateWalletParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AdminCreateWalletParams{}, ErrPassphraseIsRequired
	}

	return params, nil
}

func NewAdminCreateWallet(
	walletStore WalletStore,
) *AdminCreateWallet {
	return &AdminCreateWallet{
		walletStore: walletStore,
	}
}
