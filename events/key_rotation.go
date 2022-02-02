package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// KeyRotation ...
type KeyRotation struct {
	*Base
	NodeID      string
	OldPubKey   string
	NewPubKey   string
	BlockHeight uint64
}

func NewKeyRotationEvent(
	ctx context.Context,
	nodeID string,
	oldPubKey string,
	newPubKey string,
	blockHeight uint64,
) *KeyRotation {
	return &KeyRotation{
		Base:        newBase(ctx, KeyRotationEvent),
		NodeID:      nodeID,
		OldPubKey:   oldPubKey,
		NewPubKey:   newPubKey,
		BlockHeight: blockHeight,
	}
}

func (kr KeyRotation) KeyRotation() eventspb.KeyRotation {
	return kr.Proto()
}

func (kr KeyRotation) Proto() eventspb.KeyRotation {
	return eventspb.KeyRotation{
		NodeId:      kr.NodeID,
		OldPubKey:   kr.OldPubKey,
		NewPubKey:   kr.NewPubKey,
		BlockHeight: kr.BlockHeight,
	}
}

func (kr KeyRotation) StreamMessage() *eventspb.BusEvent {
	krproto := kr.Proto()

	busEvent := newBusEventFromBase(kr.Base)
	busEvent.Event = &eventspb.BusEvent_KeyRotation{
		KeyRotation: &krproto,
	}
	return busEvent
}

func KeyRotationEventFromStream(ctx context.Context, be *eventspb.BusEvent) *KeyRotation {
	event := be.GetKeyRotation()
	if event == nil {
		return nil
	}

	return &KeyRotation{
		Base:        newBaseFromBusEvent(ctx, KeyRotationEvent, be),
		NodeID:      event.GetNodeId(),
		OldPubKey:   event.GetOldPubKey(),
		NewPubKey:   event.GetNewPubKey(),
		BlockHeight: event.GetBlockHeight(),
	}
}
