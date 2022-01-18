package supplied

import (
	snapshotpb "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types/num"
)

func (e *Engine) HasUpdates() bool {
	return e.changed
}

func (e *Engine) ResetUpdated() {
	e.changed = false
}

func (e *Engine) Payload() *snapshotpb.Payload {
	bidCache := make([]*snapshotpb.LiquidityPriceProbabilityPair, 0, len(e.pot.bidOffset))
	for i := 0; i < len(e.pot.bidOffset); i++ {
		bidCache = append(bidCache, &snapshotpb.LiquidityPriceProbabilityPair{Price: e.pot.bidOffset[i].String(), Probability: e.pot.bidProbability[i].String()})
	}
	askCache := make([]*snapshotpb.LiquidityPriceProbabilityPair, 0, len(e.pot.askOffset))
	for i := 0; i < len(e.pot.askOffset); i++ {
		askCache = append(askCache, &snapshotpb.LiquidityPriceProbabilityPair{Price: e.pot.askOffset[i].String(), Probability: e.pot.askProbability[i].String()})
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
	bidOffsets := make([]num.Decimal, 0, len(ls.BidCache))
	bidProbs := make([]num.Decimal, 0, len(ls.BidCache))
	for _, bid := range ls.BidCache {
		bidOffsets = append(bidOffsets, num.MustDecimalFromString(bid.Price))
		bidProbs = append(bidProbs, num.MustDecimalFromString(bid.Probability))
	}
	askOffsets := make([]num.Decimal, 0, len(ls.AskCache))
	askProbs := make([]num.Decimal, 0, len(ls.AskCache))
	for _, ask := range ls.AskCache {
		askOffsets = append(askOffsets, num.MustDecimalFromString(ask.Price))
		askProbs = append(askProbs, num.MustDecimalFromString(ask.Probability))
	}

	e.pot = &probabilityOfTrading{
		bidOffset:      bidOffsets,
		bidProbability: bidProbs,
		askOffset:      askOffsets,
		askProbability: askProbs,
	}
	e.potInitialised = ls.ConsensusReached
	e.changed = true
	return nil
}
