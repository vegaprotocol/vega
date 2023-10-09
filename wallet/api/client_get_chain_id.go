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

package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	walletnode "code.vegaprotocol.io/vega/wallet/api/node"
)

type ClientGetChainIDResult struct {
	ChainID string `json:"chainID"`
}

type ClientGetChainID struct {
	nodeSelector walletnode.Selector
}

func (h *ClientGetChainID) Handle(ctx context.Context) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	currentNode, err := h.nodeSelector.Node(ctx, noNodeSelectionReporting)
	if err != nil {
		return nil, NodeCommunicationError(ErrNoHealthyNodeAvailable)
	}

	lastBlockData, err := currentNode.LastBlock(ctx)
	if err != nil {
		return nil, NodeCommunicationError(ErrCouldNotGetLastBlockInformation)
	}

	return ClientGetChainIDResult{
		ChainID: lastBlockData.ChainID,
	}, nil
}

func NewGetChainID(nodeSelector walletnode.Selector) *ClientGetChainID {
	return &ClientGetChainID{
		nodeSelector: nodeSelector,
	}
}
