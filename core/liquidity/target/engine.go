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

package target

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

var (
	// ErrTimeSequence signals that time sequence is not in a non-decreasing order.
	ErrTimeSequence = errors.New("received a time that's before the last received time")
	// ErrNegativeScalingFactor indicates that a negative scaling factor was supplied to the engine.
	ErrNegativeScalingFactor = errors.New("scaling factor can't be negative")
)

var (
	exp    = num.UintZero().Exp(num.NewUint(10), num.NewUint(5))
	exp2   = num.UintZero().Exp(num.NewUint(10), num.NewUint(10))
	expDec = num.DecimalFromUint(exp)
)

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the RangeProvider (risk model).
type Engine struct {
	marketID string

	tWindow time.Duration
	sFactor *num.Uint
	oiCalc  OpenInterestCalculator

	now               time.Time
	scheduledTruncate time.Time
	current           []uint64
	previous          []timestampedOI
	max               timestampedOI
	positionFactor    num.Decimal
}

type timestampedOI struct {
	Time time.Time
	OI   uint64
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/open_interest_calculator_mock.go -package mocks code.vegaprotocol.io/vega/core/liquidity/target OpenInterestCalculator
type OpenInterestCalculator interface {
	GetOpenInterestGivenTrades(trades []*types.Trade) uint64
}

// NewEngine returns a new instance of target stake calculation Engine.
func NewEngine(parameters types.TargetStakeParameters, oiCalc OpenInterestCalculator, marketID string, positionFactor num.Decimal) *Engine {
	factor, _ := num.UintFromDecimal(parameters.ScalingFactor.Mul(expDec))

	return &Engine{
		marketID:       marketID,
		tWindow:        time.Duration(parameters.TimeWindow) * time.Second,
		sFactor:        factor,
		oiCalc:         oiCalc,
		positionFactor: positionFactor,
	}
}

// UpdateTimeWindow updates the time windows used in target stake calculation.
func (e *Engine) UpdateTimeWindow(tWindow time.Duration) {
	e.tWindow = tWindow
}

// UpdateScalingFactor updates the scaling factor used in target stake calculation
// if it's non-negative and returns an error otherwise.
func (e *Engine) UpdateScalingFactor(sFactor num.Decimal) error {
	if sFactor.IsNegative() {
		return ErrNegativeScalingFactor
	}
	factor, _ := num.UintFromDecimal(sFactor.Mul(expDec))
	e.sFactor = factor

	return nil
}

// RecordOpenInterest records open interest history so that target stake can be calculated.
func (e *Engine) RecordOpenInterest(oi uint64, now time.Time) error {
	if now.Before(e.now) {
		return ErrTimeSequence
	}

	if oi >= e.max.OI {
		e.max = timestampedOI{Time: now, OI: oi}
	}

	if now.After(e.now) {
		// get max from current before updating timestamp
		e.previous = append(e.previous, e.getMaxFromCurrent())
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
// and the open interest time series constructed by calls to RecordOpenInterest.
func (e *Engine) GetTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint) (*num.Uint, bool) {
	var changed bool
	if minTime := e.minTime(now); minTime.After(e.max.Time) {
		e.computeMaxOI(minTime)
		changed = true
	}

	// float64(markPrice.Uint64()*e.max.OI) * math.Max(rf.Short, rf.Long) * e.sFactor
	factor := rf.Long
	if factor.LessThan(rf.Short) {
		factor = rf.Short
	}
	factorUint, _ := num.UintFromDecimal(factor.Mul(expDec))

	value, _ := num.UintFromDecimal(markPrice.ToDecimal().Mul(num.DecimalFromInt64(int64(e.max.OI))).Div(e.positionFactor))
	return num.UintZero().Div(num.UintZero().Mul(value, factorUint.Mul(factorUint, e.sFactor)), exp2), changed
}

// GetTheoreticalTargetStake returns target stake based current time, risk factors
// and the supplied trades without modifying the internal state.
func (e *Engine) GetTheoreticalTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint, trades []*types.Trade) (*num.Uint, bool) {
	var changed bool
	theoreticalOI := e.oiCalc.GetOpenInterestGivenTrades(trades)

	timeWindowStart := e.minTime(now)
	maxExpired := timeWindowStart.After(e.max.Time)
	if maxExpired {
		e.computeMaxOI(timeWindowStart)
		changed = true
	}

	maxOI := e.max.OI
	if theoreticalOI > maxOI || maxExpired {
		maxOI = theoreticalOI
	}

	factor := rf.Long
	if factor.LessThan(rf.Short) {
		factor = rf.Short
	}

	factorUint, _ := num.UintFromDecimal(factor.Mul(expDec))
	value, _ := num.UintFromDecimal(markPrice.ToDecimal().Mul(num.DecimalFromInt64(int64(maxOI))).Div(e.positionFactor))
	return num.UintZero().Div(num.UintZero().Mul(value, factorUint.Mul(factorUint, e.sFactor)), exp2), changed
}

func (e *Engine) UpdateParameters(parameters types.TargetStakeParameters) {
	factor, _ := num.UintFromDecimal(parameters.ScalingFactor.Mul(expDec))
	e.sFactor = factor
	e.tWindow = time.Duration(parameters.TimeWindow) * time.Second
}

func (e *Engine) getMaxFromCurrent() timestampedOI {
	if len(e.current) == 0 {
		return timestampedOI{Time: e.now, OI: 0}
	}
	m := e.current[0]
	for i := 1; i < len(e.current); i++ {
		if e.current[i] > m {
			m = e.current[i]
		}
	}
	return timestampedOI{Time: e.now, OI: m}
}

func (e *Engine) computeMaxOI(minTime time.Time) {
	m := e.getMaxFromCurrent()
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

// minTime returns the lower bound of the sliding time window.
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
	// Truncate at least every 2 time windows in case not called before to prevent excessive memory usage
	e.scheduledTruncate = e.now.Add(2 * e.tWindow)
}
