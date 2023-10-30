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
	ptypes "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type Acc struct {
	*Base
	a ptypes.Account
}

func NewAccountEvent(ctx context.Context, a types.Account) *Acc {
	return &Acc{
		Base: newBase(ctx, AccountEvent),
		a:    *(a.IntoProto()),
	}
}

func (a Acc) IsParty(id string) bool {
	return a.a.Owner == id
}

func (a Acc) PartyID() string {
	return a.a.Owner
}

func (a Acc) MarketID() string {
	return a.a.MarketId
}

func (a *Acc) Account() ptypes.Account {
	return a.a
}

func (a Acc) Proto() ptypes.Account {
	return a.a
}

func (a Acc) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(a.Base)
	busEvent.Event = &eventspb.BusEvent_Account{
		Account: &a.a,
	}
	return busEvent
}

func AccountEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Acc {
	return &Acc{
		Base: newBaseFromBusEvent(ctx, AccountEvent, be),
		a:    *be.GetAccount(),
	}
}
