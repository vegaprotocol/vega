package pricemonitoring

import (
	"errors"
	"sort"
	"time"
)

type priceMoveBound struct {
	MaxValidMoveUp   float64
	MinValidMoveDown float64
}

type timestampedAveragePrice struct {
	Time         time.Time
	AveragePrice float64
}

var (
	errProbabilityLevel           = errors.New("Probability level must be in the interval (0,1)")
	errTimeSequence               = errors.New("Received a time that's before the last received time")
	errPriceHistoryNotAvailable   = errors.New("Price history not available")
	errHorizonNotInFuture         = errors.New("Horizon must be represented by a positive duration")
	errUpdateFrequencyNotPositive = errors.New("Update frequency must be represented by a positive duration")
)

// PriceRangeProvider provides the minimium and maximum price corresponding to the current price level, horizon expressed as year fraction (e.g. 0.5 for 6 months) and probability level (e.g. 0.95 for 95%).
type PriceRangeProvider interface {
	PriceRange(currentPrice float64, yearFraction float64, probabilityLevel float64) (minPrice float64, maxPrice float64)
}

// PriceMonitoring allows tracking price changes and verifying them against the theoretical levels implied by the risk model.
// Reset() needs to be called after initialization and after each auction period to assure proper behaviour.
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

// NewPriceMonitoring return a new instance of PriceMonitoring.
// Note that horizonProbabilityLevelPairs will get sorte by horizon (ascending) and probabilit level (descending) to aid performance.
func NewPriceMonitoring(riskModel PriceRangeProvider, horizonProbabilityLevelPairs []HorizonProbabilityLevelPair, updateFrequency time.Duration) (*PriceMonitoring, error) {
	if updateFrequency.Nanoseconds() <= 0 {
		return nil, errUpdateFrequencyNotPositive
	}

	sort.Slice(horizonProbabilityLevelPairs,
		func(i, j int) bool {
			return horizonProbabilityLevelPairs[i].Horizon < horizonProbabilityLevelPairs[j].Horizon &&
				horizonProbabilityLevelPairs[i].ProbabilityLevel > horizonProbabilityLevelPairs[j].ProbabilityLevel
		})

	horizonsAsYearFraction := make(map[time.Duration]float64)
	nanosecondsInAYear := (365.25 * 24 * time.Hour).Nanoseconds()
	for _, p := range horizonProbabilityLevelPairs {
		if _, ok := horizonsAsYearFraction[p.Horizon]; !ok {
			horizonNano := p.Horizon.Nanoseconds()
			if horizonNano == 0 {
				return nil, errHorizonNotInFuture
			}
			horizonsAsYearFraction[p.Horizon] = float64(horizonNano) / float64(nanosecondsInAYear)
		}
	}
	return &PriceMonitoring{
		riskModel:                    riskModel,
		horizonProbabilityLevelPairs: horizonProbabilityLevelPairs,
		horizonsAsYearFraction:       horizonsAsYearFraction}, nil
}

// Reset restarts price monitoring with a new price. All previously recorded prices and previously obtained bounds get deleted.
// It should get called as the first method after initialisation for price monitoring to work as expected.
func (pm *PriceMonitoring) Reset(price uint64, now time.Time) error {
	pm.currentTime = now
	pm.pricesPerCurrentTime = []uint64{price}
	pm.averagePriceHistory = []timestampedAveragePrice{}
	return pm.updateBounds()
}

// RecordPriceChange informs price monitoring module of a price change within the same instance as specified by the last call to UpdateTime
func (pm *PriceMonitoring) RecordPriceChange(price uint64) {
	pm.pricesPerCurrentTime = append(pm.pricesPerCurrentTime, price)
}

// UpdateTime updates the time in the price monitoring module and returns an error otherwise if any problems are encountered.
func (pm *PriceMonitoring) UpdateTime(now time.Time) error {
	if now.Before(pm.currentTime) {
		return errTimeSequence // This shouldn't happen, but if it does there's something fishy going on
	}
	if now.After(pm.currentTime) {
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
		pm.currentTime = now
		if err := pm.updateBounds(); err != nil {
			return err
		}
	}
	return nil
}

