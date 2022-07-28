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

package validators

import (
	"code.vegaprotocol.io/vega/core/nodewallets"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_wallet_mock.go -package mocks code.vegaprotocol.io/vega/core/validators NodeWallets
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
