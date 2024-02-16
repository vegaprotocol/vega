package common

import (
	"sort"

	"code.vegaprotocol.io/vega/libs/num"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func (m *MarketLiquidity) GetState() *snapshot.MarketLiquidity {
	state := &snapshot.MarketLiquidity{
		Tick: m.tick,
		Amm:  make([]*snapshot.AMMValues, 0, len(m.ammStats)),
	}
	for id, vals := range m.ammStats {
		v := &snapshot.AMMValues{
			Party: id,
			Stake: vals.stake.String(),
			Score: vals.score.String(),
			Tick:  vals.lastTick,
		}
		state.Amm = append(state.Amm, v)
	}
	sort.SliceStable(state.Amm, func(i, j int) bool {
		return state.Amm[i].Party < state.Amm[j].Party
	})
	return state
}

func (m *MarketLiquidity) SetState(ml *snapshot.MarketLiquidity) error {
	if ml == nil {
		return nil
	}
	m.tick = ml.Tick
	m.ammStats = make(map[string]*AMMState, len(ml.Amm))
	for _, val := range ml.Amm {
		stake, err := num.DecimalFromString(val.Stake)
		if err != nil {
			return err
		}
		score, err := num.DecimalFromString(val.Score)
		if err != nil {
			return err
		}
		m.ammStats[val.Party] = &AMMState{
			stake:    stake,
			score:    score,
			lastTick: val.Tick,
			ltD:      num.DecimalFromInt64(val.Tick),
		}
	}
	return nil
}
