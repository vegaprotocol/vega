package target

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	// ErrTimeSequence signals that time sequence is not in a non-decreasing order
	ErrTimeSequence = errors.New("received a time that's before the last received time")
	// ErrNegativeScalingFactor indicates that a negative scaling factor was supplied to the engine
	ErrNegativeScalingFactor = errors.New("scaling factor can't be negative")
)

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the RangeProvider (risk model).
type Engine struct {
	tWindow time.Duration
	sFactor float64
	oiCalc  OpenInterestCalculator

	now               time.Time
	scheduledTruncate time.Time
	current           []uint64
	previous          []timestampedOI
	max               timestampedOI
}

type timestampedOI struct {
	Time time.Time
	OI   uint64
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/open_interest_calculator_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/target OpenInterestCalculator
type OpenInterestCalculator interface {
	GetOpenInterestGivenTrades(trades []*types.Trade) uint64
}

// NewEngine returns a new instance of target stake calculation Engine
func NewEngine(parameters types.TargetStakeParameters, oiCalc OpenInterestCalculator) *Engine {
	return &Engine{
		tWindow: time.Duration(parameters.TimeWindow) * time.Second,
		sFactor: parameters.ScalingFactor,
		oiCalc:  oiCalc,
	}
}

// UpdateTimeWindow updates the time windows used in target stake calculation
func (e *Engine) UpdateTimeWindow(tWindow time.Duration) {
	e.tWindow = tWindow
}

// UpdateScalingFactor updates the scaling factor used in target stake calculation
// if it's non-negative and returns an error otherwise
func (e *Engine) UpdateScalingFactor(sFactor float64) error {
	if sFactor < 0 {
		return ErrNegativeScalingFactor
	}
	e.sFactor = sFactor
	return nil
}

// RecordOpenInterest records open interset history so that target stake can be calculated
func (e *Engine) RecordOpenInterest(oi uint64, now time.Time) error {
	if now.Before(e.now) {
		return ErrTimeSequence
	}

	if oi >= e.max.OI {
		e.max = timestampedOI{Time: now, OI: oi}
	}

	if now.After(e.now) {
		toi := timestampedOI{Time: e.now, OI: e.getMaxFromCurrent()}
		e.previous = append(e.previous, toi)
		e.current = make([]uint64, 0, len(e.current))
		e.now = now
	}
	e.current = append(e.current, oi)

	if e.now.After(e.scheduledTruncate) {
		e.truncateHistory(e.minTime(now))
	}

	return nil
}

// GetTargetStake returns target stake based current time, risk factors
// and the open interest time series constructed by calls to RecordOpenInterest
func (e *Engine) GetTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint) num.Decimal {
	minTime := e.minTime(now)
	if minTime.After(e.max.Time) {
		e.computeMaxOI(now, minTime)
	}

	// float64(markPrice.Uint64()*e.max.OI) * math.Max(rf.Short, rf.Long) * e.sFactor
	factor := rf.Long
	if factor.LessThan(rf.Short) {
		factor = rf.Short
	}
	mp := num.DecimalFromUint(markPrice)
	return mp.Mul(num.NewUint(e.max.OI).ToDecimal()).Mul(factor.Mul(num.DecimalFromFloat(e.sFactor)))
}

//GetTheoreticalTargetStake returns target stake based current time, risk factors
//and the supplied trades without modifying the internal state
func (e *Engine) GetTheoreticalTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint, trades []*types.Trade) *num.Uint {
	theoreticalOI := e.oiCalc.GetOpenInterestGivenTrades(trades)
	minTime := e.minTime(now)
	if minTime.After(e.max.Time) {
		e.computeMaxOI(now, minTime)
	}
	maxOI := e.max.OI
	if theoreticalOI > maxOI {
		maxOI = theoreticalOI
	}

	factor := rf.Long
	if factor.LessThan(rf.Short) {
		factor = rf.Short
	}
	// float64(markPrice.Uint64()*maxOI) * math.Max(rf.Short, rf.Long) * e.sFactor
	mp := num.DecimalFromUint(markPrice).Mul(
		num.NewUint(maxOI).ToDecimal(),
	).Mul(
		factor.Mul(num.DecimalFromFloat(e.sFactor)),
	)
	// return the decimal as uint
	retVal, _ := num.UintFromDecimal(mp)
	return retVal
}

func (e *Engine) getMaxFromCurrent() uint64 {
	if len(e.current) == 0 {
		return 0
	}
	m := e.current[0]
	for i := 1; i < len(e.current); i++ {
		if e.current[i] > m {
			m = e.current[i]
		}
	}
	return m
}

func (e *Engine) computeMaxOI(now, minTime time.Time) {
	m := timestampedOI{Time: e.now, OI: e.getMaxFromCurrent()}
	e.truncateHistory(minTime)
	var j int
	for i := 0; i < len(e.previous); i++ {
		if e.previous[i].OI > m.OI {
			m = e.previous[i]
			j = i
		}
	}
	e.max = m

	// remove entries less than max as these won't ever be needed anyway
	e.previous = e.previous[j:]

}

//minTime returns the lower bound of the sliding time window
func (e *Engine) minTime(now time.Time) time.Time {
	return now.Add(-e.tWindow)
}

func (e *Engine) truncateHistory(minTime time.Time) {
	var i int
	for i = 0; i < len(e.previous); i++ {
		if !e.previous[i].Time.Before(minTime) {
			break
		}

	}
	e.previous = e.previous[i:]
	//Truncate at least every 2 time windows in case not called before to prevent excessive memory usage
	e.scheduledTruncate = e.now.Add(2 * e.tWindow)
}
