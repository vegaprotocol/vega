// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/types/num"
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

	totalVStake num.Decimal // Not needed in snapshot, we can reconstruct this from the data in snapshot already
	totalPStake num.Decimal // same as total VStake -> not included in snapshot

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
		totalVStake:  num.DecimalZero(),
		totalPStake:  num.DecimalZero(),
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
	if !es.openingAuctionEnded {
		defer es.setOpeningAuctionAVG()
	}

	v, found := es.lps[id]
	if newStakeU == nil || newStakeU.IsZero() {
		if found {
			// remove vStake from total
			es.totalVStake = es.totalVStake.Sub(v.vStake)
			es.totalPStake = es.totalPStake.Sub(v.stake)
		}
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
		es.totalVStake = es.totalVStake.Add(newStake)
		es.totalPStake = es.totalPStake.Add(newStake)
		return
	}

	delta := newStake.Sub(v.stake)
	if newStake.LessThanOrEqual(v.stake) {
		// vStake * (newStake/oldStake)
		// remove old value from totals
		es.totalVStake = es.totalVStake.Sub(v.vStake)
		es.totalPStake = es.totalPStake.Sub(v.stake)
		v.vStake = v.vStake.Mul(newStake.Div(v.stake))
		// add new values back to totals
		es.totalVStake = es.totalVStake.Add(v.vStake)
		es.totalPStake = es.totalPStake.Add(newStake)
		v.stake = newStake
		return
	}

	// delta will allways be > 0 at this point
	if es.openingAuctionEnded {
		eq := es.mustEquity(id)
		v.share = eq
		// v.avg = ((eq * v.avg) + (delta * es.mvp)) / (eq + v.stake)
		v.avg = (eq.Mul(v.avg).Add(delta.Mul(es.mvp))).Div(eq.Add(v.stake))
	}
	// update totals
	es.totalVStake = es.totalVStake.Add(delta)
	es.totalPStake = es.totalPStake.Sub(v.stake).Add(newStake)
	v.vStake = v.vStake.Add(delta) // increase
	v.stake = newStake
}

// AvgEntryValuation returns the Average Entry Valuation for a given party.
func (es *EquityShares) AvgEntryValuation(id string) num.Decimal {
	if v, ok := es.lps[id]; ok {
		// calculate average
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
		// avoid division by zero, if total vStake is zero, then v.vStake has to be zero
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
	eq, err := es.equity(party)
	if err != nil {
		panic(err)
	}
	return eq
}

// SharesExcept returns the ratio of equity for each party on the market, except
// the ones listed in parameter.
func (es *EquityShares) SharesExcept(except map[string]struct{}) map[string]num.Decimal {
	shares := make(map[string]num.Decimal, len(es.lps)-len(except))
	for id, v := range es.lps {
		if _, ok := except[id]; ok {
			continue
		}
		eq, err := es.equity(id)
		if err != nil {
			panic(err)
		}
		shares[id] = eq
		if !v.share.Equals(eq) {
			v.share = eq
			es.stateChanged = true
		}
	}
	return shares
}
