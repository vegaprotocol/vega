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

type ERC20MultiSigSignerAdded struct {
	*Base
	evt eventspb.ERC20MultiSigSignerAdded
}

func NewERC20MultiSigSignerAdded(ctx context.Context, evt eventspb.ERC20MultiSigSignerAdded) *ERC20MultiSigSignerAdded {
	return &ERC20MultiSigSignerAdded{
		Base: newBase(ctx, ERC20MultiSigSignerAddedEvent),
		evt:  evt,
	}
}

func (s ERC20MultiSigSignerAdded) ERC20MultiSigSignerAdded() eventspb.ERC20MultiSigSignerAdded {
	return s.evt
}

func (s ERC20MultiSigSignerAdded) Proto() eventspb.ERC20MultiSigSignerAdded {
	return s.evt
}

func (s ERC20MultiSigSignerAdded) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSignerAdded{
		Erc20MultisigSignerAdded: &s.evt,
	}
	return busEvent
}

func ERC20MultiSigSignerAddedFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigSignerAdded {
	return &ERC20MultiSigSignerAdded{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigSignerAddedEvent, be),
		evt:  *be.GetErc20MultisigSignerAdded(),
	}
}
