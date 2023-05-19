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
	"sort"

	"code.vegaprotocol.io/vega/core/types"
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
	// the same map as above, but used at the end of the opening auction in successor markets to carry over ELS
	parentLPs map[string]*lp

	openingAuctionEnded bool
}

func NewEquityShares(mvp num.Decimal) *EquityShares {
	return &EquityShares{
		mvp:         mvp,
		r:           num.DecimalZero(),
		totalPStake: num.DecimalZero(),
		totalVStake: num.DecimalZero(),
		lps:         map[string]*lp{},
		parentLPs:   map[string]*lp{},
	}
}

func (es *EquityShares) InheritELS(shares []*types.ELSShare) {
	for _, els := range shares {
		if current, ok := es.lps[els.PartyID]; ok {
			update, _ := num.UintFromDecimal(current.stake)
			// update the totals:
			es.totalPStake = es.totalPStake.Sub(current.stake).Add(els.SuppliedStake)
			es.totalVStake = es.totalVStake.Sub(current.vStake).Add(els.VStake)
			// make it look like the old ELS data was part of this market...
			current.stake = els.SuppliedStake
			current.vStake = els.VStake
			current.avg = els.Avg
			current.share = els.Share
			// then treat the current commitment as a change to the commitment amount:
			es.SetPartyStake(els.PartyID, update)
		} else {
			// this LP hasn't committed to this market (yet)
			es.parentLPs[els.PartyID] = &lp{
				stake:  els.SuppliedStake,
				share:  els.Share,
				vStake: els.VStake,
				avg:    els.Avg,
			}
		}
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
	if len(es.parentLPs) > 0 {
		// set the ELS in successor markets
		// es.resolveParentLPs()
		es.parentLPs = nil // ditch the old data
	}
}

func (es *EquityShares) UpdateVStake() {
	if es.r.IsZero() {
		return
	}
	total := num.DecimalZero()
	factor := num.DecimalFromFloat(1.0).Add(es.r)
	recalc := false
	for _, v := range es.lps {
		vStake := num.MaxD(v.stake, v.vStake.Mul(factor))
		v.vStake = vStake
		total = total.Add(vStake)
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
	if !es.openingAuctionEnded && len(es.parentLPs) > 0 {
		if parent, ok := es.parentLPs[id]; ok {
			// we can only reach this point if we've inherited from a parent market, and this
			// party, at that time, hadn't made a commitment yet
			es.lps[id] = parent
			// add the old ELS data to the running totals
			es.totalPStake = es.totalPStake.Add(parent.stake)
			es.totalVStake = es.totalVStake.Add(parent.vStake)
			delete(es.parentLPs, id) // we've restored this now, so remove from this map
		}
	}
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
	v.avg = v.avg.Mul(v.stake).Div(newStake).Add(es.totalVStake.Mul(delta).Div(newStake))
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

func (es *EquityShares) getCPShares() []*types.ELSShare {
	shares := make([]*types.ELSShare, 0, len(es.lps))
	for id, els := range es.lps {
		shares = append(shares, &types.ELSShare{
			PartyID:       id,
			Share:         els.share,
			SuppliedStake: els.stake,
			VStake:        els.vStake,
			Avg:           els.avg,
		})
	}
	// make sure data is sorted
	sort.SliceStable(shares, func(i, j int) bool {
		return shares[i].PartyID > shares[j].PartyID
	})
	return shares
}

func (es *EquityShares) setCPShares(shares []*types.ELSShare) {
	es.lps = make(map[string]*lp, len(shares))
	es.totalPStake = num.DecimalZero()
	es.totalVStake = num.DecimalZero()
	for _, share := range shares {
		lp := lp{
			avg:    share.Avg,
			share:  share.Share,
			stake:  share.SuppliedStake,
			vStake: share.VStake,
		}
		es.totalPStake = es.totalPStake.Add(lp.stake)
		es.totalVStake = es.totalVStake.Add(lp.vStake)
		es.lps[share.PartyID] = &lp
	}
}

func (es *EquityShares) InheritELSBatch(shares []*types.ELSShare) {
	// the simple way is to just keep track of all previous ELS data
	// once we leave opening auction, we call `resolveParentLPs` and work it all out
	// of course, we could instead check the map of parties who already provided liquidity
	// and do what we need to do for those parties here, and add the remainder to the partentLP map
	for _, els := range shares {
		lp := lp{
			stake:  els.SuppliedStake,
			share:  els.Share,
			vStake: els.VStake,
			avg:    els.Avg,
		}
		es.parentLPs[els.PartyID] = &lp
		// es.totalPStake = es.totalPStake.Add(els.SuppliedStake)
		// es.totalVStake = es.totalVStake.Add(els.VStake)
	}
	if es.openingAuctionEnded {
		es.resolveParentLPs()
	}
}

func (es *EquityShares) resolveParentLPs() {
	for id, lp := range es.parentLPs {
		if sLP, ok := es.lps[id]; ok {
			// ok, this party carries over their ELS, but the amounts may differ, first check if the amount is different:
			if sLP.stake.Equal(lp.stake) {
				// physical stake is identical, let's update the vStake total:
				es.totalVStake = es.totalVStake.Sub(sLP.vStake).Add(lp.vStake)
				// and update the vStake and avg
				sLP.vStake = lp.vStake
				sLP.avg = lp.avg
			} else {
				// now this one is a bit trickier -> we should swap out the new commitment, and treat it as a change in commitment:
				// step 1: update the the totals:
				es.totalVStake = es.totalVStake.Sub(sLP.vStake).Add(lp.vStake)
				// remove the new physical stake, add the old
				es.totalPStake = es.totalPStake.Sub(sLP.stake).Add(lp.stake)
				newCommitment, _ := num.UintFromDecimal(sLP.stake)
				es.lps[id] = lp // assign it the old LP
				// now apply the different stake as though it's an amendment of the old one
				es.SetPartyStake(id, newCommitment)
			}
		}
	}
	// recalculate all ELS
	es.updateAllELS()
	// clear the data
	es.parentLPs = map[string]*lp{}
}
