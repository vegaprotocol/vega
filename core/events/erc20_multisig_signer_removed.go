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

type ERC20MultiSigSignerRemoved struct {
	*Base
	evt eventspb.ERC20MultiSigSignerRemoved
}

func NewERC20MultiSigSignerRemoved(ctx context.Context, evt eventspb.ERC20MultiSigSignerRemoved) *ERC20MultiSigSignerRemoved {
	return &ERC20MultiSigSignerRemoved{
		Base: newBase(ctx, ERC20MultiSigSignerRemovedEvent),
		evt:  evt,
	}
}

func (s ERC20MultiSigSignerRemoved) ERC20MultiSigSignerRemoved() eventspb.ERC20MultiSigSignerRemoved {
	return s.evt
}

func (s ERC20MultiSigSignerRemoved) Proto() eventspb.ERC20MultiSigSignerRemoved {
	return s.evt
}

func (s ERC20MultiSigSignerRemoved) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSignerRemoved{
		Erc20MultisigSignerRemoved: &s.evt,
	}
	return busEvent
}

func ERC20MultiSigSignerRemovedFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigSignerRemoved {
	return &ERC20MultiSigSignerRemoved{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigSignerRemovedEvent, be),
		evt:  *be.GetErc20MultisigSignerRemoved(),
	}
}
