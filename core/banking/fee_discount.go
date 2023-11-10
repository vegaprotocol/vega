// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package banking

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/libs/num"
)

type partyAssetKey struct {
	party string
	asset string
}

type feeDiscount struct {
	paidTakerFees       []*num.Uint
	pos                 int
	accumulatedDiscount *num.Uint
	usedDiscount        *num.Uint
}

func newFeeDiscount(window int) *feeDiscount {
	return &feeDiscount{
		paidTakerFees:       make([]*num.Uint, window),
		pos:                 0,
		usedDiscount:        num.UintZero(),
		accumulatedDiscount: num.UintZero(),
	}
}

// AddTakerFee adds a new taker fee to the slice and accumulates all fees into discount.
func (r *feeDiscount) AddTakerFee(val *num.Uint) {
	r.paidTakerFees[r.pos] = val
	r.accumulateDiscount()

	if r.pos == cap(r.paidTakerFees)-1 {
		r.pos = 0
		return
	}
	r.pos++
}

func (r *feeDiscount) AccumulatedDiscount() *num.Uint {
	return r.accumulatedDiscount.Clone()
}

func (r *feeDiscount) accumulateDiscount() {
	r.accumulatedDiscount = num.UintZero()

	for _, v := range r.paidTakerFees {
		if v != nil {
			r.accumulatedDiscount.AddSum(v)
		}
	}

	r.accumulatedDiscount.Sub(r.accumulatedDiscount, r.usedDiscount)
	r.usedDiscount = num.UintZero()
}

// ApplyDiscount applies a portion of the accumulated discount to the fee and return the discounted fee amount.
func (r *feeDiscount) ApplyDiscount(theoreticalFee *num.Uint) (discountedFee, discount *num.Uint) {
	discountedFee, discount = r.CalculateDiscount(theoreticalFee)

	r.usedDiscount.AddSum(discount)
	r.accumulatedDiscount.Sub(r.accumulatedDiscount, r.usedDiscount)

	return discountedFee, discount
}

func (r *feeDiscount) CalculateDiscount(theoreticalFee *num.Uint) (discountedFee, discount *num.Uint) {
	return calculateDiscount(r.accumulatedDiscount, theoreticalFee)
}

func (r *feeDiscount) UpdateDiscountWindow(window int) {
	currentCap := cap(r.paidTakerFees)
	if currentCap == window {
		return
	}

	new := make([]*num.Uint, window)

	// decrease
	if window < currentCap {
		new = r.paidTakerFees[currentCap-window:]
		r.paidTakerFees = new
		r.pos = 0
		return
	}

	// increase
	for i := 0; i < currentCap; i++ {
		new[i] = r.paidTakerFees[i]
	}

	r.paidTakerFees = new
	r.pos = currentCap
}

func (e *Engine) updateDiscountsWindows() {
	for key := range e.feeDiscountPerPartyAndAsset {
		e.feeDiscountPerPartyAndAsset[key].UpdateDiscountWindow(e.feeDiscountNumOfEpoch)
	}
}

func (e *Engine) feeDiscountKey(asset, party string) partyAssetKey {
	return partyAssetKey{party: party, asset: asset}
}

func (e *Engine) RegisterTakerFees(ctx context.Context, asset string, feesPerParty map[string]*num.Uint) {
	updateDiscountEvents := make([]events.Event, 0, len(e.feeDiscountPerPartyAndAsset))

	updatedKeys := map[partyAssetKey]struct{}{}

	zero := num.UintZero()
	for party, fee := range feesPerParty {
		key := e.feeDiscountKey(asset, party)
		updatedKeys[key] = struct{}{}

		if _, ok := e.feeDiscountPerPartyAndAsset[key]; !ok {
			e.feeDiscountPerPartyAndAsset[key] = newFeeDiscount(e.feeDiscountNumOfEpoch)
		}

		e.feeDiscountPerPartyAndAsset[key].AddTakerFee(fee)

		updateDiscountEvents = append(updateDiscountEvents, events.NewTransferFeesDiscountUpdated(
			ctx,
			party,
			asset,
			e.feeDiscountPerPartyAndAsset[key].AccumulatedDiscount(),
			e.currentEpoch,
		))
	}

	for key := range e.feeDiscountPerPartyAndAsset {
		if _, ok := updatedKeys[key]; ok {
			continue
		}

		e.feeDiscountPerPartyAndAsset[key].AddTakerFee(zero)

		updateDiscountEvents = append(updateDiscountEvents, events.NewTransferFeesDiscountUpdated(
			ctx,
			key.party,
			asset,
			e.feeDiscountPerPartyAndAsset[key].AccumulatedDiscount(),
			e.currentEpoch,
		))
	}

	e.broker.SendBatch(updateDiscountEvents)
}

func (e *Engine) ApplyFeeDiscount(ctx context.Context, asset string, party string, fee *num.Uint) (discountedFee *num.Uint, discount *num.Uint) {
	if fee.IsZero() {
		return fee, num.UintZero()
	}

	key := e.feeDiscountKey(asset, party)

	if _, ok := e.feeDiscountPerPartyAndAsset[key]; !ok {
		return fee, num.UintZero()
	}

	defer e.broker.Send(
		events.NewTransferFeesDiscountUpdated(ctx,
			party, asset,
			e.feeDiscountPerPartyAndAsset[key].AccumulatedDiscount(),
			e.currentEpoch,
		),
	)

	return e.feeDiscountPerPartyAndAsset[key].ApplyDiscount(fee)
}

func (e *Engine) EstimateFeeDiscount(asset string, party string, fee *num.Uint) (discountedFee *num.Uint, discount *num.Uint) {
	if fee.IsZero() {
		return fee, num.UintZero()
	}

	key := e.feeDiscountKey(asset, party)

	if _, ok := e.feeDiscountPerPartyAndAsset[key]; !ok {
		return fee, num.UintZero()
	}

	return e.feeDiscountPerPartyAndAsset[key].CalculateDiscount(fee)
}

func calculateDiscount(accumulatedDiscount, theoreticalFee *num.Uint) (discountedFee, discount *num.Uint) {
	theoreticalFeeD := theoreticalFee.ToDecimal()
	// min(accumulatedDiscount-theoreticalFee,0)
	feeD := num.MinD(
		accumulatedDiscount.ToDecimal().Sub(theoreticalFee.ToDecimal()),
		num.DecimalZero(),
	).Neg()

	appliedDiscount, _ := num.UintFromDecimal(theoreticalFeeD.Sub(feeD))
	// -fee
	discountedFee, _ = num.UintFromDecimal(feeD)
	return discountedFee, appliedDiscount
}
