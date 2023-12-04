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

func (e *Engine) applyPendingFeeDiscountsUpdates(ctx context.Context) {
	assetIDs := maps.Keys(e.pendingPerAssetAndPartyFeeDiscountUpdates)
	sort.Strings(assetIDs)

	updatedKeys := map[string]map[string]struct{}{}
	for _, assetID := range assetIDs {
		feeDiscountsPerParty := e.pendingPerAssetAndPartyFeeDiscountUpdates[assetID]
		perAssetUpdatedKeys := e.updateFeeDiscountsForAsset(ctx, assetID, feeDiscountsPerParty)
		updatedKeys[assetID] = perAssetUpdatedKeys
	}

	e.pendingPerAssetAndPartyFeeDiscountUpdates = map[string]map[string]*num.Uint{}
	e.decayAllFeeDiscounts(ctx, updatedKeys)
}

func (e *Engine) decayAllFeeDiscounts(ctx context.Context, perAssetAndPartyUpdates map[string]map[string]struct{}) {
	updateDiscountEvents := make([]events.Event, 0, len(e.feeDiscountPerPartyAndAsset))

	feeDiscountPerPartyAndAssetKeys := maps.Keys(e.feeDiscountPerPartyAndAsset)
	sort.SliceStable(feeDiscountPerPartyAndAssetKeys, func(i, j int) bool {
		if feeDiscountPerPartyAndAssetKeys[i].party == feeDiscountPerPartyAndAssetKeys[j].party {
			return feeDiscountPerPartyAndAssetKeys[i].asset < feeDiscountPerPartyAndAssetKeys[j].asset
		}

		return feeDiscountPerPartyAndAssetKeys[i].party < feeDiscountPerPartyAndAssetKeys[j].party
	})

	for _, key := range feeDiscountPerPartyAndAssetKeys {
		if assetUpdate, assetOK := perAssetAndPartyUpdates[key.asset]; assetOK {
			if _, partyOK := assetUpdate[key.party]; partyOK {
				continue
			}
		}

		// ensure asset exists
		asset, err := e.assets.Get(key.asset)
		if err != nil {
			e.log.Panic("could not register taker fees, invalid asset", logging.Error(err))
		}

		assetQuantum := asset.Type().Details.Quantum

		var decayAmountD *num.Uint
		// apply decay discount amount
		decayAmount := e.decayFeeDiscountAmount(e.feeDiscountPerPartyAndAsset[key])

		// or 0 if discount is less than e.feeDiscountMinimumTrackedAmount x quantum (where quantum is the asset quantum).
		if decayAmount.LessThan(e.feeDiscountMinimumTrackedAmount.Mul(assetQuantum)) {
			decayAmountD = num.UintZero()
			delete(e.feeDiscountPerPartyAndAsset, key)
		} else {
			decayAmountD, _ = num.UintFromDecimal(decayAmount)
			e.feeDiscountPerPartyAndAsset[key] = decayAmountD
		}

		updateDiscountEvents = append(updateDiscountEvents, events.NewTransferFeesDiscountUpdated(
			ctx,
			key.party,
			key.asset,
			decayAmountD.Clone(),
			e.currentEpoch,
		))
	}

	if len(updateDiscountEvents) > 0 {
		e.broker.SendBatch(updateDiscountEvents)
	}
}

func (e *Engine) updateFeeDiscountsForAsset(
	ctx context.Context, assetID string, feeDiscountsPerParty map[string]*num.Uint,
) map[string]struct{} {
	updateDiscountEvents := make([]events.Event, 0, len(e.feeDiscountPerPartyAndAsset))

	// ensure asset exists
	asset, err := e.assets.Get(assetID)
	if err != nil {
		e.log.Panic("could not register taker fees, invalid asset", logging.Error(err))
	}

	assetQuantum := asset.Type().Details.Quantum

	feesPerPartyKeys := maps.Keys(feeDiscountsPerParty)
	sort.Strings(feesPerPartyKeys)

	updatedKeys := map[string]struct{}{}
	for _, party := range feesPerPartyKeys {
		fee := feeDiscountsPerParty[party]

		updatedKeys[party] = struct{}{}

		key := e.feeDiscountKey(assetID, party)
		if _, ok := e.feeDiscountPerPartyAndAsset[key]; !ok {
			e.feeDiscountPerPartyAndAsset[key] = num.UintZero()
		}

		// apply decay discount amount and add new fees to it
		newAmount := e.decayFeeDiscountAmount(e.feeDiscountPerPartyAndAsset[key]).Add(fee.ToDecimal())

		if newAmount.LessThan(e.feeDiscountMinimumTrackedAmount.Mul(assetQuantum)) {
			e.feeDiscountPerPartyAndAsset[key] = num.UintZero()
		} else {
			newAmountD, _ := num.UintFromDecimal(newAmount)
			e.feeDiscountPerPartyAndAsset[key] = newAmountD
		}

		updateDiscountEvents = append(updateDiscountEvents, events.NewTransferFeesDiscountUpdated(
			ctx,
			party,
			assetID,
			e.feeDiscountPerPartyAndAsset[key].Clone(),
			e.currentEpoch,
		))
	}

	if len(updateDiscountEvents) > 0 {
		e.broker.SendBatch(updateDiscountEvents)
	}

	return updatedKeys
}

func (e *Engine) RegisterTradingFees(ctx context.Context, assetID string, feesPerParty map[string]*num.Uint) {
	// ensure asset exists
	_, err := e.assets.Get(assetID)
	if err != nil {
		e.log.Panic("could not register taker fees, invalid asset", logging.Error(err))
	}

	if _, ok := e.pendingPerAssetAndPartyFeeDiscountUpdates[assetID]; !ok {
		e.pendingPerAssetAndPartyFeeDiscountUpdates[assetID] = map[string]*num.Uint{}
	}

	for partyID, fee := range feesPerParty {
		if _, ok := e.pendingPerAssetAndPartyFeeDiscountUpdates[assetID][partyID]; !ok {
			e.pendingPerAssetAndPartyFeeDiscountUpdates[assetID][partyID] = fee.Clone()
			continue
		}
		e.pendingPerAssetAndPartyFeeDiscountUpdates[assetID][partyID].AddSum(fee)
	}
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

// decayFeeDiscountAmount update current discount with: discount x e.feeDiscountDecayFraction.
func (e *Engine) decayFeeDiscountAmount(currentDiscount *num.Uint) num.Decimal {
	discount := currentDiscount.ToDecimal()
	if discount.IsZero() {
		return discount
	}
	return discount.Mul(e.feeDiscountDecayFraction)
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
