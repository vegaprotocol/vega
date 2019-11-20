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

func (m *Market) Add(ord types.Market) {
	m.buf = append(m.buf, ord)
}

func (m *Market) Flush() error {
	copyBuf := m.buf
	m.buf = make([]types.Market, 0, len(copyBuf))
	return m.store.SaveBatch(copyBuf)
}
