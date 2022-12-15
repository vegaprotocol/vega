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

type ProtocolUpgradeStarted struct {
	*Base
	pps eventspb.ProtocolUpgradeStarted
}

// NewProtocolUpgradeStarted returns a new time Update event.
func NewProtocolUpgradeStarted(ctx context.Context, bb eventspb.ProtocolUpgradeStarted) *ProtocolUpgradeStarted {
	return &ProtocolUpgradeStarted{
		Base: newBase(ctx, ProtocolUpgradeStartedEvent),
		pps:  bb,
	}
}

func (b ProtocolUpgradeStarted) ProtocolUpgradeStarted() eventspb.ProtocolUpgradeStarted {
	return b.pps
}

func (b ProtocolUpgradeStarted) Proto() eventspb.ProtocolUpgradeStarted {
	return b.pps
}

func (b ProtocolUpgradeStarted) StreamMessage() *eventspb.BusEvent {
	p := b.Proto()
	busEvent := newBusEventFromBase(b.Base)
	busEvent.Event = &eventspb.BusEvent_ProtocolUpgradeStarted{
		ProtocolUpgradeStarted: &p,
	}

	return busEvent
}

func ProtocolUpgradeStartedEventFromStream(ctx context.Context, be *eventspb.BusEvent) *ProtocolUpgradeStarted {
	return &ProtocolUpgradeStarted{
		Base: newBaseFromBusEvent(ctx, ProtocolUpgradeStartedEvent, be),
		pps:  *be.GetProtocolUpgradeStarted(),
	}
}
