package vega

import (
	"code.vegaprotocol.io/go-wallet/wallet"
	"code.vegaprotocol.io/vega/crypto"
)

type Wallet struct {
	handler    *wallet.Handler
	walletName string
	keyPair    wallet.KeyPair
	pubKey     crypto.PublicKeyOrAddress
}

func (w *Wallet) Name() string {
	return w.walletName
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
