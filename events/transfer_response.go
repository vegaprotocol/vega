package events

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type TransferResponse struct {
	*Base
	responses []*types.TransferResponse
}

// NewTransferResponse returns an event with transfer responses - this is the replacement of the transfer buffer
func NewTransferResponse(ctx context.Context, response []*types.TransferResponse) *TransferResponse {
	return &TransferResponse{
		Base:      newBase(ctx, TransferResponses),
		responses: response,
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

func (t *TransferResponse) Proto() types.TransferResponses {
	return types.TransferResponses{
		Responses: t.responses,
	}
}

func (t TransferResponse) StreamMessage() *types.BusEvent {
	p := t.Proto()
	return &types.BusEvent{
		Id:    t.eventID(),
		Block: t.TraceID(),
		Type:  t.et.ToProto(),
		Event: &types.BusEvent_TransferResponses{
			TransferResponses: &p,
		},
	}
}
