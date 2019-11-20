package buffer

import types "code.vegaprotocol.io/vega/proto"

type OrderStore interface {
	SaveBatch(order []types.Order) error
}

type Order struct {
	store OrderStore
	buf   []types.Order
}

func NewOrder(store OrderStore) *Order {
	return &Order{
		store: store,
		buf:   []types.Order{},
	}
}

func (o *Order) Add(ord types.Order) {
	o.buf = append(o.buf, ord)
}

func (o *Order) Flush() error {
	copyBuf := o.buf
	o.buf = make([]types.Order, 0, len(copyBuf))
	return o.store.SaveBatch(copyBuf)
}
