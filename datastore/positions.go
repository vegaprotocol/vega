package datastore

import (
	"math"
	"vega/proto"
)

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

func (t *memTradeStore) GetTradesBySideBuckets(party string) map[string]*MarketBucket {
	marketBuckets := make(map[string]*MarketBucket, 0)
	tradesByTimestamp, err := t.GetByParty(party, GetParams{})
	if err != nil {
		return marketBuckets
	}

	for idx, trade := range tradesByTimestamp {
		if _, ok := marketBuckets[trade.Market]; !ok {
			marketBuckets[trade.Market] = &MarketBucket{[]*Trade{}, []*Trade{}, 0, 0}
		}
		if trade.Buyer == party {
			marketBuckets[trade.Market].buys = append(marketBuckets[trade.Market].buys, &tradesByTimestamp[idx])
			marketBuckets[trade.Market].buyVolume += int64(tradesByTimestamp[idx].Size)
		}
		if trade.Seller == party {
			marketBuckets[trade.Market].sells = append(marketBuckets[trade.Market].sells, &tradesByTimestamp[idx])
			marketBuckets[trade.Market].sellVolume += int64(tradesByTimestamp[idx].Size)
		}
	}
	return marketBuckets
}

func (t *memTradeStore) CalculateVolumeEntryPriceWeightedAveragesForLong(marketBucket *MarketBucket,
	OpenContracts, ClosedContracts int64) (int64, int64) {

	var (
		buyAverageEntryPriceForClosed  int64
		sellAverageEntryPriceForClosed int64
		deltaAverageEntryPrice         int64
		avgEntryPriceForOpenContracts  int64
		thresholdController            int64
		thresholdReached               bool
	)

	// calculate avg entry price for closed and open contracts
	for _, trade := range marketBucket.buys {
		thresholdController += int64(trade.Size)
		if thresholdController <= ClosedContracts {
			buyAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
		} else {
			if thresholdReached == false {
				thresholdReached = true
				buyAverageEntryPriceForClosed += (ClosedContracts - thresholdController + int64(trade.Size)) * int64(trade.Price)
				avgEntryPriceForOpenContracts += (thresholdController - ClosedContracts) * int64(trade.Price)
			} else {
				avgEntryPriceForOpenContracts += int64(trade.Size * trade.Price)
			}
		}
	}

	for _, trade := range marketBucket.sells {
		sellAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
	}

	if ClosedContracts != 0 {
		deltaAverageEntryPrice = (sellAverageEntryPriceForClosed - buyAverageEntryPriceForClosed) / ClosedContracts
	} else {
		deltaAverageEntryPrice = 0
	}

	if OpenContracts != 0 {
		avgEntryPriceForOpenContracts = int64(math.Abs(float64(avgEntryPriceForOpenContracts / OpenContracts)))
	} else {
		avgEntryPriceForOpenContracts = 0
	}

	return deltaAverageEntryPrice, avgEntryPriceForOpenContracts
}

func (t *memTradeStore) CalculateVolumeEntryPriceWeightedAveragesForNet(marketBucket *MarketBucket,
	OpenContracts, ClosedContracts int64) (int64, int64) {

	var (
		buyAverageEntryPriceForClosed  int64
		sellAverageEntryPriceForClosed int64
		deltaAverageEntryPrice         int64
		avgEntryPriceForOpenContracts  int64
	)

	avgEntryPriceForOpenContracts = 0

	for _, trade := range marketBucket.buys {
		buyAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
	}
	for _, trade := range marketBucket.sells {
		sellAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
	}

	if ClosedContracts != 0 {
		deltaAverageEntryPrice = (sellAverageEntryPriceForClosed - buyAverageEntryPriceForClosed) / ClosedContracts
	} else {
		deltaAverageEntryPrice = 0
	}

	return deltaAverageEntryPrice, avgEntryPriceForOpenContracts
}

func (t *memTradeStore) CalculateVolumeEntryPriceWeightedAveragesForShort(marketBucket *MarketBucket,
	OpenContracts, ClosedContracts int64) (int64, int64) {

	var (
		buyAverageEntryPriceForClosed  int64
		sellAverageEntryPriceForClosed int64
		deltaAverageEntryPrice         int64
		avgEntryPriceForOpenContracts  int64
		thresholdController            int64
		thresholdReached               bool
	)

	// calculate avg entry price for closed and open contracts
	for _, trade := range marketBucket.sells {
		thresholdController += int64(trade.Size)
		if thresholdController <= ClosedContracts {
			sellAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
		} else {
			if thresholdReached == false {
				thresholdReached = true
				sellAverageEntryPriceForClosed += (ClosedContracts - thresholdController + int64(trade.Size)) * int64(trade.Price)
				avgEntryPriceForOpenContracts += (thresholdController - ClosedContracts) * int64(trade.Price)
			} else {
				avgEntryPriceForOpenContracts += int64(trade.Size * trade.Price)
			}
		}
	}

	for _, trade := range marketBucket.buys {
		buyAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
	}

	if ClosedContracts != 0 {
		deltaAverageEntryPrice = (sellAverageEntryPriceForClosed - buyAverageEntryPriceForClosed) / ClosedContracts
	} else {
		deltaAverageEntryPrice = 0
	}

	if OpenContracts != 0 {
		avgEntryPriceForOpenContracts = int64(math.Abs(float64(avgEntryPriceForOpenContracts / OpenContracts)))
	} else {
		avgEntryPriceForOpenContracts = 0
	}

	return deltaAverageEntryPrice, avgEntryPriceForOpenContracts
}

func (t *memTradeStore) GetPositionsByParty(party string) map[string]*msg.MarketPosition {
	positions := make(map[string]*msg.MarketPosition, 0)
	marketBuckets := t.GetTradesBySideBuckets(party)

	var (
		OpenVolumeSign                int8
		ClosedContracts               int64
		OpenContracts                 int64
		deltaAverageEntryPrice        int64
		avgEntryPriceForOpenContracts int64
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
			OpenContracts = marketBucket.buyVolume - marketBucket.sellVolume
		}

		// long
		if OpenVolumeSign == 1 {
			//// calculate avg entry price for closed and open contracts when position is long
			deltaAverageEntryPrice, avgEntryPriceForOpenContracts = t.CalculateVolumeEntryPriceWeightedAveragesForLong(marketBucket,
				OpenContracts, ClosedContracts)
		}

		// net
		if OpenVolumeSign == 0 {
			//// calculate avg entry price for closed and open contracts when position is net
			deltaAverageEntryPrice, avgEntryPriceForOpenContracts = t.CalculateVolumeEntryPriceWeightedAveragesForNet(marketBucket,
				OpenContracts, ClosedContracts)
		}

		// short
		if OpenVolumeSign == -1 {
			//// calculate avg entry price for closed and open contracts when position is short
			deltaAverageEntryPrice, avgEntryPriceForOpenContracts = t.CalculateVolumeEntryPriceWeightedAveragesForShort(marketBucket,
				OpenContracts, ClosedContracts)
		}

		markPrice, _ = t.GetMarkPrice(market)
		if markPrice == 0 {
			continue
		}

		positions[market] = &msg.MarketPosition{}
		positions[market].Market = market
		positions[market].RealisedVolume = int64(ClosedContracts)
		positions[market].UnrealisedVolume = int64(OpenContracts)
		positions[market].RealisedPNL = int64(ClosedContracts * deltaAverageEntryPrice)
		positions[market].UnrealisedPNL = int64(OpenContracts * (int64(markPrice) - avgEntryPriceForOpenContracts))
	}

	return positions
}
