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
