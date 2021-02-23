package execution

import (
	"fmt"
)

// lp holds LiquidityProvider stake and avg values
type lp struct {
	stake float64
	share float64
	avg   float64
}

// EquityShares module controls the Equity sharing algorithm described on the spec:
// https://github.com/vegaprotocol/product/blob/02af55e048a92a204e9ee7b7ae6b4475a198c7ff/specs/0042-setting-fees-and-rewarding-lps.md#calculating-liquidity-provider-equity-like-share
type EquityShares struct {
	// mvp is the MarketValueProxy
	mvp float64

	// lps is a map of party id to lp (LiquidityProviders)
	lps map[string]*lp
}

func NewEquityShares(mvp float64) *EquityShares {
	return &EquityShares{
		mvp: mvp,
		lps: map[string]*lp{},
	}
}

func (es *EquityShares) WithMVP(mvp float64) *EquityShares {
	es.mvp = mvp
	return es
}

// SetPartyStake sets LP values for a given party.
func (es *EquityShares) SetPartyStake(id string, newStake float64) {
	v, found := es.lps[id]
	// first time we set the newStake and mvp as avg.
	if !found {
		if newStake > 0 {
			es.lps[id] = &lp{stake: newStake, avg: es.mvp}
			return
		}
		// If we didn't previously have stake and are trying to set it to zero, just return
		return
	}

	if newStake <= 0 {
		// We are removing an existing stake
		delete(es.lps, id)
		return
	}

	if newStake <= v.stake {
		v.stake = newStake
		return
	}

	// delta will allways be > 0 at this point
	delta := newStake - v.stake
	eq := es.mustEquity(id)
	v.avg = ((eq * v.avg) + (delta * es.mvp)) / (eq + v.stake)
	v.stake = newStake
}

// AvgEntryValuation returns the Average Entry Valuation for a given party.
func (es *EquityShares) AvgEntryValuation(id string) float64 {
	if v, ok := es.lps[id]; ok {
		return v.avg
	}
	return 0
}

func (es *EquityShares) mustEquity(party string) float64 {
	eq, err := es.equity(party)
	if err != nil {
		panic(err)
	}
	return eq
}

// equity returns the following:
// (LP i stake)(n) x market_value_proxy(n) / (LP i avg_entry_valuation)(n)
// given a party id (i).
//
// Returns an error if the party has no stake.
func (es *EquityShares) equity(id string) (float64, error) {
	if v, ok := es.lps[id]; ok {
		return (v.stake * es.mvp) / v.avg, nil
	}

	return 0, fmt.Errorf("party %s has no stake", id)
}

// Shares returns the ratio of equity for a given party
func (es *EquityShares) Shares() map[string]float64 {
	// Calculate the equity for each party and the totalEquity (the sum of all
	// the equities)
	var totalEquity float64
	shares := map[string]float64{}
	for id := range es.lps {
		eq, err := es.equity(id)
		if err != nil {
			// since equity(id) returns an error when the party does not exist
			// getting an error here means we are doing something wrong cause
			// it should never happen unless `.equity()` behavior changes.
			panic(err)
		}
		shares[id] = eq
		totalEquity += eq
	}

	for id, eq := range shares {
		shares[id] = eq / totalEquity
	}

	return shares
}
