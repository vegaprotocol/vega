package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
)

type TransferResponse struct {
	*Base
	responses []*types.TransferResponse
}

// NewTransferResponse returns an event with transfer responses - this is the replacement of the transfer buffer
func NewTransferResponse(ctx context.Context, responses []*types.TransferResponse) *TransferResponse {
	trs := make([]*types.TransferResponse, len(responses))
	for i, tr := range responses {
		trs[i] = tr.DeepClone()
	}

	return &TransferResponse{
		Base:      newBase(ctx, TransferResponses),
		responses: trs,
	}
}

// TransferResponses returns the actual event payload
func (t *TransferResponse) TransferResponses() []*types.TransferResponse {
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
	return &eventspb.BusEvent{
		Id:    t.eventID(),
		Block: t.TraceID(),
		Type:  t.et.ToProto(),
		Event: &eventspb.BusEvent_TransferResponses{
			TransferResponses: &p,
		},
	}
}
