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
