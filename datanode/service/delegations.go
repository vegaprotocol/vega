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

type delegationStore interface {
	Add(ctx context.Context, d entities.Delegation) error
	GetAll(ctx context.Context) ([]entities.Delegation, error)
	Get(ctx context.Context, partyID *string, nodeID *string, epoch *int64, p entities.Pagination) ([]entities.Delegation, entities.PageInfo, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Delegation, error)
}

type Delegation struct {
	store delegationStore
}

func NewDelegation(store delegationStore, log *logging.Logger) *Delegation {
	return &Delegation{
		store: store,
	}
}

func (d *Delegation) Add(ctx context.Context, delegation entities.Delegation) error {
	err := d.store.Add(ctx, delegation)
	if err != nil {
		return err
	}
	return nil
}

func (d *Delegation) GetAll(ctx context.Context) ([]entities.Delegation, error) {
	return d.store.GetAll(ctx)
}

func (d *Delegation) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Delegation, error) {
	return d.store.GetByTxHash(ctx, txHash)
}

func (d *Delegation) Get(ctx context.Context, partyID *string, nodeID *string, epoch *int64, p entities.Pagination) ([]entities.Delegation, entities.PageInfo, error) {
	return d.store.Get(ctx, partyID, nodeID, epoch, p)
}
