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
	"fmt"

	proto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/core/types"
)

type MarketCreated struct {
	*Base
	m  types.Market
	pm proto.Market
}

func NewMarketCreatedEvent(ctx context.Context, m types.Market) *MarketCreated {
	pm := m.IntoProto()
	return &MarketCreated{
		Base: newBase(ctx, MarketCreatedEvent),
		m:    m,
		pm:   *pm,
	}
}

// MarketEvent -> is needs to be logged as a market event.
func (m MarketCreated) MarketEvent() string {
	return fmt.Sprintf("Market ID %s created (%s)", m.m.ID, m.pm.String())
}

func (m MarketCreated) MarketID() string {
	return m.m.ID
}

func (m MarketCreated) Market() proto.Market {
	return m.pm
}

func (m MarketCreated) Proto() proto.Market {
	return m.pm
}

func (m MarketCreated) MarketProto() eventspb.MarketEvent {
	return eventspb.MarketEvent{
		MarketId: m.m.ID,
		Payload:  m.MarketEvent(),
	}
}

func (m MarketCreated) StreamMessage() *eventspb.BusEvent {
	market := m.Proto()

	busEvent := newBusEventFromBase(m.Base)
	busEvent.Event = &eventspb.BusEvent_MarketCreated{
		MarketCreated: &market,
	}

	return busEvent
}

func (m MarketCreated) StreamMarketMessage() *eventspb.BusEvent {
	return m.StreamMessage()
}

func MarketCreatedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *MarketCreated {
	m := be.GetMarketCreated()
	return &MarketCreated{
		Base: newBaseFromBusEvent(ctx, MarketCreatedEvent, be),
		m:    types.Market{ID: m.Id},
		pm:   *m,
	}
}
