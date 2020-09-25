package pricemonitoring

import (
	"errors"
	"sort"
	"time"
)

var (
	// ErrProbabilityLevel gets thrown when probability is outside the (0,1)
	ErrProbabilityLevel = errors.New("probability level must be in the interval (0,1)")
	// ErrTimeSequence signals that time sequence is not in a non-decreasing order
	ErrTimeSequence = errors.New("received a time that's before the last received time")
	// ErrHorizonNotInFuture signals that the time horizon is not positive
	ErrHorizonNotInFuture = errors.New("horizon must be represented by a positive duration")
	// ErrUpdateFrequencyNotPositive signals that update frequency isn't positive.
	ErrUpdateFrequencyNotPositive = errors.New("update frequency must be represented by a positive duration")
)

// HorizonProbabilityLevelPair ties the horizon τ and probability p level.
// It's used to check if price over τ has exceeded the probability level p implied by the risk model
// (e.g. τ = 1 hour, p = 95%)
type HorizonProbabilityLevelPair struct {
	Horizon          time.Duration
	ProbabilityLevel float64
}

// NewHorizonProbabilityLevelPair returns a new instance of HorizonProbabilityLevelPair
// if probability level is in the range (0,1) and horizon is in the future and an error otherwise
func NewHorizonProbabilityLevelPair(horizon time.Duration, probabilityLevel float64) (*HorizonProbabilityLevelPair, error) {
	p := HorizonProbabilityLevelPair{Horizon: horizon, ProbabilityLevel: probabilityLevel}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Validate returns an error if probability level is not the range (0,1) or horizon is not in the future and nil otherwise
func (p HorizonProbabilityLevelPair) Validate() error {
	if p.ProbabilityLevel <= 0 || p.ProbabilityLevel >= 1 {
		return ErrProbabilityLevel
	}
	if p.Horizon <= 0 {
		return ErrHorizonNotInFuture
	}
	return nil
}

type priceMoveBound struct {
	MaxValidMoveUp   float64
	MinValidMoveDown float64
}

type timestampedAveragePrice struct {
	Time         time.Time
	AveragePrice float64
}

// PriceRangeProvider provides the minimium and maximum future price corresponding to the current price level, horizon expressed as year fraction (e.g. 0.5 for 6 months) and probability level (e.g. 0.95 for 95%).
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_range_provider_mock.go -package mocks code.vegaprotocol.io/vega/pricemonitoring PriceRangeProvider
type PriceRangeProvider interface {
	PriceRange(currentPrice float64, yearFraction float64, probabilityLevel float64) (minPrice float64, maxPrice float64)
}

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the PriceRangeProvider (risk model).
type Engine struct {
	riskModel                    PriceRangeProvider
	horizonProbabilityLevelPairs []HorizonProbabilityLevelPair
	updateFrequency              time.Duration

	horizonsAsYearFraction map[time.Duration]float64
	currentTime            time.Time
	updateTime             time.Time
	pricesPerCurrentTime   []uint64
	averagePriceHistory    []timestampedAveragePrice
	priceMoveBounds        map[HorizonProbabilityLevelPair]priceMoveBound
}

// NewPriceMonitoring returns a new instance of PriceMonitoring.
func NewPriceMonitoring(riskModel PriceRangeProvider, horizonProbabilityLevelPairs []HorizonProbabilityLevelPair, updateFrequency time.Duration, currentPrice uint64, currentTime time.Time) (*Engine, error) {
	if updateFrequency <= 0 {
		return nil, ErrUpdateFrequencyNotPositive
	}

	// Other functions depend on this sorting
	sort.Slice(horizonProbabilityLevelPairs,
		func(i, j int) bool {
			return horizonProbabilityLevelPairs[i].Horizon < horizonProbabilityLevelPairs[j].Horizon &&
				horizonProbabilityLevelPairs[i].ProbabilityLevel >= horizonProbabilityLevelPairs[j].ProbabilityLevel
		})

	h := make(map[time.Duration]float64)
	nanosecondsInAYear := 365.25 * 24 * time.Hour
	for _, p := range horizonProbabilityLevelPairs {
		if err := p.Validate(); err != nil {
			return nil, err
		}
		if _, ok := h[p.Horizon]; !ok {
			if p.Horizon == 0 {
				return nil, ErrHorizonNotInFuture
			}
			h[p.Horizon] = float64(p.Horizon) / float64(nanosecondsInAYear)
		}
	}
	e := &Engine{
		riskModel:                    riskModel,
		horizonProbabilityLevelPairs: horizonProbabilityLevelPairs,
		horizonsAsYearFraction:       h,
		updateFrequency:              updateFrequency,
	}
	e.Reset(currentPrice, currentTime)
	return e, nil
}

// Reset restarts price monitoring with a new price. All previously recorded prices and previously obtained bounds get deleted.
func (e *Engine) Reset(currentPrice uint64, currentTime time.Time) {
	e.currentTime = currentTime
	e.pricesPerCurrentTime = []uint64{currentPrice}
	e.averagePriceHistory = []timestampedAveragePrice{}
	e.priceMoveBounds = map[HorizonProbabilityLevelPair]priceMoveBound{}
	e.updateTime = currentTime
	e.updateBounds()
}

// RecordPriceChange informs price monitoring module of a price change within the same instance as specified by the last call to UpdateTime
func (e *Engine) RecordPriceChange(price uint64) {
	e.pricesPerCurrentTime = append(e.pricesPerCurrentTime, price)
}

// RecordTimeChange updates the time in the price monitoring module and returns an error if any problems are encountered.
func (e *Engine) RecordTimeChange(currentTime time.Time) error {
	if currentTime.Before(e.currentTime) {
		return ErrTimeSequence // This shouldn't happen, but if it does there's something fishy going on
	}
	if currentTime.After(e.currentTime) {
		var sum uint64 = 0
		for _, x := range e.pricesPerCurrentTime {
			sum += x
		}
		e.averagePriceHistory = append(e.averagePriceHistory,
			timestampedAveragePrice{
				Time:         e.currentTime,
				AveragePrice: float64(sum) / float64(len(e.pricesPerCurrentTime)),
			})
		e.pricesPerCurrentTime = make([]uint64, 0, cap(e.pricesPerCurrentTime))
		e.currentTime = currentTime
		e.updateBounds()
	}
	return nil
}

// CheckBoundViolations returns a map of horizon and probability level pair to boolean.
// A true value indicates that a bound corresponding to a given horizon and probability level pair has been violated.
func (e *Engine) CheckBoundViolations(price uint64) map[HorizonProbabilityLevelPair]bool {
	fpPrice := float64(price)
	checks := make(map[HorizonProbabilityLevelPair]bool, len(e.horizonProbabilityLevelPairs))
	prevHorizon := 0 * time.Nanosecond
	var referencePrice float64
	for _, p := range e.horizonProbabilityLevelPairs {
		// horizonProbabilityLevelPairs are sorted by Horizon to avoid repeated price lookup
		if p.Horizon != prevHorizon {
			referencePrice = e.getReferencePrice(e.currentTime.Add(-p.Horizon))
			prevHorizon = p.Horizon
		}

		priceDiff := fpPrice - referencePrice
		bounds := e.priceMoveBounds[p]
		checks[p] = priceDiff < bounds.MinValidMoveDown || priceDiff > bounds.MaxValidMoveUp
	}
	return checks
}

func (e *Engine) updateBounds() {
	if e.currentTime.Before(e.updateTime) {
		return
	}

	e.updateTime = e.currentTime.Add(e.updateFrequency)
	var latestPrice float64
	if len(e.averagePriceHistory) == 0 {
		latestPrice = float64(e.pricesPerCurrentTime[len(e.pricesPerCurrentTime)-1])
	} else {
		latestPrice = e.averagePriceHistory[len(e.averagePriceHistory)-1].AveragePrice
	}
	for _, p := range e.horizonProbabilityLevelPairs {

		minPrice, maxPrice := e.riskModel.PriceRange(latestPrice, e.horizonsAsYearFraction[p.Horizon], p.ProbabilityLevel)
		e.priceMoveBounds[p] = priceMoveBound{MinValidMoveDown: minPrice - latestPrice, MaxValidMoveUp: maxPrice - latestPrice}
	}
	// Remove redundant average prices
	maxTau := e.horizonProbabilityLevelPairs[len(e.horizonProbabilityLevelPairs)-1].Horizon
	minRequiredHorizon := e.currentTime.Add(-maxTau)
	var i int
	// Make sure at least one entry is left hence the "len(..) - 1"
	for i = 0; i < len(e.averagePriceHistory)-1; i++ {
		if !e.averagePriceHistory[i].Time.Before(minRequiredHorizon) {
			break
		}
		e.averagePriceHistory[i] = timestampedAveragePrice{} //TODO (WG): Confirm if this is needed to reclaim memory
	}
	e.averagePriceHistory = e.averagePriceHistory[i:]
}

func (e *Engine) getReferencePrice(referenceTime time.Time) float64 {
	var referencePrice float64
	if len(e.averagePriceHistory) < 1 {
		referencePrice = float64(e.pricesPerCurrentTime[0])
	} else {
		referencePrice = e.averagePriceHistory[0].AveragePrice
	}
	for _, p := range e.averagePriceHistory {
		if p.Time.After(referenceTime) {
			break
		}
		referencePrice = p.AveragePrice
	}
	return referencePrice
}
