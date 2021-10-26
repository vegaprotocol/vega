package price

import (
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func NewMonitorFromSnapshot(
	pm *types.PriceMonitor,
	settings *types.PriceMonitoringSettings,
	riskModel RangeProvider,
) (*Engine, error) {
	if riskModel == nil {
		return nil, ErrNilRangeProvider
	}
	if settings == nil {
		return nil, ErrNilPriceMonitoringSettings
	}

	e := &Engine{
		riskModel:           riskModel,
		updateFrequency:     time.Duration(settings.UpdateFrequency) * time.Second,
		initialised:         pm.Initialised,
		fpHorizons:          keyDecimalPairToMap(pm.FPHorizons),
		now:                 pm.Now,
		update:              pm.Update,
		priceRangeCacheTime: pm.PriceRangeCacheTime,
		refPriceCache:       keyDecimalPairToMap(pm.RefPriceCache),
		refPriceCacheTime:   pm.RefPriceCacheTime,
		bounds:              priceBoundsToBounds(pm.Bounds),
		priceRangesCache:    newPriceRangeCacheFromSlice(pm.PriceRangeCache),
		stateChanged:        true,
	}
	// hack to work around the update frequency being 0 causing an infinite loop
	// for now, this will do
	// @TODO go through integration and system tests once we validate this properly
	if settings.UpdateFrequency == 0 {
		e.updateFrequency = time.Second
	}
	return e, nil
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

	return dm
}

func keyDecimalPairToMap(dms []*types.KeyDecimalPair) map[int64]num.Decimal {
	m := map[int64]num.Decimal{}

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
	return prc
}

func (e Engine) Changed() bool {
	return e.stateChanged
}

func (e *Engine) GetState() *types.PriceMonitor {
	pm := &types.PriceMonitor{
		Initialised:         e.initialised,
		FPHorizons:          mapToKeyDecimalPair(e.fpHorizons),
		Now:                 e.now,
		Update:              e.update,
		Bounds:              e.serialiseBounds(),
		PriceRangeCache:     e.serialisePriceRanges(),
		PriceRangeCacheTime: e.priceRangeCacheTime,
		RefPriceCache:       mapToKeyDecimalPair(e.refPriceCache),
		RefPriceCacheTime:   e.refPriceCacheTime,
	}

	e.stateChanged = false

	return pm
}
