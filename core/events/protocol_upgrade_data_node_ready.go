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

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type ProtocolUpgradeDataNodeReady struct {
	*Base
	pdr eventspb.ProtocolUpgradeDataNodeReady
}

func NewProtocolUpgradeDataNodeReady(ctx context.Context, lastBlockHeight int64) *ProtocolUpgradeDataNodeReady {
	return &ProtocolUpgradeDataNodeReady{
		Base: newBase(ctx, ProtocolUpgradeDataNodeReadyEvent),
		pdr: eventspb.ProtocolUpgradeDataNodeReady{
			LastBlockHeight: uint64(lastBlockHeight),
		},
	}
}

func (b ProtocolUpgradeDataNodeReady) Proto() eventspb.ProtocolUpgradeDataNodeReady {
	return b.pdr
}

func (b ProtocolUpgradeDataNodeReady) StreamMessage() *eventspb.BusEvent {
	p := b.Proto()
	busEvent := newBusEventFromBase(b.Base)
	busEvent.Event = &eventspb.BusEvent_ProtocolUpgradeDataNodeReady{
		ProtocolUpgradeDataNodeReady: &p,
	}

	return busEvent
}

func ProtocolUpgradeDataNodeReadyEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ProtocolUpgradeDataNodeReady {
	return &ProtocolUpgradeDataNodeReady{
		Base: newBaseFromBusEvent(ctx, ProtocolUpgradeDataNodeReadyEvent, be),
		pdr:  *be.GetProtocolUpgradeDataNodeReady(),
	}
}
