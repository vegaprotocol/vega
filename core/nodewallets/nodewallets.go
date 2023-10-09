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

package nodewallets

import (
	"errors"

	"code.vegaprotocol.io/vega/core/nodewallets/eth"
	"code.vegaprotocol.io/vega/core/nodewallets/vega"
)

var (
	ErrVegaWalletIsMissing       = errors.New("the Vega node wallet is missing")
	ErrEthereumWalletIsMissing   = errors.New("the Ethereum node wallet is missing")
	ErrTendermintPubkeyIsMissing = errors.New("the Tendermint pubkey is missing")
)

type TendermintPubkey struct {
	Pubkey string
}

type NodeWallets struct {
	Vega       *vega.Wallet
	Ethereum   *eth.Wallet
	Tendermint *TendermintPubkey
}

func (w *NodeWallets) SetEthereumWallet(ethWallet *eth.Wallet) {
	w.Ethereum = ethWallet
}

func (w *NodeWallets) Verify() error {
	if w.Vega == nil {
		return ErrVegaWalletIsMissing
	}
	if w.Ethereum == nil {
		return ErrEthereumWalletIsMissing
	}
	if w.Tendermint == nil {
		return ErrTendermintPubkeyIsMissing
	}
	return nil
}
