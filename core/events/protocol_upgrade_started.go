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

type ProtocolUpgradeStarted struct {
	*Base
	pps eventspb.ProtocolUpgradeStarted
}

// NewProtocolUpgradeStarted returns a new time Update event.
func NewProtocolUpgradeStarted(ctx context.Context, bb eventspb.ProtocolUpgradeStarted) *ProtocolUpgradeStarted {
	return &ProtocolUpgradeStarted{
		Base: newBase(ctx, ProtocolUpgradeStartedEvent),
		pps:  bb,
	}
}

func (b ProtocolUpgradeStarted) ProtocolUpgradeStarted() eventspb.ProtocolUpgradeStarted {
	return b.pps
}

func (b ProtocolUpgradeStarted) Proto() eventspb.ProtocolUpgradeStarted {
	return b.pps
}

func (b ProtocolUpgradeStarted) StreamMessage() *eventspb.BusEvent {
	p := b.Proto()
	busEvent := newBusEventFromBase(b.Base)
	busEvent.Event = &eventspb.BusEvent_ProtocolUpgradeStarted{
		ProtocolUpgradeStarted: &p,
	}

	return busEvent
}

func ProtocolUpgradeStartedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ProtocolUpgradeStarted {
	return &ProtocolUpgradeStarted{
		Base: newBaseFromBusEvent(ctx, ProtocolUpgradeStartedEvent, be),
		pps:  *be.GetProtocolUpgradeStarted(),
	}
}
