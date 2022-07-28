// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
)

type MarketDepthService interface {
	AddOrder(order *types.Order, vegaTime time.Time, sequenceNumber uint64)
}

type MarketDepth struct {
	subscriber
	depthService MarketDepthService
}

// NewMarketDepthBuilder constructor to create a market depth subscriber
func NewMarketDepth(depthService MarketDepthService) *MarketDepth {
	m := MarketDepth{
		depthService: depthService,
	}

	return &m
}

func (m *MarketDepth) Types() []events.Type {
	return []events.Type{events.OrderEvent}
}

func (m *MarketDepth) Push(ctx context.Context, evt events.Event) error {
	m.consume(ctx, evt.(OrderEvent))
	return nil
}

func (m *MarketDepth) consume(ctx context.Context, event OrderEvent) {
	order, err := types.OrderFromProto(event.Order())
	if err != nil {
		panic(err)
	}
	m.depthService.AddOrder(order, m.vegaTime, event.Sequence())
}
