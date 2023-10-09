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

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func NewEquitySharesFromSnapshot(state *types.EquityShare) *EquityShares {
	lps := map[string]*lp{}

	totalPStake, totalVStake := num.DecimalZero(), num.DecimalZero()
	for _, slp := range state.Lps {
		lps[slp.ID] = &lp{
			stake:  slp.Stake,
			share:  slp.Share,
			avg:    slp.Avg,
			vStake: slp.VStake,
		}
		totalPStake = totalPStake.Add(slp.Stake)
		totalVStake = totalVStake.Add(slp.VStake)
	}

	return &EquityShares{
		mvp:                 state.Mvp,
		r:                   state.R,
		totalVStake:         totalVStake,
		totalPStake:         totalPStake,
		openingAuctionEnded: state.OpeningAuctionEnded,
		lps:                 lps,
	}
}

func (es EquityShares) Changed() bool {
	return true
}

func (es *EquityShares) GetState() *types.EquityShare {
	lps := make([]*types.EquityShareLP, 0, len(es.lps))
	for id, lp := range es.lps {
		lps = append(lps, &types.EquityShareLP{
			ID:     id,
			Stake:  lp.stake,
			Share:  lp.share,
			Avg:    lp.avg,
			VStake: lp.vStake,
		})
	}

	// Need to make sure the items are correctly sorted to make this deterministic
	sort.Slice(lps, func(i, j int) bool {
		return lps[i].ID < lps[j].ID
	})

	return &types.EquityShare{
		Mvp:                 es.mvp,
		R:                   es.r,
		OpeningAuctionEnded: es.openingAuctionEnded,
		Lps:                 lps,
	}
}
