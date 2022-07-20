// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package keystore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/nodewallets/registry"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type loader interface {
	Load(walletName, passphrase string) (*Wallet, error)
}

type Wallet struct {
	loader     loader
	name       string
	acc        accounts.Account
	ks         *keystore.KeyStore
	passphrase string
	address    crypto.PublicKey
	mut        sync.Mutex
}

func newWallet(loader loader, walletName, passphrase string, data []byte) (*Wallet, error) {
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
		loader:     loader,
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

func (w *Wallet) Reload(details registry.EthereumWalletDetails) error {
	d, ok := details.(registry.EthereumKeyStoreWallet)
	if !ok {
		// this would mean an implementation error
		panic(fmt.Errorf("failed to get EthereumKeyStoreWallet"))
	}

	nW, err := w.loader.Load(d.Name, d.Passphrase)
	if err != nil {
		return fmt.Errorf("failed to load new wallet: %w", err)
	}

	w.mut.Lock()
	defer w.mut.Unlock()

	w.name = nW.name
	w.acc = nW.acc
	w.ks = nW.ks
	w.passphrase = nW.passphrase
	w.address = nW.address

	return nil
}
