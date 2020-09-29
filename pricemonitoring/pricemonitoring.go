package pricemonitoring

import (
	"context"
	"errors"
	"sort"
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrProbability gets thrown when probability is outside the (0,1)
	ErrProbability = errors.New("probability level must be in the interval (0,1)")
	// ErrTimeSequence signals that time sequence is not in a non-decreasing order
	ErrTimeSequence = errors.New("received a time that's before the last received time")
	// ErrHorizonNotInFuture signals that the time horizon is not positive
	ErrHorizonNotInFuture = errors.New("horizon must be represented by a positive duration")
	// ErrUpdateFrequencyNotPositive signals that update frequency isn't positive.
	ErrUpdateFrequencyNotPositive = errors.New("update frequency must be represented by a positive duration")
)

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
	ActionStart() bool
	// start a price-related auction, extend a current auction, or end it
	StartPriceAuction(t time.Time, d *types.AuctionDuration)
	ExtendAuction(delta types.AuctionDuration)
	EndAuction()
	// get parameters for current auction
	Start() time.Time
	Duration() types.AuctionDuration
}

// PriceProjection ties the horizon τ and probability p level.
// It's used to check if price over τ has exceeded the probability level p implied by the risk model
// (e.g. τ = 1 hour, p = 95%)
type PriceProjection struct {
	Horizon     time.Duration
	Probability float64
}

