package datastore

type MarketBucket struct {
	Buys                []*Trade
	Sells               []*Trade
	BuyVolume           int64
	SellVolume          int64
	MinimumContractSize int64
}

func (t *memTradeStore) GetTradesBySideBuckets(party string) map[string]*MarketBucket {
	marketBuckets := make(map[string]*MarketBucket, 0)
	tradesByTimestamp, err := t.GetByParty(party, GetParams{})
	if err != nil {
		return marketBuckets
	}

	for idx, trade := range tradesByTimestamp {
		if _, ok := marketBuckets[trade.Market]; !ok {
			marketBuckets[trade.Market] = &MarketBucket{[]*Trade{}, []*Trade{}, 0, 0, 1}
		}
		if trade.Buyer == party {
			marketBuckets[trade.Market].Buys = append(marketBuckets[trade.Market].Buys, &tradesByTimestamp[idx])
			marketBuckets[trade.Market].BuyVolume += int64(tradesByTimestamp[idx].Size)
		}
		if trade.Seller == party {
			marketBuckets[trade.Market].Sells = append(marketBuckets[trade.Market].Sells, &tradesByTimestamp[idx])
			marketBuckets[trade.Market].SellVolume += int64(tradesByTimestamp[idx].Size)
		}
	}
	return marketBuckets
}
