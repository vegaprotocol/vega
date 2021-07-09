package metrics

import (
	"time"
)

// TimeCounter holds a time.Time and a list of label values, hiding the start time from being accidentally
// overwritten, and removing the need to duplicate the label values.
type TimeCounter struct {
	labelValues []string
	start       time.Time
}

// NewTimeCounter returns a new TimeCounter, with the start time already recorded.
func NewTimeCounter(labelValues ...string) *TimeCounter {
	return &TimeCounter{
		labelValues: labelValues,
		start:       time.Now(),
	}
}

/*
EngineTimeCounterAdd is used to time a function.
e.g.
	func DoSomething() {
		timer := metrics.NewTimeCounter("x", "y", "z")
		// do something
		timer.EngineTimeCounterAdd()
	}
*/
func (tc *TimeCounter) EngineTimeCounterAdd() {
	// Check that the metric has been set up. (Testing does not use metrics.)
	if engineTime == nil {
		return
	}
	engineTime.WithLabelValues(tc.labelValues...).Add(time.Since(tc.start).Seconds())
}
