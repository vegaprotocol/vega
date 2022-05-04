package execution

import (
	"fmt"

	"code.vegaprotocol.io/vega/types/num"
)

// lp holds LiquidityProvider stake and avg values.
type lp struct {
	stake  num.Decimal
	share  num.Decimal
	avg    num.Decimal
	vStake num.Decimal //@TODO add this
}

// EquityShares module controls the Equity sharing algorithm described on the spec:
// https://github.com/vegaprotocol/product/blob/02af55e048a92a204e9ee7b7ae6b4475a198c7ff/specs/0042-setting-fees-and-rewarding-lps.md#calculating-liquidity-provider-equity-like-share
type EquityShares struct {
	// mvp is the MarketValueProxy
	mvp num.Decimal
	r   num.Decimal // market growth @TODO add this to snapshots

	// lps is a map of party id to lp (LiquidityProviders)
	lps map[string]*lp

	openingAuctionEnded bool

	stateChanged bool
}

func NewEquityShares(mvp num.Decimal) *EquityShares {
	return &EquityShares{
		mvp:          mvp,
		r:            num.DecimalZero(),
		lps:          map[string]*lp{},
		stateChanged: true,
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
	es.stateChanged = true
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

func (es *EquityShares) UpdateVirtualStake() {
	growth := num.NewDecimalFromFloat(1.0).Mul(es.r)
	for _, v := range es.lps {
		v.vStake = num.MaxD(v.stake, growth.Mul(v.vStake))
	}
	es.stateChanged = true
}

func (es *EquityShares) WithMVP(mvp num.Decimal) *EquityShares {
	es.r = mvp.Sub(es.mvp).Div(es.mvp)
	es.mvp = mvp
	es.stateChanged = true
	return es
}

// SetPartyStake sets LP values for a given party.
func (es *EquityShares) SetPartyStake(id string, newStakeU *num.Uint) {
	if !es.openingAuctionEnded {
		defer es.setOpeningAuctionAVG()
	}

	v, found := es.lps[id]
	if newStakeU == nil || newStakeU.IsZero() {
		delete(es.lps, id)
		es.stateChanged = (es.stateChanged || found)
		return
	}
	defer func() {
		es.stateChanged = true
	}()
	newStake := num.DecimalFromUint(newStakeU)
	// first time we set the newStake and mvp as avg.
	if !found {
		avg := num.DecimalZero()
		if es.openingAuctionEnded {
			avg = es.mvp
		}
		es.lps[id] = &lp{
			stake:  newStake,
			avg:    avg,
			vStake: newStake,
		}
		return
	}

	delta := newStake.Sub(v.stake)
	if newStake.LessThanOrEqual(v.stake) {
		// vStake * (newStake/oldStake)
		v.vStake = v.vStake.Mul(newStake.Div(v.stake))
		v.stake = newStake
		return
	}

	// delta will allways be > 0 at this point
	if es.openingAuctionEnded {
		eq := es.mustEquity(id)
		// v.avg = ((eq * v.avg) + (delta * es.mvp)) / (eq + v.stake)
		v.avg = (eq.Mul(v.avg).Add(delta.Mul(es.mvp))).Div(eq.Add(v.stake))
	}
	v.vStake = v.vStake.Add(delta) // increase
	v.stake = newStake
}

// AvgEntryValuation returns the Average Entry Valuation for a given party.
func (es *EquityShares) AvgEntryValuation(id string) num.Decimal {
	if v, ok := es.lps[id]; ok {
		es.stateChanged = true
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
		return (v.vStake.Mul(es.mvp)).Div(v.avg), nil
	}

	return num.DecimalZero(), fmt.Errorf("party %s has no stake", id)
}

// AllShares returns the ratio of equity for each party on the market.
func (es *EquityShares) AllShares() map[string]num.Decimal {
	return es.SharesExcept(map[string]struct{}{})
}

// SharesFromParty returns the equity-like shares of a given party on the market.
func (es *EquityShares) SharesFromParty(party string) num.Decimal {
	totalEquity := num.DecimalZero()
	partyELS := num.DecimalZero()
	for id := range es.lps {
		eq, err := es.equity(id)
		if err != nil {
			// since equity(id) returns an error when the party does not exist
			// getting an error here means we are doing something wrong cause
			// it should never happen unless `.equity()` behavior changes.
			panic(err)
		}
		if id == party {
			partyELS = eq
		}
		totalEquity = totalEquity.Add(eq)
	}

	if partyELS.Equal(num.DecimalZero()) || totalEquity.Equal(num.DecimalZero()) {
		return num.DecimalZero()
	}

	return partyELS.Div(totalEquity)
}

// SharesExcept returns the ratio of equity for each party on the market, except
// the ones listed in parameter.
func (es *EquityShares) SharesExcept(except map[string]struct{}) map[string]num.Decimal {
	// Calculate the equity for each party and the totalEquity (the sum of all
	// the equities)
	var totalEquity num.Decimal
	shares := map[string]num.Decimal{}
	for id := range es.lps {
		// If the party is not one of the deployed parties, we just skip.
		if _, ok := except[id]; ok {
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

	if len(shares) > 0 {
		es.stateChanged = true
	}

	return shares
}
