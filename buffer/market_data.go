package buffer

import (
	"code.vegaprotocol.io/vega/proto"
)

type MarketDataPlugin interface {
	SaveBatch([]proto.MarketData)
}

type MarketData struct {
	plugins []MarketDataPlugin
	mds     []proto.MarketData
}

func NewMarketData() *MarketData {
	return &MarketData{
		plugins: []MarketDataPlugin{},
		mds:     []proto.MarketData{},
	}
}

func (m *MarketData) Register(plug MarketDataPlugin) {
	m.plugins = append(m.plugins, plug)
}

func (m *MarketData) Add(mp proto.MarketData) {
	m.mds = append(m.mds, mp)
}

func (m *MarketData) Flush() {
	mds := m.mds
	m.mds = m.mds[:0]
	for _, v := range m.plugins {
		v.SaveBatch(mds)
	}
}
