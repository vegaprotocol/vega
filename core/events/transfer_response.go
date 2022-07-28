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

	ptypes "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/types"
)

type TransferResponse struct {
	*Base
	responses []*ptypes.TransferResponse
}

// NewTransferResponse returns an event with transfer responses - this is the replacement of the transfer buffer.
func NewTransferResponse(ctx context.Context, responses []*types.TransferResponse) *TransferResponse {
	return &TransferResponse{
		Base:      newBase(ctx, TransferResponses),
		responses: types.TransferResponses(responses).IntoProto(),
	}
}

// TransferResponses returns the actual event payload.
func (t *TransferResponse) TransferResponses() []*ptypes.TransferResponse {
	return t.responses
}

func (t TransferResponse) IsParty(id string) bool {
	for _, r := range t.responses {
		for _, e := range r.Transfers {
			if e.FromAccount == id || e.ToAccount == id {
				return true
			}
		}
	}
	return false
}

func (t *TransferResponse) Proto() eventspb.TransferResponses {
	return eventspb.TransferResponses{
		Responses: t.responses,
	}
}

func (t TransferResponse) StreamMessage() *eventspb.BusEvent {
	p := t.Proto()
	busEvent := newBusEventFromBase(t.Base)
	busEvent.Event = &eventspb.BusEvent_TransferResponses{
		TransferResponses: &p,
	}

	return busEvent
}

func TransferResponseEventFromStream(ctx context.Context, be *eventspb.BusEvent) *TransferResponse {
	return &TransferResponse{
		Base:      newBaseFromBusEvent(ctx, TransferResponses, be),
		responses: be.GetTransferResponses().Responses,
	}
}
