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
