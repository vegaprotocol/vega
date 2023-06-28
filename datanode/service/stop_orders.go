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
)

type StopOrderStore interface {
	Add(order entities.StopOrder) error
	Flush(ctx context.Context) ([]entities.StopOrder, error)
	GetStopOrder(ctx context.Context, orderID string) (entities.StopOrder, error)
	ListStopOrders(ctx context.Context, filter entities.StopOrderFilter, p entities.CursorPagination) ([]entities.StopOrder, entities.PageInfo, error)
}

type StopOrders struct {
	store StopOrderStore
}

func NewStopOrders(store StopOrderStore) *StopOrders {
	return &StopOrders{
		store: store,
	}
}

func (s *StopOrders) Add(order entities.StopOrder) error {
	return s.store.Add(order)
}

func (s *StopOrders) Flush(ctx context.Context) error {
	_, err := s.store.Flush(ctx)
	return err
}

func (s *StopOrders) GetStopOrder(ctx context.Context, orderID string) (entities.StopOrder, error) {
	return s.store.GetStopOrder(ctx, orderID)
}

func (s *StopOrders) ListStopOrders(ctx context.Context, filter entities.StopOrderFilter, p entities.CursorPagination) ([]entities.StopOrder, entities.PageInfo, error) {
	return s.store.ListStopOrders(ctx, filter, p)
}
