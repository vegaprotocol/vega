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

package vega

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/core/nodewallets/registry"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/wallet/wallet"
)

type loader interface {
	Load(walletName, passphrase string) (*Wallet, error)
}

type Wallet struct {
	loader   loader
	name     string
	keyPair  wallet.KeyPair
	pubKey   crypto.PublicKey
	walletID crypto.PublicKey
	mut      sync.Mutex
}

func (w *Wallet) Name() string {
	return w.name
}

func (w *Wallet) Chain() string {
	return "vega"
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	return w.keyPair.SignAny(data)
}

func (w *Wallet) Algo() string {
	return w.keyPair.AlgorithmName()
}

func (w *Wallet) Version() uint32 {
	return w.keyPair.AlgorithmVersion()
}

func (w *Wallet) PubKey() crypto.PublicKey {
	return w.pubKey
}

func (w *Wallet) Index() uint32 {
	return w.keyPair.Index()
}

func (w *Wallet) ID() crypto.PublicKey {
	return w.walletID
}

func (w *Wallet) Reload(rw registry.RegisteredVegaWallet) error {
	nW, err := w.loader.Load(rw.Name, rw.Passphrase)
	if err != nil {
		return fmt.Errorf("failed to load wallet: %w", err)
	}

	w.mut.Lock()
	defer w.mut.Unlock()

	w.name = nW.name
	w.keyPair = nW.keyPair
	w.pubKey = nW.pubKey
	w.walletID = nW.walletID

	return nil
}
