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
	bidCache := make([]*snapshotpb.LiquidityPriceProbabilityPair, 0, len(e.pot.bidPrice))
	for i := 0; i < len(e.pot.bidPrice); i++ {
		bidCache = append(bidCache, &snapshotpb.LiquidityPriceProbabilityPair{Price: e.pot.bidPrice[i].String(), Probability: e.pot.bidProbability[i].String()})
	}
	askCache := make([]*snapshotpb.LiquidityPriceProbabilityPair, 0, len(e.pot.askPrice))
	for i := 0; i < len(e.pot.askPrice); i++ {
		askCache = append(askCache, &snapshotpb.LiquidityPriceProbabilityPair{Price: e.pot.askPrice[i].String(), Probability: e.pot.askProbability[i].String()})
	}

	return &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquiditySupplied{
			LiquiditySupplied: &snapshotpb.LiquiditySupplied{
				MarketId: e.marketID,
				BidCache: bidCache,
				AskCache: askCache,
			},
		},
	}
}

func (e *Engine) Reload(ls *snapshotpb.LiquiditySupplied) error {
	bidPrices := make([]num.Decimal, 0, len(ls.BidCache))
	bidProbs := make([]num.Decimal, 0, len(ls.BidCache))
	for _, bid := range ls.BidCache {
		bidPrices = append(bidPrices, num.MustDecimalFromString(bid.Price))
		bidProbs = append(bidProbs, num.MustDecimalFromString(bid.Probability))
	}
	askPrices := make([]num.Decimal, 0, len(ls.AskCache))
	askProbs := make([]num.Decimal, 0, len(ls.AskCache))
	for _, ask := range ls.AskCache {
		askPrices = append(askPrices, num.MustDecimalFromString(ask.Price))
		askProbs = append(askProbs, num.MustDecimalFromString(ask.Probability))
	}

	e.pot = &probabilityOfTrading{
		bidPrice:       bidPrices,
		bidProbability: bidProbs,
		askPrice:       askPrices,
		askProbability: askProbs,
	}
	e.changed = true
	return nil
}
