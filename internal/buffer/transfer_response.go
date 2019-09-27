package buffer

import types "code.vegaprotocol.io/vega/proto"

//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_response_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/buffer TransferResponseStore
type TransferResponseStore interface {
	SaveBatch([]*types.TransferResponse) error
}

type TransferResponse struct {
	store TransferResponseStore
	trs   []*types.TransferResponse
}

func NewTransferResponse(store TransferResponseStore) *TransferResponse {
	return &TransferResponse{
		store: store,
		trs:   []*types.TransferResponse{},
	}
}

func (t *TransferResponse) Add(trs []*types.TransferResponse) {
	t.trs = append(t.trs, trs...)
}

func (t *TransferResponse) Flush() error {
	trs := t.trs
	t.trs = []*types.TransferResponse{}
	return t.store.SaveBatch(trs)
}
