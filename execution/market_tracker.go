package execution

import (
	"code.vegaprotocol.io/vega/types/num"
)

type marketTracker struct {
	volumeTraded  *num.Uint
	proposersPaid bool
	proposer      string
}

type MarketTracker struct {
	marketIDMarketTracker map[string]*marketTracker
}

func NewMarketTracker() *MarketTracker {
	return &MarketTracker{
		marketIDMarketTracker: map[string]*marketTracker{},
	}
}

func (m *MarketTracker) MarketProposed(marketID, proposer string) {
	m.marketIDMarketTracker[marketID] = &marketTracker{
		proposer:      proposer,
		proposersPaid: false,
		volumeTraded:  num.Zero(),
	}
}

func (m *MarketTracker) AddValueTraded(marketID string, value *num.Uint) {
	if _, ok := m.marketIDMarketTracker[marketID]; !ok {
		return
	}
	m.marketIDMarketTracker[marketID].volumeTraded.AddSum(value)
}
