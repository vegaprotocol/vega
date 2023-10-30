// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package faucet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/wallet"
	storev1 "code.vegaprotocol.io/vega/wallet/wallet/store/v1"
)

// ErrFaucetHasNoKeyInItsWallet is returned when trying to get the wallet
// key of the faucet whereas no key has been generated or added to the
// faucet's wallet.
var ErrFaucetHasNoKeyInItsWallet = errors.New("faucet has no key in its wallet")

type faucetWallet struct {
	// publicKey is the one used to retrieve the private key to sign messages.
	publicKey string
	wallet    wallet.Wallet
}

func (w *faucetWallet) Sign(message []byte) ([]byte, string, error) {
	sig, err := w.wallet.SignAny(w.publicKey, message)
	if err != nil {
		return nil, "", fmt.Errorf("could not sign the message: %w", err)
	}

	return sig, w.publicKey, nil
}

type WalletGenerationResult struct {
	Mnemonic  string
	FilePath  string
	Name      string
	PublicKey string
}

func GenerateWallet(vegaPaths paths.Paths, passphrase string) (*WalletGenerationResult, error) {
	ctx := context.Background()

	walletsHome, err := vegaPaths.CreateDataDirFor(paths.FaucetWalletsDataHome)
	if err != nil {
		return nil, fmt.Errorf("could not get directory for %s: %w", paths.FaucetWalletsDataHome, err)
	}

	store, err := storev1.InitialiseStore(walletsHome, false)
	if err != nil {
		return nil, fmt.Errorf("could not initialise faucet wallet store at %s: %w", walletsHome, err)
	}
	defer store.Close()

	walletName := fmt.Sprintf("vega.%v", time.Now().UnixNano())

	if exists, err := store.WalletExists(ctx, walletName); err != nil {
		return nil, fmt.Errorf("couldn't verify the wallet existence: %w", err)
	} else if exists {
		return nil, wallet.ErrWalletAlreadyExists
	}

	w, recoveryPhrase, err := wallet.NewHDWallet(walletName)
	if err != nil {
		return nil, fmt.Errorf("could not generate faucet wallet: %w", err)
	}

	keyPair, err := w.GenerateKeyPair([]wallet.Metadata{})
	if err != nil {
		return nil, fmt.Errorf("could not generate key pair for faucet wallet %s: %w", walletName, err)
	}

	if err := store.CreateWallet(ctx, w, passphrase); err != nil {
		return nil, fmt.Errorf("could not save the generated faucet wallet: %w", err)
	}

	return &WalletGenerationResult{
		Mnemonic:  recoveryPhrase,
		FilePath:  store.GetWalletPath(walletName),
		Name:      walletName,
		PublicKey: keyPair.PublicKey(),
	}, nil
}

func loadWallet(vegaPaths paths.Paths, walletName, passphrase string) (*faucetWallet, error) {
	ctx := context.Background()

	walletsHome, err := vegaPaths.CreateDataDirFor(paths.FaucetWalletsDataHome)
	if err != nil {
		return nil, fmt.Errorf("could not get directory for %q: %w", paths.FaucetWalletsDataHome, err)
	}

	store, err := storev1.InitialiseStore(walletsHome, false)
	if err != nil {
		return nil, fmt.Errorf("could not initialise faucet wallet store at %q: %w", walletsHome, err)
	}
	defer store.Close()

	if exists, err := store.WalletExists(ctx, walletName); err != nil {
		return nil, fmt.Errorf("could not verify the faucet wallet existence: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("the faucet wallet %q does not exist", walletName)
	}

	if err := store.UnlockWallet(ctx, walletName, passphrase); err != nil {
		if errors.Is(err, wallet.ErrWrongPassphrase) {
			return nil, err
		}
		return nil, fmt.Errorf("could not unlock the faucet wallet %q: %w", walletName, err)
	}

	w, err := store.GetWallet(ctx, walletName)
	if err != nil {
		return nil, fmt.Errorf("could not get the faucet wallet %q: %w", walletName, err)
	}

	keyPairs := w.ListKeyPairs()

	if len(keyPairs) == 0 {
		return nil, ErrFaucetHasNoKeyInItsWallet
	}

	return &faucetWallet{
		wallet:    w,
		publicKey: keyPairs[0].PublicKey(),
	}, nil
}
