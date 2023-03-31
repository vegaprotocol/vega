// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package price

import (
	"context"
	"errors"
	"log"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var (
	// ErrNilRangeProvider signals that nil was supplied in place of RangeProvider.
	ErrNilRangeProvider = errors.New("nil RangeProvider")
	// ErrTimeSequence signals that time sequence is not in a non-decreasing order.
	ErrTimeSequence = errors.New("received a time that's before the last received time")
	// ErrExpiresAtNotSet indicates price monitoring auction is endless somehow.
	ErrExpiresAtNotSet = errors.New("price monitoring auction with no end time")
	// ErrNilPriceMonitoringSettings signals that nil was supplied in place of PriceMonitoringSettings.
	ErrNilPriceMonitoringSettings = errors.New("nil PriceMonitoringSettings")
)

// can't make this one constant...
var (
	secondsPerYear = num.DecimalFromFloat(365.25 * 24 * 60 * 60)
	tolerance, _   = num.DecimalFromString("1e-6")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/auction_state_mock.go -package mocks code.vegaprotocol.io/vega/core/monitor/price AuctionState
//nolint:interfacebloat
type AuctionState interface {
	// What is the current trading mode of the market, is it in auction
	Mode() types.MarketTradingMode
	InAuction() bool
	// What type of auction are we dealing with
	IsOpeningAuction() bool
	IsLiquidityAuction() bool
	IsPriceAuction() bool
	IsPriceExtension() bool
	IsLiquidityExtension() bool
	IsFBA() bool
	// is it the start/end of the auction
	CanLeave() bool
	AuctionStart() bool
	// start a price-related auction, extend a current auction, or end it
	StartPriceAuction(t time.Time, d *types.AuctionDuration)
	ExtendAuctionPrice(delta types.AuctionDuration)
	SetReadyToLeave()
	// get parameters for current auction
	Start() time.Time
	Duration() types.AuctionDuration // currently not used - might be useful when extending an auction
	ExpiresAt() *time.Time
}

// bound holds the limits for the valid price movement.
type bound struct {
	Active     bool
	UpFactor   num.Decimal
	DownFactor num.Decimal
	Trigger    *types.PriceMonitoringTrigger
}

type boundFactors struct {
	up   []num.Decimal
	down []num.Decimal
}

var (
	defaultDownFactor = num.MustDecimalFromString("0.9")
	defaultUpFactor   = num.MustDecimalFromString("1.1")
)

type priceRange struct {
	MinPrice       num.WrappedDecimal
	MaxPrice       num.WrappedDecimal
	ReferencePrice num.Decimal
}

type pastPrice struct {
	Time                time.Time
	VolumeWeightedPrice num.Decimal
}

type currentPrice struct {
	Price  *num.Uint
	Volume uint64
}

// RangeProvider provides the minimum and maximum future price corresponding to the current price level, horizon expressed as year fraction (e.g. 0.5 for 6 months) and probability level (e.g. 0.95 for 95%).
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_range_provider_mock.go -package mocks code.vegaprotocol.io/vega/core/monitor/price RangeProvider
type RangeProvider interface {
	PriceRange(price, yearFraction, probability num.Decimal) (num.Decimal, num.Decimal)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/state_var_mock.go -package mocks code.vegaprotocol.io/vega/core/monitor/price StateVarEngine
type StateVarEngine interface {
	RegisterStateVariable(asset, market, name string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.EventType, result func(context.Context, statevar.StateVariableResult) error) error
}

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the RangeProvider (risk model).
type Engine struct {
	log          *logging.Logger
	riskModel    RangeProvider
	auctionState AuctionState
	minDuration  time.Duration

	initialised bool
	fpHorizons  map[int64]num.Decimal
	now         time.Time
	update      time.Time
	pricesNow   []currentPrice
	pricesPast  []pastPrice
	bounds      []*bound

	priceRangeCacheTime time.Time
	priceRangesCache    map[*bound]priceRange

	refPriceCacheTime time.Time
	refPriceCache     map[int64]num.Decimal
	refPriceLock      sync.RWMutex

	boundFactorsInitialised bool

	stateChanged   bool
	stateVarEngine StateVarEngine
	market         string
	asset          string
}

func (e *Engine) UpdateSettings(riskModel risk.Model, settings *types.PriceMonitoringSettings) {
	e.riskModel = riskModel
	e.fpHorizons, e.bounds = computeBoundsAndHorizons(settings)
	e.initialised = false
	e.boundFactorsInitialised = false
	e.priceRangesCache = make(map[*bound]priceRange, len(e.bounds)) // clear the cache
	// reset reference cache
	e.refPriceCacheTime = time.Time{}
	e.refPriceCache = map[int64]num.Decimal{}
	_ = e.getCurrentPriceRanges(true) // force bound recalc
}

// Initialised returns true if the engine already saw at least one price.
func (e *Engine) Initialised() bool {
	return e.initialised
}

// NewMonitor returns a new instance of PriceMonitoring.
func NewMonitor(asset, mktID string, riskModel RangeProvider, auctionState AuctionState, settings *types.PriceMonitoringSettings, stateVarEngine StateVarEngine, log *logging.Logger) (*Engine, error) {
	if riskModel == nil {
		return nil, ErrNilRangeProvider
	}
	if settings == nil {
		return nil, ErrNilPriceMonitoringSettings
	}

	// Other functions depend on this sorting
	horizons, bounds := computeBoundsAndHorizons(settings)

	e := &Engine{
		riskModel:               riskModel,
		auctionState:            auctionState,
		fpHorizons:              horizons,
		bounds:                  bounds,
		stateChanged:            true,
		stateVarEngine:          stateVarEngine,
		boundFactorsInitialised: false,
		log:                     log,
		market:                  mktID,
		asset:                   asset,
	}

	stateVarEngine.RegisterStateVariable(asset, mktID, "bound-factors", boundFactorsConverter{}, e.startCalcPriceRanges, []statevar.EventType{statevar.EventTypeTimeTrigger, statevar.EventTypeAuctionEnded, statevar.EventTypeOpeningAuctionFirstUncrossingPrice}, e.updatePriceBounds)
	return e, nil
}

func (e *Engine) SetMinDuration(d time.Duration) {
	e.minDuration = d
	e.stateChanged = true
}

// GetHorizonYearFractions returns horizons of all the triggers specified, expressed as year fraction, sorted in ascending order.
func (e *Engine) GetHorizonYearFractions() []num.Decimal {
	h := make([]num.Decimal, 0, len(e.bounds))
	for _, v := range e.fpHorizons {
		h = append(h, v)
	}

	sort.Slice(h, func(i, j int) bool { return h[i].LessThan(h[j]) })
	return h
}

// GetValidPriceRange returns the range of prices that won't trigger the price monitoring auction.
func (e *Engine) GetValidPriceRange() (num.WrappedDecimal, num.WrappedDecimal) {
	min := num.NewWrappedDecimal(num.UintZero(), num.DecimalZero())
	m := num.MaxUint()
	max := num.NewWrappedDecimal(m, m.ToDecimal())
	for _, pr := range e.getCurrentPriceRanges(false) {
		if pr.MinPrice.Representation().GT(min.Representation()) {
			min = pr.MinPrice
		}
		if !pr.MaxPrice.Representation().IsZero() && pr.MaxPrice.Representation().LT(max.Representation()) {
			max = pr.MaxPrice
		}
	}
	if min.Original().LessThan(num.DecimalZero()) {
		min = num.NewWrappedDecimal(num.UintZero(), num.DecimalZero())
	}
	return min, max
}

// GetCurrentBounds returns a list of valid price ranges per price monitoring trigger. Note these are subject to change as the time progresses.
func (e *Engine) GetCurrentBounds() []*types.PriceMonitoringBounds {
	priceRanges := e.getCurrentPriceRanges(false)
	ret := make([]*types.PriceMonitoringBounds, 0, len(priceRanges))
	for b, pr := range priceRanges {
		if b.Active {
			ret = append(ret,
				&types.PriceMonitoringBounds{
					MinValidPrice:  pr.MinPrice.Representation(),
					MaxValidPrice:  pr.MaxPrice.Representation(),
					Trigger:        b.Trigger,
					ReferencePrice: pr.ReferencePrice,
				})
		}
	}

	sort.SliceStable(ret,
		func(i, j int) bool {
			if ret[i].Trigger.Horizon == ret[j].Trigger.Horizon {
				return ret[i].Trigger.Probability.LessThan(ret[j].Trigger.Probability)
			}
			return ret[i].Trigger.Horizon < ret[j].Trigger.Horizon
		})

	return ret
}

func (e *Engine) OnTimeUpdate(now time.Time) {
	e.recordTimeChange(now)
}

// CheckPrice checks how current price, volume and time should impact the auction state and modifies it accordingly: start auction, end auction, extend ongoing auction,
// "true" gets returned if non-persistent order should be rejected.
func (e *Engine) CheckPrice(ctx context.Context, as AuctionState, trades []*types.Trade, persistent bool) bool {
	// initialise with the first price & time provided, otherwise there won't be any bounds
	wasInitialised := e.initialised
	if !wasInitialised {
		// Volume of 0, do nothing
		if len(trades) == 0 {
			return false
		}
		// only reset history if there isn't any (we need to initialise the engine) or we're still in opening auction as in that case it's based on previous indicative prices which are no longer relevant
		if e.noHistory() || as.IsOpeningAuction() {
			e.resetPriceHistory(trades)
		}
		e.initialised = true
	}
	// market is not in auction, or in batch auction
	if fba := as.IsFBA(); !as.InAuction() || fba {
		bounds := e.checkBounds(trades)
		// no bounds violations - update price, and we're done (unless we initialised as part of this call, then price has alrady been updated)
		if len(bounds) == 0 {
			if wasInitialised {
				e.recordPriceChanges(trades)
			}
			return false
		}
		if !persistent {
			// we're going to stay in continuous trading, make sure we still have bounds
			e.reactivateBounds()
			return true
		}
		duration := types.AuctionDuration{}
		for _, b := range bounds {
			duration.Duration += b.AuctionExtension
		}
		// we're dealing with a batch auction that's about to end -> extend it?
		if fba && as.CanLeave() {
			// bounds were violated, based on the values in the bounds slice, we can calculate how long the auction should last
			as.ExtendAuctionPrice(duration)
			return false
		}
		if min := int64(e.minDuration / time.Second); duration.Duration < min {
			duration.Duration = min
		}

		as.StartPriceAuction(e.now, &duration)
		return false
	}
	// market is in auction
	// opening auction -> ignore
	if as.IsOpeningAuction() {
		e.resetPriceHistory(trades)
		return false
	}

	bounds := e.checkBounds(trades)
	if len(bounds) == 0 {
		// current auction is price monitoring
		// check for end of auction, reset monitoring, and end auction
		if as.IsPriceAuction() || as.IsPriceExtension() {
			end := as.ExpiresAt()
			if !e.now.After(*end) {
				return false
			}
			// auction can be terminated
			as.SetReadyToLeave()
			// reset the engine
			e.resetPriceHistory(trades)
			return false
		}
		// liquidity auction, and it was safe to end -> book is OK, price was OK, reset the engine
		if as.CanLeave() {
			e.reactivateBounds()
		}
		return false
	}

	var duration int64
	for _, b := range bounds {
		duration += b.AuctionExtension
	}

	// extend the current auction
	as.ExtendAuctionPrice(types.AuctionDuration{
		Duration: duration,
	})

	return false
}

// resetPriceHistory deletes existing price history and starts it afresh with the supplied value.
func (e *Engine) resetPriceHistory(trades []*types.Trade) {
	e.update = e.now
	if len(trades) > 0 {
		pricesNow := make([]currentPrice, 0, len(trades))
		for _, t := range trades {
			pricesNow = append(pricesNow, currentPrice{Price: t.Price, Volume: t.Size})
		}
		e.pricesNow = pricesNow
		e.pricesPast = []pastPrice{}
	} else {
		// If there's a price history than use the most recent
		if len(e.pricesPast) > 0 {
			e.pricesPast = e.pricesPast[len(e.pricesPast)-1:]
		} else { // Otherwise can't initialise
			e.initialised = false
			e.stateChanged = true
			return
		}
	}
	e.priceRangeCacheTime = time.Time{}
	e.refPriceCacheTime = time.Time{}
	// we're not reseetting the down/up factors - they will be updated as triggered by auction end/time
	e.reactivateBounds()
	e.stateChanged = true
}

// reactivateBounds reactivates all bounds.
func (e *Engine) reactivateBounds() {
	for _, b := range e.bounds {
		if !b.Active {
			e.stateChanged = true
		}
		b.Active = true
	}
	e.priceRangeCacheTime = time.Time{}
}

// recordPriceChange informs price monitoring module of a price change within the same instance as specified by the last call to UpdateTime.
func (e *Engine) recordPriceChanges(trades []*types.Trade) {
	for _, t := range trades {
		if t.Size > 0 {
			e.pricesNow = append(e.pricesNow, currentPrice{Price: t.Price.Clone(), Volume: t.Size})
			e.stateChanged = true
		}
	}
}

// recordTimeChange updates the current time and moves prices from current prices to past prices by calculating their corresponding vwp.
func (e *Engine) recordTimeChange(now time.Time) {
	if now.Before(e.now) {
		log.Panic("invalid state enecountered in price monitoring engine",
			logging.Error(ErrTimeSequence))
	}
	if now.Equal(e.now) {
		return
	}

	if len(e.pricesNow) > 0 {
		totalWeightedPrice, totalVol := num.UintZero(), num.UintZero()
		for _, x := range e.pricesNow {
			v := num.NewUint(x.Volume)
			totalVol.AddSum(v)
			totalWeightedPrice.AddSum(v.Mul(v, x.Price))
		}
		e.pricesPast = append(e.pricesPast,
			pastPrice{
				Time:                e.now,
				VolumeWeightedPrice: totalWeightedPrice.ToDecimal().Div(totalVol.ToDecimal()),
			})
	}
	e.pricesNow = e.pricesNow[:0]
	e.now = now
	e.clearStalePrices()
	e.stateChanged = true
}

// checkBounds checks if the price is within price range for each of the bound and return trigger for each bound that it's not.
func (e *Engine) checkBounds(trades []*types.Trade) []*types.PriceMonitoringTrigger {
	ret := []*types.PriceMonitoringTrigger{} // returned price projections, empty if all good
	if len(trades) == 0 {
		return ret // volume 0 so no bounds violated
	}
	priceRanges := e.getCurrentPriceRanges(false)
	for _, t := range trades {
		if t.Size == 0 {
			continue
		}
		for _, b := range e.bounds {
			if !b.Active {
				continue
			}
			p := t.Price
			priceRange := priceRanges[b]
			if p.LT(priceRange.MinPrice.Representation()) || p.GT(priceRange.MaxPrice.Representation()) {
				ret = append(ret, b.Trigger)
				// deactivate the bound that just got violated so it doesn't prevent auction from terminating
				b.Active = false
			}
		}
	}
	return ret
}

// getCurrentPriceRanges calculates price ranges from current reference prices and bound down/up factors.
func (e *Engine) getCurrentPriceRanges(force bool) map[*bound]priceRange {
	if !force && e.priceRangeCacheTime == e.now && len(e.priceRangesCache) > 0 {
		return e.priceRangesCache
	}
	ranges := make(map[*bound]priceRange, len(e.priceRangesCache))
	if e.noHistory() {
		return ranges
	}
	for _, b := range e.bounds {
		if !b.Active {
			continue
		}
		if e.monitoringAuction() && len(e.pricesPast)+len(e.pricesNow) > 0 {
			triggerLookback := e.now.Add(time.Duration(-b.Trigger.Horizon) * time.Second)
			// check if trigger's not stale (newest reference price older than horizon lookback time)
			var mostRecentObservation time.Time
			if len(e.pricesNow) > 0 {
				mostRecentObservation = e.now
			} else {
				x := e.pricesPast[len(e.pricesPast)-1]
				mostRecentObservation = x.Time
			}
			if mostRecentObservation.Before(triggerLookback) {
				b.Active = false
				continue
			}
		}
		ref := e.getRefPrice(b.Trigger.Horizon, force)
		var min, max num.Decimal

		if e.boundFactorsInitialised {
			min = ref.Mul(b.DownFactor)
			max = ref.Mul(b.UpFactor)
		} else {
			min = ref.Mul(defaultDownFactor)
			max = ref.Mul(defaultUpFactor)
		}

		ranges[b] = priceRange{
			MinPrice:       wrapPriceRange(min, true),
			MaxPrice:       wrapPriceRange(max, false),
			ReferencePrice: ref,
		}
	}
	e.priceRangesCache = ranges
	e.priceRangeCacheTime = e.now
	e.stateChanged = true
	return e.priceRangesCache
}

func (e *Engine) monitoringAuction() bool {
	return e.auctionState.IsLiquidityAuction() || e.auctionState.IsPriceAuction()
}

// clearStalePrices updates the pricesPast slice to hold only as many prices as implied by the horizon.
func (e *Engine) clearStalePrices() {
	if e.now.Before(e.update) || len(e.bounds) == 0 || len(e.pricesPast) == 0 {
		return
	}

	// Remove redundant average prices
	minRequiredHorizon := e.now
	if len(e.bounds) > 0 {
		maxTau := e.bounds[len(e.bounds)-1].Trigger.Horizon
		minRequiredHorizon = e.now.Add(time.Duration(-maxTau) * time.Second)
	}

	// Make sure at least one entry is left hence the "len(..) - 1"
	for i := 0; i < len(e.pricesPast)-1; i++ {
		if !e.pricesPast[i].Time.Before(minRequiredHorizon) {
			e.pricesPast = e.pricesPast[i:]
			return
		}
	}
	e.pricesPast = e.pricesPast[len(e.pricesPast)-1:]
}

// getRefPrice caches and returns the ref price for a given horizon. The cache is invalidated when block changes.
func (e *Engine) getRefPrice(horizon int64, force bool) num.Decimal {
	e.refPriceLock.Lock()
	defer e.refPriceLock.Unlock()
	if e.refPriceCache == nil || e.refPriceCacheTime != e.now || force {
		e.refPriceCache = make(map[int64]num.Decimal, len(e.refPriceCache))
		e.stateChanged = true
		e.refPriceCacheTime = e.now
	}

	if _, ok := e.refPriceCache[horizon]; !ok {
		e.refPriceCache[horizon] = e.calculateRefPrice(horizon)
		e.stateChanged = true
	}
	return e.refPriceCache[horizon]
}

func (e *Engine) getRefPriceNoUpdate(horizon int64) num.Decimal {
	e.refPriceLock.RLock()
	defer e.refPriceLock.RUnlock()
	if e.refPriceCacheTime == e.now {
		if _, ok := e.refPriceCache[horizon]; !ok {
			return e.calculateRefPrice(horizon)
		}
		return e.refPriceCache[horizon]
	}
	return e.calculateRefPrice(horizon)
}

// calculateRefPrice returns theh last VolumeWeightedPrice with time preceding currentTime - horizon seconds. If there's only one price it returns the Price.
func (e *Engine) calculateRefPrice(horizon int64) num.Decimal {
	t := e.now.Add(time.Duration(-horizon) * time.Second)
	if len(e.pricesPast) < 1 {
		return e.pricesNow[0].Price.ToDecimal()
	}
	ref := e.pricesPast[0].VolumeWeightedPrice
	for _, p := range e.pricesPast {
		if p.Time.After(t) {
			break
		}
		ref = p.VolumeWeightedPrice
	}
	return ref
}

func (e *Engine) noHistory() bool {
	return len(e.pricesPast) == 0 && len(e.pricesNow) == 0
}

func computeBoundsAndHorizons(settings *types.PriceMonitoringSettings) (map[int64]num.Decimal, []*bound) {
	parameters := make([]*types.PriceMonitoringTrigger, 0, len(settings.Parameters.Triggers))
	for _, p := range settings.Parameters.Triggers {
		p := *p
		parameters = append(parameters, &p)
	}
	sort.Slice(parameters,
		func(i, j int) bool {
			return parameters[i].Horizon < parameters[j].Horizon &&
				parameters[i].Probability.GreaterThanOrEqual(parameters[j].Probability)
		})

	horizons := map[int64]num.Decimal{}
	bounds := make([]*bound, 0, len(parameters))
	for _, p := range parameters {
		bounds = append(bounds, &bound{
			Active:  true,
			Trigger: p,
		})
		if _, ok := horizons[p.Horizon]; !ok {
			horizons[p.Horizon] = p.HorizonDec.Div(secondsPerYear)
		}
	}
	return horizons, bounds
}

func wrapPriceRange(b num.Decimal, isMin bool) num.WrappedDecimal {
	var r *num.Uint
	if isMin {
		r, _ = num.UintFromDecimal(b.Ceil())
	} else {
		r, _ = num.UintFromDecimal(b.Floor())
	}
	return num.NewWrappedDecimal(r, b)
}
