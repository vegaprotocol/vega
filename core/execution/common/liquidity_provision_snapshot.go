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
	"sort"

	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func NewMarketLiquidityFromSnapshot(
	log *logging.Logger,
	liquidityEngine LiquidityEngine,
	collateral Collateral,
	broker Broker,
	orderBook liquidity.OrderBook,
	equityShares EquityLikeShares,
	marketActivityTracker *MarketActivityTracker,
	fee *fee.Engine,
	marketType marketType,
	marketID string,
	asset string,
	priceFactor num.Decimal,
	state *snapshot.MarketLiquidity,
	amm AMM,
) (*MarketLiquidity, error) {
	priceRange, _ := num.DecimalFromString(state.PriceRange)

	ml := &MarketLiquidity{
		log:                   log,
		liquidityEngine:       liquidityEngine,
		collateral:            collateral,
		broker:                broker,
		orderBook:             orderBook,
		equityShares:          equityShares,
		marketActivityTracker: marketActivityTracker,
		fee:                   fee,
		marketType:            marketType,
		marketID:              marketID,
		asset:                 asset,
		priceFactor:           priceFactor,
		priceRange:            priceRange,
		amm:                   amm,
		tick:                  state.Tick,
	}

	ml.ammStats = make(map[string]*AMMState, len(state.Amm))
	for _, val := range state.Amm {
		stake, err := num.DecimalFromString(val.Stake)
		if err != nil {
			return nil, err
		}
		score, err := num.DecimalFromString(val.Score)
		if err != nil {
			return nil, err
		}
		ml.ammStats[val.Party] = &AMMState{
			stake:    stake,
			score:    score,
			lastTick: val.Tick,
			ltD:      num.DecimalFromInt64(val.Tick),
		}
	}

	return ml, nil
}

func (m *MarketLiquidity) GetState() *snapshot.MarketLiquidity {
	state := &snapshot.MarketLiquidity{
		PriceRange: m.priceRange.String(),
		Tick:       m.tick,
		Amm:        make([]*snapshot.AMMValues, 0, len(m.ammStats)),
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

	return nil
}
