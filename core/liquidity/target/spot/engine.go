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

package spot

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

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the RangeProvider (risk model).
type Engine struct {
	marketID string

	tWindow time.Duration
	sFactor num.Decimal

	now               time.Time
	scheduledTruncate time.Time
	current           []uint64
	previous          []timestampedTotalStake
	max               timestampedTotalStake
	positionFactor    num.Decimal
}

type timestampedTotalStake struct {
	Time       time.Time
	TotalStake uint64
}

// NewEngine returns a new instance of target stake calculation Engine.
func NewEngine(parameters types.TargetStakeParameters, marketID string, positionFactor num.Decimal) *Engine {
	return &Engine{
		marketID:       marketID,
		tWindow:        time.Duration(parameters.TimeWindow) * time.Second,
		sFactor:        parameters.ScalingFactor,
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
	e.sFactor = sFactor
	return nil
}

// RecordTotalStake records open interest history so that target stake can be calculated.
func (e *Engine) RecordTotalStake(ts uint64, now time.Time) error {
	if now.Before(e.now) {
		return ErrTimeSequence
	}

	if ts >= e.max.TotalStake {
		e.max = timestampedTotalStake{Time: now, TotalStake: ts}
	}

	if now.After(e.now) {
		// get max from current before updating timestamp
		e.previous = append(e.previous, e.getMaxFromCurrent())
		e.current = make([]uint64, 0, len(e.current))
		e.now = now
	}
	e.current = append(e.current, ts)
	if e.now.After(e.scheduledTruncate) {
		e.truncateHistory(e.minTime(now))
	}

	return nil
}

// GetTargetStake returns target stake based current time
func (e *Engine) GetTargetStake(now time.Time) *num.Uint {
	if minTime := e.minTime(now); minTime.After(e.max.Time) {
		e.computeMaxTotalStake(minTime)
	}

	value, _ := num.UintFromDecimal(num.DecimalFromInt64(int64(e.max.TotalStake)).Mul(e.sFactor).Div(e.positionFactor))
	return value
}

func (e *Engine) UpdateParameters(parameters types.TargetStakeParameters) {
	e.sFactor = parameters.ScalingFactor
	e.tWindow = time.Duration(parameters.TimeWindow) * time.Second
}

func (e *Engine) getMaxFromCurrent() timestampedTotalStake {
	if len(e.current) == 0 {
		return timestampedTotalStake{Time: e.now, TotalStake: 0}
	}
	m := e.current[0]
	for i := 1; i < len(e.current); i++ {
		if e.current[i] > m {
			m = e.current[i]
		}
	}
	return timestampedTotalStake{Time: e.now, TotalStake: m}
}

func (e *Engine) computeMaxTotalStake(minTime time.Time) {
	m := e.getMaxFromCurrent()
	e.truncateHistory(minTime)
	var j int
	for i := 0; i < len(e.previous); i++ {
		if e.previous[i].TotalStake > m.TotalStake {
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
