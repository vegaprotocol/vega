// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package common

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

func (fs *FeeSplitter) TradeValue() *num.Uint {
	return fs.tradeValue.Clone()
}

func (fs *FeeSplitter) AddTradeValue(tv *num.Uint) {
	fs.tradeValue.AddSum(tv)
	fs.changed = true
}

func (fs *FeeSplitter) SetTradeValue(tv *num.Uint) {
	fs.tradeValue = tv.Clone()
}

// TimeWindowStart starts or restarts (if active) a current time window.
// This sets the internal timers to `t` and resets the accumulated trade values.
func (fs *FeeSplitter) TimeWindowStart(t time.Time) {
	// if we have an average value, that means we left the opening auction
	// and we can increase the window to the next value
	if !fs.avg.IsZero() {
		fs.window++
	} else if !fs.tradeValue.IsZero() {
		// if tradeValue is set, but the average hasn't been updated, it means we're currently leaving opening auction
		// we should set the average accordingly: avg == trade_value, but keep the window as-is.
		// next time we calculate the avg for window == 1, the value should be avg + trade_val (opening auction + trade value)
		fs.avg = num.DecimalFromUint(fs.tradeValue)
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

func (fs *FeeSplitter) SetElapsed(e time.Duration) {
	fs.timeWindowStart = fs.currentTime.Add(-e)
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
	// end of 1st window after opening auction
	if fs.window == 1 {
		fs.avg = fs.avg.Add(tv)
		if !tv.IsZero() {
			fs.changed = true
		}
		return fs.avg
	}
	fs.changed = true
	// n == 2 or more
	n := num.DecimalFromInt64(int64(fs.window))
	// nmin == 1 or more
	nmin := num.DecimalFromInt64(int64(fs.window - 1))
	// avg = avg * ((n-1)/n) + tv/n
	fs.avg = fs.avg.Mul(nmin.Div(n)).Add(tv.Div(n))
	return fs.avg
	// return tv
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
