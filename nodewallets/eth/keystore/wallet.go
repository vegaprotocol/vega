package keystore

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/nodewallets/registryloader"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type Wallet struct {
	homeDir    string
	name       string
	acc        accounts.Account
	ks         *keystore.KeyStore
	passphrase string
	address    crypto.PublicKey
}

func newWallet(walletHome, walletName, passphrase string, data []byte) (*Wallet, error) {
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
		homeDir:    walletHome,
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

func (w *Wallet) Reload(details registryloader.EthereumWalletDetails) error {
	d, ok := details.(registryloader.EthereumKeyStoreWallet)
	if !ok {
		return fmt.Errorf("failed to get EthereumKeyStoreWallet")
	}

	data, err := fs.ReadFile(os.DirFS(w.homeDir), w.name)
	if err != nil {
		return fmt.Errorf("couldn't read wallet file: %v", err)
	}

	nW, err := newWallet(w.homeDir, d.Name, d.Passphrase, data)
	if err != nil {
		return fmt.Errorf("couldn't create wallet: %w", err)
	}

	w.homeDir = nW.homeDir
	w.name = nW.name
	w.acc = nW.acc
	w.ks = nW.ks
	w.passphrase = nW.passphrase
	w.address = nW.address

	return nil
}
