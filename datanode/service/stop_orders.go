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
