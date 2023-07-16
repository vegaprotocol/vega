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
