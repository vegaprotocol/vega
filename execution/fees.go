package execution

import (
	"errors"
	"math"
	"time"
)

type FeeSplitter struct {
	timeWindowStart time.Time
	currentTime     time.Time
	tradeValue      uint64
}

func (fs *FeeSplitter) MustSetCurrentTime(t time.Time) {
	if err := fs.SetCurrentTime(t); err != nil {
		panic(err)
	}
}

func (fs *FeeSplitter) SetCurrentTime(t time.Time) error {
	if t.Before(fs.timeWindowStart) {
		return errors.New("current time can't be before openingAuctionEnded time")
	}
	fs.currentTime = t
	return nil
}

// TimeWindowStart starts or restarts (if active) a current time window.
// This sets the internal timers to `t` and resets the accumulated trade values.
func (fs *FeeSplitter) TimeWindowStart(t time.Time) {
	// reset the trade value for this window
	fs.tradeValue = 0

	// reset both timers
	fs.timeWindowStart = t
	fs.SetCurrentTime(t)
}

// Elapsed returns the distance (in duration) from TimeWindowStart and
// CurrentTime.
func (fs *FeeSplitter) Elapsed() time.Duration {
	return fs.currentTime.Sub(fs.timeWindowStart)
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func (fs *FeeSplitter) activeWindowLength(mvw time.Duration) time.Duration {
	// t is the distance between current time and the openingActionEnded time
	t := fs.currentTime.Sub(fs.timeWindowStart)
	return t - maxDuration(t-mvw, 0)
}

// MarketValueProxy returns the market value proxy according to the spec:
// https://github.com/vegaprotocol/product/blob/master/specs/0042-setting-fees-and-rewarding-lps.md
func (fs *FeeSplitter) MarketValueProxy(mvwl time.Duration, totalStake float64) float64 {
	// t is the distance between
	awl := fs.activeWindowLength(mvwl)
	if awl > 0 {
		factor := mvwl.Seconds() / awl.Seconds()
		tv := fs.tradeValue
		return math.Max(totalStake, factor*float64(tv))
	}
	return totalStake
}

func (fs *FeeSplitter) AddTradeValue(v uint64) {
	fs.tradeValue += v
}
