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
	"sort"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func NewEquitySharesFromSnapshot(state *types.EquityShare) *EquityShares {
	lps := map[string]*lp{}

	totalV, totalP := num.DecimalZero(), num.DecimalZero()
	for _, slp := range state.Lps {
		lps[slp.ID] = &lp{
			stake:  slp.Stake,
			share:  slp.Share,
			avg:    slp.Avg,
			vStake: slp.VStake,
		}
		totalV = total.Add(slp.VStake)
		totalP = totalP.Add(slp.Stake)
	}

	return &EquityShares{
		mvp:                 state.Mvp,
		pMvp:                state.PMvp,
		r:                   state.R,
		totalVStake:         totalV,
		totalPStake:         totalP,
		openingAuctionEnded: state.OpeningAuctionEnded,
		lps:                 lps,
		stateChanged:        true,
	}
}

func (es EquityShares) Changed() bool {
	return es.stateChanged
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

	es.stateChanged = false

	return &types.EquityShare{
		Mvp:                 es.mvp,
		PMvp:                es.pMvp,
		R:                   es.r,
		OpeningAuctionEnded: es.openingAuctionEnded,
		Lps:                 lps,
	}
}
