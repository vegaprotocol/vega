// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package keystore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"code.vegaprotocol.io/vega/core/nodewallets/registry"
	"code.vegaprotocol.io/vega/libs/crypto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

const KeyStoreAlgoType = "eth"

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
	return KeyStoreAlgoType
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
