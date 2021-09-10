package eth

import (
	"context"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"

	"code.vegaprotocol.io/vega/crypto"
	crypto2 "code.vegaprotocol.io/vega/libs/crypto"
	vgfs "code.vegaprotocol.io/vega/libs/fs"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

// ETHClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/eth_client_mock.go -package mocks code.vegaprotocol.io/vega/nodewallet/eth ETHClient
type ETHClient interface {
	bind.ContractBackend
	ChainID(context.Context) (*big.Int, error)
	NetworkID(context.Context) (*big.Int, error)
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

type WalletLoader struct {
	walletRootPath string
	ethClient      ETHClient
}

func NewWalletLoader(walletRootPath string, ethClient ETHClient) *WalletLoader {
	return &WalletLoader{
		walletRootPath: walletRootPath,
		ethClient:      ethClient,
	}
}

func (l *WalletLoader) Initialise() error {
	return vgfs.EnsureDir(l.walletRootPath)
}

func (l *WalletLoader) Generate(passphrase string) (*Wallet, error) {
	ks := keystore.NewKeyStore(l.walletRootPath, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(passphrase)
	if err != nil {
		return nil, err
	}

	_, fileName := filepath.Split(acc.URL.Path)

	data, err := vgfs.ReadFile(acc.URL.Path)
	if err != nil {
		return nil, fmt.Errorf("unable to read store file: %w", err)
	}

	return l.newWallet(fileName, passphrase, data)
}

func (l *WalletLoader) Load(walletName, passphrase string) (*Wallet, error) {
	data, err := fs.ReadFile(os.DirFS(l.walletRootPath), walletName)
	if err != nil {
		return nil, fmt.Errorf("unable to read store file: %v", err)
	}

	return l.newWallet(walletName, passphrase, data)
}

func (l *WalletLoader) Import(sourceFilePath, passphrase string) (*Wallet, error) {
	data, err := vgfs.ReadFile(sourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read store file: %w", err)
	}

	_, fileName := filepath.Split(sourceFilePath)

	err = vgfs.WriteFile(filepath.Join(l.walletRootPath, fileName), data)
	if err != nil {
		return nil, err
	}

	return l.newWallet(fileName, passphrase, data)
}

func (l *WalletLoader) newWallet(walletName, passphrase string, data []byte) (*Wallet, error) {
	// NewKeyStore always create a new wallet key store file
	// we create this in tmp as we do not want to impact the original one.
	tempFile := filepath.Join(os.TempDir(), crypto2.RandomStr(10))
	ks := keystore.NewKeyStore(tempFile, keystore.StandardScryptN, keystore.StandardScryptP)

	acc, err := ks.Import(data, passphrase, passphrase)
	if err != nil {
		return nil, err
	}

	if err := ks.Unlock(acc, passphrase); err != nil {
		return nil, errors.Wrap(err, "unable to unlock wallet")
	}

	address := crypto.NewPublicKeyOrAddress(acc.Address.Hex(), acc.Address.Bytes())

	return &Wallet{
		name:       walletName,
		acc:        acc,
		ks:         ks,
		clt:        l.ethClient,
		passphrase: passphrase,
		address:    address,
	}, nil
}
