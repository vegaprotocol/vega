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

	"code.vegaprotocol.io/vega/core/types"
)

type ERC20MultiSigSigner struct {
	*Base
	evt eventspb.ERC20MultiSigSignerEvent
}

func NewERC20MultiSigSigner(ctx context.Context, evt types.SignerEvent) *ERC20MultiSigSigner {
	return &ERC20MultiSigSigner{
		Base: newBase(ctx, ERC20MultiSigSignerEvent),
		evt:  *(evt.IntoProto()),
	}
}

func (s ERC20MultiSigSigner) ERC20MultiSigSigner() eventspb.ERC20MultiSigSignerEvent {
	return s.evt
}

func (s ERC20MultiSigSigner) Proto() eventspb.ERC20MultiSigSignerEvent {
	return s.evt
}

func (s ERC20MultiSigSigner) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSignerEvent{
		Erc20MultisigSignerEvent: &s.evt,
	}

	return busEvent
}

func ERC20MultiSigSignerFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigSigner {
	return &ERC20MultiSigSigner{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigSignerEvent, be),
		evt:  *be.GetErc20MultisigSignerEvent(),
	}
}
