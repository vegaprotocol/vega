package vega

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vegawallet/wallet"
	storev1 "code.vegaprotocol.io/vegawallet/wallet/store/v1"
	"code.vegaprotocol.io/vegawallet/wallets"
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

	_, err = handler.GenerateKeyPair(walletName, passphrase, []wallet.Meta{})
	if err != nil {
		return nil, nil, err
	}

	w, err := newWallet(store, walletName, passphrase)
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

	return newWallet(store, walletName, passphrase)
}

func (l *WalletLoader) Import(sourceFilePath string, passphrase string) (*Wallet, map[string]string, error) {
	sourcePath, sourceWalletName := filepath.Split(sourceFilePath)

	sourceStore, err := storev1.InitialiseStore(sourcePath)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialise source wallet store: %w", err)
	}

	w, err := sourceStore.GetWallet(sourceWalletName, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get source wallet %s: %w", sourceWalletName, err)
	}

	destStore, err := storev1.InitialiseStore(l.walletHome)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialise destination wallet store: %w", err)
	}

	destWalletName := fmt.Sprintf("vega.%v", time.Now().UnixNano())
	w.SetName(destWalletName)
	err = destStore.SaveWallet(w, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't save the wallet %s: %w", destWalletName, err)
	}

	destWallet, err := newWallet(destStore, destWalletName, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't create wallet: %w", err)
	}

	data := map[string]string{
		"walletFilePath": destStore.GetWalletPath(destWalletName),
	}

	return destWallet, data, nil
}

func newWallet(store *storev1.Store, walletName, passphrase string) (*Wallet, error) {
	w, err := store.GetWallet(walletName, passphrase)
	if err != nil {
		return nil, fmt.Errorf("could not get wallet `%s`: %w", walletName, err)
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
		walletName: walletName,
		keyPair:    keyPair,
		pubKey:     pubKey,
		walletID:   walletID,
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
