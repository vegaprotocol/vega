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
