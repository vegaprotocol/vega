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
