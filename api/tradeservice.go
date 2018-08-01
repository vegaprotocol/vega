package api

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/golang/go/src/pkg/fmt"
	"vega/core"
	"vega/datastore"
	"vega/msg"
	"vega/risk"
)

type TradeService interface {
	Init(app *core.Vega, tradeStore datastore.TradeStore, riskEngine risk.RiskEngine)
	GetByMarket(ctx context.Context, market string, limit uint64) (trades []*msg.Trade, err error)
	GetByParty(ctx context.Context, party string, limit uint64) (trades []*msg.Trade, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (trade *msg.Trade, err error)
	GetByPartyAndId(ctx context.Context, party string, id string) (trade *msg.Trade, err error)
	GetCandles(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error)
	GetPositionsByParty(ctx context.Context, party string) (positions []*msg.MarketPosition, err error)
}

type tradeService struct {
	app        *core.Vega
	tradeStore datastore.TradeStore
	riskEngine risk.RiskEngine
}

func NewTradeService() TradeService {
	return &tradeService{}
}

func (t *tradeService) Init(app *core.Vega, tradeStore datastore.TradeStore, riskEngine risk.RiskEngine) {
	t.app = app
	t.tradeStore = tradeStore
	t.riskEngine = riskEngine
}

func (t *tradeService) GetByMarket(ctx context.Context, market string, limit uint64) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByMarket(market, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]*msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetByParty(ctx context.Context, party string, limit uint64) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByParty(party, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]*msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetByMarketAndId(ctx context.Context, market string, id string) (trade *msg.Trade, err error) {
	tr, err := t.tradeStore.GetByMarketAndId(market, id)
	if err != nil {
		return &msg.Trade{}, err
	}
	return tr.ToProtoMessage(), err
}

func (t *tradeService) GetByPartyAndId(ctx context.Context, party string, id string) (trade *msg.Trade, err error) {
	tr, err := t.tradeStore.GetByPartyAndId(party, id)
	if err != nil {
		return &msg.Trade{}, err
	}
	return tr.ToProtoMessage(), err
}

func (t *tradeService) GetCandles(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error) {
	// compare time and translate it into timestamps
	appCurrentTime := t.app.GetTime()

	delta := appCurrentTime.Sub(since)
	deltaInSeconds := int64(delta.Seconds())
	if deltaInSeconds < 0 {
		return msg.Candles{}, errors.New("INVALID_REQUEST")
	}

	sinceBlock := t.app.GetAbciHeight() - deltaInSeconds
	if sinceBlock < 0 {
		sinceBlock = 0
	}

	c, err := t.tradeStore.GetCandles(market, uint64(sinceBlock), uint64(t.app.GetAbciHeight()), interval)
	if err != nil {
		return msg.Candles{}, err
	}

	aggregationStartTime := appCurrentTime.Add(-delta)
	for i, candle := range c.Candles {
		candleDuration := time.Duration(i*int(interval)) * time.Second
		candle.Date = aggregationStartTime.Add(candleDuration).Format(time.RFC3339)
	}

	return c, nil
}

func (t *tradeService) GetPositionsByParty(ctx context.Context, party string) (positions []*msg.MarketPosition, err error) {
	marketBuckets := t.tradeStore.GetTradesBySideBuckets(party)

	var (
		OpenVolumeSign                int8
		ClosedContracts               int64
		OpenContracts                 int64
		deltaAverageEntryPrice        int64
		avgEntryPriceForOpenContracts int64
		markPrice                     uint64
		riskFactor                    float64
		forwardRiskMargin             float64
	)

	for market, marketBucket := range marketBuckets {
		if marketBucket.BuyVolume > marketBucket.SellVolume {
			OpenVolumeSign = 1
			ClosedContracts = marketBucket.SellVolume
			OpenContracts = marketBucket.BuyVolume - marketBucket.SellVolume
		}

		if marketBucket.BuyVolume == marketBucket.SellVolume {
			OpenVolumeSign = 0
			ClosedContracts = marketBucket.SellVolume
			OpenContracts = 0
		}

		if marketBucket.BuyVolume < marketBucket.SellVolume {
			OpenVolumeSign = -1
			ClosedContracts = marketBucket.BuyVolume
			OpenContracts = marketBucket.BuyVolume - marketBucket.SellVolume
		}

		// long
		if OpenVolumeSign == 1 {
			//// calculate avg entry price for closed and open contracts when position is long
			deltaAverageEntryPrice, avgEntryPriceForOpenContracts =
				t.calculateVolumeEntryPriceWeightedAveragesForLong(marketBucket, OpenContracts, ClosedContracts)
		}

		// net
		if OpenVolumeSign == 0 {
			//// calculate avg entry price for closed and open contracts when position is net
			deltaAverageEntryPrice, avgEntryPriceForOpenContracts =
				t.calculateVolumeEntryPriceWeightedAveragesForNet(marketBucket, OpenContracts, ClosedContracts)
		}

		// short
		if OpenVolumeSign == -1 {
			//// calculate avg entry price for closed and open contracts when position is short
			deltaAverageEntryPrice, avgEntryPriceForOpenContracts =
				t.calculateVolumeEntryPriceWeightedAveragesForShort(marketBucket, OpenContracts, ClosedContracts)
		}

		markPrice, _ = t.tradeStore.GetMarkPrice(market)
		if markPrice == 0 {
			continue
		}

		riskFactor = t.getRiskFactorByMarketAndPositionSign(ctx, market, OpenVolumeSign)

		marketPositions := &msg.MarketPosition{}
		marketPositions.Market = market
		marketPositions.RealisedVolume = int64(ClosedContracts)
		marketPositions.UnrealisedVolume = int64(OpenContracts)
		marketPositions.RealisedPNL = int64(ClosedContracts * deltaAverageEntryPrice)
		marketPositions.UnrealisedPNL = int64(OpenContracts * (int64(markPrice) - avgEntryPriceForOpenContracts))

		forwardRiskMargin = float64(marketPositions.UnrealisedVolume) * float64(markPrice) *
			riskFactor * float64(marketBucket.MinimumContractSize)

		// deliberately loose precision for minimum margin requirement to operate on int64 on the API

		//if minimumMargin is a negative number it means that trader is in credit towards vega
		//if minimumMargin is a positive number it means that trader is in debit towards vega
		marketPositions.MinimumMargin = -marketPositions.UnrealisedPNL + int64(math.Abs(forwardRiskMargin))

		positions = append(positions, marketPositions)
	}

	return positions, nil
}

func (t *tradeService) getRiskFactorByMarketAndPositionSign(ctx context.Context, market string, openVolumeSign int8) float64 {
	riskFactorLong, riskFactorShort, err := t.riskEngine.GetRiskFactors(market)
	if err != nil {
		fmt.Errorf("failed to obtain risk factors from risk engine for market: %s", market)
	}
	var riskFactor float64
	if openVolumeSign == 1 {
		riskFactor = riskFactorLong
	}

	if openVolumeSign == 0 {
		riskFactor = 0
	}

	if openVolumeSign == -1 {
		riskFactor = riskFactorShort
	}

	return riskFactor
}

func (t *tradeService) calculateVolumeEntryPriceWeightedAveragesForLong(marketBucket *datastore.MarketBucket,
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
	for _, trade := range marketBucket.Buys {
		thresholdController += int64(trade.Size)
		if thresholdController <= ClosedContracts {
			buyAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
		} else {
			if thresholdReached == false {
				thresholdReached = true
				buyAverageEntryPriceForClosed +=
					(ClosedContracts - thresholdController + int64(trade.Size)) * int64(trade.Price)
				avgEntryPriceForOpenContracts +=
					(thresholdController - ClosedContracts) * int64(trade.Price)
			} else {
				avgEntryPriceForOpenContracts += int64(trade.Size * trade.Price)
			}
		}
	}

	for _, trade := range marketBucket.Sells {
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

func (t *tradeService) calculateVolumeEntryPriceWeightedAveragesForNet(marketBucket *datastore.MarketBucket,
	OpenContracts, ClosedContracts int64) (int64, int64) {

	var (
		buyAverageEntryPriceForClosed  int64
		sellAverageEntryPriceForClosed int64
		deltaAverageEntryPrice         int64
		avgEntryPriceForOpenContracts  int64
	)

	avgEntryPriceForOpenContracts = 0

	for _, trade := range marketBucket.Buys {
		buyAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
	}
	for _, trade := range marketBucket.Sells {
		sellAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
	}

	if ClosedContracts != 0 {
		deltaAverageEntryPrice = (sellAverageEntryPriceForClosed - buyAverageEntryPriceForClosed) / ClosedContracts
	} else {
		deltaAverageEntryPrice = 0
	}

	return deltaAverageEntryPrice, avgEntryPriceForOpenContracts
}

func (t *tradeService) calculateVolumeEntryPriceWeightedAveragesForShort(marketBucket *datastore.MarketBucket,
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
	for _, trade := range marketBucket.Sells {
		thresholdController += int64(trade.Size)
		if thresholdController <= ClosedContracts {
			sellAverageEntryPriceForClosed += int64(trade.Size * trade.Price)
		} else {
			if thresholdReached == false {
				thresholdReached = true
				sellAverageEntryPriceForClosed +=
					(ClosedContracts - thresholdController + int64(trade.Size)) * int64(trade.Price)
				avgEntryPriceForOpenContracts +=
					(thresholdController - ClosedContracts) * int64(trade.Price)
			} else {
				avgEntryPriceForOpenContracts += int64(trade.Size * trade.Price)
			}
		}
	}

	for _, trade := range marketBucket.Buys {
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

//func (t *memTradeStore) CalculateMarginRequirements() {
//	riskFactorLong, riskFactorShort, err := v.riskEngine.GetRiskFactors()
//	// for every party in the system calculate open volume on all the markets
//	parties, err := v.PartyStore.GetAllParties()
//	if err != nil {
//		return
//	}
//
//	var forwardRiskFactor uint64
//
//	for _, party := range parties {
//		forwardRiskFactor = 0
//		positionsMap := v.TradeStore.GetPositionsByParty(party)
//		for _, positions := range positionsMap {
//			openVolume := positions.UnrealisedVolume
//			if openVolume > int64(0) {
//				forwardRiskFactor = v.CalculateMarginRequirementsForLong(riskFactorLong, openVolume)
//			}
//			if openVolume < int64(0) {
//				forwardRiskFactor = v.CalculateMarginRequirementsForShort(riskFactorShort, openVolume)
//			}
//
//		}
//	}
//}
//
//func (v *Vega) CalculateMarginRequirementsForLong(riskFactorLong, openVolume int64) uint64 {
//	return uint64(riskFactorLong * int64(math.Abs(float64(openVolume))))
//}
//
//func (v *Vega) CalculateMarginRequirementsForShort(riskFactorShort, openVolume int64) uint64 {
//	return uint64(riskFactorShort * int64(math.Abs(float64(openVolume))))
//}
