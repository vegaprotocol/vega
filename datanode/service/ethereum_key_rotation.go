// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
