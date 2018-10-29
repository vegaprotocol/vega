package api

import (
	"context"
	"math"
	"time"

	"vega/core"
	"vega/datastore"
	"vega/msg"
	"vega/log"
	"vega/vegatime"
	"vega/filters"
)

type TradeService interface {
	Init(app *core.Vega)
	ObserveTrades(ctx context.Context, market *string, party *string) (orders <-chan []msg.Trade, ref uint64)

	GetByMarket(ctx context.Context, market string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error)
	GetByParty(ctx context.Context, party string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (trade *msg.Trade, err error)
	GetByPartyAndId(ctx context.Context, party string, id string) (trade *msg.Trade, err error)

	GetCandles(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error)

	GetLastCandles(ctx context.Context, market string, last uint64, interval uint64) (candles msg.Candles, err error)
	GetCandleSinceBlock(ctx context.Context, market string, sinceBlock uint64) (candle *msg.Candle, time time.Time, err error)

	GetLatestBlock() (blockNow uint64)

	GetPositionsByParty(ctx context.Context, party string) (positions []*msg.MarketPosition, err error)
	ObservePositions(ctx context.Context, party string) (positions <-chan msg.MarketPosition, ref uint64)
}

type tradeService struct {
	app        *core.Vega
	tradeStore datastore.TradeStore
}

func NewTradeService() TradeService {
	return &tradeService{}
}

func (t *tradeService) Init(app *core.Vega) {
	t.app = app
	//t.tradeStore = tradeStore
}

func (t *tradeService) GetByMarket(ctx context.Context, market string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByMarket(market, filters)
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]*msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetByParty(ctx context.Context, party string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByParty(party, filters)
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
	vtc := vegatime.NewVegaTimeConverter(t.app)
	sinceBlock := vtc.TimeToBlock(since)

	c, err := t.tradeStore.GetCandles(market, sinceBlock, uint64(t.app.GetChainHeight()), interval)
	if err != nil {
		return msg.Candles{}, err
	}

	for _, candle := range c.Candles {
		candle.Date = vtc.BlockToTime(candle.OpenBlockNumber).Format(time.RFC3339)
	}
	return c, nil
}

func (t *tradeService) GetLastCandles(ctx context.Context, market string, last uint64, interval uint64) (candles msg.Candles, err error) {
	vtc := vegatime.NewVegaTimeConverter(t.app)
	
	// Convert last N candles to vega-time
	latestBlock := uint64(t.GetLatestBlock())
	offset := uint64(interval) * uint64(last)
	sinceBlock := uint64(0)
	if offset < latestBlock {
		sinceBlock = latestBlock - offset
	}
	
	c, err := t.tradeStore.GetCandles(market, sinceBlock, latestBlock, interval)
	if err != nil {
		return msg.Candles{}, err
	}

	for _, candle := range c.Candles {
		candle.Date = vtc.BlockToTime(candle.OpenBlockNumber).Format(time.RFC3339)
	}
	return c, nil
}

// GetCandleSinceBlock will return exactly one candle for the last interval (seconds) from the current VEGA time.
// It can return an empty candle if there was no trading activity in the last interval (seconds)
// This function is designed to be used in partnership with a streaming endpoint where the candle is filled up
// with a fixed interval e.g. sixty seconds
func (t *tradeService) GetCandleSinceBlock(ctx context.Context, market string, sinceBlock uint64) (*msg.Candle, time.Time, error) {
	vtc := vegatime.NewVegaTimeConverter(t.app)
	height := t.GetLatestBlock()
	c, err := t.tradeStore.GetCandle(market, sinceBlock, uint64(height))
	if err != nil {
		return nil, vtc.BlockToTime(sinceBlock), err
	}
	c.Date = vtc.BlockToTime(c.OpenBlockNumber).Format(time.RFC3339)
	return c, vtc.BlockToTime(sinceBlock), nil
}

// GetLatestBlock is a helper function for now that will allow the caller to provide a sinceBlock to the GetCandleSinceBlock
// function. TODO when we have the VEGA time package we can do all kinds of fantastic block->real time ops without this call
func (t *tradeService) GetLatestBlock() uint64 {
	height := t.app.GetChainHeight()
	return uint64(height)
}

func (t *tradeService) ObserveTrades(ctx context.Context, market *string, party *string) (<-chan []msg.Trade, uint64) {
	trades := make(chan []msg.Trade)
	internal := make(chan []datastore.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []datastore.Trade) {
		<-ctx.Done()
		log.Debugf("TradeService -> Subscriber closed connection: %d", id)
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing when context.Done() on TradeService for id: %d", id)
		}
	}(ref, internal)

	go func(id uint64) {
		var validatedTrades []msg.Trade
		for v := range internal {
			validatedTrades = nil
			for _, item := range v {
				if market != nil && item.Market != *market {
					continue
				}
				if party != nil && (item.Seller != *party || item.Buyer != *party) {
					continue
				}

				validatedTrades = append(validatedTrades, *item.ToProtoMessage())
			}
			trades <- validatedTrades
		}
		log.Debugf("TradeService -> Channel for subscriber %d has been closed", ref)
	}(ref)

	return trades, ref
}

