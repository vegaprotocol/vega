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

package events

import (
	"context"

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type SettleMarket struct {
	*Base
	marketID       string
	settledPrice   *num.Uint
	positionFactor num.Decimal
	ts             int64
}

func NewMarketSettled(ctx context.Context, marketID string, ts int64, settledPrice *num.Uint, positionFactor num.Decimal) *SettleMarket {
	return &SettleMarket{
		Base:           newBase(ctx, SettleMarketEvent),
		marketID:       marketID,
		settledPrice:   settledPrice.Clone(),
		positionFactor: positionFactor,
		ts:             ts,
	}
}

func (m SettleMarket) MarketID() string {
	return m.marketID
}

// PartyID will return an empty string as this is only required to satisfy an interface
// for identifying events that can affect positions in the data-node.
func (m SettleMarket) PartyID() string {
	return ""
}

func (m SettleMarket) SettledPrice() *num.Uint {
	return m.settledPrice.Clone()
}

func (m SettleMarket) PositionFactor() num.Decimal {
	return m.positionFactor
}

func (m SettleMarket) Timestamp() int64 {
	return m.ts
}

func (m SettleMarket) Proto() *eventspb.SettleMarket {
	return &eventspb.SettleMarket{
		MarketId:       m.marketID,
		Price:          m.settledPrice.String(),
		PositionFactor: m.positionFactor.String(),
	}
}

func (m SettleMarket) StreamMessage() *eventspb.BusEvent {
	p := m.Proto()
	busEvent := newBusEventFromBase(m.Base)
	busEvent.Event = &eventspb.BusEvent_SettleMarket{SettleMarket: p}
	return busEvent
}

func SettleMarketEventFromStream(ctx context.Context, be *eventspb.BusEvent) *SettleMarket {
	sm := be.GetSettleMarket()
	smPrice, _ := num.UintFromString(sm.Price, 10)
	positionFactor := num.MustDecimalFromString(sm.PositionFactor)

	return &SettleMarket{
		Base:           newBaseFromBusEvent(ctx, SettleMarketEvent, be),
		marketID:       sm.MarketId,
		settledPrice:   smPrice,
		positionFactor: positionFactor,
	}
}
