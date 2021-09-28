package execution

import (
	"fmt"

	"code.vegaprotocol.io/vega/types/num"
)

// lp holds LiquidityProvider stake and avg values
type lp struct {
	stake num.Decimal
	share num.Decimal
	avg   num.Decimal
}

// EquityShares module controls the Equity sharing algorithm described on the spec:
// https://github.com/vegaprotocol/product/blob/02af55e048a92a204e9ee7b7ae6b4475a198c7ff/specs/0042-setting-fees-and-rewarding-lps.md#calculating-liquidity-provider-equity-like-share
type EquityShares struct {
	// mvp is the MarketValueProxy
	mvp num.Decimal

	// lps is a map of party id to lp (LiquidityProviders)
	lps map[string]*lp

	openingAuctionEnded bool
}

func NewEquityShares(mvp num.Decimal) *EquityShares {
	return &EquityShares{
		mvp: mvp,
		lps: map[string]*lp{},
	}
}

// OpeningAuctionEnded signal to the EquityShare that the
// opening auction has ended.
func (es *EquityShares) OpeningAuctionEnded() {
	// we should never call this twice
	if es.openingAuctionEnded {
		panic("market already left opening auction")
	}
	es.openingAuctionEnded = true
	es.setOpeningAuctionAVG()
}

// we just the average entry valuation to the same value
// for every LP during opening auction.
func (es *EquityShares) setOpeningAuctionAVG() {
	// now set average entry valuation for all of them.
	for _, v := range es.lps {
		v.avg = es.mvp
	}
}

func (es *EquityShares) WithMVP(mvp num.Decimal) *EquityShares {
	es.mvp = mvp
	return es
}

// SetPartyStake sets LP values for a given party.
func (es *EquityShares) SetPartyStake(id string, newStakeU *num.Uint) {
	if !es.openingAuctionEnded {
		defer es.setOpeningAuctionAVG()
	}

	newStake := num.DecimalFromUint(newStakeU)
	v, found := es.lps[id]
	// first time we set the newStake and mvp as avg.
	if !found {
		if avg := num.DecimalZero(); newStake.GreaterThan(avg) {
			if es.openingAuctionEnded {
				avg = es.mvp
			}
			es.lps[id] = &lp{stake: newStake, avg: avg}
			return
		}
		// If we didn't previously have stake and are trying to set it to zero, just return
		return
	}

	if newStake.IsZero() {
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
	if es.openingAuctionEnded {
		v.avg = (eq.Mul(v.avg).Add(delta.Mul(es.mvp))).Div(eq.Add(v.stake))
	}
	v.stake = newStake
}

// AvgEntryValuation returns the Average Entry Valuation for a given party.
func (es *EquityShares) AvgEntryValuation(id string) num.Decimal {
	if v, ok := es.lps[id]; ok {
		return v.avg
	}
	return num.DecimalZero()
}

func (es *EquityShares) mustEquity(party string) num.Decimal {
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
func (es *EquityShares) equity(id string) (num.Decimal, error) {
	if v, ok := es.lps[id]; ok {
		return (v.stake.Mul(es.mvp)).Div(v.avg), nil
	}

	return num.DecimalZero(), fmt.Errorf("party %s has no stake", id)
}

// Shares returns the ratio of equity for a given party
func (es *EquityShares) Shares(undeployed map[string]struct{}) map[string]num.Decimal {
	// Calculate the equity for each party and the totalEquity (the sum of all
	// the equities)
	var totalEquity num.Decimal
	shares := map[string]num.Decimal{}
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
