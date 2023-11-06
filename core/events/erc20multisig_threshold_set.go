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

	"code.vegaprotocol.io/vega/core/types"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
