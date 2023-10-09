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
	proto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type Withdrawal struct {
	*Base
	w proto.Withdrawal
}

func NewWithdrawalEvent(ctx context.Context, w types.Withdrawal) *Withdrawal {
	return &Withdrawal{
		Base: newBase(ctx, WithdrawalEvent),
		w:    *w.IntoProto(),
	}
}

func (w *Withdrawal) Withdrawal() proto.Withdrawal {
	return w.w
}

func (w Withdrawal) IsParty(id string) bool {
	return w.w.PartyId == id
}

func (w Withdrawal) PartyID() string { return w.w.PartyId }

func (w Withdrawal) Proto() proto.Withdrawal {
	return w.w
}

func (w Withdrawal) StreamMessage() *eventspb.BusEvent {
	wit := w.w

	busEvent := newBusEventFromBase(w.Base)
	busEvent.Event = &eventspb.BusEvent_Withdrawal{
		Withdrawal: &wit,
	}

	return busEvent
}

func WithdrawalEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Withdrawal {
	return &Withdrawal{
		Base: newBaseFromBusEvent(ctx, WithdrawalEvent, be),
		w:    *be.GetWithdrawal(),
	}
}
