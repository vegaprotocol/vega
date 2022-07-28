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

	"code.vegaprotocol.io/vega/types"
)

type ERC20MultiSigThresholdSet struct {
	*Base
	evt eventspb.ERC20MultiSigThresholdSetEvent
}

func NewERC20MultiSigThresholdSet(ctx context.Context, evt types.SignerThresholdSetEvent) *ERC20MultiSigThresholdSet {
	return &ERC20MultiSigThresholdSet{
		Base: newBase(ctx, ERC20MultiSigThresholdSetEvent),
		evt:  *(evt.IntoProto()),
	}
}

func (s ERC20MultiSigThresholdSet) ERC20MultiSigThresholdSet() eventspb.ERC20MultiSigThresholdSetEvent {
	return s.evt
}

func (s ERC20MultiSigThresholdSet) Proto() eventspb.ERC20MultiSigThresholdSetEvent {
	return s.evt
}

func (s ERC20MultiSigThresholdSet) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(s.Base)
	busEvent.Event = &eventspb.BusEvent_Erc20MultisigSetThresholdEvent{
		Erc20MultisigSetThresholdEvent: &s.evt,
	}

	return busEvent
}

func ERC20MultiSigThresholdSetFromStream(ctx context.Context, be *eventspb.BusEvent) *ERC20MultiSigThresholdSet {
	return &ERC20MultiSigThresholdSet{
		Base: newBaseFromBusEvent(ctx, ERC20MultiSigThresholdSetEvent, be),
		evt:  *be.GetErc20MultisigSetThresholdEvent(),
	}
}
