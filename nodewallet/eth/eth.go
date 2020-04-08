package eth

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ETHClient interface {
	bind.ContractBackend
}

type Wallet struct {
	acc accounts.Account
	ks  *keystore.KeyStore
	clt *ethclient.Client
}

func DevInit(path, passphrase string) (string, error) {
	ks := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(passphrase)
	if err != nil {
		return "", err
	}
	return acc.URL.Path, nil
}

func New(cfg Config, path, passphrase string) (*Wallet, error) {
	// NewKeyStore always create a new wallet key store file
	// we create this in tmp as we do not want to impact the original one.
	ks := keystore.NewKeyStore(
		os.TempDir(), keystore.StandardScryptN, keystore.StandardScryptP)
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// now instanciate the client
	clt, err := ethclient.Dial(cfg.Address)
	if err != nil {
		return nil, err
	}

	// just trying to call to make sure there's not issue
	_, err = clt.ChainID(context.Background())
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
		clt: clt,
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

func (w *Wallet) Client() ETHClient {
	return w.clt
}
