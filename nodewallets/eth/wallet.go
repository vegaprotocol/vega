package eth

import (
	"code.vegaprotocol.io/vega/crypto"
)

type wallet interface {
	Cleanup() error
	Name() string
	Chain() string
	Sign(data []byte) ([]byte, error)
	Algo() string
	Version() string
	PubKeyOrAddress() crypto.PublicKeyOrAddress
}

type Wallet struct {
	name       string
	acc        accounts.Account
	ks         *keystore.KeyStore
	passphrase string
	address    crypto.PublicKey
	w          wallet
}

func NewWallet(w wallet) *Wallet {
	return &Wallet{w}
}

func (w *Wallet) Cleanup() error {
	// just remove the wallet from the tmp file
	return w.w.Cleanup()
}

func (w *Wallet) Name() string {
	return w.w.Name()
}

func (w *Wallet) Chain() string {
	return w.w.Chain()
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	return w.w.Sign(data)
}

func (w *Wallet) Algo() string {
	return w.w.Algo()
}

func (w *Wallet) Version() string {
	return w.w.Version()
}

func (w *Wallet) PubKey() crypto.PublicKey {
	return w.address
}
