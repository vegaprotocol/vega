package execution

import (
	"fmt"
	"strconv"

	types "code.vegaprotocol.io/vega/proto"
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
		es.lps[id] = &lp{stake: newStake, avg: es.mvp}
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
	eq, err := es.Equity(party)
	if err != nil {
		panic(err)
	}
	return eq
}

// Equity returns the following:
// (LP i stake)(n) x market_value_proxy(n) / (LP i avg_entry_valuation)(n)
// given a party id (i).
//
// Returns an error if the party has no stake.
func (es *EquityShares) Equity(id string) (float64, error) {
	if v, ok := es.lps[id]; ok {
		return (v.stake * es.mvp) / v.avg, nil
	}

	return 0, fmt.Errorf("party %s has no stake", id)
}

// Shares returns the ratio of equity for a given party
func (es *EquityShares) Shares(party string) (float64, error) {
	var totalEquity float64
	for id := range es.lps {
		eq, err := es.Equity(id)
		if err != nil {
			return 0, err
		}
		totalEquity += eq
	}

	eq, err := es.Equity(party)
	if err != nil {
		return 0, err
	}

	es.lps[party].share = eq / totalEquity
	return es.lps[party].share, nil
}

func (es *EquityShares) ToLiquidityProviderFeeShare() []*types.LiquidityProviderFeeShare {
	out := make([]*types.LiquidityProviderFeeShare, 0, len(es.lps))
	for k, v := range es.lps {
		out = append(out, &types.LiquidityProviderFeeShare{
			Party:                 k,
			EquityLikeShare:       strconv.FormatFloat(v.share, 'f', -1, 64),
			AverageEntryValuation: strconv.FormatFloat(v.avg, 'f', -1, 64),
		})
	}
	return out
}
