package vega

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/nodewallets/registry"
	"code.vegaprotocol.io/vegawallet/wallet"
)

type loader interface {
	Load(walletName, passphrase string) (*Wallet, error)
}

type Wallet struct {
	loader   loader
	name     string
	keyPair  wallet.KeyPair
	pubKey   crypto.PublicKey
	walletID crypto.PublicKey
	mut      sync.Mutex
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

func (w *Wallet) Reload(rw registry.RegisteredVegaWallet) error {
	nW, err := w.loader.Load(rw.Name, rw.Passphrase)
	if err != nil {
		return fmt.Errorf("failed to load wallet: %w", err)
	}

	w.mut.Lock()
	defer w.mut.Unlock()

	w.name = nW.name
	w.keyPair = nW.keyPair
	w.pubKey = nW.pubKey
	w.walletID = nW.walletID

	return nil
}
