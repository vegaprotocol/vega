package datastore

func (store *memTradeStore) GetNetPositionsByParty(party string) map[string]int64 {
	positions := make(map[string]int64, 0)
	tradesByTimestamp, err := store.GetByParty(party, GetParams{})
	if err != nil {
		return positions
	}
	for _, trade := range tradesByTimestamp {
		//fmt.Printf("T: %+v\n", trade)
		if exposure, ok := positions[trade.Market]; ok {
			if trade.Buyer == party {
				exposure += int64(trade.Price * trade.Size)
			}
			if trade.Seller == party {
				exposure -= int64(trade.Price * trade.Size)
			}
			positions[trade.Market] = exposure
			//fmt.Printf("positions.positions[trade.Market] = %+v\n", positions.positions[trade.Market])
		} else {
			positions[trade.Market] = 0
			if trade.Buyer == party {
				exposure += int64(trade.Price * trade.Size)
			}
			if trade.Seller == party {
				exposure -= int64(trade.Price * trade.Size)
			}
			positions[trade.Market] = exposure
			//fmt.Printf("positions.positions[trade.Market] = %+v\n", positions.positions[trade.Market])
		}
	}
	return positions
}
