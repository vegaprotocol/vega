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

package gql

import (
	"context"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/vegatime"
	vega "code.vegaprotocol.io/vega/protos/vega/api/v1"
)

type statisticsResolver VegaResolverRoot

func (s *statisticsResolver) CurrentTime(ctx context.Context, obj *vega.Statistics) (int64, error) {
	t, err := vegatime.Parse(obj.CurrentTime)
	if err != nil {
		return 0, err
	}
	return t.UnixNano(), nil
}

func (s *statisticsResolver) GenesisTime(ctx context.Context, obj *vega.Statistics) (int64, error) {
	t, err := vegatime.Parse(obj.GenesisTime)
	if err != nil {
		return 0, err
	}
	return t.UnixNano(), nil
}

func (s *statisticsResolver) VegaTime(ctx context.Context, obj *vega.Statistics) (int64, error) {
	t, err := vegatime.Parse(obj.VegaTime)
	if err != nil {
		return 0, err
	}
	return t.UnixNano(), nil
}

func (s *statisticsResolver) BlockHeight(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.BlockHeight, 10), nil
}

func (s *statisticsResolver) BacklogLength(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.BacklogLength, 10), nil
}

func (s *statisticsResolver) TotalPeers(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.TotalPeers, 10), nil
}

func (s *statisticsResolver) TxPerBlock(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.TxPerBlock, 10), nil
}

func (s *statisticsResolver) AverageTxBytes(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.AverageTxBytes, 10), nil
}

func (s *statisticsResolver) AverageOrdersPerBlock(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.AverageOrdersPerBlock, 10), nil
}

func (s *statisticsResolver) TradesPerSecond(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.TradesPerSecond, 10), nil
}

func (s *statisticsResolver) OrdersPerSecond(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.OrdersPerSecond, 10), nil
}

func (s *statisticsResolver) TotalMarkets(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.TotalMarkets, 10), nil
}

func (s *statisticsResolver) TotalAmendOrder(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.TotalAmendOrder, 10), nil
}

func (s *statisticsResolver) TotalCancelOrder(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.TotalCancelOrder, 10), nil
}

func (s *statisticsResolver) TotalCreateOrder(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.TotalCreateOrder, 10), nil
}

func (s *statisticsResolver) EventCount(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.EventCount, 10), nil
}

func (s *statisticsResolver) EventsPerSecond(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.EventsPerSecond, 10), nil
}

func (s *statisticsResolver) TotalOrders(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.TotalOrders, 10), nil
}

func (s *statisticsResolver) TotalTrades(ctx context.Context, obj *vega.Statistics) (string, error) {
	return strconv.FormatUint(obj.TotalTrades, 10), nil
}

func (s *statisticsResolver) BlockDuration(ctx context.Context, obj *vega.Statistics) (string, error) {
	return time.Duration(obj.BlockDuration).String(), nil
}

func (s *statisticsResolver) Status(ctx context.Context, obj *vega.Statistics) (string, error) {
	return obj.Status.String(), nil
}
