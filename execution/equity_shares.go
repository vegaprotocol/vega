package execution

import (
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

// lp holds LiquidityProvider stake and avg values
type lp struct {
	stake decimal.Decimal
	share decimal.Decimal
	avg   decimal.Decimal
}

// EquityShares module controls the Equity sharing algorithm described on the spec:
// https://github.com/vegaprotocol/product/blob/02af55e048a92a204e9ee7b7ae6b4475a198c7ff/specs/0042-setting-fees-and-rewarding-lps.md#calculating-liquidity-provider-equity-like-share
type EquityShares struct {
	// mvp is the MarketValueProxy
	mvp decimal.Decimal

	// lps is a map of party id to lp (LiquidityProviders)
	lps map[string]*lp
}

func NewEquityShares(mvp decimal.Decimal) *EquityShares {
	return &EquityShares{
		mvp: mvp,
		lps: map[string]*lp{},
	}
}

func (es *EquityShares) WithMVP(mvp decimal.Decimal) *EquityShares {
	es.mvp = mvp
	return es
}

// SetPartyStake sets LP values for a given party.
func (es *EquityShares) SetPartyStake(id string, newStakeU64 uint64) {
	newStake := decimal.NewFromBigInt(new(big.Int).SetUint64(newStakeU64), 0)
	v, found := es.lps[id]
	// first time we set the newStake and mvp as avg.
	if !found {
		if newStake.GreaterThan(decimal.Zero) {
			// if marketValueProxy == 0
			// we assume mvp will be our stake?
			if es.mvp.Equal(decimal.Zero) {
				es.mvp = newStake
			}
			es.lps[id] = &lp{stake: newStake, avg: es.mvp}
			return
		}
		// If we didn't previously have stake and are trying to set it to zero, just return
		return
	}

	if newStake.Equal(decimal.Zero) {
		// We are removing an existing stake
		delete(es.lps, id)
		return
	}

	if newStake.LessThanOrEqual(v.stake) {
		v.stake = newStake
		return
	}

	// delta will allways be > 0 at this point
	delta := newStake.Sub(v.stake)
	eq := es.mustEquity(id)
	// v.avg = ((eq * v.avg) + (delta * es.mvp)) / (eq + v.stake)
	v.avg = (eq.Mul(v.avg).Add(delta.Mul(es.mvp))).Div(eq.Add(v.stake))
	v.stake = newStake
}

// AvgEntryValuation returns the Average Entry Valuation for a given party.
func (es *EquityShares) AvgEntryValuation(id string) decimal.Decimal {
	if v, ok := es.lps[id]; ok {
		return v.avg
	}
	return decimal.Zero
}

func (es *EquityShares) mustEquity(party string) decimal.Decimal {
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
func (es *EquityShares) equity(id string) (decimal.Decimal, error) {
	if v, ok := es.lps[id]; ok {
		return (v.stake.Mul(es.mvp)).Div(v.avg), nil
	}

	return decimal.Zero, fmt.Errorf("party %s has no stake", id)
}

// Shares returns the ratio of equity for a given party
func (es *EquityShares) Shares(undeployed map[string]struct{}) map[string]decimal.Decimal {
	// Calculate the equity for each party and the totalEquity (the sum of all
	// the equities)
	var totalEquity decimal.Decimal
	shares := map[string]decimal.Decimal{}
	for id := range es.lps {
		// if the party is not one of the deployed parties,
		// we just skip
		if _, ok := undeployed[id]; ok {
			continue
		}
		eq, err := es.equity(id)
		if err != nil {
			// since equity(id) returns an error when the party does not exist
			// getting an error here means we are doing something wrong cause
			// it should never happen unless `.equity()` behavior changes.
			panic(err)
		}
		shares[id] = eq
		totalEquity = totalEquity.Add(eq)
	}

	for id, eq := range shares {
		eqshare := eq.Div(totalEquity)
		shares[id] = eqshare
		es.lps[id].share = eqshare
	}

	return shares
}
