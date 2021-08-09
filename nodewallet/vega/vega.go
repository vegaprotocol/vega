package vega

import (
	"encoding/hex"
	"fmt"
	"path/filepath"

	"code.vegaprotocol.io/go-wallet/wallet"
	storev1 "code.vegaprotocol.io/go-wallet/wallet/store/v1"
	"code.vegaprotocol.io/vega/crypto"
)

const (
	defaultVegaWalletOwner = "vega-node"
)

type Wallet struct {
	handler    *wallet.Handler
	walletName string
	keyPair    wallet.KeyPair
	pubKey     crypto.PublicKeyOrAddress
}

func DevInit(path, passphrase string) (string, error) {
	store, err := storev1.NewStore(path)
	if err != nil {
		return "", err
	}

	err = store.Initialise()
	if err != nil {
		return "", err
	}

	handler := wallet.NewHandler(store)

	// we ignore the mnemonic as this wallet is one-shot.
	_, err = handler.CreateWallet(defaultVegaWalletOwner, passphrase)
	if err != nil {
		return "", err
	}

	meta := []wallet.Meta{{Key: "env", Value: "dev"}}
	_, err = handler.GenerateKeyPair(defaultVegaWalletOwner, passphrase, meta)
	if err != nil {
		return "", err
	}

	return filepath.Join(path, defaultVegaWalletOwner), nil
}

func New(walletFilePath, passphrase string) (*Wallet, error) {
	path, walletName := filepath.Split(walletFilePath)

	store, err := storev1.NewStore(path)
	if err != nil {
		return nil, err
	}

	err = store.Initialise()
	if err != nil {
		return nil, err
	}

	handler := wallet.NewHandler(store)

	err = handler.LoginWallet(walletName, passphrase)
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

func (w *Wallet) Chain() string {
	return "vega"
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	return w.handler.SignAny(w.walletName, data, w.keyPair.PublicKey())
}

func (w *Wallet) Algo() string {
	return w.keyPair.AlgorithmName()
}

func (w *Wallet) Version() uint32 {
	return w.keyPair.AlgorithmVersion()
}

func (w *Wallet) PubKeyOrAddress() crypto.PublicKeyOrAddress {
	return w.pubKey
}
