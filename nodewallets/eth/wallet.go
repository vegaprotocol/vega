package eth

import (
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/nodewallets/registryloader"
)

type wallet interface {
	Cleanup() error
	Name() string
	Chain() string
	Sign(data []byte) ([]byte, error)
	Algo() string
	Version() (string, error)
	PubKey() crypto.PublicKey
	Reload(details registryloader.EthereumWalletDetails) error
}

type Wallet struct {
	w wallet
}

func NewWallet(w wallet) *Wallet {
	return &Wallet{w}
}

func (w *Wallet) Cleanup() error {
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

func (w *Wallet) Version() (string, error) {
	return w.w.Version()
}

func (w *Wallet) PubKey() crypto.PublicKey {
	return w.w.PubKey()
}

func (w *Wallet) Reload(details registryloader.EthereumWalletDetails) error {
	return w.w.Reload(details)
}
