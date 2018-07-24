package datastore

import "vega/proto"

//type Exposure struct {
//	Position int64
//	Volume   int64
//}

//func (store *memTradeStore) GetNetPositionsByParty(party string) map[string]Exposure {
//	positions := make(map[string]Exposure, 0)
//	tradesByTimestamp, err := store.GetByParty(party, GetParams{})
//	if err != nil {
//		return positions
//	}
//	for _, trade := range tradesByTimestamp {
//		//fmt.Printf("T: %+v\n", trade)
//		if exposure, ok := positions[trade.Market]; ok {
//			if trade.Buyer == party {
//				exposure.Position += int64(trade.Price * trade.Size)
//				exposure.Volume += int64(trade.Size)
//			}
//			if trade.Seller == party {
//				exposure.Position -= int64(trade.Price * trade.Size)
//				exposure.Volume -= int64(trade.Size)
//			}
//			positions[trade.Market] = exposure
//			//fmt.Printf("positions.positions[trade.Market] = %+v\n", positions.positions[trade.Market])
//		} else {
//			positions[trade.Market] = Exposure{}
//			if trade.Buyer == party {
//				exposure.Position += int64(trade.Price * trade.Size)
//				exposure.Volume += int64(trade.Size)
//			}
//			if trade.Seller == party {
//				exposure.Position -= int64(trade.Price * trade.Size)
//				exposure.Volume -= int64(trade.Size)
//			}
//			positions[trade.Market] = exposure
//			//fmt.Printf("positions.positions[trade.Market] = %+v\n", positions.positions[trade.Market])
//		}
//	}
//	return positions
//}

//func (t *memTradeStore) GetPositionByParty(party string) map[string]uint64 {
//	positions := make(map[string]uint64, 0)
//	tradesByTimestamp, err := t.GetByParty(party, GetParams{})
//	if err != nil {
//		return positions
//	}
//
//	for _, trade := range tradesByTimestamp {
//		if _, ok := positions[trade.Market]; ok {
//			positions[trade.Market] += trade.Size
//		} else {
//			positions[trade.Market] = trade.Size
//		}
//	}
//	return positions
//}

//type TradeMatcher struct {
//	market    string
//	trade     *Trade
//	remaining uint64
//	closedAt  uint64
//}

//func (t *memTradeStore) GetProfitAndLossByParty1(party string) map[string]int64 {
//	/*
//	   	1. get all the trades by timestamp and pack into new struct with remaining flag
//	   	2. iterate over new struct take each element if remaining !=0 and start from begining comparing with trades inside the struct until you hit yourself
//	   	3. when you compare check if type of trade is OPPOSITE, else keep iterating
//	   	4. if type is opposite subtract remaining on both trades that matched.
//	       5. And keep iterating until the source trade is not fully netted or hits itself.
//	*/
//	positions := make(map[string]int64, 0)
//	tradesByTimestamp, err := t.GetByParty(party, GetParams{})
//	if err != nil {
//		return positions
//	}
//
//	var fifoTradeMatcher map[string][]*TradeMatcher
//	for _, trade := range tradesByTimestamp {
//		fifoTradeMatcher[trade.Market] = append(fifoTradeMatcher[trade.Market], &TradeMatcher{trade.Market, &trade, trade.Size, 0})
//	}
//
//	for market := range fifoTradeMatcher {
//		for mainIdx := range fifoTradeMatcher[market] {
//			for innerIdx := range fifoTradeMatcher[market] {
//
//				if fifoTradeMatcher[market][innerIdx].trade.Id == fifoTradeMatcher[market][mainIdx].trade.Id {
//					break
//				}
//
//				if fifoTradeMatcher[market][innerIdx].trade.Buyer == fifoTradeMatcher[market][mainIdx].trade.Buyer ||
//					fifoTradeMatcher[market][innerIdx].trade.Seller == fifoTradeMatcher[market][mainIdx].trade.Seller {
//					continue
//				}
//
//				if fifoTradeMatcher[market][innerIdx].remaining > fifoTradeMatcher[market][mainIdx].remaining {
//					fifoTradeMatcher[market][innerIdx].remaining -= fifoTradeMatcher[market][mainIdx].remaining
//					fifoTradeMatcher[market][innerIdx].remaining = 0
//				} else {
//					fifoTradeMatcher[market][mainIdx].remaining -= fifoTradeMatcher[market][innerIdx].remaining
//					fifoTradeMatcher[market][mainIdx].remaining = 0
//				}
//
//				if fifoTradeMatcher[market][mainIdx].remaining == 0 {
//					break
//				}
//			}
//		}
//	}
//	return positions
//}

