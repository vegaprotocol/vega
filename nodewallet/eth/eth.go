package eth

import (
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type Wallet struct {
	acc accounts.Account
	ks  *keystore.KeyStore
}

func DevInit(path, passphrase string) (string, error) {
	ks := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(passphrase)
	if err != nil {
		return "", err
	}
	return acc.URL.Path, nil
}

func New(path, passphrase string) (*Wallet, error) {
	// NewKeyStore always create a new wallet key store file
	// we create this in tmp as we do not want to impact the original one.
	ks := keystore.NewKeyStore(
		os.TempDir(), keystore.StandardScryptN, keystore.StandardScryptP)
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	acc, err := ks.Import(jsonBytes, passphrase, passphrase)
	if err != nil {
		return nil, err
	}
	return &Wallet{
		acc: acc,
		ks:  ks,
	}, nil
}

func (w *Wallet) Chain() string {
	return "ethereum"
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	return w.ks.SignHash(w.acc, accounts.TextHash(data))
}

func (w *Wallet) PubKeyOrAddress() []byte {
	return w.acc.Address.Bytes()
}