func (t *tradeService) ObservePositions(ctx context.Context, party string) (<-chan msg.MarketPosition, uint64) {
	positions := make(chan msg.MarketPosition)
	internal := make(chan []datastore.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []datastore.Trade) {
		<-ctx.Done()
		log.Debugf("TradeService -> Positions subscriber closed connection: %d", id)
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing positions when context.Done() on TradeService for id: %d", id, err)
		}
	}(ref, internal)

	go func(id uint64) {
		for range internal {
			mapOfMarketPositions, err := t.GetPositionsByParty(ctx, party)
			if err != nil {
				log.Errorf("Error getting positions by party on TradeService for id: %d", id, err)
			}
			for _, marketPositions := range mapOfMarketPositions {
				positions <- *marketPositions
			}
		}
		log.Debugf("TradeService -> Channel for positions subscriber %d has been closed", ref)
	}(ref)

	return positions, ref
}

func (t *tradeService) GetPositionsByParty(ctx context.Context, party string) (positions []*msg.MarketPosition, err error) {
	marketBuckets := t.tradeStore.GetTradesBySideBuckets(party)

	var (
		OpenVolumeSign                int8
		ClosedContracts               int64
		OpenContracts                 int64
		deltaAverageEntryPrice        float64
		avgEntryPriceForOpenContracts float64
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
		marketPositions.RealisedPNL = int64(float64(ClosedContracts) * deltaAverageEntryPrice)
		marketPositions.UnrealisedPNL = int64(float64(OpenContracts) * (float64(markPrice) - avgEntryPriceForOpenContracts))
		marketPositions.AverageEntryPrice = uint64(avgEntryPriceForOpenContracts)

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
	riskFactorLong, riskFactorShort, err := t.app.GetRiskFactors(market)
	if err != nil {
		log.Errorf("failed to obtain risk factors from risk engine for market: %s", market)
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
	OpenContracts, ClosedContracts int64) (float64, float64) {

	var (
		buyAggregateEntryPriceForClosed       int64
		sellAggregateEntryPriceForClosed      int64
		deltaAverageEntryPrice              float64
		aggregateEntryPriceForOpenContracts int64
		avgEntryPriceForOpenContracts       float64
		thresholdController                 int64
		thresholdReached                    bool
	)

	// calculate avg entry price for closed and open contracts
	for _, trade := range marketBucket.Buys {
		thresholdController += int64(trade.Size)
		if thresholdController <= ClosedContracts {
			buyAggregateEntryPriceForClosed += int64(trade.Size * trade.Price)
		} else {
			if thresholdReached == false {
				thresholdReached = true
				buyAggregateEntryPriceForClosed +=
					(ClosedContracts - thresholdController + int64(trade.Size)) * int64(trade.Price)
				aggregateEntryPriceForOpenContracts +=
					(thresholdController - ClosedContracts) * int64(trade.Price)
			} else {
				aggregateEntryPriceForOpenContracts += int64(trade.Size * trade.Price)
			}
		}
	}

	for _, trade := range marketBucket.Sells {
		sellAggregateEntryPriceForClosed += int64(trade.Size * trade.Price)
	}

	if ClosedContracts != 0 {
		deltaAverageEntryPrice = float64(sellAggregateEntryPriceForClosed-buyAggregateEntryPriceForClosed) / float64(ClosedContracts)
	} else {
		deltaAverageEntryPrice = 0
	}

	if OpenContracts != 0 {
		avgEntryPriceForOpenContracts = float64(math.Abs(float64(aggregateEntryPriceForOpenContracts) / float64(OpenContracts)))
	} else {
		avgEntryPriceForOpenContracts = 0
	}

	return deltaAverageEntryPrice, avgEntryPriceForOpenContracts
}

func (t *tradeService) calculateVolumeEntryPriceWeightedAveragesForNet(marketBucket *datastore.MarketBucket,
	OpenContracts, ClosedContracts int64) (float64, float64) {

	var (
		buyAggregateEntryPriceForClosed  int64
		sellAggregateEntryPriceForClosed int64
		deltaAverageEntryPrice           float64
		avgEntryPriceForOpenContracts    float64
	)

	avgEntryPriceForOpenContracts = 0

	for _, trade := range marketBucket.Buys {
		buyAggregateEntryPriceForClosed += int64(trade.Size * trade.Price)
	}
	for _, trade := range marketBucket.Sells {
		sellAggregateEntryPriceForClosed += int64(trade.Size * trade.Price)
	}

	if ClosedContracts != 0 {
		deltaAverageEntryPrice = float64(sellAggregateEntryPriceForClosed-buyAggregateEntryPriceForClosed) / float64(ClosedContracts)
	} else {
		deltaAverageEntryPrice = 0
	}

	return deltaAverageEntryPrice, avgEntryPriceForOpenContracts
}

func (t *tradeService) calculateVolumeEntryPriceWeightedAveragesForShort(marketBucket *datastore.MarketBucket,
	OpenContracts, ClosedContracts int64) (float64, float64) {

	var (
		buyAggregateEntryPriceForClosed     int64
		sellAggregateEntryPriceForClosed    int64
		deltaAverageEntryPrice              float64
		aggregateEntryPriceForOpenContracts int64
		avgEntryPriceForOpenContracts       float64
		thresholdController                 int64
		thresholdReached                    bool
	)

	// calculate avg entry price for closed and open contracts
	for _, trade := range marketBucket.Sells {
		thresholdController += int64(trade.Size)
		if thresholdController <= ClosedContracts {
			sellAggregateEntryPriceForClosed += int64(trade.Size * trade.Price)
		} else {
			if thresholdReached == false {
				thresholdReached = true
				sellAggregateEntryPriceForClosed +=
					(ClosedContracts - thresholdController + int64(trade.Size)) * int64(trade.Price)
				aggregateEntryPriceForOpenContracts +=
					(thresholdController - ClosedContracts) * int64(trade.Price)
			} else {
				aggregateEntryPriceForOpenContracts += int64(trade.Size * trade.Price)
			}
		}
	}

	for _, trade := range marketBucket.Buys {
		buyAggregateEntryPriceForClosed += int64(trade.Size * trade.Price)
	}

	if ClosedContracts != 0 {
		deltaAverageEntryPrice = float64(sellAggregateEntryPriceForClosed-buyAggregateEntryPriceForClosed) / float64(ClosedContracts)
	} else {
		deltaAverageEntryPrice = 0
	}

	if OpenContracts != 0 {
		avgEntryPriceForOpenContracts = math.Abs(float64(aggregateEntryPriceForOpenContracts) / float64(OpenContracts))
	} else {
		avgEntryPriceForOpenContracts = 0
	}

	return deltaAverageEntryPrice, avgEntryPriceForOpenContracts
}
