package price

import (
	"context"
	"errors"
	"sort"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
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

//go:generate go run github.com/golang/mock/mockgen -destination mocks/auction_state_mock.go -package mocks code.vegaprotocol.io/vega/monitor/price AuctionState
type AuctionState interface {
	// What is the current trading mode of the market, is it in auction
	Mode() types.MarketTradingMode
	InAuction() bool
	// What type of auction are we dealing with
	IsOpeningAuction() bool
	IsLiquidityAuction() bool
	IsPriceAuction() bool
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

type boundFactorsConverter struct{}

func (boundFactorsConverter) BundleToInterface(kvb *statevar.KeyValueBundle) statevar.StateVariableResult {
	return &boundFactors{
		up:   kvb.KVT[0].Val.(*statevar.DecimalVector).Val,
		down: kvb.KVT[1].Val.(*statevar.DecimalVector).Val,
	}
}

func (boundFactorsConverter) InterfaceToBundle(res statevar.StateVariableResult) *statevar.KeyValueBundle {
	value := res.(*boundFactors)
	return &statevar.KeyValueBundle{
		KVT: []statevar.KeyValueTol{
			{Key: "up", Val: &statevar.DecimalVector{Val: value.up}, Tolerance: tolerance},
			{Key: "down", Val: &statevar.DecimalVector{Val: value.down}, Tolerance: tolerance},
		},
	}
}

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
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_range_provider_mock.go -package mocks code.vegaprotocol.io/vega/monitor/price RangeProvider
type RangeProvider interface {
	PriceRange(price, yearFraction, probability num.Decimal) (num.Decimal, num.Decimal)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/state_var_mock.go -package mocks code.vegaprotocol.io/vega/monitor/price StateVarEngine
type StateVarEngine interface {
	AddStateVariable(asset, market string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.StateVarEventType, result func(context.Context, statevar.StateVariableResult) error) error
}

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the RangeProvider (risk model).
type Engine struct {
	riskModel   RangeProvider
	minDuration time.Duration

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

	boundFactorsConsensusDone bool

	stateChanged   bool
	stateVarEngine StateVarEngine
}

// NewMonitor returns a new instance of PriceMonitoring.
func NewMonitor(asset, mktID string, riskModel RangeProvider, settings *types.PriceMonitoringSettings, stateVarEngine StateVarEngine) (*Engine, error) {
	if riskModel == nil {
		return nil, ErrNilRangeProvider
	}
	if settings == nil {
		return nil, ErrNilPriceMonitoringSettings
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
				parameters[i].Probability.GreaterThanOrEqual(parameters[j].Probability)
		})

	h := map[int64]num.Decimal{}
	bounds := make([]*bound, 0, len(parameters))
	for _, p := range parameters {
		bounds = append(bounds, &bound{
			Active:  true,
			Trigger: p,
		})
		if _, ok := h[p.Horizon]; !ok {
			h[p.Horizon] = p.HorizonDec.Div(secondsPerYear)
		}
	}

	e := &Engine{
		riskModel:                 riskModel,
		fpHorizons:                h,
		bounds:                    bounds,
		stateChanged:              true,
		stateVarEngine:            stateVarEngine,
		boundFactorsConsensusDone: false,
	}

	stateVarEngine.AddStateVariable(asset, mktID, boundFactorsConverter{}, e.startCalcPriceRanges, []statevar.StateVarEventType{statevar.StateVarEventTypeTimeTrigger, statevar.StateVarEventTypeAuctionEnded, statevar.StateVarEventTypeOpeningAuctionFirstUncrossingPrice}, e.updatePriceBounds)
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
	min := num.NewWrappedDecimal(num.Zero(), num.DecimalZero())
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
	// don't like this use of floats here, still
	sort.SliceStable(ret,
		func(i, j int) bool {
			return ret[i].Trigger.Horizon <= ret[j].Trigger.Horizon &&
				ret[i].Trigger.Probability.LessThanOrEqual(ret[j].Trigger.Probability)
		})
	return ret
}

// CheckPrice checks how current price, volume and time should impact the auction state and modifies it accordingly: start auction, end auction, extend ongoing auction.
func (e *Engine) CheckPrice(ctx context.Context, as AuctionState, p *num.Uint, v uint64, now time.Time, persistent bool) error {
	// initialise with the first price & time provided, otherwise there won't be any bounds
	wasInitialised := e.initialised
	if !wasInitialised {
		// Volume of 0, do nothing
		if v == 0 {
			return nil
		}
		e.reset(p, v, now)
		e.initialised = true
		e.stateChanged = true
	}

	last := currentPrice{
		Price:  p.Clone(),
		Volume: v,
	}
	if len(e.pricesNow) > 0 {
		last = e.pricesNow[len(e.pricesNow)-1]
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
		if !persistent {
			// we're going to stay in continuous trading, make sure we still have bounds
			e.reset(last.Price, last.Volume, now)
			return proto.ErrNonPersistentOrderOutOfBounds
		}
		duration := types.AuctionDuration{}
		for _, b := range bounds {
			duration.Duration += b.AuctionExtension
		}
		// we're dealing with a batch auction that's about to end -> extend it?
		if fba && as.CanLeave() {
			// bounds were violated, based on the values in the bounds slice, we can calculate how long the auction should last
			as.ExtendAuctionPrice(duration)
			return nil
		}
		if min := int64(e.minDuration / time.Second); duration.Duration < min {
			duration.Duration = min
		}

		as.StartPriceAuction(now, &duration)
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
			as.SetReadyToLeave()
			// reset the engine
			e.reset(p, v, now)
			return nil
		}
		// liquidity auction, and it was safe to end -> book is OK, price was OK, reset the engine
		if as.CanLeave() {
			e.reset(p, v, now)
		}
		return nil
	}

	var duration int64
	for _, b := range bounds {
		duration += b.AuctionExtension
	}

	// extend the current auction
	as.ExtendAuctionPrice(types.AuctionDuration{
		Duration: duration,
	})

	return nil
}

// reset restarts price monitoring with a new price. All previously recorded prices and previously obtained bounds get deleted.
func (e *Engine) reset(price *num.Uint, volume uint64, now time.Time) {
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
			e.stateChanged = true
			return
		}
	}
	e.priceRangeCacheTime = time.Time{}
	e.resetBounds()
	e.clearStalePrices()
	e.stateChanged = true
}

