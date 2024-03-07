package common

import "code.vegaprotocol.io/vega/libs/num"

type AMMState struct {
	stake    num.Decimal // stake during epoch
	score    num.Decimal // running liquidity score
	lastTick int64       // last time update, not used currently, but useful when we want to start caching this stuff
	ltD      num.Decimal // last time update, just in decimal because it saves on pointless conversions between int64 and num.Decimal
}

func newAMMState(count int64) *AMMState {
	// prevent underflow
	if count == 0 {
		count = 1
	}
	return &AMMState{
		stake:    num.DecimalZero(),
		score:    num.DecimalZero(),
		lastTick: count - 1,
		ltD:      num.DecimalZero(),
	}
}

// UpdateTick is equivalent to calls to UpdateStake, UpdateScore, and IncrementTick.
func (a *AMMState) UpdateTick(stake, score num.Decimal) {
	tick := a.ltD.Add(num.DecimalOne())
	a.stake = a.ltD.Mul(a.stake).Add(stake).Div(tick)
	a.score = a.ltD.Mul(a.score).Add(score).Div(tick)
	a.lastTick++
	a.ltD = a.ltD.Add(num.DecimalOne())
}

// UpdateStake updates the time-weighted average stake during epoch.
func (a *AMMState) UpdateStake(stake num.Decimal) {
	tick := a.ltD.Add(num.DecimalOne())
	// ((current_tick - 1) * old_stake + new_stake)/current_tick
	// ((1 * 100) + 100)/ 2 == 100, checks out
	a.stake = a.ltD.Mul(a.stake).Add(stake).Div(tick)
}

// UpdateScore updates the current epoch score.
func (a *AMMState) UpdateScore(score num.Decimal) {
	tick := a.ltD.Add(num.DecimalOne())
	// ((current_tick - 1) * old_score + new_score)/current_tick
	// (( 2 * 50 ) + 200) / 3 = 100, checks out
	a.score = a.ltD.Mul(a.score).Add(score).Div(tick)
}

// IncrementTick increments the internal tick/time counter.
func (a *AMMState) IncrementTick() {
	a.lastTick++
	a.ltD.Add(num.DecimalOne())
}

// StartEpoch resets the internal tick counter, ready for the new epoch to start
func (a *AMMState) StartEpoch() {
	a.lastTick = 0
	a.ltD = num.DecimalZero()
}
