package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// Indicates the end of events for a given block
type BlockEnd struct {
	*Base
}

// NewBlockEnd returns a new block end event.
func NewBlockEnd(ctx context.Context) *BlockEnd {
	return &BlockEnd{
		Base: newBase(ctx, BlockEndEvent),
	}
}

func (t BlockEnd) Proto() eventspb.BlockEnd {
	return eventspb.BlockEnd{}
}

func (t BlockEnd) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_BlockEnd{}

	return busEvent
}

func BlockEndFromStream(ctx context.Context, be *eventspb.BusEvent) *BlockEnd {
	return &BlockEnd{
		Base: newBaseFromBusEvent(ctx, BlockEndEvent, be),
	}
}
