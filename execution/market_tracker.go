package execution

import (
	"sort"

	"code.vegaprotocol.io/vega/types/num"
)

type marketTracker struct {
	volumeTraded  *num.Uint
	proposersPaid bool
	proposer      string
}

type EligibilityChecker interface {
	IsEligibleForProposerBonus(marketID string, volumeTraded *num.Uint) bool
}

type MarketTracker struct {
	marketIDMarketTracker map[string]*marketTracker
	eligibilityChecker    EligibilityChecker
	ss                    *marketTrackerSnapshotState
}

func (m *MarketTracker) SetEligibilityChecker(eligibilityChecker EligibilityChecker) {
	m.eligibilityChecker = eligibilityChecker
}

func NewMarketTracker() *MarketTracker {
	return &MarketTracker{
		marketIDMarketTracker: map[string]*marketTracker{},
		ss:                    &marketTrackerSnapshotState{changed: true},
	}
}

func (m *MarketTracker) MarketProposed(marketID, proposer string) {
	// if we already know about this market don't re-add it
	if _, ok := m.marketIDMarketTracker[marketID]; ok {
		return
	}
	m.marketIDMarketTracker[marketID] = &marketTracker{
		proposer:      proposer,
		proposersPaid: false,
		volumeTraded:  num.Zero(),
	}
	m.ss.changed = true
}

func (m *MarketTracker) AddValueTraded(marketID string, value *num.Uint) {
	if _, ok := m.marketIDMarketTracker[marketID]; !ok {
		return
	}
	m.marketIDMarketTracker[marketID].volumeTraded.AddSum(value)
	m.ss.changed = true
}

func (m *MarketTracker) GetAndResetEligibleProposers(market string) []string {
	if _, ok := m.marketIDMarketTracker[market]; !ok {
		return []string{}
	}
	t := m.marketIDMarketTracker[market]
	if !t.proposersPaid && m.eligibilityChecker.IsEligibleForProposerBonus(market, t.volumeTraded) {
		t.proposersPaid = true
		m.ss.changed = true
		return []string{t.proposer}
	}
	return []string{}
}

func (m *MarketTracker) GetAllMarketIDs() []string {
	mIDs := make([]string, 0, len(m.marketIDMarketTracker))
	for k := range m.marketIDMarketTracker {
		mIDs = append(mIDs, k)
	}

	sort.Strings(mIDs)
	return mIDs
}

func (m *MarketTracker) removeMarket(marketID string) {
	delete(m.marketIDMarketTracker, marketID)
	m.ss.changed = true
}
