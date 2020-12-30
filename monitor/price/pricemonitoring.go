package price

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrNilRangeProvider signals that nil was supplied in place of RangeProvider
	ErrNilRangeProvider = errors.New("nil RangeProvider")
	// ErrTimeSequence signals that time sequence is not in a non-decreasing order
	ErrTimeSequence = errors.New("received a time that's before the last received time")
	// ErrExpiresAtNotSet indicates price monitoring auction is endless somehow
	ErrExpiresAtNotSet = errors.New("price monitoring auction with no end time")
)

const (
	secondsPerYear = 365.25 * 24 * 60 * 60
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/auction_state_mock.go -package mocks code.vegaprotocol.io/vega/monitor/price AuctionState
type AuctionState interface {
	// What is the current trading mode of the market, is it in auction
	Mode() types.MarketState
	InAuction() bool
	// What type of auction are we dealing with
	IsOpeningAuction() bool
	IsLiquidityAuction() bool
	IsPriceAuction() bool
	IsFBA() bool
	// is it the start/end of the auction
	AuctionEnd() bool
	AuctionStart() bool
	// start a price-related auction, extend a current auction, or end it
	StartPriceAuction(t time.Time, d *types.AuctionDuration)
	ExtendAuction(delta types.AuctionDuration)
	EndAuction()
	// get parameters for current auction
	Start() time.Time
	Duration() types.AuctionDuration // currently not used - might be useful when extending an auction
	ExpiresAt() *time.Time
}

// bound holds the limits for the valid price movement
type bound struct {
	Active     bool
	UpFactor   float64
	DownFactor float64
	Trigger    *types.PriceMonitoringTrigger
}

type priceRange struct {
	MinPrice       float64
	MaxPrice       float64
	ReferencePrice float64
}

type pastPrice struct {
	Time                time.Time
	VolumeWeightedPrice float64
}

type currentPrice struct {
	Price  uint64
	Volume uint64
}

// RangeProvider provides the minimium and maximum future price corresponding to the current price level, horizon expressed as year fraction (e.g. 0.5 for 6 months) and probability level (e.g. 0.95 for 95%).
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_range_provider_mock.go -package mocks code.vegaprotocol.io/vega/monitor/price RangeProvider
type RangeProvider interface {
	PriceRange(price, yearFraction, probability float64) (float64, float64)
}

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the RangeProvider (risk model).
type Engine struct {
	riskModel       RangeProvider
	updateFrequency time.Duration

	initialised bool
	fpHorizons  map[int64]float64
	now         time.Time
	update      time.Time
	pricesNow   []currentPrice
	pricesPast  []pastPrice
	bounds      []*bound

	priceRangeCacheTime time.Time
	priceRangesCache    map[*bound]priceRange

	refPriceCacheTime time.Time
	refPriceCache     map[int64]float64
}

// NewMonitor returns a new instance of PriceMonitoring.
func NewMonitor(riskModel RangeProvider, settings types.PriceMonitoringSettings) (*Engine, error) {
	if riskModel == nil {
		return nil, ErrNilRangeProvider
	}

	parameters := make([]*types.PriceMonitoringTrigger, 0, len(settings.Parameters.Triggers))
	for _, p := range settings.Parameters.Triggers {
		p := *p
		parameters = append(parameters, &p)
	}

	// Other functions depend on this sorting
	sort.Slice(parameters,
		func(i, j int) bool {
			return parameters[i].Horizon < parameters[j].Horizon &&
				parameters[i].Probability >= parameters[j].Probability
		})

	h := map[int64]float64{}
	for _, p := range parameters {
		if _, ok := h[p.Horizon]; !ok {
			h[p.Horizon] = float64(p.Horizon) / secondsPerYear
		}
	}

	bounds := make([]*bound, 0, len(parameters))
	for _, p := range parameters {
		bounds = append(bounds, &bound{Active: true, Trigger: p})
	}

	e := &Engine{
		riskModel:       riskModel,
		fpHorizons:      h,
		updateFrequency: time.Duration(settings.UpdateFrequency) * time.Second,
		bounds:          bounds,
	}
	return e, nil
}

// GetHorizonYearFractions returns horizons of all the triggers specifed, expressed as year fraction, sorted in ascending order.
func (e *Engine) GetHorizonYearFractions() []float64 {
	h := make([]float64, 0, len(e.bounds))
	for _, v := range e.fpHorizons {
		h = append(h, v)
	}

	sort.Slice(h, func(i, j int) bool { return h[i] < h[j] })
	return h
}

// GetValidPriceRange returns the range of prices that won't trigger the price monitoring auction
func (e *Engine) GetValidPriceRange() (float64, float64) {
	min := -math.MaxFloat64
	max := math.MaxFloat64
	for _, pr := range e.getCurrentPriceRanges() {
		if pr.MinPrice > min {
			min = pr.MinPrice
		}
		if pr.MaxPrice < max {
			max = pr.MaxPrice
		}
	}
	return min, max
}

// GetCurrentBounds returns a list of valid price ranges per price monitoring trigger. Note these are subject to change as the time progresses.
func (e *Engine) GetCurrentBounds() []*types.PriceMonitoringBounds {
	priceRanges := e.getCurrentPriceRanges()
	ret := make([]*types.PriceMonitoringBounds, 0, len(priceRanges))
	for b, pr := range priceRanges {
		if b.Active {
			ret = append(ret,
				&types.PriceMonitoringBounds{
					MinValidPrice:  uint64(math.Ceil(pr.MinPrice)),
					MaxValidPrice:  uint64(math.Floor(pr.MaxPrice)),
					Trigger:        b.Trigger,
					ReferencePrice: pr.ReferencePrice})
		}
	}
	sort.SliceStable(ret,
		func(i, j int) bool {
			return ret[i].Trigger.Horizon <= ret[j].Trigger.Horizon &&
				ret[i].Trigger.Probability <= ret[j].Trigger.Probability
		})
	return ret
}

// CheckPrice checks how current price, volume and time should impact the auction state and modifies it accordingly: start auction, end auction, extend ongoing auction
func (e *Engine) CheckPrice(ctx context.Context, as AuctionState, p uint64, v uint64, now time.Time) error {
	// initialise with the first price & time provided, otherwise there won't be any bounds
	wasInitialised := e.initialised
	if !wasInitialised {
		//Volume of 0, do nothing
		if v == 0 {
			return nil
		}
		e.reset(p, v, now)
		e.initialised = true
	}

	// market is not in auction, or in batch auction
	if fba := as.IsFBA(); !as.InAuction() || fba {
		if err := e.recordTimeChange(now); err != nil {
			return err
		}
		bounds := e.checkBounds(ctx, p, v)
		// no bounds violations - update price, and we're done (unless we initialised as part of this call, then price has alrady been updated)
		if len(bounds) == 0 {
			if wasInitialised {
				e.recordPriceChange(p, v)
			}
			return nil
		}
		// bounds were violated, based on the values in the bounds slice, we can calculate how long the auction should last
		var duration int64
		for _, b := range bounds {
			duration += b.AuctionExtension
		}

		end := types.AuctionDuration{
			Duration: duration,
		}
		// we're dealing with a batch auction that's about to end -> extend it?
		if fba && as.AuctionEnd() {
			as.ExtendAuction(end)
			return nil // we could return an error here to indicate the batch auction was altered?
		}
		// setup auction
		as.StartPriceAuction(now, &end)
		return nil
	}
	// market is in auction

	// opening auction -> ignore
	if as.IsOpeningAuction() {
		return nil
	}

	if err := e.recordTimeChange(now); err != nil {
		return err
	}

	bounds := e.checkBounds(ctx, p, v)
	if len(bounds) == 0 {
		// current auction is price monitoring
		// check for end of auction, reset monitoring, and end auction
		if as.IsPriceAuction() {
			end := as.ExpiresAt()
			if end == nil {
				return ErrExpiresAtNotSet
			}
			if !now.After(*end) {
				return nil
			}
			// auction can be terminated
			as.EndAuction()
			// reset the engine
			e.reset(p, v, now)
		}
		return nil
	}

	var duration int64
	for _, b := range bounds {
		duration += b.AuctionExtension
	}

	// extend the current auction
	as.ExtendAuction(types.AuctionDuration{
		Duration: duration,
	})

	return nil
}

// reset restarts price monitoring with a new price. All previously recorded prices and previously obtained bounds get deleted.
func (e *Engine) reset(price uint64, volume uint64, now time.Time) {
	e.now = now
	e.update = now
	if volume > 0 {
		e.pricesNow = []currentPrice{{Price: price, Volume: volume}}
		e.pricesPast = []pastPrice{}
	} else {
		// If there's a price history than use the most recent
		if len(e.pricesPast) > 0 {
			e.pricesPast = e.pricesPast[len(e.pricesPast)-1:]
		} else { // Otherwise can't initialise
			e.initialised = false
			return
		}
	}
	e.priceRangeCacheTime = time.Time{}
	e.resetBounds()
	e.updateBounds()

}

func (e *Engine) resetBounds() {
	for _, b := range e.bounds {
		b.Active = true
		b.DownFactor = 0
		b.UpFactor = 0
	}
}

// recordPriceChange informs price monitoring module of a price change within the same instance as specified by the last call to UpdateTime
func (e *Engine) recordPriceChange(price uint64, volume uint64) {
	if volume > 0 {
		e.pricesNow = append(e.pricesNow, currentPrice{Price: price, Volume: volume})
	}
}

// recordTimeChange updates the time in the price monitoring module and returns an error if any problems are encountered.
func (e *Engine) recordTimeChange(now time.Time) error {
	if now.Before(e.now) {
		return ErrTimeSequence // This shouldn't happen, but if it does there's something fishy going on
	}
	if now.After(e.now) {
		if len(e.pricesNow) > 0 {
			var sumProduct uint64 = 0
			var volumeSum uint64 = 0
			for _, x := range e.pricesNow {
				sumProduct += x.Price * x.Volume
				volumeSum += x.Volume
			}
			e.pricesPast = append(e.pricesPast,
				pastPrice{
					Time:                e.now,
					VolumeWeightedPrice: float64(sumProduct) / float64(volumeSum),
				})
		}
		e.pricesNow = e.pricesNow[:0]
		e.now = now
		e.updateBounds()
	}
	return nil
}

func (e *Engine) checkBounds(ctx context.Context, p uint64, v uint64) []*types.PriceMonitoringTrigger {
	var ret []*types.PriceMonitoringTrigger = []*types.PriceMonitoringTrigger{} // returned price projections, empty if all good
	if v == 0 {
		return ret //volume 0 so no bounds violated
	}
	var fp float64 = float64(p) // price as float
	priceRanges := e.getCurrentPriceRanges()
	for _, b := range e.bounds {
		if !b.Active {
			continue
		}
		priceRange := priceRanges[b]
		if fp < priceRange.MinPrice || fp > priceRange.MaxPrice {
			ret = append(ret, b.Trigger)
			// Disactivate the bound that just got violated so it doesn't prevent auction from terminating
			b.Active = false
		}
	}
	return ret
}

func (e *Engine) getCurrentPriceRanges() map[*bound]priceRange {
	if e.priceRangeCacheTime != e.now {
		e.priceRangesCache = make(map[*bound]priceRange, len(e.priceRangesCache))

		for _, b := range e.bounds {
			if !b.Active {
				continue
			}
			ref := e.getRefPrice(b.Trigger.Horizon)
			e.priceRangesCache[b] = priceRange{MinPrice: ref * b.DownFactor, MaxPrice: ref * b.UpFactor, ReferencePrice: ref}
		}
		e.priceRangeCacheTime = e.now
	}
	return e.priceRangesCache
}

func (e *Engine) updateBounds() {
	if e.now.Before(e.update) || len(e.bounds) == 0 {
		return
	}

	// Iterate update time until in the future
	for !e.update.After(e.now) {
		e.update = e.update.Add(e.updateFrequency)
	}

	for _, b := range e.bounds {
		if !b.Active {
			continue
		}
		ref := e.getRefPrice(b.Trigger.Horizon)
		minPrice, maxPrice := e.riskModel.PriceRange(ref, e.fpHorizons[b.Trigger.Horizon], b.Trigger.Probability)
		b.DownFactor = minPrice / ref
		b.UpFactor = maxPrice / ref
	}
	// Remove redundant average prices
	minRequiredHorizon := e.now
	if len(e.bounds) > 0 {
		maxTau := e.bounds[len(e.bounds)-1].Trigger.Horizon
		minRequiredHorizon = e.now.Add(time.Duration(-maxTau) * time.Second)
	}

	var i int
	// Make sure at least one entry is left hence the "len(..) - 1"
	for i = 0; i < len(e.pricesPast)-1; i++ {
		if !e.pricesPast[i].Time.Before(minRequiredHorizon) {
			break
		}
	}
	e.pricesPast = e.pricesPast[i:]
}

func (e *Engine) getRefPrice(horizon int64) float64 {
	if e.refPriceCacheTime != e.now {
		e.refPriceCache = make(map[int64]float64, len(e.refPriceCache))
	}

	if _, ok := e.refPriceCache[horizon]; !ok {
		e.refPriceCache[horizon] = e.calculateRefPrice(horizon)
	}
	return e.refPriceCache[horizon]
}

func (e *Engine) calculateRefPrice(horizon int64) float64 {
	t := e.now.Add(time.Duration(-horizon) * time.Second)
	var ref float64
	if len(e.pricesPast) < 1 {
		ref = float64(e.pricesNow[0].Price)
	} else {
		ref = e.pricesPast[0].VolumeWeightedPrice
	}
	for _, p := range e.pricesPast {
		if p.Time.After(t) {
			break
		}
		ref = p.VolumeWeightedPrice
	}
	return ref
}
