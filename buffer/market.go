package buffer

import types "code.vegaprotocol.io/vega/proto"

type MarketStore interface {
	SaveBatch(order []types.Market) error
}

type Market struct {
	store MarketStore
	buf   []types.Market
}

func NewMarket(store MarketStore) *Market {
	return &Market{
		store: store,
		buf:   []types.Market{},
	}
}

func (o *Market) Add(ord types.Market) {
	o.buf = append(o.buf, ord)
}

func (o *Market) Flush() error {
	copyBuf := o.buf
	o.buf = make([]types.Market, 0, len(copyBuf))
	return o.store.SaveBatch(copyBuf)
}
