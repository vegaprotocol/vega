package faucet

import (
	"errors"
	"path/filepath"

	"code.vegaprotocol.io/go-wallet/wallet"
	storev1 "code.vegaprotocol.io/go-wallet/wallet/store/v1"
	"code.vegaprotocol.io/go-wallet/wallets"
)

var (
	// ErrFaucetHasNoKeyInItsWallet is returned when trying to get the wallet
	// key of the faucet whereas no key has been generated or added to the
	// faucet's wallet.
	ErrFaucetHasNoKeyInItsWallet = errors.New("faucet has no key in its wallet")
)

type faucetWallet struct {
	handler *wallets.Handler
	// publicKey is the one used to retrieve the private key to sign messages.
	publicKey  string
	walletName string
}

func loadWallet(walletFilePath, passphrase string) (*faucetWallet, error) {
	walletDir, walletName := filepath.Split(walletFilePath)

	store, err := storev1.InitialiseStore(walletDir)
	if err != nil {
		return nil, err
	}

	handler := wallets.NewHandler(store)

	err = handler.LoginWallet(walletName, passphrase)
	if err != nil {
		return nil, err
	}

	keyPairs, err := handler.ListKeyPairs(walletName)
	if err != nil {
		return nil, err
	}

	if len(keyPairs) == 0 {
		return nil, ErrFaucetHasNoKeyInItsWallet
	}

	return &faucetWallet{
		handler:    handler,
		publicKey:  keyPairs[0].PublicKey(),
		walletName: walletName,
	}, nil
}

func (w *faucetWallet) Sign(message []byte) ([]byte, string, error) {
	sig, err := w.handler.SignAny(w.walletName, message, w.publicKey)
	if err != nil {
		return nil, "", err
	}
	return sig, w.publicKey, nil
}

func initialiseWallet(walletFilePath, passphrase string) (string, error) {
	walletDir, walletName := filepath.Split(walletFilePath)

	store, err := storev1.InitialiseStore(walletDir)
	if err != nil {
		return "", err
	}

	handler := wallets.NewHandler(store)

	// we ignore the mnemonic as this wallet is one-shot.
	_, err = handler.CreateWallet(walletName, passphrase)
	if err != nil {
		return "", err
	}

	keyPair, err := handler.GenerateKeyPair(walletName, passphrase, []wallet.Meta{})
	if err != nil {
		return "", err
	}

	return keyPair.PublicKey(), nil
}
