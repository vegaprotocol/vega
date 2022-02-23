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

func (m *MarketTracker) GetAndResetEligibleProposers() []string {
	eligibleProposers := []string{}
	for ID, t := range m.marketIDMarketTracker {
		if !t.proposersPaid && m.eligibilityChecker.IsEligibleForProposerBonus(ID, t.volumeTraded) {
			eligibleProposers = append(eligibleProposers, t.proposer)
			t.proposersPaid = true
			m.ss.changed = true
		}
	}
	sort.Strings(eligibleProposers)
	return eligibleProposers
}

func (m *MarketTracker) removeMarket(marketID string) {
	delete(m.marketIDMarketTracker, marketID)
	m.ss.changed = true
}
