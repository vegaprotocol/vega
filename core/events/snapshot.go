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
	"code.vegaprotocol.io/vega/version"
)

type SnapshotTakenEvent struct {
	*Base
	SnapshotBlockHeight  uint64
	SnapshotBlockHash    string
	VegaCoreVersion      string
	ProtocolUpgradeBlock bool
}

func NewSnapshotEventEvent(ctx context.Context, blockHeight uint64, blockHash string, protocolUpgradeBlock bool) *SnapshotTakenEvent {
	return &SnapshotTakenEvent{
		Base:                 newBase(ctx, CoreSnapshotEvent),
		SnapshotBlockHeight:  blockHeight,
		SnapshotBlockHash:    blockHash,
		VegaCoreVersion:      version.Get(),
		ProtocolUpgradeBlock: protocolUpgradeBlock,
	}
}

func (ste SnapshotTakenEvent) Proto() eventspb.CoreSnapshotData {
	return eventspb.CoreSnapshotData{
		BlockHeight:          ste.SnapshotBlockHeight,
		BlockHash:            ste.SnapshotBlockHash,
		CoreVersion:          ste.VegaCoreVersion,
		ProtocolUpgradeBlock: ste.ProtocolUpgradeBlock,
	}
}

func (ste SnapshotTakenEvent) SnapshotTakenEvent() eventspb.CoreSnapshotData {
	return ste.Proto()
}

func (ste SnapshotTakenEvent) StreamMessage() *eventspb.BusEvent {
	p := ste.Proto()
	busEvent := newBusEventFromBase(ste.Base)
	busEvent.Event = &eventspb.BusEvent_CoreSnapshotEvent{
		CoreSnapshotEvent: &p,
	}

	return busEvent
}

func SnapthostTakenEventFromStream(ctx context.Context, be *eventspb.BusEvent) *SnapshotTakenEvent {
	event := be.GetCoreSnapshotEvent()
	if event == nil {
		return nil
	}

	return &SnapshotTakenEvent{
		Base:                 newBaseFromBusEvent(ctx, CoreSnapshotEvent, be),
		SnapshotBlockHeight:  event.BlockHeight,
		SnapshotBlockHash:    event.BlockHash,
		VegaCoreVersion:      event.CoreVersion,
		ProtocolUpgradeBlock: event.ProtocolUpgradeBlock,
	}
}
