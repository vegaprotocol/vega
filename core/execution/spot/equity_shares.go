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

package spot

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
)

// lp holds LiquidityProvider stake and avg values.
type lp struct {
	// physical stake
	buyStake      num.Decimal
	sellStake     num.Decimal
	physicalStake num.Decimal
	// virtual stake
	buyVStake    num.Decimal
	sellVStake   num.Decimal
	virtualStake num.Decimal

	share num.Decimal
	avg   num.Decimal
}

type EquityShares struct {
	// market value proxy
	mvp num.Decimal
	// growth factor
	r num.Decimal

	totalVStake num.Decimal
	totalPStake num.Decimal
	lps         map[string]*lp

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

func (es *EquityShares) updateVStake(markPrice num.Decimal) {
	if es.r.IsZero() {
		return
	}
	total := num.DecimalZero()
	factor := num.DecimalOne().Add(es.r)
	recalc := false
	for _, v := range es.lps {
		v.buyVStake = num.MaxD(v.buyStake, v.buyVStake.Mul(factor))
		v.sellVStake = num.MaxD(v.sellStake, v.sellVStake.Mul(factor))
		v.virtualStake = num.MinD(v.buyVStake, v.sellVStake.Mul(markPrice))
		total = total.Add(v.virtualStake)
	}
	// some vStake changed, force recalc of ELS values.
	if !es.totalVStake.Equals(total) {
		recalc = true
	}
	es.totalVStake = total
	if recalc {
		es.updateAllELS()
	}
}

func (es *EquityShares) GetTotalVStake() num.Decimal {
	return es.totalVStake
}

func (es *EquityShares) GetTotalStake() num.Decimal {
	return es.totalPStake
}

func (es *EquityShares) AvgTradeValue(avg num.Decimal, markPrice num.Decimal) *EquityShares {
	if avg.IsZero() {
		return es
	}
	if !es.mvp.IsZero() {
		es.r = avg.Sub(es.mvp).Div(es.mvp)
	} else {
		es.r = num.DecimalZero()
	}
	es.updateVStake(markPrice)
	es.mvp = avg
	return es
}

// SetPartyStake sets LP values for a given party.
func (es *EquityShares) SetPartyStake(id string, newBuyStakeU *num.Uint, newSellStakeU *num.Uint, markPrice num.Decimal) {
	v, found := es.lps[id]
	if (newBuyStakeU == nil || newBuyStakeU.IsZero()) && (newSellStakeU == nil || newSellStakeU.IsZero()) {
		if found {
			es.totalVStake = es.totalVStake.Sub(v.virtualStake)
			es.totalPStake = es.totalPStake.Sub(v.physicalStake)
		}
		delete(es.lps, id)
		return
	}
	newBuyStake := num.DecimalZero()
	if newBuyStakeU != nil {
		newBuyStake = newBuyStakeU.ToDecimal()
	}
	newSellStake := num.DecimalZero()
	if newSellStakeU != nil {
		newSellStake = newSellStakeU.ToDecimal()
	}
	newStake := num.MinD(newBuyStake, newSellStake.Mul(markPrice))
	if found && newStake.Equal(v.physicalStake) && newBuyStake.Equal(v.buyStake) && newSellStake.Equal(v.sellStake) {
		// stake didn't change? there's nothing to do really
		return
	}
	// first time we set the newStake and mvp as avg.
	if !found {
		// this is technically a delta == new stake -> calculate avg accordingly
		v = &lp{}
		es.lps[id] = v
		es.updateAvgPosDelta(v, newStake, newBuyStake, newSellStake, markPrice) // delta == new stake
		return
	}

	delta := newStake.Sub(v.physicalStake)
	println("delta", delta.String())
	if newStake.LessThan(v.physicalStake) {
		// commitment decreased
		es.totalVStake = es.totalVStake.Sub(v.virtualStake)
		es.totalPStake = es.totalPStake.Sub(v.physicalStake)
		// vStake * (newStake/oldStake) or vStake * (old_stake + (-delta))/old_stake
		// e.g. 8000 => 5000 == vStakle * ((8000 + (-3000))/8000) == vStake * (5000/8000)

		v.buyVStake = v.buyVStake.Mul(newBuyStake.Div(v.buyStake))
		v.sellVStake = v.sellVStake.Mul(newSellStake.Div(v.sellStake))
		v.virtualStake = num.MinD(v.buyVStake, v.sellVStake.Mul(markPrice))

		v.buyStake = newBuyStake
		v.sellStake = newSellStake
		v.physicalStake = num.MinD(newBuyStake, newSellStake.Mul(markPrice))
		es.totalVStake = es.totalVStake.Add(v.virtualStake)
		es.totalPStake = es.totalPStake.Add(v.physicalStake)
		return
	}

	es.updateAvgPosDelta(v, delta, newBuyStake, newSellStake, markPrice)
}

func (es *EquityShares) updateAvgPosDelta(v *lp, delta, newBuyStake, newSellStake num.Decimal, markPrice num.Decimal) {
	// entry valuation == total Virtual stake (before delta is applied)
	// (average entry valuation) <- (average entry valuation) x S / (S + Delta S) + (entry valuation) x (Delta S) / (S + Delta S)
	// S being the LP's physical stake, Delta S being the amount by which the stake is increased
	newStake := num.MinD(newBuyStake, newSellStake.Mul(markPrice))
	es.totalVStake = es.totalVStake.Sub(v.virtualStake)
	es.totalPStake = es.totalPStake.Add(delta)

	deltaBuy := newBuyStake.Sub(v.buyStake)
	deltaSell := newSellStake.Sub(v.sellStake)

	if deltaBuy.IsNegative() {
		v.buyVStake = v.buyVStake.Mul(newBuyStake.Div(v.buyStake))
	} else {
		v.buyVStake = v.buyVStake.Add(deltaBuy)
	}
	if deltaSell.IsNegative() {
		v.sellVStake = v.sellVStake.Mul(newSellStake.Div(v.sellStake))
	} else {
		v.sellVStake = v.sellVStake.Add(deltaSell)
	}
	newVStake := num.MinD(v.buyVStake, v.sellVStake.Mul(markPrice))
	deltaVStake := newVStake.Sub(v.virtualStake)
	v.virtualStake = newVStake
	es.totalVStake = es.totalVStake.Add(v.virtualStake)
	v.avg = v.avg.Mul(v.physicalStake).Div(newStake).Add(es.totalVStake.Mul(deltaVStake).Div(newStake))
	v.physicalStake = newStake
	v.buyStake = newBuyStake
	v.sellStake = newSellStake
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
		return v.virtualStake.Div(es.totalVStake), nil
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
			es.totalPStake = es.totalPStake.Sub(v.physicalStake)
			es.totalVStake = es.totalVStake.Sub(v.virtualStake)
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

func (es *EquityShares) updateAllELS() {
	for id, v := range es.lps {
		eq, err := es.equity(id)
		if err != nil {
			panic(err)
		}
		if !v.share.Equals(eq) {
			v.share = eq
		}
	}
}
