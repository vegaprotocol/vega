package eth

import (
	"code.vegaprotocol.io/vega/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type Wallet struct {
	name       string
	acc        accounts.Account
	ks         *keystore.KeyStore
	passphrase string
	address    crypto.PublicKey
}

func (w *Wallet) Cleanup() error {
	// just remove the wallet from the tmp file
	return w.ks.Delete(w.acc, w.passphrase)
}

func (w *Wallet) Name() string {
	return w.name
}

func (w *Wallet) Chain() string {
	return "ethereum"
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	return w.ks.SignHash(w.acc, data)
}

func (w *Wallet) Algo() string {
	return "eth"
}

func (w *Wallet) Version() string {
	return "0"
}

func (w *Wallet) PubKey() crypto.PublicKey {
	return w.address
}
