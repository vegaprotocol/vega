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
