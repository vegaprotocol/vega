// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	proto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type MarketData struct {
	*Base
	md proto.MarketData
}

func NewMarketDataEvent(ctx context.Context, md types.MarketData) *MarketData {
	return &MarketData{
		Base: newBase(ctx, MarketDataEvent),
		md:   *md.IntoProto(),
	}
}

func (m MarketData) MarketID() string {
	return m.md.Market
}

func (m MarketData) MarketData() proto.MarketData {
	return m.md
}

func (m MarketData) Proto() proto.MarketData {
	return m.md
}

func (m MarketData) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(m.Base)
	busEvent.Event = &eventspb.BusEvent_MarketData{
		MarketData: &m.md,
	}

	return busEvent
}

func MarketDataEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketData {
	return &MarketData{
		Base: newBaseFromBusEvent(ctx, MarketDataEvent, be),
		md:   *be.GetMarketData(),
	}
}
