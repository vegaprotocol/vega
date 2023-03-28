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

package eth

import (
	"code.vegaprotocol.io/vega/core/nodewallets/registry"
	"code.vegaprotocol.io/vega/libs/crypto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/nodewallets/eth EthereumWallet
type EthereumWallet interface {
	Cleanup() error
	Name() string
	Chain() string
	Sign(data []byte) ([]byte, error)
	Algo() string
	Version() (string, error)
	PubKey() crypto.PublicKey
	Reload(details registry.EthereumWalletDetails) error
}

type Wallet struct {
	w EthereumWallet
}

func NewWallet(w EthereumWallet) *Wallet {
	return &Wallet{w}
}

func (w *Wallet) Cleanup() error {
	return w.w.Cleanup()
}

func (w *Wallet) Name() string {
	return w.w.Name()
}

func (w *Wallet) Chain() string {
	return w.w.Chain()
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	return w.w.Sign(data)
}

func (w *Wallet) Algo() string {
	return w.w.Algo()
}

func (w *Wallet) Version() (string, error) {
	return w.w.Version()
}

func (w *Wallet) PubKey() crypto.PublicKey {
	return w.w.PubKey()
}

func (w *Wallet) Reload(details registry.EthereumWalletDetails) error {
	return w.w.Reload(details)
}
