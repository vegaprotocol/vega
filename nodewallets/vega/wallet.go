package vega

import (
	"code.vegaprotocol.io/go-wallet/wallet"
	"code.vegaprotocol.io/go-wallet/wallets"
	"code.vegaprotocol.io/vega/crypto"
)

type Wallet struct {
	handler    *wallets.Handler
	walletName string
	keyPair    wallet.KeyPair
	pubKey     crypto.PublicKeyOrAddress
	walletID   crypto.PublicKeyOrAddress
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

func (w *Wallet) ID() crypto.PublicKeyOrAddress {
	return w.walletID
}
