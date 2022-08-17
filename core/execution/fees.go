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

package execution

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type FeeSplitter struct {
	timeWindowStart time.Time
	currentTime     time.Time
	tradeValue      *num.Uint
	changed         bool
	avg             num.Decimal
	window          uint64
}

func NewFeeSplitter() *FeeSplitter {
	return &FeeSplitter{
		tradeValue: num.UintZero(),
		changed:    true,
		window:     1, // initialise as 1 otherwise the average value calculation ends up being borked
		avg:        num.DecimalZero(),
	}
}

func (fs *FeeSplitter) SetCurrentTime(t time.Time) error {
	if t.Before(fs.timeWindowStart) {
		return errors.New("current time can't be before current window time")
	}
	// we're past the opening auction, or we have a trade value (ie we're leaving opening auction)
	fs.currentTime = t
	return nil
}

// TimeWindowStart starts or restarts (if active) a current time window.
// This sets the internal timers to `t` and resets the accumulated trade values.
func (fs *FeeSplitter) TimeWindowStart(t time.Time) {
	// if we have an average value, that means we left the opening auction
	// and we can increase the window to the next value
	if !fs.avg.IsZero() {
		fs.window++
	}
	// reset the trade value for this window
	fs.tradeValue = num.UintZero()

	// reset both timers
	fs.timeWindowStart = t
	fs.SetCurrentTime(t)
	fs.changed = true
}

// Elapsed returns the distance (in duration) from TimeWindowStart and
// CurrentTime.
func (fs *FeeSplitter) Elapsed() time.Duration {
	return fs.currentTime.Sub(fs.timeWindowStart)
}

func (fs *FeeSplitter) activeWindowLength(mvw time.Duration) time.Duration {
	t := fs.Elapsed()
	return t - num.MaxV(t-mvw, 0)
}

// MarketValueProxy returns the market value proxy according to the spec:
// https://github.com/vegaprotocol/product/blob/master/specs/0042-setting-fees-and-rewarding-lps.md
func (fs *FeeSplitter) MarketValueProxy(mvwl time.Duration, totalStakeU *num.Uint) num.Decimal {
	totalStake := num.DecimalFromUint(totalStakeU)
	// t is the distance between
	awl := fs.activeWindowLength(mvwl)
	if awl > 0 {
		factor := num.DecimalFromInt64(mvwl.Nanoseconds()).Div(
			num.DecimalFromInt64(awl.Nanoseconds()))
		tv := num.DecimalFromUint(fs.tradeValue)
		return num.MaxD(totalStake, factor.Mul(tv))
	}
	return totalStake
}

func (fs *FeeSplitter) AvgTradeValue() num.Decimal {
	tv := num.DecimalFromUint(fs.tradeValue)
	if fs.avg.IsZero() {
		if !tv.IsZero() {
			fs.changed = true
		}
		fs.avg = tv
		return tv
	}
	fs.changed = true
	n := num.NewDecimalFromFloat(float64(fs.window))
	nmin := num.NewDecimalFromFloat(float64(fs.window - 1))
	fs.avg = fs.avg.Mul(nmin.Div(n)).Add(tv.Div(n))
	return fs.avg
	// return tv
}

func (fs *FeeSplitter) AddTradeValue(v *num.Uint) {
	fs.tradeValue.AddSum(v)
	fs.changed = true
}

func NewFeeSplitterFromSnapshot(fs *types.FeeSplitter, now time.Time) *FeeSplitter {
	return &FeeSplitter{
		timeWindowStart: fs.TimeWindowStart,
		currentTime:     now,
		tradeValue:      fs.TradeValue,
		changed:         true,
		avg:             fs.Avg,
		window:          fs.Window,
	}
}

func (fs *FeeSplitter) GetState() *types.FeeSplitter {
	fs.changed = false
	return &types.FeeSplitter{
		TimeWindowStart: fs.timeWindowStart,
		TradeValue:      fs.tradeValue,
		Avg:             fs.avg,
		Window:          fs.window,
	}
}

func (fs *FeeSplitter) Changed() bool {
	return fs.changed
}
