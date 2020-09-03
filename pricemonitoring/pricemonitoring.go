package pricemonitoring

import (
	"time"

	"code.vegaprotocol.io/vega/risk"
)

type PriceMonitoring struct {
	riskModel                    risk.Model
	horizonProbabilityLevelPairs []HorizonProbabilityLevelPair
}

// NewPriceMonitoring return a new instance of PriceMonitoring
func NewPriceMonitoring(riskModel risk.Model, horizonProbabilityLevelPairs []HorizonProbabilityLevelPair) (*PriceMonitoring, error) {
	return &PriceMonitoring{riskModel: riskModel, horizonProbabilityLevelPairs: horizonProbabilityLevelPairs}, nil
}

// RecordPriceChange informs price monitoring module of a price change
func (pm *PriceMonitoring) RecordPriceChange(price uint64, time time.Time) {

}

// CheckPrice returns an array of booleans, each corresponding to a given horizon and probability level pair
func (pm PriceMonitoring) CheckPrice(price uint64, time time.Time) []bool {

}

func (pm PriceMonitoring) GetHorizonProbablityLevelPairs() []HorizonProbabilityLevelPair {
	return pm.horizonProbabilityLevelPairs
}

type HorizonProbabilityLevelPair struct {
	Horizon          time.Duration
	ProbabilityLevel float32
}

// NewHorizonProbabilityLevelPair returns a new instance of HorizonProbabilityLevelPair
// if probability level is in the range (0,1) and an error otherwise
func NewHorizonProbabilityLevelPair(horizon time.Duration, probabilityLevel float32) (*HorizonProbabilityLevelPair, error) {
	return &HorizonProbabilityLevelPair{Horizon: horizon, ProbabilityLevel: probabilityLevel}
}
