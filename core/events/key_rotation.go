// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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
