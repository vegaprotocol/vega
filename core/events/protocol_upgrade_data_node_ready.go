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

type ProtocolUpgradeDataNodeReady struct {
	*Base
	pdr eventspb.ProtocolUpgradeDataNodeReady
}

func NewProtocolUpgradeDataNodeReady(ctx context.Context, lastBlockHeight int64) *ProtocolUpgradeDataNodeReady {
	return &ProtocolUpgradeDataNodeReady{
		Base: newBase(ctx, ProtocolUpgradeDataNodeReadyEvent),
		pdr: eventspb.ProtocolUpgradeDataNodeReady{
			LastBlockHeight: uint64(lastBlockHeight),
		},
	}
}

func (b ProtocolUpgradeDataNodeReady) Proto() eventspb.ProtocolUpgradeDataNodeReady {
	return b.pdr
}

func (b ProtocolUpgradeDataNodeReady) StreamMessage() *eventspb.BusEvent {
	p := b.Proto()
	busEvent := newBusEventFromBase(b.Base)
	busEvent.Event = &eventspb.BusEvent_ProtocolUpgradeDataNodeReady{
		ProtocolUpgradeDataNodeReady: &p,
	}

	return busEvent
}

func ProtocolUpgradeDataNodeReadyEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ProtocolUpgradeDataNodeReady {
	return &ProtocolUpgradeDataNodeReady{
		Base: newBaseFromBusEvent(ctx, ProtocolUpgradeDataNodeReadyEvent, be),
		pdr:  *be.GetProtocolUpgradeDataNodeReady(),
	}
}
