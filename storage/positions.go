package storage

import (
	"context"
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

// MarketBucket ...
type MarketBucket struct {
	Buys                []*types.Trade
	Sells               []*types.Trade
	BuyVolume           int64
	SellVolume          int64
	MinimumContractSize int64
}

// GetTradesBySideBuckets ...
func (ts *Trade) GetTradesBySideBuckets(ctx context.Context, party string) map[string]*MarketBucket {

	marketBuckets := make(map[string]*MarketBucket)
	tradesByTimestamp, err := ts.GetByParty(ctx, party, 0, 0, false, nil)

	if err != nil {
		return marketBuckets
	}

	if ts.LogPositionStoreDebug {
		ts.log.Debug(fmt.Sprintf("Total trades by timestamp for party %s = %d", party, len(tradesByTimestamp)))
	}

	for idx, trade := range tradesByTimestamp {
		if _, ok := marketBuckets[trade.MarketId]; !ok {
			marketBuckets[trade.MarketId] = &MarketBucket{[]*types.Trade{}, []*types.Trade{}, 0, 0, 1}
		}
		if trade.Buyer == party {
			marketBuckets[trade.MarketId].Buys = append(marketBuckets[trade.MarketId].Buys, tradesByTimestamp[idx])
			marketBuckets[trade.MarketId].BuyVolume += int64(tradesByTimestamp[idx].Size)
		}
		if trade.Seller == party {
			marketBuckets[trade.MarketId].Sells = append(marketBuckets[trade.MarketId].Sells, tradesByTimestamp[idx])
			marketBuckets[trade.MarketId].SellVolume += int64(tradesByTimestamp[idx].Size)
		}
	}

	return marketBuckets
}
