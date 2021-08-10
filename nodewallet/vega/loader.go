package vega

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/go-wallet/wallet"
	storev1 "code.vegaprotocol.io/go-wallet/wallet/store/v1"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/fsutil"
)

type WalletLoader struct {
	walletRootPath string
}

func NewWalletLoader(walletRootPath string) *WalletLoader {
	return &WalletLoader{
		walletRootPath: walletRootPath,
	}
}

func (l *WalletLoader) Initialise() error {
	return fsutil.EnsureDir(l.walletRootPath)
}

func (l *WalletLoader) Generate(passphrase string) (*Wallet, error) {
	store, err := storev1.NewStore(l.walletRootPath)
	if err != nil {
		return nil, err
	}

	err = store.Initialise()
	if err != nil {
		return nil, err
	}

	handler := wallet.NewHandler(store)

	// we ignore the mnemonic as this wallet is one-shot.
	walletName := fmt.Sprintf("vega.%v", time.Now().UnixNano())
	_, err = handler.CreateWallet(walletName, passphrase)
	if err != nil {
		return nil, err
	}

	_, err = handler.GenerateKeyPair(walletName, passphrase, []wallet.Meta{})
	if err != nil {
		return nil, err
	}

	return newWallet(store, walletName, passphrase)
}

func (l *WalletLoader) Load(walletName, passphrase string) (*Wallet, error) {
	store, err := storev1.NewStore(l.walletRootPath)
	if err != nil {
		return nil, err
	}

	err = store.Initialise()
	if err != nil {
		return nil, err
	}

	return newWallet(store, walletName, passphrase)
}

func (l *WalletLoader) Import(sourceFilePath, passphrase string) (*Wallet, error) {
	sourcePath, sourceWalletName := filepath.Split(sourceFilePath)

	sourceStore, err := storev1.NewStore(sourcePath)
	if err != nil {
		return nil, err
	}

	err = sourceStore.Initialise()
	if err != nil {
		return nil, err
	}

	w, err := sourceStore.GetWallet(sourceWalletName, passphrase)
	if err != nil {
		return nil, err
	}

	destStore, err := storev1.NewStore(l.walletRootPath)
	if err != nil {
		return nil, err
	}

	err = destStore.Initialise()
	if err != nil {
		return nil, err
	}

	destWalletName := fmt.Sprintf("vega.%v", time.Now().UnixNano())
	w.SetName(destWalletName)
	err = destStore.SaveWallet(w, passphrase)
	if err != nil {
		return nil, err
	}

	return newWallet(destStore, destWalletName, passphrase)
}

func newWallet(store *storev1.Store, walletName, passphrase string) (*Wallet, error) {
	handler := wallet.NewHandler(store)

	err := handler.LoginWallet(walletName, passphrase)
	if err != nil {
		return nil, err
	}

	keyPairs, err := handler.ListKeyPairs(walletName)
	if err != nil {
		return nil, err
	}

	keyPairCount := len(keyPairs)
	if keyPairCount == 0 {
		return nil, fmt.Errorf("vega wallet for node requires to have 1 key pair, none found")
	} else if keyPairCount != 1 {
		return nil, fmt.Errorf("vega wallet for node requires to have max 1 key pair, found %v", keyPairCount)
	}

	keyPair := keyPairs[0]

	pubKey, err := getPubKey(keyPair)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		handler:    handler,
		walletName: walletName,
		keyPair:    keyPair,
		pubKey:     pubKey,
	}, nil
}

func getPubKey(keyPair wallet.KeyPair) (crypto.PublicKeyOrAddress, error) {
	decodedPubKey, err := hex.DecodeString(keyPair.PublicKey())
	if err != nil {
		return crypto.PublicKeyOrAddress{}, err
	}

	return crypto.NewPublicKeyOrAddress(keyPair.PublicKey(), decodedPubKey), nil
}
