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

	"code.vegaprotocol.io/vega/core/types"
	ptypes "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type TransferInstructionResponse struct {
	*Base
	responses []*ptypes.TransferInstructionResponse
}

// NewTransferInstructionResponse returns an event with transfer responses - this is the replacement of the transfer buffer.
func NewTransferInstructionResponse(ctx context.Context, responses []*types.TransferInstructionResponse) *TransferInstructionResponse {
	return &TransferInstructionResponse{
		Base:      newBase(ctx, TransferInstructionResponses),
		responses: types.TransferInstructionResponses(responses).IntoProto(),
	}
}

// TransferInstructionResponses returns the actual event payload.
func (t *TransferInstructionResponse) TransferInstructionResponses() []*ptypes.TransferInstructionResponse {
	return t.responses
}

func (t TransferInstructionResponse) IsParty(id string) bool {
	for _, r := range t.responses {
		for _, e := range r.Transfers {
			if e.FromAccount == id || e.ToAccount == id {
				return true
			}
		}
	}
	return false
}

func (t *TransferInstructionResponse) Proto() eventspb.TransferInstructionResponses {
	return eventspb.TransferInstructionResponses{
		Responses: t.responses,
	}
}

func (t TransferInstructionResponse) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TransferInstructionResponses{
		TransferInstructionResponses: &p,
	}

	return busEvent
}

func TransferInstructionResponseEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TransferInstructionResponse {
	return &TransferInstructionResponse{
		Base:      newBaseFromBusEvent(ctx, TransferInstructionResponses, be),
		responses: be.GetTransferInstructionResponses().Responses,
	}
}
