package events

import (
	"context"
	"encoding/hex"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/types"
)

type Checkpoint struct {
	*Base
	data eventspb.CheckpointEvent
}

func NewCheckpointEvent(ctx context.Context, snap *types.Snapshot) *Checkpoint {
	height, _ := vgcontext.BlockHeightFromContext(ctx)
	_, block := vgcontext.TraceIDFromContext(ctx)
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
		Version: eventspb.Version,
		Id:      e.eventID(),
		Block:   e.TraceID(),
		Type:    e.et.ToProto(),
		Event: &eventspb.BusEvent_Checkpoint{
			Checkpoint: &e.data,
		},
	}
}

func CheckpointEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Checkpoint {
	event := be.GetCheckpoint()
	if event == nil {
		return nil
	}

	return &Checkpoint{
		Base: newBaseFromStream(ctx, CheckpointEvent, be),
		data: *event,
	}
}
