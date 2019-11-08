package buffer

import types "code.vegaprotocol.io/vega/proto"

// TransferResponseStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_response_store_mock.go -package mocks code.vegaprotocol.io/vega/buffer TransferResponseStore
type TransferResponseStore interface {
	SaveBatch([]*types.TransferResponse) error
}

// TransferResponse is a buffer for all transfer responses
// produced during a block by vega
type TransferResponse struct {
	store TransferResponseStore
	trs   []*types.TransferResponse
}

// NewTransferResponse instanciate a new buffer
func NewTransferResponse(store TransferResponseStore) *TransferResponse {
	return &TransferResponse{
		store: store,
		trs:   []*types.TransferResponse{},
	}
}

// Add adds a slice of transfer responses to the buffer
func (t *TransferResponse) Add(trs []*types.TransferResponse) {
	t.trs = append(t.trs, trs...)
}

// Flush will save all the buffered element into the stores
func (t *TransferResponse) Flush() error {
	trs := t.trs
	t.trs = []*types.TransferResponse{}
	return t.store.SaveBatch(trs)
}
