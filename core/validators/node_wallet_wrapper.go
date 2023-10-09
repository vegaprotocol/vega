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

package validators

import (
	"code.vegaprotocol.io/vega/core/nodewallets"
)

type NodeWallets interface {
	GetVega() Wallet
	GetTendermintPubkey() string
	GetEthereumAddress() string
	GetEthereum() Signer
}

type NodeWalletsWrapper struct {
	*nodewallets.NodeWallets
}

func WrapNodeWallets(nw *nodewallets.NodeWallets) *NodeWalletsWrapper {
	return &NodeWalletsWrapper{nw}
}

func (w *NodeWalletsWrapper) GetVega() Wallet {
	return w.Vega
}

func (w *NodeWalletsWrapper) GetEthereum() Signer {
	return w.Ethereum
}

func (w *NodeWalletsWrapper) GetEthereumAddress() string {
	return w.Ethereum.PubKey().Hex()
}

func (w *NodeWalletsWrapper) GetTendermintPubkey() string {
	return w.Tendermint.Pubkey
}
