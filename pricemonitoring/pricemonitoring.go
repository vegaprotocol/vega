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
	if p.Horizon.Nanoseconds() <= 0 {
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

// PriceMonitoring allows tracking price changes and verifying them against the theoretical levels implied by the PriceRangeProvider (risk model).
type PriceMonitoring struct {
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
func NewPriceMonitoring(riskModel PriceRangeProvider, horizonProbabilityLevelPairs []HorizonProbabilityLevelPair, updateFrequency time.Duration, currentPrice uint64, currentTime time.Time) (*PriceMonitoring, error) {
	if updateFrequency.Nanoseconds() <= 0 {
		return nil, ErrUpdateFrequencyNotPositive
	}

	// Other functions depend on this sorting
	sort.Slice(horizonProbabilityLevelPairs,
		func(i, j int) bool {
			return horizonProbabilityLevelPairs[i].Horizon < horizonProbabilityLevelPairs[j].Horizon &&
				horizonProbabilityLevelPairs[i].ProbabilityLevel >= horizonProbabilityLevelPairs[j].ProbabilityLevel
		})

	horizonsAsYearFraction := make(map[time.Duration]float64)
	nanosecondsInAYear := (365.25 * 24 * time.Hour).Nanoseconds()
	for _, p := range horizonProbabilityLevelPairs {
		if err := p.Validate(); err != nil {
			return nil, err
		}
		if _, ok := horizonsAsYearFraction[p.Horizon]; !ok {
			horizonNano := p.Horizon.Nanoseconds()
			if horizonNano == 0 {
				return nil, ErrHorizonNotInFuture
			}
			horizonsAsYearFraction[p.Horizon] = float64(horizonNano) / float64(nanosecondsInAYear)
		}
	}
	pm := &PriceMonitoring{
		riskModel:                    riskModel,
		horizonProbabilityLevelPairs: horizonProbabilityLevelPairs,
		horizonsAsYearFraction:       horizonsAsYearFraction,
		updateFrequency:              updateFrequency,
	}
	pm.Reset(currentPrice, currentTime)
	return pm, nil
}

// Reset restarts price monitoring with a new price. All previously recorded prices and previously obtained bounds get deleted.
func (pm *PriceMonitoring) Reset(currentPrice uint64, currentTime time.Time) {
	pm.currentTime = currentTime
	pm.pricesPerCurrentTime = []uint64{currentPrice}
	pm.averagePriceHistory = []timestampedAveragePrice{}
	pm.priceMoveBounds = make(map[HorizonProbabilityLevelPair]priceMoveBound)
	pm.updateTime = currentTime
	pm.updateBounds()
}

// RecordPriceChange informs price monitoring module of a price change within the same instance as specified by the last call to UpdateTime
func (pm *PriceMonitoring) RecordPriceChange(price uint64) {
	pm.pricesPerCurrentTime = append(pm.pricesPerCurrentTime, price)
}

// RecordTimeChange updates the time in the price monitoring module and returns an error if any problems are encountered.
func (pm *PriceMonitoring) RecordTimeChange(currentTime time.Time) error {
	if currentTime.Before(pm.currentTime) {
		return ErrTimeSequence // This shouldn't happen, but if it does there's something fishy going on
	}
	if currentTime.After(pm.currentTime) {
		var sum uint64 = 0
		for _, x := range pm.pricesPerCurrentTime {
			sum += x
		}
		pm.averagePriceHistory = append(pm.averagePriceHistory,
			timestampedAveragePrice{
				Time:         pm.currentTime,
				AveragePrice: float64(sum) / float64(len(pm.pricesPerCurrentTime)),
			})
		pm.pricesPerCurrentTime = make([]uint64, 0)
		pm.currentTime = currentTime
		pm.updateBounds()
	}
	return nil
}

// CheckBoundViolations returns a map of horizon and probability level pair to boolean.
// A true value indicates that a bound corresponding to a given horizon and probability level pair has been violated.
func (pm *PriceMonitoring) CheckBoundViolations(price uint64) map[HorizonProbabilityLevelPair]bool {
	fpPrice := float64(price)
	checks := make(map[HorizonProbabilityLevelPair]bool, len(pm.horizonProbabilityLevelPairs))
	prevHorizon := 0 * time.Nanosecond
	var referencePrice float64
	for _, p := range pm.horizonProbabilityLevelPairs {
		// horizonProbabilityLevelPairs are sorted by Horizon to avoid repeated price lookup
		if p.Horizon != prevHorizon {
			referencePrice = pm.getReferencePrice(pm.currentTime.Add(-p.Horizon))
			prevHorizon = p.Horizon
		}

		priceDiff := fpPrice - referencePrice
		bounds := pm.priceMoveBounds[p]
		checks[p] = priceDiff < bounds.MinValidMoveDown || priceDiff > bounds.MaxValidMoveUp
	}
	return checks
}

func (pm *PriceMonitoring) updateBounds() {
	if !pm.currentTime.Before(pm.updateTime) {
		pm.updateTime = pm.currentTime.Add(pm.updateFrequency)
		var latestPrice float64
		if len(pm.averagePriceHistory) < 1 {
			latestPrice = float64(pm.pricesPerCurrentTime[len(pm.pricesPerCurrentTime)-1])
		} else {
			latestPrice = pm.averagePriceHistory[len(pm.averagePriceHistory)-1].AveragePrice
		}
		for _, p := range pm.horizonProbabilityLevelPairs {

			minPrice, maxPrice := pm.riskModel.PriceRange(latestPrice, pm.horizonsAsYearFraction[p.Horizon], p.ProbabilityLevel)
			pm.priceMoveBounds[p] = priceMoveBound{MinValidMoveDown: minPrice - latestPrice, MaxValidMoveUp: maxPrice - latestPrice}
		}
		// Remove redundant average prices
		maxTau := pm.horizonProbabilityLevelPairs[len(pm.horizonProbabilityLevelPairs)-1].Horizon
		minRequiredHorizon := pm.currentTime.Add(-maxTau)
		var i int
		// Make sure at least one entry is left hence the "len(..) - 1"
		for i = 0; i < len(pm.averagePriceHistory)-1; i++ {
			if !pm.averagePriceHistory[i].Time.Before(minRequiredHorizon) {
				break
			}
			pm.averagePriceHistory[i] = timestampedAveragePrice{} //TODO (WG): Confirm if this is needed to reclaim memory
		}
		pm.averagePriceHistory = pm.averagePriceHistory[i:]

	}
}

func (pm *PriceMonitoring) getReferencePrice(referenceTime time.Time) float64 {
	var referencePrice float64
	if len(pm.averagePriceHistory) < 1 {
		referencePrice = float64(pm.pricesPerCurrentTime[0])
	} else {
		referencePrice = pm.averagePriceHistory[0].AveragePrice
	}
	for _, p := range pm.averagePriceHistory {
		if p.Time.After(referenceTime) {
			break
		}
		referencePrice = p.AveragePrice
	}
	return referencePrice
}
