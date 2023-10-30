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

// KeyRotation ...
type KeyRotation struct {
	*Base
	NodeID      string
	OldPubKey   string
	NewPubKey   string
	BlockHeight uint64
}

func NewVegaKeyRotationEvent(
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
