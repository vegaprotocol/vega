// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package vega

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/core/crypto"
	"code.vegaprotocol.io/vega/core/nodewallets/registry"
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
