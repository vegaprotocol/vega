package buffer

import types "code.vegaprotocol.io/vega/proto"

type TradeStore interface {
	SaveBatch(order []types.Trade) error
}

type Trade struct {
	store TradeStore
	buf   []types.Trade
}

func NewTrade(store TradeStore) *Trade {
	return &Trade{
		store: store,
		buf:   []types.Trade{},
	}
}

func (o *Trade) Add(ord types.Trade) {
	o.buf = append(o.buf, ord)
}

func (o *Trade) Flush() error {
	copyBuf := o.buf
	o.buf = make([]types.Trade, 0, len(copyBuf))
	return o.store.SaveBatch(copyBuf)
}
