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
