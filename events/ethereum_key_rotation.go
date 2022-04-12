package events

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

// EthereumKeyRotation ...
type EthereumKeyRotation struct {
	*Base
	NodeID      string
	OldAddr     string
	NewAddr     string
	BlockHeight uint64
}

func NewEthereumKeyRotationEvent(
	ctx context.Context,
	nodeID string,
	oldAddr string,
	newAddr string,
	blockHeight uint64,
) *EthereumKeyRotation {
	return &EthereumKeyRotation{
		Base:        newBase(ctx, EthereumKeyRotationEvent),
		NodeID:      nodeID,
		OldAddr:     oldAddr,
		NewAddr:     newAddr,
		BlockHeight: blockHeight,
	}
}

func (kr EthereumKeyRotation) EthereumKeyRotation() eventspb.EthereumKeyRotation {
	return kr.Proto()
}

func (kr EthereumKeyRotation) Proto() eventspb.EthereumKeyRotation {
	return eventspb.EthereumKeyRotation{
		NodeId:      kr.NodeID,
		OldAddress:  kr.OldAddr,
		NewAddress:  kr.NewAddr,
		BlockHeight: kr.BlockHeight,
	}
}

func (kr EthereumKeyRotation) StreamMessage() *eventspb.BusEvent {
	krproto := kr.Proto()

	busEvent := newBusEventFromBase(kr.Base)
	busEvent.Event = &eventspb.BusEvent_EthereumKeyRotation{
		EthereumKeyRotation: &krproto,
	}
	return busEvent
}

func EthereumKeyRotationEventFromStream(ctx context.Context, be *eventspb.BusEvent) *EthereumKeyRotation {
	event := be.GetEthereumKeyRotation()
	if event == nil {
		return nil
	}

	return &EthereumKeyRotation{
		Base:        newBaseFromBusEvent(ctx, EthereumKeyRotationEvent, be),
		NodeID:      event.GetNodeId(),
		OldAddr:     event.GetOldAddress(),
		NewAddr:     event.GetNewAddress(),
		BlockHeight: event.GetBlockHeight(),
	}
}
