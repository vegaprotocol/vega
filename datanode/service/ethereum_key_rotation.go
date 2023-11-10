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

package service

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

type ethereumKeyRotationsStore interface {
	Add(context.Context, entities.EthereumKeyRotation) error
	List(context.Context, entities.NodeID, entities.CursorPagination) ([]entities.EthereumKeyRotation, entities.PageInfo, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.EthereumKeyRotation, error)
}

type EthereumKeyRotation struct {
	store ethereumKeyRotationsStore
}

func NewEthereumKeyRotation(service ethereumKeyRotationsStore, log *logging.Logger) *EthereumKeyRotation {
	return &EthereumKeyRotation{store: service}
}

func (e *EthereumKeyRotation) Add(ctx context.Context, kr entities.EthereumKeyRotation) error {
	return e.store.Add(ctx, kr)
}

func (e *EthereumKeyRotation) List(ctx context.Context,
	nodeID entities.NodeID,
	pagination entities.CursorPagination,
) ([]entities.EthereumKeyRotation, entities.PageInfo, error) {
	return e.store.List(ctx, nodeID, pagination)
}

func (e *EthereumKeyRotation) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.EthereumKeyRotation, error) {
	return e.store.GetByTxHash(ctx, txHash)
}
