package price

import (
	"sort"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
)

func NewMonitorFromSnapshot(
	marketID string,
	asset string,
	pm *types.PriceMonitor,
	settings *types.PriceMonitoringSettings,
	riskModel RangeProvider,
	stateVarEngine StateVarEngine,
	log *logging.Logger,
) (*Engine, error) {
	if riskModel == nil {
		return nil, ErrNilRangeProvider
	}
	if settings == nil {
		return nil, ErrNilPriceMonitoringSettings
	}

	e := &Engine{
		log:                 log,
		riskModel:           riskModel,
		initialised:         pm.Initialised,
		fpHorizons:          keyDecimalPairToMap(pm.FPHorizons),
		now:                 pm.Now,
		update:              pm.Update,
		priceRangeCacheTime: pm.PriceRangeCacheTime,
		refPriceCache:       keyDecimalPairToMap(pm.RefPriceCache),
		refPriceCacheTime:   pm.RefPriceCacheTime,
		bounds:              priceBoundsToBounds(pm.Bounds),
		priceRangesCache:    newPriceRangeCacheFromSlice(pm.PriceRangeCache),
		pricesNow:           pricesNowToInternal(pm.PricesNow),
		pricesPast:          pricesPastToInternal(pm.PricesPast),
		stateChanged:        true,
	}
	e.boundFactorsInitialised = pm.PriceBoundsConsensusReached
	stateVarEngine.RegisterStateVariable(asset, marketID, "bound-factors", boundFactorsConverter{}, e.startCalcPriceRanges, []statevar.StateVarEventType{statevar.StateVarEventTypeTimeTrigger, statevar.StateVarEventTypeAuctionEnded, statevar.StateVarEventTypeOpeningAuctionFirstUncrossingPrice}, e.updatePriceBounds)
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

func wrappedDecimalFromDecimal(d num.Decimal) num.WrappedDecimal {
	uit, _ := num.UintFromDecimal(d)
	return num.NewWrappedDecimal(uit, d)
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

func newPriceRangeCacheFromSlice(prs []*types.PriceRangeCache) map[*bound]priceRange {
	priceRangesCache := map[*bound]priceRange{}
	for _, pr := range prs {
		priceRangesCache[priceBoundTypeToInternal(pr.Bound)] = priceRange{
			MinPrice:       wrappedDecimalFromDecimal(pr.Range.Min),
			MaxPrice:       wrappedDecimalFromDecimal(pr.Range.Max),
			ReferencePrice: pr.Range.Ref,
		}
	}
	return priceRangesCache
}

func (e Engine) serialisePriceRanges() []*types.PriceRangeCache {
	prc := make([]*types.PriceRangeCache, 0, len(e.priceRangesCache))
	for bound, priceRange := range e.priceRangesCache {
		prc = append(prc, &types.PriceRangeCache{
			Bound: internalBoundToPriceBoundType(bound),
			Range: &types.PriceRange{
				Min: priceRange.MinPrice.Original(),
				Max: priceRange.MaxPrice.Original(),
				Ref: priceRange.ReferencePrice,
			},
		})
	}

	sort.Slice(prc, func(i, j int) bool {
		if prc[i].Bound.UpFactor.Equal(prc[j].Bound.UpFactor) {
			return prc[j].Bound.DownFactor.GreaterThan(prc[i].Bound.DownFactor)
		}

		return prc[j].Bound.UpFactor.GreaterThan(prc[i].Bound.UpFactor)
	})

	return prc
}

func (e Engine) Changed() bool {
	return e.stateChanged
}

func (e Engine) serialisePricesNow() []*types.CurrentPrice {
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

func (e Engine) serialisePricesPast() []*types.PastPrice {
	pps := make([]*types.PastPrice, 0, len(e.pricesPast))
	for _, pp := range pps {
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
