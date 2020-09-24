package eth

import (
	"context"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/pkg/errors"
)

// ETHClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_client_mock.go -package mocks code.vegaprotocol.io/vega/nodewallet/eth ETHClient
type ETHClient interface {
	bind.ContractBackend
	ChainID(context.Context) (*big.Int, error)
}

type Wallet struct {
	cfg           Config
	acc           accounts.Account
	ks            *keystore.KeyStore
	clt           ETHClient
	bridgeAddress string
	passphrase    string
}

func DevInit(path, passphrase string) (string, error) {
	ks := keystore.NewKeyStore(path, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(passphrase)
	if err != nil {
		return "", err
	}
	return acc.URL.Path, nil
}

func New(cfg Config, path, passphrase string, ethclt ETHClient) (*Wallet, error) {
	// NewKeyStore always create a new wallet key store file
	// we create this in tmp as we do not want to impact the original one.
	ks := keystore.NewKeyStore(
		os.TempDir(), keystore.StandardScryptN, keystore.StandardScryptP)
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// just trying to call to make sure there's not issue
	_, err = ethclt.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	acc, err := ks.Import(jsonBytes, passphrase, passphrase)
	if err != nil {
		return nil, err
	}

	if err := ks.Unlock(acc, passphrase); err != nil {
		return nil, errors.Wrap(err, "unable to unlock wallet")
	}

	return &Wallet{
		cfg:        cfg,
		acc:        acc,
		ks:         ks,
		clt:        ethclt,
		passphrase: passphrase,
	}, nil

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

func (w *Wallet) Version() uint64 {
	return 0
}

func (w *Wallet) PubKeyOrAddress() []byte {
	return w.acc.Address.Bytes()
}

func (w *Wallet) Client() ETHClient {
	return w.clt
}

func (w Wallet) BridgeAddress() string {
	return w.bridgeAddress
}

func (w *Wallet) SetERC20BridgeAddress(addr string) {
	w.bridgeAddress = addr
}
