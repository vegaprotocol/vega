package keystore

import (
	"fmt"
	"os"
	"path/filepath"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
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

func newWallet(walletName, passphrase string, data []byte) (*Wallet, error) {
	// NewKeyStore always create a new wallet key store file
	// we create this in tmp as we do not want to impact the original one.
	tempFile := filepath.Join(os.TempDir(), vgrand.RandomStr(10))
	ks := keystore.NewKeyStore(tempFile, keystore.StandardScryptN, keystore.StandardScryptP)

	acc, err := ks.Import(data, passphrase, passphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't import Ethereum wallet in keystore: %w", err)
	}

	if err := ks.Unlock(acc, passphrase); err != nil {
		return nil, fmt.Errorf("couldn't unlock Ethereum wallet: %w", err)
	}

	address := crypto.NewPublicKey(acc.Address.Hex(), acc.Address.Bytes())

	return &Wallet{
		name:       walletName,
		acc:        acc,
		ks:         ks,
		passphrase: passphrase,
		address:    address,
	}, nil
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

func (w *Wallet) Version() (string, error) {
	return "0", nil
}

func (w *Wallet) PubKey() crypto.PublicKey {
	return w.address
}