type MarketBucket struct {
	buys       []*Trade
	sells      []*Trade
	buyVolume  int64
	sellVolume int64
}

func (t *memTradeStore) GetPositionsByParty(party string) map[string]*msg.MarketPosition {
	positions := make(map[string]*msg.MarketPosition, 0)
	tradesByTimestamp, err := t.GetByParty(party, GetParams{})
	if err != nil {
		return positions
	}

	marketBuckets := make(map[string]*MarketBucket, 0)
	for _, trade := range tradesByTimestamp {
		if _, ok := marketBuckets[trade.Market]; !ok {
			marketBuckets[trade.Market] = &MarketBucket{[]*Trade{}, []*Trade{}, 0, 0}
		}
		if trade.Buyer == party {
			marketBuckets[trade.Market].buys = append(marketBuckets[trade.Market].buys, &trade)
			marketBuckets[trade.Market].buyVolume += int64(trade.Size)
		}
		if trade.Seller == party {
			marketBuckets[trade.Market].sells = append(marketBuckets[trade.Market].sells, &trade)
			marketBuckets[trade.Market].sellVolume += int64(trade.Size)
		}
	}

	var (
		OpenVolumeSign                 int8
		ClosedContracts                int64
		OpenContracts                  int64
		buyAverageEntryPriceForClosed  int64
		sellAverageEntryPriceForClosed int64
		deltaAverageEntryPrice         int64

		avgEntryPriceForOpenContracts int64
		threshold                     int64
		markPrice                     uint64
	)

	for market, marketBucket := range marketBuckets {
		if marketBucket.buyVolume > marketBucket.sellVolume {
			OpenVolumeSign = 1
			ClosedContracts = marketBucket.sellVolume
			OpenContracts = marketBucket.buyVolume - marketBucket.sellVolume
		}

		if marketBucket.buyVolume == marketBucket.sellVolume {
			OpenVolumeSign = 0
			ClosedContracts = marketBucket.sellVolume
			OpenContracts = 0
		}

		if marketBucket.buyVolume < marketBucket.sellVolume {
			OpenVolumeSign = -1
			ClosedContracts = marketBucket.buyVolume
			OpenContracts = marketBucket.sellVolume - marketBucket.buyVolume
		}

		// long
		if OpenVolumeSign == 1 {
			// calculate avg entry price for closed and open contracts
			for _, trade := range marketBucket.buys {
				threshold += int64(trade.Size)
				if threshold <= ClosedContracts {
					buyAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
				} else {
					avgEntryPriceForOpenContracts += int64(trade.Size * trade.Price)
				}
			}

			for _, trade := range marketBucket.sells {
				sellAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
			}

			deltaAverageEntryPrice = (buyAverageEntryPriceForClosed - sellAverageEntryPriceForClosed) / ClosedContracts
		}

		// net
		if OpenVolumeSign == 0 {
			avgEntryPriceForOpenContracts = 0
			for _, trade := range marketBucket.sells {
				sellAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
			}

			deltaAverageEntryPrice = sellAverageEntryPriceForClosed / ClosedContracts

		}

		// short
		if OpenVolumeSign == -1 {
			// calculate avg entry price for closed and open contracts
			for _, trade := range marketBucket.sells {
				threshold += int64(trade.Size)
				if threshold <= ClosedContracts {
					sellAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
				} else {
					avgEntryPriceForOpenContracts += int64(trade.Size * trade.Price)
				}
			}

			for _, trade := range marketBucket.sells {
				buyAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
			}

			deltaAverageEntryPrice = (sellAverageEntryPriceForClosed - buyAverageEntryPriceForClosed) / ClosedContracts
		}

		markPrice, _ = t.GetMarkPrice(market)
		if markPrice == 0 {
			continue
		}
		positions[market] = &msg.MarketPosition{}
		positions[market].RealisedVolume = uint64(ClosedContracts)
		positions[market].UnrealisedVolume = uint64(OpenContracts)
		positions[market].RealisedPNL = uint64(ClosedContracts * deltaAverageEntryPrice
		positions[market].UnrealisedPNL = OpenContracts * (int64(markPrice) - avgEntryPriceForOpenContracts)
	}

	return positions
}
