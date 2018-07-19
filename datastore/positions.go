package datastore

//type Position struct {
//	market string
//	startPrice uint64
//	closePrice uint64
//	startSide msg.Side
//	closeSide msg.Side
//}

type Positions struct {
	positions map[string]uint64
}

func (store *memTradeStore) calculateNetPositions(party string) (positions *Positions){
	tradesByTimestamp, err := store.GetByParty(party, GetParams{})
	if err != nil {
		return &Positions{}
	}
	for _, trade := range tradesByTimestamp {
		if position, ok := positions.positions[trade.Market]; ok {
			if trade.Buyer == party {
				position += trade.Price * trade.Size
			}
			if trade.Seller == party {
				position -= trade.Price * trade.Size
			}
			positions.positions[party] = position
		} else {
			positions.positions[trade.Market] = 0
			if trade.Buyer == party {
				position += trade.Price * trade.Size
			}
			if trade.Seller == party {
				position -= trade.Price * trade.Size
			}
			positions.positions[party] = position
		}
	}
	return &Positions{}
}
