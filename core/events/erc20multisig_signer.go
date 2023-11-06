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
