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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/version"
)

type SnapshotTakenEvent struct {
	*Base
	SnapshotBlockHeight uint64
	SnapshotBlockHash   string
	VegaCoreVersion     string
}

func NewSnapshotEventEvent(ctx context.Context, blockHeight uint64, blockHash string) *SnapshotTakenEvent {
	return &SnapshotTakenEvent{
		Base:                newBase(ctx, CoreSnapshotEvent),
		SnapshotBlockHeight: blockHeight,
		SnapshotBlockHash:   blockHash,
		VegaCoreVersion:     version.Get(),
	}
}

func (ste SnapshotTakenEvent) Proto() eventspb.CoreSnapshotData {
	return eventspb.CoreSnapshotData{
		BlockHeight: ste.SnapshotBlockHeight,
		BlockHash:   ste.SnapshotBlockHash,
		CoreVersion: ste.VegaCoreVersion,
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
		Base:                newBaseFromBusEvent(ctx, CoreSnapshotEvent, be),
		SnapshotBlockHeight: event.BlockHeight,
		SnapshotBlockHash:   event.BlockHash,
		VegaCoreVersion:     event.CoreVersion,
	}
}
