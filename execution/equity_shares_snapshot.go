package execution

import (
	"sort"

	"code.vegaprotocol.io/vega/types"
)

func NewEquitySharesFromSnapshot(state *types.EquityShare) *EquityShares {
	lps := map[string]*lp{}

	for _, slp := range state.Lps {
		lps[slp.ID] = &lp{
			stake: slp.Stake,
			share: slp.Share,
			avg:   slp.Avg,
		}
	}

	return &EquityShares{
		mvp:                 state.Mvp,
		openingAuctionEnded: state.OpeningAuctionEnded,
		lps:                 lps,
		stateChanged:        true,
	}
}

func (es EquityShares) Changed() bool {
	return es.stateChanged
}

func (es EquityShares) GetState() *types.EquityShare {
	lps := make([]*types.EquityShareLP, 0, len(es.lps))
	for id, lp := range es.lps {
		lps = append(lps, &types.EquityShareLP{
			ID:    id,
			Stake: lp.stake,
			Share: lp.share,
			Avg:   lp.avg,
		})
	}

	// Need to make sure the items are correctly sorted to make this deterministic
	sort.Slice(lps, func(i, j int) bool {
		return lps[i].ID < lps[j].ID
	})

	es.stateChanged = false

	return &types.EquityShare{
		Mvp:                 es.mvp,
		OpeningAuctionEnded: es.openingAuctionEnded,
		Lps:                 lps,
	}
}
