package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type AdminImportWalletParams struct {
	Wallet               string `json:"wallet"`
	RecoveryPhrase       string `json:"recoveryPhrase"`
	KeyDerivationVersion uint32 `json:"keyDerivationVersion"`
	Passphrase           string `json:"passphrase"`
}

type AdminImportWalletResult struct {
	Wallet AdminImportedWallet `json:"wallet"`
	Key    AdminFirstPublicKey `json:"key"`
}

type AdminImportedWallet struct {
	Name                 string `json:"name"`
	KeyDerivationVersion uint32 `json:"keyDerivationVersion"`
}

type AdminImportWallet struct {
	walletStore WalletStore
}

func (h *AdminImportWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateImportWalletParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, InternalError(fmt.Errorf("could not verify the wallet exists: %w", err))
	} else if exist {
		return nil, InvalidParams(ErrWalletAlreadyExists)
	}

	w, err := wallet.ImportHDWallet(params.Wallet, params.RecoveryPhrase, params.KeyDerivationVersion)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not import the wallet: %w", err))
	}

	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		return nil, InternalError(fmt.Errorf("could not generate first key: %w", err))
	}

	if err := h.walletStore.CreateWallet(ctx, w, params.Passphrase); err != nil {
		return nil, InternalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return AdminImportWalletResult{
		Wallet: AdminImportedWallet{
			Name:                 w.Name(),
			KeyDerivationVersion: w.KeyDerivationVersion(),
		},
		Key: AdminFirstPublicKey{
			PublicKey: kp.PublicKey(),
			Algorithm: wallet.Algorithm{
				Name:    kp.AlgorithmName(),
				Version: kp.AlgorithmVersion(),
			},
			Metadata: kp.Metadata(),
		},
	}, nil
}

func validateImportWalletParams(rawParams jsonrpc.Params) (AdminImportWalletParams, error) {
	if rawParams == nil {
		return AdminImportWalletParams{}, ErrParamsRequired
	}

	params := AdminImportWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminImportWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return AdminImportWalletParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return AdminImportWalletParams{}, ErrPassphraseIsRequired
	}

	if params.RecoveryPhrase == "" {
		return AdminImportWalletParams{}, ErrRecoveryPhraseIsRequired
	}

	if params.KeyDerivationVersion == 0 {
		return AdminImportWalletParams{}, ErrWalletKeyDerivationVersionIsRequired
	}

	return params, nil
}

func NewAdminImportWallet(
	walletStore WalletStore,
) *AdminImportWallet {
	return &AdminImportWallet{
		walletStore: walletStore,
	}
}
