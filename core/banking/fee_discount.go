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

func (e *Engine) feeDiscountKey(asset, party string) partyAssetKey {
	return partyAssetKey{party: party, asset: asset}
}

func (e *Engine) RegisterTakerFees(ctx context.Context, asset string, feesPerParty map[string]*num.Uint) {
	updateDiscountEvents := make([]events.Event, 0, len(e.feeDiscountPerPartyAndAsset))

	updatedKeys := map[partyAssetKey]struct{}{}
	for party, fee := range feesPerParty {
		key := e.feeDiscountKey(asset, party)
		updatedKeys[key] = struct{}{}

		if _, ok := e.feeDiscountPerPartyAndAsset[key]; !ok {
			e.feeDiscountPerPartyAndAsset[key] = num.UintZero()
		}

		// apply decay if not zero
		if !e.feeDiscountPerPartyAndAsset[key].IsZero() {
			decayedDiscount, _ := num.UintFromDecimal(e.feeDiscountPerPartyAndAsset[key].ToDecimal().Mul(e.feeDiscountDecayFraction))
			e.feeDiscountPerPartyAndAsset[key] = decayedDiscount
		}

		// add fees
		e.feeDiscountPerPartyAndAsset[key].Add(e.feeDiscountPerPartyAndAsset[key], fee)

		updateDiscountEvents = append(updateDiscountEvents, events.NewTransferFeesDiscountUpdated(
			ctx,
			party,
			asset,
			e.feeDiscountPerPartyAndAsset[key].Clone(),
			e.currentEpoch,
		))
	}
	for key := range e.feeDiscountPerPartyAndAsset {
		if _, ok := updatedKeys[key]; ok {
			continue
		}

		// apply decay if not zero
		if !e.feeDiscountPerPartyAndAsset[key].IsZero() {
			decayedDiscount, _ := num.UintFromDecimal(e.feeDiscountPerPartyAndAsset[key].ToDecimal().Mul(e.feeDiscountDecayFraction))
			e.feeDiscountPerPartyAndAsset[key] = decayedDiscount
		}

		updateDiscountEvents = append(updateDiscountEvents, events.NewTransferFeesDiscountUpdated(
			ctx,
			key.party,
			asset,
			e.feeDiscountPerPartyAndAsset[key].Clone(),
			e.currentEpoch,
		))
	}

	e.broker.SendBatch(updateDiscountEvents)
}

func (e *Engine) ApplyFeeDiscount(ctx context.Context, asset string, party string, fee *num.Uint) (discountedFee *num.Uint, discount *num.Uint) {
	discountedFee, discount = e.EstimateFeeDiscount(asset, party, fee)
	if discount.IsZero() {
		return discountedFee, discount
	}

	key := e.feeDiscountKey(asset, party)
	defer e.broker.Send(
		events.NewTransferFeesDiscountUpdated(ctx,
			party, asset,
			e.feeDiscountPerPartyAndAsset[key].Clone(),
			e.currentEpoch,
		),
	)

	e.feeDiscountPerPartyAndAsset[key].Sub(e.feeDiscountPerPartyAndAsset[key], discount)

	return discountedFee, discount
}

func (e *Engine) EstimateFeeDiscount(asset string, party string, fee *num.Uint) (discountedFee *num.Uint, discount *num.Uint) {
	if fee.IsZero() {
		return fee, num.UintZero()
	}

	key := e.feeDiscountKey(asset, party)
	accumulatedDiscount, ok := e.feeDiscountPerPartyAndAsset[key]
	if !ok {
		return fee, num.UintZero()
	}

	return calculateDiscount(accumulatedDiscount, fee)
}

func (e *Engine) AvailableFeeDiscount(asset string, party string) *num.Uint {
	key := e.feeDiscountKey(asset, party)

	if discount, ok := e.feeDiscountPerPartyAndAsset[key]; ok {
		return discount.Clone()
	}

	return num.UintZero()
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
