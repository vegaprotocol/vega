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
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"golang.org/x/exp/maps"
)

type partyAssetKey struct {
	party string
	asset string
}

func (e *Engine) feeDiscountKey(asset, party string) partyAssetKey {
	return partyAssetKey{party: party, asset: asset}
}

func (e *Engine) RegisterTradingFees(ctx context.Context, assetID string, feesPerParty map[string]*num.Uint) {
	fmt.Printf("------ RegisterTradingFees: %s, %+v \n", assetID, feesPerParty)

	updateDiscountEvents := make([]events.Event, 0, len(e.feeDiscountPerPartyAndAsset))

	// ensure asset exists
	asset, err := e.assets.Get(assetID)
	if err != nil {
		e.log.Panic("could not register taker fees, invalid asset", logging.Error(err))
	}

	assetQuantum := asset.Type().Details.Quantum

	feesPerPartyKeys := maps.Keys(feesPerParty)
	sort.Strings(feesPerPartyKeys)

	updatedKeys := map[partyAssetKey]struct{}{}
	for _, party := range feesPerPartyKeys {
		fee := feesPerParty[party]

		key := e.feeDiscountKey(assetID, party)
		updatedKeys[key] = struct{}{}

		if _, ok := e.feeDiscountPerPartyAndAsset[key]; !ok {
			e.feeDiscountPerPartyAndAsset[key] = num.UintZero()
		}

		fmt.Println("-------- before decay e.feeDiscountPerPartyAndAsset[key] ", key, e.feeDiscountPerPartyAndAsset[key])

		// apply decay discount amount
		e.feeDiscountPerPartyAndAsset[key] = e.decayFeeDiscountAmount(e.feeDiscountPerPartyAndAsset[key], assetQuantum)

		fmt.Println("-------- after e.feeDiscountPerPartyAndAsset[key] ", key, e.feeDiscountPerPartyAndAsset[key])

		// add fees
		e.feeDiscountPerPartyAndAsset[key].AddSum(fee)

		updateDiscountEvents = append(updateDiscountEvents, events.NewTransferFeesDiscountUpdated(
			ctx,
			party,
			assetID,
			e.feeDiscountPerPartyAndAsset[key].Clone(),
			e.currentEpoch,
		))
	}

	feeDiscountPerPartyAndAssetKeys := maps.Keys(e.feeDiscountPerPartyAndAsset)
	sort.SliceStable(feeDiscountPerPartyAndAssetKeys, func(i, j int) bool {
		if feeDiscountPerPartyAndAssetKeys[i].party == feeDiscountPerPartyAndAssetKeys[j].party {
			return feeDiscountPerPartyAndAssetKeys[i].asset < feeDiscountPerPartyAndAssetKeys[j].asset
		}

		return feeDiscountPerPartyAndAssetKeys[i].party < feeDiscountPerPartyAndAssetKeys[j].party
	})

	for _, key := range feeDiscountPerPartyAndAssetKeys {
		if _, ok := updatedKeys[key]; ok {
			continue
		}

		// apply decay discount amount
		e.feeDiscountPerPartyAndAsset[key] = e.decayFeeDiscountAmount(e.feeDiscountPerPartyAndAsset[key], assetQuantum)

		fmt.Println("-------- non updated after e.feeDiscountPerPartyAndAsset[key]:", key, e.feeDiscountPerPartyAndAsset[key])

		updateDiscountEvents = append(updateDiscountEvents, events.NewTransferFeesDiscountUpdated(
			ctx,
			key.party,
			assetID,
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

// decayFeeDiscountAmount update current discount with: discount x e.feeDiscountDecayFraction
// or 0 if discount is less than e.feeDiscountMinimumTrackedAmount x quantum (where quantum is the asset quantum).
func (e *Engine) decayFeeDiscountAmount(currentDiscount *num.Uint, assetQuantum num.Decimal) *num.Uint {
	if currentDiscount.IsZero() {
		return currentDiscount
	}

	fmt.Println("------- currentDiscount, e.feeDiscountDecayFraction:", currentDiscount, e.feeDiscountDecayFraction)
	decayedAmount := currentDiscount.ToDecimal().Mul(e.feeDiscountDecayFraction)

	if decayedAmount.LessThan(e.feeDiscountMinimumTrackedAmount.Mul(assetQuantum)) {
		return num.UintZero()
	}

	decayedAmountUint, _ := num.UintFromDecimal(decayedAmount)
	return decayedAmountUint
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
