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

package supplied

import (
	"code.vegaprotocol.io/vega/libs/num"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func (e *Engine) Payload() *snapshotpb.Payload {
	bidCache := make([]*snapshotpb.LiquidityOffsetProbabilityPair, 0, len(e.pot.bidOffset))
	for i := 0; i < len(e.pot.bidOffset); i++ {
		bidCache = append(bidCache, &snapshotpb.LiquidityOffsetProbabilityPair{Offset: e.pot.bidOffset[i], Probability: e.pot.bidProbability[i].String()})
	}
	askCache := make([]*snapshotpb.LiquidityOffsetProbabilityPair, 0, len(e.pot.askOffset))
	for i := 0; i < len(e.pot.askOffset); i++ {
		askCache = append(askCache, &snapshotpb.LiquidityOffsetProbabilityPair{Offset: e.pot.askOffset[i], Probability: e.pot.askProbability[i].String()})
	}

	return &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquiditySupplied{
			LiquiditySupplied: &snapshotpb.LiquiditySupplied{
				MarketId:         e.marketID,
				BidCache:         bidCache,
				AskCache:         askCache,
				ConsensusReached: e.potInitialised,
			},
		},
	}
}

func (e *Engine) Reload(ls *snapshotpb.LiquiditySupplied) error {
	bidOffsets := make([]uint32, 0, len(ls.BidCache))
	bidProbs := make([]num.Decimal, 0, len(ls.BidCache))
	for _, bid := range ls.BidCache {
		bidOffsets = append(bidOffsets, bid.Offset)
		bidProbs = append(bidProbs, num.MustDecimalFromString(bid.Probability))
	}
	askOffsets := make([]uint32, 0, len(ls.AskCache))
	askProbs := make([]num.Decimal, 0, len(ls.AskCache))
	for _, ask := range ls.AskCache {
		askOffsets = append(askOffsets, ask.Offset)
		askProbs = append(askProbs, num.MustDecimalFromString(ask.Probability))
	}

	e.pot = &probabilityOfTrading{
		bidOffset:      bidOffsets,
		bidProbability: bidProbs,
		askOffset:      askOffsets,
		askProbability: askProbs,
	}
	e.potInitialised = ls.ConsensusReached
	return nil
}
