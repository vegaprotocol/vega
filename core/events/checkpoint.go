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
	"encoding/hex"

	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type Checkpoint struct {
	*Base
	data eventspb.CheckpointEvent
}

func NewCheckpointEvent(ctx context.Context, snap *types.CheckpointState) *Checkpoint {
	height, _ := vgcontext.BlockHeightFromContext(ctx)
	_, block := vgcontext.TraceIDFromContext(ctx)
	return &Checkpoint{
		Base: newBase(ctx, CheckpointEvent),
		data: eventspb.CheckpointEvent{
			Hash:        hex.EncodeToString(snap.Hash),
			BlockHash:   block,
			BlockHeight: height,
		},
	}
}

func (e Checkpoint) Proto() eventspb.CheckpointEvent {
	return e.data
}

func (e Checkpoint) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(e.Base)
	busEvent.Event = &eventspb.BusEvent_Checkpoint{
		Checkpoint: &e.data,
	}
	return busEvent
}

func CheckpointEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Checkpoint {
	if event := be.GetCheckpoint(); event != nil {
		return &Checkpoint{
			Base: newBaseFromBusEvent(ctx, CheckpointEvent, be),
			data: *event,
		}
	}
	return nil
}
