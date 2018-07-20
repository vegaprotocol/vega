package datastore

type Exposure struct {
	Position int64
	Volume int64
}


func (store *memTradeStore) GetNetPositionsByParty(party string) map[string]Exposure {
	positions := make(map[string]Exposure, 0)
	tradesByTimestamp, err := store.GetByParty(party, GetParams{})
	if err != nil {
		return positions
	}
	for _, trade := range tradesByTimestamp {
		//fmt.Printf("T: %+v\n", trade)
		if exposure, ok := positions[trade.Market]; ok {
			if trade.Buyer == party {
				exposure.Position += int64(trade.Price * trade.Size)
				exposure.Volume += int64(trade.Size)
			}
			if trade.Seller == party {
				exposure.Position -= int64(trade.Price * trade.Size)
				exposure.Volume -= int64(trade.Size)
			}
			positions[trade.Market] = exposure
			//fmt.Printf("positions.positions[trade.Market] = %+v\n", positions.positions[trade.Market])
		} else {
			positions[trade.Market] = Exposure{}
			if trade.Buyer == party {
				exposure.Position += int64(trade.Price * trade.Size)
				exposure.Volume += int64(trade.Size)
			}
			if trade.Seller == party {
				exposure.Position -= int64(trade.Price * trade.Size)
				exposure.Volume -= int64(trade.Size)
			}
			positions[trade.Market] = exposure
			//fmt.Printf("positions.positions[trade.Market] = %+v\n", positions.positions[trade.Market])
		}
	}
	return positions
}

//func (store *memTradeStore) GetProfitAndLoss(party string, market string) map[string]int64 {
///*
//	Get your all trades for this market
//
//
// */
//}