// resetBounds reactivates all bounds.
func (e *Engine) resetBounds() {
	for _, b := range e.bounds {
		if !b.Active {
			e.stateChanged = true
		}
		b.Active = true
	}
}

// recordPriceChange informs price monitoring module of a price change within the same instance as specified by the last call to UpdateTime.
func (e *Engine) recordPriceChange(price *num.Uint, volume uint64) {
	if volume > 0 {
		e.pricesNow = append(e.pricesNow, currentPrice{Price: price.Clone(), Volume: volume})
		e.stateChanged = true
	}
}

// recordTimeChange updates the current time and moves prices from current prices to past prices by calculating their corresponding vwp.
func (e *Engine) recordTimeChange(now time.Time) error {
	if now.Before(e.now) {
		return ErrTimeSequence // This shouldn't happen, but if it does there's something fishy going on
	}
	if now.Equal(e.now) {
		return nil
	}

	if len(e.pricesNow) > 0 {
		totalWeightedPrice, totalVol := num.Zero(), num.Zero()
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

	return nil
}

// checkBounds checks if the price is within price range for each of the bound and return trigger for each bound that it's not.
func (e *Engine) checkBounds(ctx context.Context, p *num.Uint, v uint64) []*types.PriceMonitoringTrigger {
	ret := []*types.PriceMonitoringTrigger{} // returned price projections, empty if all good
	if v == 0 {
		return ret // volume 0 so no bounds violated
	}
	priceRanges := e.getCurrentPriceRanges(false)
	for _, b := range e.bounds {
		if !b.Active {
			continue
		}
		priceRange := priceRanges[b]
		if p.LT(priceRange.MinPrice.Representation()) || p.GT(priceRange.MaxPrice.Representation()) {
			ret = append(ret, b.Trigger)
			// deactivate the bound that just got violated so it doesn't prevent auction from terminating
			b.Active = false
		}
	}
	return ret
}

// getCurrentPriceRanges calculates price ranges from current reference prices and bound down/up factors.
func (e *Engine) getCurrentPriceRanges(force bool) map[*bound]priceRange {
	if e.priceRangeCacheTime == e.now && !force {
		return e.priceRangesCache
	}
	e.priceRangesCache = make(map[*bound]priceRange, len(e.priceRangesCache))

	for _, b := range e.bounds {
		if !b.Active {
			continue
		}
		ref := e.getRefPrice(b.Trigger.Horizon)
		var min, max num.Decimal
		if e.boundFactorsConsensusDone {
			min = ref.Mul(b.DownFactor)
			max = ref.Mul(b.UpFactor)
		} else {
			min = ref.Mul(defaultDownFactor)
			max = ref.Mul(defaultUpFactor)
		}

		minUint, _ := num.UintFromDecimal(min.Ceil())
		maxUint, _ := num.UintFromDecimal(max.Floor())
		e.priceRangesCache[b] = priceRange{
			MinPrice:       num.NewWrappedDecimal(minUint, min),
			MaxPrice:       num.NewWrappedDecimal(maxUint, max),
			ReferencePrice: ref,
		}
	}
	e.priceRangeCacheTime = e.now
	e.stateChanged = true
	return e.priceRangesCache
}

// startCalcPriceRanges kicks off the bounds factors factors calculation, done asynchronously for illustration.
func (e *Engine) startCalcPriceRanges(eventID string, endOfCalcCallback statevar.FinaliseCalculation) {
	down := make([]num.Decimal, 0, len(e.bounds))
	up := make([]num.Decimal, 0, len(e.bounds))
	for _, b := range e.bounds {
		ref := e.getRefPrice(b.Trigger.Horizon)
		minPrice, maxPrice := e.riskModel.PriceRange(ref, e.fpHorizons[b.Trigger.Horizon], b.Trigger.Probability)
		down = append(down, minPrice.Div(ref))
		up = append(up, maxPrice.Div(ref))
	}
	res := &boundFactors{
		down: down,
		up:   up,
	}
	endOfCalcCallback.CalculationFinished(eventID, res, nil)
}

// updatePriceBounds is called back from the state variable consensus engine when consensus is reached for the down/up factors and updates the price bounds.
func (e *Engine) updatePriceBounds(ctx context.Context, res statevar.StateVariableResult) error {
	bRes := res.(*boundFactors)
	e.updateFactors(bRes.down, bRes.up)
	return nil
}

func (e *Engine) updateFactors(down, up []num.Decimal) {
	for i, b := range e.bounds {
		if !b.Active {
			continue
		}

		b.DownFactor = down[i]
		b.UpFactor = up[i]
	}
	e.boundFactorsConsensusDone = true
	// force invalidation of the price range cache
	if len(e.pricesNow) > 0 {
		e.getCurrentPriceRanges(true)
	}

	e.clearStalePrices()
	e.stateChanged = true
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
func (e *Engine) getRefPrice(horizon int64) num.Decimal {
	if e.refPriceCacheTime != e.now {
		e.refPriceCache = make(map[int64]num.Decimal, len(e.refPriceCache))
		e.stateChanged = true
	}

	if _, ok := e.refPriceCache[horizon]; !ok {
		e.refPriceCache[horizon] = e.calculateRefPrice(horizon)
		e.stateChanged = true
	}
	return e.refPriceCache[horizon]
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
