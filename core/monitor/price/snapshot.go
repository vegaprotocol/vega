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

package price

import (
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

func NewMonitorFromSnapshot(
	marketID string,
	asset string,
	pm *types.PriceMonitor,
	settings *types.PriceMonitoringSettings,
	riskModel RangeProvider,
	auctionState AuctionState,
	stateVarEngine StateVarEngine,
	log *logging.Logger,
) (*Engine, error) {
	if riskModel == nil {
		return nil, ErrNilRangeProvider
	}
	if settings == nil {
		return nil, ErrNilPriceMonitoringSettings
	}

	priceRangesCache, needRecalc := newPriceRangeCacheFromSlice(pm.PriceRangeCache)

	e := &Engine{
		market:              marketID,
		log:                 log,
		riskModel:           riskModel,
		auctionState:        auctionState,
		initialised:         pm.Initialised,
		fpHorizons:          keyDecimalPairToMap(pm.FPHorizons),
		now:                 pm.Now,
		update:              pm.Update,
		priceRangeCacheTime: pm.PriceRangeCacheTime,
		refPriceCache:       keyDecimalPairToMap(pm.RefPriceCache),
		refPriceCacheTime:   pm.RefPriceCacheTime,
		bounds:              priceBoundsToBounds(pm.Bounds),
		priceRangesCache:    priceRangesCache,
		pricesNow:           pricesNowToInternal(pm.PricesNow),
		pricesPast:          pricesPastToInternal(pm.PricesPast),
		stateChanged:        true,
		asset:               asset,
	}
	e.boundFactorsInitialised = pm.PriceBoundsConsensusReached
	stateVarEngine.RegisterStateVariable(asset, marketID, "bound-factors", boundFactorsConverter{}, e.startCalcPriceRanges, []statevar.EventType{statevar.EventTypeTimeTrigger, statevar.EventTypeAuctionEnded, statevar.EventTypeOpeningAuctionFirstUncrossingPrice}, e.updatePriceBounds)

	if needRecalc {
		e.getCurrentPriceRanges(true)
	}
	return e, nil
}

func pricesNowToInternal(cps []*types.CurrentPrice) []currentPrice {
	cpsi := make([]currentPrice, 0, len(cps))
	for _, cp := range cps {
		cpsi = append(cpsi, currentPrice{
			Price:  cp.Price.Clone(),
			Volume: cp.Volume,
		})
	}
	return cpsi
}

func pricesPastToInternal(pps []*types.PastPrice) []pastPrice {
	ppsi := make([]pastPrice, 0, len(pps))
	for _, pp := range pps {
		ppsi = append(ppsi, pastPrice{
			Time:                pp.Time,
			VolumeWeightedPrice: pp.VolumeWeightedPrice,
		})
	}
	return ppsi
}

func internalBoundToPriceBoundType(b *bound) *types.PriceBound {
	return &types.PriceBound{
		Active:     b.Active,
		UpFactor:   b.UpFactor,
		DownFactor: b.DownFactor,
		Trigger:    b.Trigger.DeepClone(),
	}
}

func priceBoundTypeToInternal(pb *types.PriceBound) *bound {
	return &bound{
		Active:     pb.Active,
		UpFactor:   pb.UpFactor,
		DownFactor: pb.DownFactor,
		Trigger:    pb.Trigger.DeepClone(),
	}
}

func mapToKeyDecimalPair(m map[int64]num.Decimal) []*types.KeyDecimalPair {
	dm := make([]*types.KeyDecimalPair, 0, len(m))

	for k, v := range m {
		dm = append(dm, &types.KeyDecimalPair{
			Key: k,
			Val: v,
		})
	}

	sort.Slice(dm, func(i, j int) bool {
		return dm[i].Key < dm[j].Key
	})

	return dm
}

func keyDecimalPairToMap(dms []*types.KeyDecimalPair) map[int64]num.Decimal {
	m := make(map[int64]num.Decimal, len(dms))

	for _, dm := range dms {
		m[dm.Key] = dm.Val
	}

	return m
}

func priceBoundsToBounds(pbs []*types.PriceBound) []*bound {
	bounds := make([]*bound, 0, len(pbs))
	for _, pb := range pbs {
		bounds = append(bounds, priceBoundTypeToInternal(pb))
	}
	return bounds
}

func (e *Engine) serialiseBounds() []*types.PriceBound {
	bounds := make([]*types.PriceBound, 0, len(e.bounds))
	for _, b := range e.bounds {
		bounds = append(bounds, internalBoundToPriceBoundType(b))
	}

	return bounds
}

func newPriceRangeCacheFromSlice(prs []*types.PriceRangeCache) (map[int]priceRange, bool) {
	priceRangesCache := map[int]priceRange{}
	needsRecalc := false
	for _, pr := range prs {
		if pr.BoundIndex < 0 {
			needsRecalc = true
			break
		}
		priceRangesCache[pr.BoundIndex] = priceRange{
			MinPrice:       wrapPriceRange(pr.Range.Min, true),
			MaxPrice:       wrapPriceRange(pr.Range.Max, false),
			ReferencePrice: pr.Range.Ref,
		}
	}

	return priceRangesCache, needsRecalc
}

func (e *Engine) serialisePriceRanges() []*types.PriceRangeCache {
	prc := make([]*types.PriceRangeCache, 0, len(e.priceRangesCache))
	for ind, priceRange := range e.priceRangesCache {
		prc = append(prc, &types.PriceRangeCache{
			BoundIndex: ind,
			Range: &types.PriceRange{
				Min: priceRange.MinPrice.Original(),
				Max: priceRange.MaxPrice.Original(),
				Ref: priceRange.ReferencePrice,
			},
		})
	}

	sort.Slice(prc, func(i, j int) bool { return prc[i].BoundIndex < prc[j].BoundIndex })

	return prc
}

func (e *Engine) Changed() bool {
	return e.stateChanged
}

func (e *Engine) serialisePricesNow() []*types.CurrentPrice {
	psn := make([]*types.CurrentPrice, 0, len(e.pricesNow))
	for _, pn := range e.pricesNow {
		psn = append(psn, &types.CurrentPrice{
			Price:  pn.Price.Clone(),
			Volume: pn.Volume,
		})
	}

	sort.Slice(psn, func(i, j int) bool {
		if psn[i].Price.EQ(psn[j].Price) {
			return psn[i].Volume < psn[j].Volume
		}

		return psn[i].Price.LT(psn[j].Price)
	})

	return psn
}

func (e *Engine) serialisePricesPast() []*types.PastPrice {
	pps := make([]*types.PastPrice, 0, len(e.pricesPast))
	for _, pp := range e.pricesPast {
		pps = append(pps, &types.PastPrice{
			Time:                pp.Time,
			VolumeWeightedPrice: pp.VolumeWeightedPrice,
		})
	}

	sort.Slice(pps, func(i, j int) bool {
		if pps[i].Time.Equal(pps[j].Time) {
			return pps[j].VolumeWeightedPrice.GreaterThan(pps[i].VolumeWeightedPrice)
		}

		return pps[i].Time.Before(pps[j].Time)
	})

	return pps
}

func (e *Engine) GetState() *types.PriceMonitor {
	pm := &types.PriceMonitor{
		Initialised:                 e.initialised,
		FPHorizons:                  mapToKeyDecimalPair(e.fpHorizons),
		Now:                         e.now,
		Update:                      e.update,
		Bounds:                      e.serialiseBounds(),
		PriceRangeCache:             e.serialisePriceRanges(),
		PricesNow:                   e.serialisePricesNow(),
		PricesPast:                  e.serialisePricesPast(),
		PriceRangeCacheTime:         e.priceRangeCacheTime,
		RefPriceCache:               mapToKeyDecimalPair(e.refPriceCache),
		RefPriceCacheTime:           e.refPriceCacheTime,
		PriceBoundsConsensusReached: e.boundFactorsInitialised,
	}

	e.stateChanged = false

	return pm
}
