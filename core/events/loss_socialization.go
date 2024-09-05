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

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type LossSoc struct {
	*Base
	partyID  string
	marketID string
	amount   *num.Int
	ts       int64
	lType    types.LossType
}

func NewLossSocializationEvent(ctx context.Context, partyID, marketID string, amount *num.Uint, neg bool, ts int64, lType types.LossType) *LossSoc {
	signedAmount := num.NewIntFromUint(amount)
	if neg {
		signedAmount.FlipSign()
	}
	return &LossSoc{
		Base:     newBase(ctx, LossSocializationEvent),
		partyID:  partyID,
		marketID: marketID,
		amount:   signedAmount,
		ts:       ts,
		lType:    lType,
	}
}

func (l LossSoc) IsFunding() bool {
	return l.lType == types.LossTypeFunding
}

func (l LossSoc) IsParty(id string) bool {
	return l.partyID == id
}

func (l LossSoc) PartyID() string {
	return l.partyID
}

func (l LossSoc) MarketID() string {
	return l.marketID
}

func (l LossSoc) Negative() bool {
	return l.amount.IsNegative()
}

func (l LossSoc) Amount() *num.Int {
	return l.amount.Clone()
}

func (l LossSoc) Timestamp() int64 {
	return l.ts
}

func (l LossSoc) Proto() eventspb.LossSocialization {
	return eventspb.LossSocialization{
		MarketId: l.marketID,
		PartyId:  l.partyID,
		Amount:   l.amount.String(),
		LossType: l.lType,
	}
}

func (l LossSoc) StreamMessage() *eventspb.BusEvent {
	p := l.Proto()

	busEvent := newBusEventFromBase(l.Base)
	busEvent.Event = &eventspb.BusEvent_LossSocialization{
		LossSocialization: &p,
	}

	return busEvent
}

func LossSocializationEventFromStream(ctx context.Context, be *eventspb.BusEvent) *LossSoc {
	ls := be.GetLossSocialization()
	lse := &LossSoc{
		Base:     newBaseFromBusEvent(ctx, LossSocializationEvent, be),
		partyID:  ls.PartyId,
		marketID: ls.MarketId,
		lType:    ls.LossType,
	}

	lse.amount, _ = num.IntFromString(ls.Amount, 10)
	return lse
}
