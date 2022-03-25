package vega

import (
	"fmt"

	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/nodewallets/registryloader"
	"code.vegaprotocol.io/vegawallet/wallet"
	storev1 "code.vegaprotocol.io/vegawallet/wallet/store/v1"
)

type Wallet struct {
	homeDir  string
	name     string
	keyPair  wallet.KeyPair
	pubKey   crypto.PublicKey
	walletID crypto.PublicKey
}

func (w *Wallet) Name() string {
	return w.name
}

func (w *Wallet) Chain() string {
	return "vega"
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	return w.keyPair.SignAny(data)
}

func (w *Wallet) Algo() string {
	return w.keyPair.AlgorithmName()
}

func (w *Wallet) Version() uint32 {
	return w.keyPair.AlgorithmVersion()
}

func (w *Wallet) PubKey() crypto.PublicKey {
	return w.pubKey
}

func (w *Wallet) Index() uint32 {
	return w.keyPair.Index()
}

func (w *Wallet) ID() crypto.PublicKey {
	return w.walletID
}

func (w *Wallet) Reload(rw registryloader.RegisteredVegaWallet) error {
	store, err := storev1.InitialiseStore(w.homeDir)
	if err != nil {
		return fmt.Errorf("failed to initialise store: %w", err)
	}

	nW, err := newWallet(store, w.homeDir, rw.Name, rw.Passphrase)
	if err != nil {
		return fmt.Errorf("failed to create new wallet: %w", err)
	}

	w.name = nW.name
	w.keyPair = nW.keyPair
	w.pubKey = nW.pubKey
	w.walletID = nW.walletID

	return nil
}
