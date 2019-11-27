package buffer

import (
	"code.vegaprotocol.io/vega/proto"
)

type MarginLevelsPlugin interface {
	SaveMarginLevelsBatch([]proto.MarginLevels)
}

type MarginLevels struct {
	plugins []MarginLevelsPlugin
	mls     map[string]map[string]proto.MarginLevels
}

func NewMarginLevels() *MarginLevels {
	return &MarginLevels{
		plugins: []MarginLevelsPlugin{},
		mls:     map[string]map[string]proto.MarginLevels{},
	}
}

func (m *MarginLevels) Register(plug MarginLevelsPlugin) {
	m.plugins = append(m.plugins, plug)
}

func (m *MarginLevels) Add(ml proto.MarginLevels) {
	if _, ok := m.mls[ml.PartyID]; !ok {
		m.mls[ml.PartyID] = map[string]proto.MarginLevels{}
	}
	m.mls[ml.PartyID][ml.MarketID] = ml
}

func (m *MarginLevels) Flush() {
	out := make([]proto.MarginLevels, 0, len(m.mls))
	for _, markets := range m.mls {
		for _, margin := range markets {
			out = append(out, margin)
		}
	}
	m.mls = map[string]map[string]proto.MarginLevels{}
	for _, v := range m.plugins {
		v.SaveMarginLevelsBatch(out)
	}
}
