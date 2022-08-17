// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package execution

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
)

// lp holds LiquidityProvider stake and avg values.
type lp struct {
	stake  num.Decimal
	share  num.Decimal
	avg    num.Decimal
	vStake num.Decimal
}

// EquityShares module controls the Equity sharing algorithm described on the spec:
// https://github.com/vegaprotocol/product/blob/02af55e048a92a204e9ee7b7ae6b4475a198c7ff/specs/0042-setting-fees-and-rewarding-lps.md#calculating-liquidity-provider-equity-like-share
type EquityShares struct {
	// mvp is the MarketValueProxy
	mvp  num.Decimal
	pMvp num.Decimal // @TODO add to snapshot
	r    num.Decimal

	totalVStake num.Decimal
	totalPStake num.Decimal
	// lps is a map of party id to lp (LiquidityProviders)
	lps map[string]*lp

	openingAuctionEnded bool

	stateChanged bool
}

func NewEquityShares(mvp num.Decimal) *EquityShares {
	return &EquityShares{
		mvp:          mvp,
		pMvp:         num.DecimalZero(),
		r:            num.DecimalZero(),
		totalPStake:  num.DecimalZero(),
		totalVStake:  num.DecimalZero(),
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
	es.pMvp = num.DecimalZero()
	es.r = num.DecimalZero()
}

// we just the average entry valuation to the same value
// for every LP during opening auction.
func (es *EquityShares) setOpeningAuctionAVG() {
	// now set average entry valuation for all of them.
	factor := es.totalPStake.Div(es.totalVStake)
	for _, v := range es.lps {
		if v.stake.GreaterThan(v.vStake) {
			v.vStake = v.stake
		}
		v.avg = v.vStake.Mul(factor) // perhaps we ought to move this to a separate loop once the totals have all been updated
		// v.avg = es.mvp
	}
	es.stateChanged = true
}

func (es *EquityShares) UpdateVStake() {
	es.stateChanged = true
	if es.r.IsZero() {
		for _, v := range es.lps {
			v.vStake = v.stake
		}
		es.totalVStake = es.totalPStake
		return
	}
	total := num.DecimalZero()
	factor := num.DecimalFromFloat(1.0).Add(es.r)
	for _, v := range es.lps {
		vStake := num.MaxD(v.stake, v.vStake.Mul(factor))
		v.vStake = vStake
		total = total.Add(vStake)
	}
	es.totalVStake = total
}

func (es *EquityShares) UpdateVirtualStake() {
	defer func() {
		es.stateChanged = true
		es.recalcAverages()
	}()
	if es.mvp.IsZero() || es.pMvp.IsZero() {
		for _, v := range es.lps {
			es.totalVStake = es.totalVStake.Sub(v.vStake).Add(v.stake)
			v.vStake = v.stake
		}
		return
	}
	growth := es.r.Add(num.NewDecimalFromFloat(1.0))
	for _, v := range es.lps {
		vStake := num.MaxD(v.stake, growth.Mul(v.vStake))
		es.totalVStake = es.totalVStake.Sub(v.vStake).Add(vStake)
		v.vStake = vStake
	}
	// now that the total Virtual stake has been corrected, update the averages
}

func (es *EquityShares) recalcAverages() {
	factor := es.totalPStake.Div(es.totalVStake)
	for _, v := range es.lps {
		v.avg = v.vStake.Mul(factor)
	}
}

func (es *EquityShares) UpdateVirtualStakeOld() {
	// this isn't used if we have to set vStake to physical stake
	growth := es.r
	// if A(n) == 0 or A(n-1) == 0, vStake = physical stake
	setPhysical := (es.mvp.IsZero() || es.pMvp.IsZero())
	if !setPhysical {
		growth = num.NewDecimalFromFloat(1.0).Add(growth)
	}
	for _, v := range es.lps {
		// default to physical stake
		vStake := v.stake
		if !setPhysical {
			// unless A(n) != 0 || A(n-1) != 0
			// then set vStake = max(physical stake, ((r+1)*vStqake))
			vStake = num.MaxD(v.stake, growth.Mul(v.vStake))
		}
		// if virtual stake doesn't change, then stateChanged shouldn't be toggled
		es.stateChanged = (es.stateChanged || !vStake.Equals(v.vStake))
		v.vStake = vStake
	}
}

func (es *EquityShares) AvgTradeValue(avg num.Decimal) *EquityShares {
	if !es.mvp.IsZero() && !avg.IsZero() {
		growth := avg.Sub(es.mvp).Div(es.mvp)
		es.stateChanged = (es.stateChanged || !growth.Equals(es.r))
		es.r = growth
	} else {
		es.r = num.DecimalZero()
	}
	es.stateChanged = true
	es.pMvp = es.mvp
	es.mvp = avg
	return es
}

func (es *EquityShares) WithMVP(mvp num.Decimal) *EquityShares {
	// growth always defaults to 0
	es.r = num.DecimalZero()
	if !es.mvp.IsZero() && !mvp.IsZero() {
		// Spec notation: r = (A(n) - A(n-1))/A(n-1)
		growth := mvp.Sub(es.mvp).Div(es.mvp)
		// toggle state changed if growth rate has changed
		es.stateChanged = (es.stateChanged || !growth.Equals(es.r))
		es.r = growth
	}
	// only flip state changed if growth rate and/or mvp has changed
	// previous mvp can still change, so we need to check that, too
	es.stateChanged = (es.stateChanged || !es.mvp.Equals(mvp) || !es.mvp.Equals(es.pMvp))
	// pMvp would otherwise be A(n-2) -> update to A(n-1)
	es.pMvp = es.mvp
	es.mvp = mvp
	return es
}

// SetPartyStake sets LP values for a given party.
func (es *EquityShares) SetPartyStake(id string, newStakeU *num.Uint) {
	v, found := es.lps[id]
	if newStakeU == nil || newStakeU.IsZero() {
		if found {
			es.totalVStake = es.totalVStake.Sub(v.vStake)
			es.totalPStake = es.totalPStake.Sub(v.stake)
		}
		delete(es.lps, id)
		es.stateChanged = (es.stateChanged || found)
		return
	}
	newStake := num.DecimalFromUint(newStakeU)
	if found && newStake.Equals(v.stake) {
		// stake didn't change? there's nothing to do really
		return
	}
	defer func() {
		es.stateChanged = true
	}()
	// first time we set the newStake and mvp as avg.
	if !found {
		avg := es.mvp
		es.lps[id] = &lp{
			stake:  newStake,
			avg:    avg,
			vStake: newStake,
		}
		es.totalPStake = es.totalPStake.Add(newStake)
		es.totalVStake = es.totalVStake.Add(newStake)
		if es.openingAuctionEnded {
			es.lps[id].avg = es.AvgEntryValuation(id)
		}
		return
	}

	delta := newStake.Sub(v.stake)
	if newStake.LessThan(v.stake) {
		// commitment decreased
		es.totalVStake = es.totalVStake.Sub(v.vStake)
		es.totalPStake = es.totalPStake.Sub(v.stake)
		// vStake * (newStake/oldStake) or vStake * (old_stake + (-delta))/old_stake
		// e.g. 8000 => 5000 == vStakle * ((8000 + (-3000))/8000) == vStake * (5000/8000)
		v.vStake = v.vStake.Mul(newStake.Div(v.stake))
		v.stake = newStake
		es.totalVStake = es.totalVStake.Add(v.vStake)
		es.totalPStake = es.totalPStake.Add(v.stake)
		v.avg = v.vStake.Mul(es.totalPStake.Div(es.totalVStake))
		return
	}

	es.totalVStake = es.totalVStake.Add(delta)
	es.totalPStake = es.totalPStake.Sub(v.stake).Add(newStake)
	// stakes were originally assigned _after_ the mustEquity call
	// average was calculated in the if es.openingAuctionEnded bit
	// v.avg = v.vStake.Mul(es.totalPStake.Div(es.totalVStake))
	v.vStake = v.vStake.Add(delta) // increase
	v.stake = newStake
	v.avg = v.vStake.Mul(es.totalPStake.Div(es.totalVStake))
}

// AvgEntryValuation returns the Average Entry Valuation for a given party.
func (es *EquityShares) AvgEntryValuation(id string) num.Decimal {
	if v, ok := es.lps[id]; ok {
		avg := v.vStake.Mul(es.totalPStake.Div(es.totalVStake))
		if !avg.Equal(v.avg) {
			v.avg = avg
			es.stateChanged = true
		}
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
		if es.totalVStake.IsZero() {
			return num.DecimalZero(), nil
		}
		return v.vStake.Div(es.totalVStake), nil
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
	shares := make(map[string]num.Decimal, len(es.lps)-len(except))
	allTotal, allTotalV := es.totalPStake, es.totalVStake
	all := true
	for k := range except {
		if v, ok := es.lps[k]; ok {
			es.totalPStake = es.totalPStake.Sub(v.stake)
			es.totalVStake = es.totalVStake.Sub(v.vStake)
			all = false
		}
	}
	for id, v := range es.lps {
		if _, ok := except[id]; ok {
			continue
		}
		eq, err := es.equity(id)
		if err != nil {
			panic(err)
		}
		shares[id] = eq
		if all && !v.share.Equals(eq) {
			v.share = eq
			es.stateChanged = true
		}
	}
	if !all {
		es.totalPStake = allTotal
		es.totalVStake = allTotalV
	}
	return shares
}
