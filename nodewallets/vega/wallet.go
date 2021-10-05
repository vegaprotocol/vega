package vega

import (
	"code.vegaprotocol.io/go-wallet/wallet"
	"code.vegaprotocol.io/vega/crypto"
)

type Wallet struct {
	walletName string
	keyPair    wallet.KeyPair
	pubKey     crypto.PublicKey
	walletID   crypto.PublicKey
}

func (w *Wallet) Name() string {
	return w.walletName
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

func (w *Wallet) ID() crypto.PublicKey {
	return w.walletID
}
