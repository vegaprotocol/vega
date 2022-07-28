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

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type ERC20MultiSigSignerRemoved struct {
	*Base
	evt eventspb.ERC20MultiSigSignerRemoved
}

func NewERC20MultiSigSignerRemoved(ctx context.Context, evt eventspb.ERC20MultiSigSignerRemoved) *ERC20MultiSigSignerRemoved {
	return &ERC20MultiSigSignerRemoved{
		Base: newBase(ctx, ERC20MultiSigSignerRemovedEvent),
		evt:  evt,
	}
}

func (s ERC20MultiSigSignerRemoved) ERC20MultiSigSignerRemoved() eventspb.ERC20MultiSigSignerRemoved {
	return s.evt
}

func (s ERC20MultiSigSignerRemoved) Proto() eventspb.ERC20MultiSigSignerRemoved {
	return s.evt
}

func (s ERC20MultiSigSignerRemoved) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSignerRemoved{
		Erc20MultisigSignerRemoved: &s.evt,
	}
	return busEvent
}

func ERC20MultiSigSignerRemovedFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigSignerRemoved {
	return &ERC20MultiSigSignerRemoved{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigSignerRemovedEvent, be),
		evt:  *be.GetErc20MultisigSignerRemoved(),
	}
}
