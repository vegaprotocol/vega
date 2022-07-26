// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var ErrTimeWindowNotExpired = errors.New("time window has not expired")

type FeeSplitter struct {
	timeWindowStart time.Time
	currentTime     time.Time
	tradeValue      *num.Uint
	changed         bool
}

func NewFeeSplitter() *FeeSplitter {
	return &FeeSplitter{
		tradeValue: num.Zero(),
		changed:    true,
	}
}

func (fs *FeeSplitter) SetCurrentTime(t time.Time) error {
	if t.Before(fs.timeWindowStart) {
		return errors.New("current time can't be before current window time")
	}
	fs.currentTime = t
	return nil
}

// TimeWindowStart starts or restarts (if active) a current time window.
// This sets the internal timers to `t` and resets the accumulated trade values.
func (fs *FeeSplitter) TimeWindowStart(t time.Time) {
	// reset the trade value for this window
	fs.tradeValue = num.Zero()

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
	return num.MinV(fs.Elapsed(), mvw)
	// t := fs.Elapsed()
	// return t - num.MaxV(t-mvw, 0)
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

func (fs *FeeSplitter) ActualTradeVoluem() num.Decimal {
	return num.DecimalFromUint(fs.tradeValue)
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
	}
}

func (fs *FeeSplitter) GetState() *types.FeeSplitter {
	fs.changed = false
	return &types.FeeSplitter{
		TimeWindowStart: fs.timeWindowStart,
		TradeValue:      fs.tradeValue,
	}
}

func (fs *FeeSplitter) Changed() bool {
	return fs.changed
}
