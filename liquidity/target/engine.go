package target

import (
	"errors"
	"math"
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrTimeSequence = errors.New("received a time that's before the last received time")
)

// Engine allows tracking price changes and verifying them against the theoretical levels implied by the RangeProvider (risk model).
type Engine struct {
	tWindow       time.Duration
	scalingFactor float64

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

func (e *Engine) RecordOpenInterest(oi uint64, now time.Time) error {
	if now.Before(e.now) {
		return ErrTimeSequence // This shouldn't happen, but if it does there's something fishy going on
	}

	if oi >= e.max.OI {
		e.max = timestampedOI{Time: now, OI: oi}
	}

	if now.Equal(e.now) {
		e.current = append(e.current, oi)
	} else {
		toi := timestampedOI{Time: e.now, OI: e.getMaxFromCurrent()}
		e.previous = append(e.previous, toi)
		e.current = make([]uint64, 0, len(e.current))
		e.now = now
	}

	if e.now.After(e.scheduledTruncate) {
		e.truncateHistory()
	}

	return nil
}

func (e *Engine) getMaxFromCurrent() uint64 {
	m := e.current[0]
	for i := 1; i < len(e.current); i++ {
		if e.current[i] > m {
			m = e.current[i]
		}
	}
	return m
}

func (e *Engine) GetTargetStake(now time.Time, rf types.RiskFactor) float64 {
	minTime := now.Add(-e.tWindow)
	if minTime.After(e.max.Time) {
		e.computeMaxOI(now)
	}

	return float64(e.max.OI) * e.scalingFactor * math.Max(rf.Short, rf.Long)
}

func (e *Engine) computeMaxOI(now time.Time) {
	m := timestampedOI{Time: e.now, OI: e.getMaxFromCurrent()}
	e.truncateHistory()
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

func (e *Engine) truncateHistory() {
	minTime := e.now.Add(-e.tWindow)
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
