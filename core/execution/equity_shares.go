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
	mvp num.Decimal
	r   num.Decimal

	totalVStake num.Decimal
	totalPStake num.Decimal
	// lps is a map of party id to lp (LiquidityProviders)
	lps map[string]*lp

	openingAuctionEnded bool
}

func NewEquityShares(mvp num.Decimal) *EquityShares {
	return &EquityShares{
		mvp:         mvp,
		r:           num.DecimalZero(),
		totalPStake: num.DecimalZero(),
		totalVStake: num.DecimalZero(),
		lps:         map[string]*lp{},
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
	es.r = num.DecimalZero()
}

func (es *EquityShares) UpdateVStake() {
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

func (es *EquityShares) GetTotalVStake() num.Decimal {
	return es.totalVStake
}

func (es *EquityShares) AvgTradeValue(avg num.Decimal) *EquityShares {
	if avg.IsZero() {
		// if es.openingAuctionEnded {
		// 	// this should not be possible IRL, however unit tests like amend_lp_orders
		// 	// rely on the EndOpeningAuction call and can end opening auction without setting a price
		// 	// ie -> end opening auction without a trade value
		// 	panic("opening auction ended, and avg trade value hit zero somehow?")
		// }
		return es
	}
	if !es.mvp.IsZero() {
		es.r = avg.Sub(es.mvp).Div(es.mvp)
	} else {
		es.r = num.DecimalZero()
	}
	es.UpdateVStake()
	es.mvp = avg
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
		return
	}
	newStake := num.DecimalFromUint(newStakeU)
	if found && newStake.Equals(v.stake) {
		// stake didn't change? there's nothing to do really
		return
	}
	// first time we set the newStake and mvp as avg.
	if !found {
		// this is technically a delta == new stake -> calculate avg accordingly
		v = &lp{}
		es.lps[id] = v
		es.updateAvgPosDelta(v, newStake, newStake) // delta == new stake
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
		return
	}

	es.updateAvgPosDelta(v, delta, newStake)
}

func (es *EquityShares) updateAvgPosDelta(v *lp, delta, newStake num.Decimal) {
	// entry valuation == total Virtual stake (before delta is applied)
	// (average entry valuation) <- (average entry valuation) x S / (S + Delta S) + (entry valuation) x (Delta S) / (S + Delta S)
	// S being the LP's physical stake, Delta S being the amount by which the stake is increased
	es.totalVStake = es.totalVStake.Add(delta)
	es.totalPStake = es.totalPStake.Sub(v.stake).Add(newStake)
	v.avg = v.avg.Mul(v.vStake).Div(newStake).Add(es.totalVStake.Mul(delta).Div(newStake))
	// this is the first LP -> no vStake yet
	v.vStake = v.vStake.Add(delta)
	v.stake = newStake
}

// AvgEntryValuation returns the Average Entry Valuation for a given party.
func (es *EquityShares) AvgEntryValuation(id string) num.Decimal {
	if v, ok := es.lps[id]; ok {
		return v.avg
	}
	return num.DecimalZero()
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
		}
	}
	if !all {
		es.totalPStake = allTotal
		es.totalVStake = allTotalV
	}
	return shares
}
