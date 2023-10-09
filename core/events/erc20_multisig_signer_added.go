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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ERC20MultiSigSignerAdded struct {
	*Base
	evt eventspb.ERC20MultiSigSignerAdded
}

func NewERC20MultiSigSignerAdded(ctx context.Context, evt eventspb.ERC20MultiSigSignerAdded) *ERC20MultiSigSignerAdded {
	return &ERC20MultiSigSignerAdded{
		Base: newBase(ctx, ERC20MultiSigSignerAddedEvent),
		evt:  evt,
	}
}

func (s ERC20MultiSigSignerAdded) ERC20MultiSigSignerAdded() eventspb.ERC20MultiSigSignerAdded {
	return s.evt
}

func (s ERC20MultiSigSignerAdded) Proto() eventspb.ERC20MultiSigSignerAdded {
	return s.evt
}

func (s ERC20MultiSigSignerAdded) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSignerAdded{
		Erc20MultisigSignerAdded: &s.evt,
	}
	return busEvent
}

func ERC20MultiSigSignerAddedFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigSignerAdded {
	return &ERC20MultiSigSignerAdded{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigSignerAddedEvent, be),
		evt:  *be.GetErc20MultisigSignerAdded(),
	}
}