func (pm *PriceMonitoring) updateBounds() error { //TODO: Think if this really needs to return an error
	if pm.currentTime.After(pm.updateTime) {
		pm.updateTime = pm.currentTime.Add(pm.updateFrequency)
		// Do the update stuffz
		if len(pm.averagePriceHistory) < 1 {
			return errPriceHistoryNotAvailable
		}
		latestPrice := pm.averagePriceHistory[len(pm.averagePriceHistory)-1].AveragePrice
		for _, p := range pm.horizonProbabilityLevelPairs {

			minPrice, maxPrice := pm.riskModel.PriceRange(latestPrice, pm.horizonsAsYearFraction[p.Horizon], p.ProbabilityLevel)
			pm.priceMoveBounds[p] = priceMoveBound{MinValidMoveDown: minPrice - latestPrice, MaxValidMoveUp: maxPrice - latestPrice}
		}

		// Remove redundant average prices
		maxTau := pm.horizonProbabilityLevelPairs[len(pm.horizonProbabilityLevelPairs)].Horizon
		minRequiredHorizon := pm.currentTime.Add(-maxTau)
		var i int
		for i = 0; i < len(pm.averagePriceHistory); i++ {
			if !pm.averagePriceHistory[i].Time.Before(minRequiredHorizon) {
				break
			}
		}
		pm.averagePriceHistory = pm.averagePriceHistory[i:]

	}
	return nil
}

// CheckBoundViolations returns an array of booleans, each corresponding to a given horizon and probability level pair.
// A true value indicates that a bound corresponding to a given horizon and probability level pair has been violated.
// It should be called after Reset has been called at least once
func (pm *PriceMonitoring) CheckBoundViolations(price uint64) ([]bool, error) {
	fpPrice := float64(price)
	checks := make([]bool, len(pm.horizonProbabilityLevelPairs))
	prevHorizon := 0 * time.Nanosecond
	referencePrice := -1.0
	var err error
	for i, p := range pm.horizonProbabilityLevelPairs {
		// horizonProbabilityLevelPairs are sorted by Horizon to avoid repeated price lookup
		if p.Horizon != prevHorizon {
			referencePrice, err = pm.getReferencePrice(pm.currentTime.Add(-p.Horizon))
			if err != nil {
				return nil, err
			}
			prevHorizon = p.Horizon
		}

		priceDiff := fpPrice - referencePrice
		bounds := pm.priceMoveBounds[p]
		checks[i] = priceDiff < bounds.MinValidMoveDown || priceDiff > bounds.MaxValidMoveUp
	}
	return checks, nil
}

func (pm *PriceMonitoring) getReferencePrice(referenceTime time.Time) (float64, error) {
	if len(pm.averagePriceHistory) < 1 {
		return -1, errPriceHistoryNotAvailable
	}
	refPrice := pm.averagePriceHistory[0].AveragePrice
	for _, p := range pm.averagePriceHistory {
		if p.Time.After(referenceTime) {
			break
		}
		refPrice = p.AveragePrice
	}
	return refPrice, nil

}

//GetHorizonProbablityLevelPairs return horizon and probability level pairs that the module uses
func (pm *PriceMonitoring) GetHorizonProbablityLevelPairs() []HorizonProbabilityLevelPair {
	return pm.horizonProbabilityLevelPairs
}

// HorizonProbabilityLevelPair ties the horizon τ and probability p level.
// It's used to check if price over τ has exceeded the probability level p implied by the risk model
// (e.g. τ = 1 hour, p = 95%)
type HorizonProbabilityLevelPair struct {
	Horizon          time.Duration
	ProbabilityLevel float64
}

// NewHorizonProbabilityLevelPair returns a new instance of HorizonProbabilityLevelPair
// if probability level is in the range (0,1) and an error otherwise
func NewHorizonProbabilityLevelPair(horizon time.Duration, probabilityLevel float64) (*HorizonProbabilityLevelPair, error) {
	if probabilityLevel <= 0 || probabilityLevel >= 1 {
		return nil, errProbabilityLevel
	}
	return &HorizonProbabilityLevelPair{Horizon: horizon, ProbabilityLevel: probabilityLevel}, nil
}
