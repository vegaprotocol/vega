package events

import (
	"context"
	"encoding/hex"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/types"
)

type Checkpoint struct {
	*Base
	data eventspb.CheckpointEvent
}

func NewCheckpointEvent(ctx context.Context, snap *types.Snapshot) *Checkpoint {
	height, _ := contextutil.BlockHeightFromContext(ctx)
	_, block := contextutil.TraceIDFromContext(ctx)
	return &Checkpoint{
		Base: newBase(ctx, CheckpointEvent),
		data: eventspb.CheckpointEvent{
			Hash:        hex.EncodeToString(snap.Hash),
			BlockHash:   block,
			BlockHeight: uint64(height),
		},
	}
}

func (e Checkpoint) Proto() eventspb.CheckpointEvent {
	return e.data
}

func (e Checkpoint) StreamMessage() *eventspb.BusEvent {
	return &eventspb.BusEvent{
		Id:    e.eventID(),
		Block: e.TraceID(),
		Type:  e.et.ToProto(),
		Event: &eventspb.BusEvent_Checkpoint{
			Checkpoint: &e.data,
		},
	}
}
