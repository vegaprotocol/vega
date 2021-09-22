package eth

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/crypto"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/pkg/errors"
)

type ClefWalletLoader struct {
	walletHome string
	ethClient  ETHClient
}

func InitialiseClefWalletLoader(ethClient ETHClient) (*ClefWalletLoader, error) {
	walletHome, err := vegaPaths.DataDirFor(paths.EthereumNodeWalletsDataHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get the directory path for %s: %w", paths.EthereumNodeWalletsDataHome, err)
	}

	return &ClefWalletLoader{
		walletHome: walletHome,
		ethClient:  ethClient,
	}, nil
}

func (l *ClefWalletLoader) Generate(passphrase string) (*Wallet, map[string]string, error) {
	ks := keystore.NewKeyStore(l.walletHome, keystore.StandardScryptN, keystore.StandardScryptP)
	acc, err := ks.NewAccount(passphrase)
	if err != nil {
		return nil, nil, err
	}

	_, fileName := filepath.Split(acc.URL.Path)

	content, err := vgfs.ReadFile(acc.URL.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read file %s: %w", acc.URL.Path, err)
	}

	w, err := l.newWallet(fileName, passphrase, content)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't create wallet: %w", err)
	}

	data := map[string]string{
		"walletFilePath": acc.URL.Path,
	}

	return w, data, nil
}

func (l *ClefWalletLoader) Load(walletName, passphrase string) (*Wallet, error) {
	data, err := fs.ReadFile(os.DirFS(l.walletHome), walletName)
	if err != nil {
		return nil, fmt.Errorf("unable to read store file: %v", err)
	}

	return l.newWallet(walletName, passphrase, data)
}

func (l *ClefWalletLoader) Import(sourceFilePath, passphrase string) (*Wallet, map[string]string, error) {
	content, err := vgfs.ReadFile(sourceFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't read file %s: %w", sourceFilePath, err)
	}

	_, fileName := filepath.Split(sourceFilePath)

	walletFilePath := filepath.Join(l.walletHome, fileName)
	err = vgfs.WriteFile(walletFilePath, content)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't write file %s: %w", walletFilePath, err)
	}

	w, err := l.newWallet(fileName, passphrase, content)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't create wallet: %w", err)
	}

	data := map[string]string{
		"walletFilePath": walletFilePath,
	}

	return w, data, nil
}

func (l *ClefWalletLoader) newWallet(walletName, passphrase string, data []byte) (*ClefWallet, error) {
	// NewKeyStore always create a new wallet key store file
	// we create this in tmp as we do not want to impact the original one.
	tempFile := filepath.Join(os.TempDir(), vgrand.RandomStr(10))
	ks := keystore.NewKeyStore(tempFile, keystore.StandardScryptN, keystore.StandardScryptP)

	acc, err := ks.Import(data, passphrase, passphrase)
	if err != nil {
		return nil, err
	}

	if err := ks.Unlock(acc, passphrase); err != nil {
		return nil, errors.Wrap(err, "unable to unlock wallet")
	}

	address := crypto.NewPublicKeyOrAddress(acc.Address.Hex(), acc.Address.Bytes())

	return &ClefWallet{
		name:       walletName,
		acc:        acc,
		ks:         ks,
		clt:        l.ethClient,
		passphrase: passphrase,
		address:    address,
	}, nil
}
