package price

import (
	"context"
	"errors"
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
	MaxMoveUp   float64
	MinMoveDown float64
}

type pastPrice struct {
	Time         time.Time
	AveragePrice float64
}

// RangeProvider provides the minimium and maximum future price corresponding to the current price level, horizon expressed as year fraction (e.g. 0.5 for 6 months) and probability level (e.g. 0.95 for 95%).
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_range_provider_mock.go -package mocks code.vegaprotocol.io/vega/monitor/price RangeProvider
type RangeProvider interface {
	PriceRange(price, yearFraction, probability float64) (float64, float64)
}

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the RangeProvider (risk model).
type Engine struct {
	riskModel       RangeProvider
	parameters      []*types.PriceMonitoringTrigger
	updateFrequency time.Duration

	initialised bool
	fpHorizons  map[int64]float64
	now         time.Time
	update      time.Time
	pricesNow   []uint64
	pricesPast  []pastPrice
	bounds      map[*types.PriceMonitoringTrigger]bound
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
	e := &Engine{
		riskModel:       riskModel,
		parameters:      parameters,
		fpHorizons:      h,
		updateFrequency: time.Duration(settings.UpdateFrequency) * time.Second,
	}
	return e, nil
}

// CheckPrice checks how current price and time should impact the auction state and modifies it accordingly: start auction, end auction, extend ongoing auction
func (e *Engine) CheckPrice(ctx context.Context, as AuctionState, p uint64, now time.Time) error {
	// initialise with the first price & time provided, otherwise there won't be any bounds
	wasInitialised := e.initialised
	if !wasInitialised {
		e.reset(p, now)
		e.initialised = true
	}

	// market is not in auction, or in batch auction
	if fba := as.IsFBA(); !as.InAuction() || fba {
		if err := e.recordTimeChange(now); err != nil {
			return err
		}
		bounds := e.checkBounds(ctx, p)
		// no bounds violations - update price, and we're done (unless we initialised as part of this call, then price has alrady been updated)
		if len(bounds) == 0 {
			if wasInitialised {
				e.recordPriceChange(p)
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

	bounds := e.checkBounds(ctx, p)
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
			e.reset(p, now)
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
func (e *Engine) reset(price uint64, now time.Time) {
	e.now = now
	e.update = now
	e.pricesNow = []uint64{price}
	e.pricesPast = []pastPrice{}
	e.initilizeBounds()
	e.updateBounds()
}

func (e *Engine) initilizeBounds() {
	e.bounds = make(map[*types.PriceMonitoringTrigger]bound, len(e.parameters))
	for _, p := range e.parameters {
		e.bounds[p] = bound{}
	}
}

// recordPriceChange informs price monitoring module of a price change within the same instance as specified by the last call to UpdateTime
func (e *Engine) recordPriceChange(price uint64) {
	e.pricesNow = append(e.pricesNow, price)
}

// recordTimeChange updates the time in the price monitoring module and returns an error if any problems are encountered.
func (e *Engine) recordTimeChange(now time.Time) error {
	if now.Before(e.now) {
		return ErrTimeSequence // This shouldn't happen, but if it does there's something fishy going on
	}
	if now.After(e.now) {
		if len(e.pricesNow) > 0 {
			var sum uint64 = 0
			for _, x := range e.pricesNow {
				sum += x
			}
			e.pricesPast = append(e.pricesPast,
				pastPrice{
					Time:         e.now,
					AveragePrice: float64(sum) / float64(len(e.pricesNow)),
				})
		}
		e.pricesNow = e.pricesNow[:0]
		e.now = now
		e.updateBounds()
	}
	return nil
}

func (e *Engine) checkBounds(ctx context.Context, p uint64) []*types.PriceMonitoringTrigger {
	var (
		fp  float64                         = float64(p)                        // price as float
		ph  int64                                                               // previous horizon
		ref float64                                                             // reference price
		ret []*types.PriceMonitoringTrigger = []*types.PriceMonitoringTrigger{} // returned price projections, empty if all good
	)
	for _, p := range e.parameters {
		b, ok := e.bounds[p]
		if !ok {
			continue
		}

		if p.Horizon != ph {
			ph = p.Horizon
			ref = e.getReferencePrice(e.now.Add(time.Duration(-ph) * time.Second))
		}

		diff := fp - ref
		if diff < b.MinMoveDown || diff > b.MaxMoveUp {
			ret = append(ret, p)
			// Remove bound that gets violated so it doesn't prevent auction from terminating
			delete(e.bounds, p)
		}
	}
	return ret
}

func (e *Engine) updateBounds() {
	if e.now.Before(e.update) || len(e.parameters) == 0 {
		return
	}

	// Iterate update time until in the future
	for !e.update.After(e.now) {
		e.update = e.update.Add(e.updateFrequency)
	}

	var latestPrice float64
	if len(e.pricesPast) == 0 {
		latestPrice = float64(e.pricesNow[len(e.pricesNow)-1])
	} else {
		latestPrice = e.pricesPast[len(e.pricesPast)-1].AveragePrice
	}
	for p := range e.bounds {

		minPrice, maxPrice := e.riskModel.PriceRange(latestPrice, e.fpHorizons[p.Horizon], p.Probability)
		e.bounds[p] = bound{MinMoveDown: minPrice - latestPrice, MaxMoveUp: maxPrice - latestPrice}
	}
	// Remove redundant average prices
	minRequiredHorizon := e.now
	if len(e.parameters) > 0 {
		maxTau := e.parameters[len(e.parameters)-1].Horizon
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
