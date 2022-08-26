package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type ImportWalletParams struct {
	Wallet         string `json:"wallet"`
	RecoveryPhrase string `json:"recoveryPhrase"`
	Version        uint32 `json:"version"`
	Passphrase     string `json:"passphrase"`
}

type ImportWalletResult struct {
	Wallet ImportedWallet `json:"wallet"`
	Key    FirstPublicKey `json:"key"`
}

type ImportedWallet struct {
	Name     string `json:"name"`
	Version  uint32 `json:"version"`
	FilePath string `json:"filePath"`
}

type ImportWallet struct {
	walletStore WalletStore
}

// Handle creates a wallet and generates its first key.
func (h *ImportWallet) Handle(ctx context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateImportWalletParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.walletStore.WalletExists(ctx, params.Wallet); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the wallet existence: %w", err))
	} else if exist {
		return nil, invalidParams(ErrWalletAlreadyExists)
	}

	w, err := wallet.ImportHDWallet(params.Wallet, params.RecoveryPhrase, params.Version)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not import the wallet: %w", err))
	}

	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not generate first key: %w", err))
	}

	if err := h.walletStore.SaveWallet(ctx, w, params.Passphrase); err != nil {
		return nil, internalError(fmt.Errorf("could not save the wallet: %w", err))
	}

	return ImportWalletResult{
		Wallet: ImportedWallet{
			Name:     w.Name(),
			Version:  w.Version(),
			FilePath: h.walletStore.GetWalletPath(w.Name()),
		},
		Key: FirstPublicKey{
			PublicKey: kp.PublicKey(),
			Algorithm: wallet.Algorithm{
				Name:    kp.AlgorithmName(),
				Version: kp.AlgorithmVersion(),
			},
			Meta: kp.Metadata(),
		},
	}, nil
}

func validateImportWalletParams(rawParams jsonrpc.Params) (ImportWalletParams, error) {
	if rawParams == nil {
		return ImportWalletParams{}, ErrParamsRequired
	}

	params := ImportWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ImportWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Wallet == "" {
		return ImportWalletParams{}, ErrWalletIsRequired
	}

	if params.Passphrase == "" {
		return ImportWalletParams{}, ErrPassphraseIsRequired
	}

	if params.RecoveryPhrase == "" {
		return ImportWalletParams{}, ErrRecoveryPhraseIsRequired
	}

	if params.Version == 0 {
		return ImportWalletParams{}, ErrWalletVersionIsRequired
	}

	return params, nil
}

func NewImportWallet(
	walletStore WalletStore,
) *ImportWallet {
	return &ImportWallet{
		walletStore: walletStore,
	}
}