// NewPriceProjection returns a new instance of PriceProjection
// if probability level is in the range (0,1) and horizon is in the future and an error otherwise
func NewPriceProjection(horizon time.Duration, probability float64) (*PriceProjection, error) {
	p := PriceProjection{Horizon: horizon, Probability: probability}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Validate returns an error if probability level is not the range (0,1) or horizon is not in the future and nil otherwise
func (p PriceProjection) Validate() error {
	if p.Probability <= 0 || p.Probability >= 1 {
		return ErrProbability
	}
	if p.Horizon <= 0 {
		return ErrHorizonNotInFuture
	}
	return nil
}

type priceMoveBound struct {
	MaxMoveUp   float64
	MinMoveDown float64
}

type pastPrice struct {
	Time         time.Time
	AveragePrice float64
}

// PriceRangeProvider provides the minimium and maximum future price corresponding to the current price level, horizon expressed as year fraction (e.g. 0.5 for 6 months) and probability level (e.g. 0.95 for 95%).
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_range_provider_mock.go -package mocks code.vegaprotocol.io/vega/pricemonitoring PriceRangeProvider
type PriceRangeProvider interface {
	PriceRange(price float64, yearFraction float64, probability float64) (minPrice float64, maxPrice float64)
}

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the PriceRangeProvider (risk model).
type Engine struct {
	riskModel       PriceRangeProvider
	projections     []PriceProjection
	updateFrequency time.Duration

	fpHorizons map[time.Duration]float64
	now        time.Time
	update     time.Time
	pricesNow  []uint64
	pricesPast []pastPrice
	bounds     map[PriceProjection]priceMoveBound
}

// NewPriceMonitoring returns a new instance of PriceMonitoring.
func NewPriceMonitoring(riskModel PriceRangeProvider, projections []PriceProjection, updateFrequency time.Duration, price uint64, now time.Time) (*Engine, error) {
	if updateFrequency <= 0 {
		return nil, ErrUpdateFrequencyNotPositive
	}

	// Other functions depend on this sorting
	sort.Slice(projections,
		func(i, j int) bool {
			return projections[i].Horizon < projections[j].Horizon &&
				projections[i].Probability >= projections[j].Probability
		})

	h := make(map[time.Duration]float64)
	year := 365.25 * 24 * time.Hour
	for _, p := range projections {
		if err := p.Validate(); err != nil {
			return nil, err
		}
		if _, ok := h[p.Horizon]; !ok {
			if p.Horizon == 0 {
				return nil, ErrHorizonNotInFuture
			}
			h[p.Horizon] = float64(p.Horizon) / float64(year)
		}
	}
	e := &Engine{
		riskModel:       riskModel,
		projections:     projections,
		fpHorizons:      h,
		updateFrequency: updateFrequency,
	}
	e.Reset(price, now)
	return e, nil
}

func (e *Engine) CheckPrice(ctx context.Context, as AuctionState, p uint64, now time.Time) error {
	// market is not in auction, or in batch auction
	if fba := as.IsFBA(); !as.InAuction() || fba {
		if err := e.RecordTimeChange(now); err != nil {
			return err
		}
		bounds := e.checkBounds(ctx, p)
		// no bounds violations - update price, and we're done
		if len(bounds) == 0 {
			e.RecordPriceChange(p)
			return nil
		}
		// bounds were violated, based on the values in the bounds slice, we can calculate how long the auction should last
		// @TODO placeholder - this data comes from market def, but let's just say 5 min per bound here:
		end := types.AuctionDuration{
			Duration: int64(len(bounds)) * 5 * int64(time.Minute),
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
	// current auction is price monitoring
	// check for end of auction, reset monitoring, and end auction
	if as.IsPriceAuction() {
		start, dur := as.Start(), as.Duration()
		// auction still hasn't ended yet
		if end := start.Add(time.Duration(dur.Duration)); end.After(now) {
			return nil
		}
		// auction can be terminated
		as.EndAuction()
		// reset the engine
		e.Reset(p, now)
		return nil
	}
	// market is in auction mode, liquidity
	if err := e.RecordTimeChange(now); err != nil {
		return err
	}
	bounds := e.checkBounds(ctx, p)
	if len(bounds) == 0 {
		return nil
	}
	// let's say we need to extend this auction (in reality, liquidity can extend price, but not the other way around IIRC)
	as.ExtendAuction(types.AuctionDuration{
		Duration: int64(len(bounds)) * int64(time.Minute),
	})
	return nil
}

// Reset restarts price monitoring with a new price. All previously recorded prices and previously obtained bounds get deleted.
func (e *Engine) Reset(price uint64, now time.Time) {
	e.now = now
	e.pricesNow = []uint64{price}
	e.pricesPast = []pastPrice{}
	e.bounds = map[PriceProjection]priceMoveBound{}
	e.update = now
	e.updateBounds()
}

// RecordPriceChange informs price monitoring module of a price change within the same instance as specified by the last call to UpdateTime
func (e *Engine) RecordPriceChange(price uint64) {
	e.pricesNow = append(e.pricesNow, price)
}

// RecordTimeChange updates the time in the price monitoring module and returns an error if any problems are encountered.
func (e *Engine) RecordTimeChange(now time.Time) error {
	if now.Before(e.now) {
		return ErrTimeSequence // This shouldn't happen, but if it does there's something fishy going on
	}
	if now.After(e.now) {
		var sum uint64 = 0
		for _, x := range e.pricesNow {
			sum += x
		}
		e.pricesPast = append(e.pricesPast,
			pastPrice{
				Time:         e.now,
				AveragePrice: float64(sum) / float64(len(e.pricesNow)),
			})
		e.pricesNow = make([]uint64, 0, cap(e.pricesNow))
		e.now = now
		e.updateBounds()
	}
	return nil
}

func (e *Engine) checkBounds(ctx context.Context, p uint64) []PriceProjection {
	fp := float64(p)
	ret := []PriceProjection{} // returned price projections, empty if all good
	var (
		ph  time.Duration // previous horizon
		ref float64       // reference price
	)
	for _, p := range e.projections {
		if p.Horizon != ph {
			ph = p.Horizon
			ref = e.getReferencePrice(e.now.Add(-ph))
		}

		diff := fp - ref
		b := e.bounds[p]
		if diff < b.MinMoveDown || diff > b.MaxMoveUp {
			ret = append(ret, p)
		}
	}
	return ret
}

// CheckBoundViolations returns a map of horizon and probability level pair to boolean.
// A true value indicates that a bound corresponding to a given horizon and probability level pair has been violated.
func (e *Engine) CheckBoundViolations(price uint64) map[PriceProjection]bool {
	fpPrice := float64(price)
	checks := make(map[PriceProjection]bool, len(e.projections))
	var prevHorizon time.Duration
	var ref float64
	for _, p := range e.projections {
		// horizonProbabilityLevelPairs are sorted by Horizon to avoid repeated price lookup
		if p.Horizon != prevHorizon {
			ref = e.getReferencePrice(e.now.Add(-p.Horizon))
			prevHorizon = p.Horizon
		}

		priceDiff := fpPrice - ref
		bounds := e.bounds[p]
		checks[p] = priceDiff < bounds.MinMoveDown || priceDiff > bounds.MaxMoveUp
	}
	return checks
}

func (e *Engine) updateBounds() {
	if e.now.Before(e.update) {
		return
	}

	e.update = e.now.Add(e.updateFrequency)
	var latestPrice float64
	if len(e.pricesPast) == 0 {
		latestPrice = float64(e.pricesNow[len(e.pricesNow)-1])
	} else {
		latestPrice = e.pricesPast[len(e.pricesPast)-1].AveragePrice
	}
	for _, p := range e.projections {

		minPrice, maxPrice := e.riskModel.PriceRange(latestPrice, e.fpHorizons[p.Horizon], p.Probability)
		e.bounds[p] = priceMoveBound{MinMoveDown: minPrice - latestPrice, MaxMoveUp: maxPrice - latestPrice}
	}
	// Remove redundant average prices
	maxTau := e.projections[len(e.projections)-1].Horizon
	minRequiredHorizon := e.now.Add(-maxTau)
	var i int
	// Make sure at least one entry is left hence the "len(..) - 1"
	for i = 0; i < len(e.pricesPast)-1; i++ {
		if !e.pricesPast[i].Time.Before(minRequiredHorizon) {
			break
		}
		e.pricesPast[i] = pastPrice{} //TODO (WG): Confirm if this is needed to reclaim memory
	}
	e.pricesPast = e.pricesPast[i:]
}

func (e *Engine) getReferencePrice(t time.Time) float64 {
	var ref float64
	if len(e.pricesPast) < 1 {
		ref = float64(e.pricesNow[0])
	} else {
		ref = e.pricesPast[0].AveragePrice
	}
	for _, p := range e.pricesPast {
		if p.Time.After(t) {
			break
		}
		ref = p.AveragePrice
	}
	return ref
}
