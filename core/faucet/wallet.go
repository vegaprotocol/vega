// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package faucet

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/wallet"
	storev1 "code.vegaprotocol.io/vega/wallet/wallet/store/v1"
	"code.vegaprotocol.io/vega/wallet/wallets"
)

// ErrFaucetHasNoKeyInItsWallet is returned when trying to get the wallet
// key of the faucet whereas no key has been generated or added to the
// faucet's wallet.
var ErrFaucetHasNoKeyInItsWallet = errors.New("faucet has no key in its wallet")

type faucetWallet struct {
	handler *wallets.Handler
	// publicKey is the one used to retrieve the private key to sign messages.
	publicKey  string
	walletName string
}

func (w *faucetWallet) Sign(message []byte) ([]byte, string, error) {
	sig, err := w.handler.SignAny(w.walletName, message, w.publicKey)
	if err != nil {
		return nil, "", err
	}

	return sig, w.publicKey, nil
}

type WalletGenerationResult struct {
	Mnemonic  string
	FilePath  string
	Name      string
	PublicKey string
}

type WalletLoader struct {
	store   *storev1.FileStore
	handler *wallets.Handler
}

func InitialiseWalletLoader(vegaPaths paths.Paths) (*WalletLoader, error) {
	walletsHome, err := vegaPaths.CreateDataDirFor(paths.FaucetWalletsDataHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get directory for %s: %w", paths.FaucetWalletsDataHome, err)
	}

	store, err := storev1.InitialiseStore(walletsHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise store at %s: %w", walletsHome, err)
	}

	return &WalletLoader{
		store:   store,
		handler: wallets.NewHandler(store),
	}, nil
}

func (l *WalletLoader) GenerateWallet(passphrase string) (*WalletGenerationResult, error) {
	walletName := fmt.Sprintf("vega.%v", time.Now().UnixNano())
	mnemonic, err := l.handler.CreateWallet(walletName, passphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't create wallet %s: %w", walletName, err)
	}

	keyPair, err := l.handler.GenerateKeyPair(walletName, passphrase, []wallet.Metadata{})
	if err != nil {
		return nil, fmt.Errorf("couldn't generate key pair for wallet %s: %w", walletName, err)
	}

	return &WalletGenerationResult{
		Mnemonic:  mnemonic,
		FilePath:  l.store.GetWalletPath(walletName),
		Name:      walletName,
		PublicKey: keyPair.PublicKey(),
	}, nil
}

func (l *WalletLoader) load(walletName, passphrase string) (*faucetWallet, error) {
	if err := l.handler.LoginWallet(walletName, passphrase); err != nil {
		return nil, fmt.Errorf("couldn't login to wallet %s: %w", walletName, err)
	}

	keyPairs, err := l.handler.ListKeyPairs(walletName)
	if err != nil {
		return nil, err
	}

	if len(keyPairs) == 0 {
		return nil, ErrFaucetHasNoKeyInItsWallet
	}

	return &faucetWallet{
		handler:    l.handler,
		publicKey:  keyPairs[0].PublicKey(),
		walletName: walletName,
	}, nil
}
