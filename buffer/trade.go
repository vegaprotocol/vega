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

func (t *Trade) Add(ord types.Trade) {
	t.buf = append(t.buf, ord)
}

func (t *Trade) Flush() error {
	copyBuf := t.buf
	t.buf = make([]types.Trade, 0, len(copyBuf))
	return t.store.SaveBatch(copyBuf)
}
