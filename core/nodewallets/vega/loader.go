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

package vega

import (
	"context"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/wallet/wallet"
	storev1 "code.vegaprotocol.io/vega/wallet/wallet/store/v1"
	"code.vegaprotocol.io/vega/wallet/wallets"
)

type WalletLoader struct {
	walletHome string
}

func InitialiseWalletLoader(vegaPaths paths.Paths) (*WalletLoader, error) {
	walletHome, err := vegaPaths.CreateDataDirFor(paths.VegaNodeWalletsDataHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get the directory path for %s: %w", paths.VegaNodeWalletsDataHome, err)
	}

	return &WalletLoader{
		walletHome: walletHome,
	}, nil
}

func (l *WalletLoader) Generate(passphrase string) (*Wallet, map[string]string, error) {
	data := map[string]string{}
	store, err := storev1.InitialiseStore(l.walletHome)
	if err != nil {
		return nil, nil, err
	}

	handler := wallets.NewHandler(store)

	walletName := fmt.Sprintf("vega.%v", time.Now().UnixNano())
	mnemonic, err := handler.CreateWallet(walletName, passphrase)
	if err != nil {
		return nil, nil, err
	}
	data["mnemonic"] = mnemonic
	data["walletFilePath"] = store.GetWalletPath(walletName)

	_, err = handler.GenerateKeyPair(walletName, passphrase, []wallet.Metadata{})
	if err != nil {
		return nil, nil, err
	}

	w, err := newWallet(l, store, walletName, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't create wallet: %w", err)
	}
	return w, data, nil
}

func (l *WalletLoader) Load(walletName, passphrase string) (*Wallet, error) {
	store, err := storev1.InitialiseStore(l.walletHome)
	if err != nil {
		return nil, err
	}

	return newWallet(l, store, walletName, passphrase)
}

func (l *WalletLoader) Import(sourceFilePath string, passphrase string) (*Wallet, map[string]string, error) {
	ctx := context.Background()

	sourcePath, sourceWalletName := filepath.Split(sourceFilePath)

	sourceStore, err := storev1.InitialiseStore(sourcePath)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialise source wallet store: %w", err)
	}

	if err := sourceStore.UnlockWallet(ctx, sourceWalletName, passphrase); err != nil {
		return nil, nil, fmt.Errorf("couldn't unlock the source wallet: %w", err)
	}

	w, err := sourceStore.GetWallet(ctx, sourceWalletName)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get source wallet %s: %w", sourceWalletName, err)
	}

	destStore, err := storev1.InitialiseStore(l.walletHome)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialise destination wallet store: %w", err)
	}

	destWalletName := fmt.Sprintf("vega.%v", time.Now().UnixNano())
	w.SetName(destWalletName)
	err = destStore.CreateWallet(ctx, w, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't save the wallet %s: %w", destWalletName, err)
	}

	destWallet, err := newWallet(l, destStore, destWalletName, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't create wallet: %w", err)
	}

	data := map[string]string{
		"walletFilePath": destStore.GetWalletPath(destWalletName),
	}

	return destWallet, data, nil
}

func newWallet(loader loader, store *storev1.FileStore, walletName, passphrase string) (*Wallet, error) {
	ctx := context.Background()
	if err := store.UnlockWallet(ctx, walletName, passphrase); err != nil {
		return nil, fmt.Errorf("could not unlock the wallet %q: %w", walletName, err)
	}

	w, err := store.GetWallet(ctx, walletName)
	if err != nil {
		return nil, fmt.Errorf("could not get wallet %q: %w", walletName, err)
	}

	keyPairs := w.ListKeyPairs()

	if keyPairCount := len(keyPairs); keyPairCount == 0 {
		return nil, fmt.Errorf("vega wallet for node requires to have 1 key pair, none found")
	} else if keyPairCount != 1 {
		return nil, fmt.Errorf("vega wallet for node requires to have max 1 key pair, found %v", keyPairCount)
	}

	keyPair := keyPairs[0]

	pubKey, err := getPubKey(keyPair)
	if err != nil {
		return nil, fmt.Errorf("couldn't get public key: %w", err)
	}

	walletID, err := getID(w)
	if err != nil {
		return nil, fmt.Errorf("couldn't get wallet ID: %w", err)
	}

	return &Wallet{
		loader:   loader,
		name:     walletName,
		keyPair:  keyPair,
		pubKey:   pubKey,
		walletID: walletID,
	}, nil
}

func getPubKey(keyPair wallet.KeyPair) (crypto.PublicKey, error) {
	decodedPubKey, err := hex.DecodeString(keyPair.PublicKey())
	if err != nil {
		return crypto.PublicKey{}, fmt.Errorf("couldn't decode public key as hexadecimal: %w", err)
	}

	return crypto.NewPublicKey(keyPair.PublicKey(), decodedPubKey), nil
}

func getID(w wallet.Wallet) (crypto.PublicKey, error) {
	decodedID, err := hex.DecodeString(w.ID())
	if err != nil {
		return crypto.PublicKey{}, fmt.Errorf("couldn't decode wallet ID as hexadecimal: %w", err)
	}

	return crypto.NewPublicKey(w.ID(), decodedID), nil
}
