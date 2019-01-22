package api

import (
	"context"
	"math"
	"vega/core"
	"vega/datastore"
	"vega/msg"
	"vega/log"
	"vega/filters"
)

type TradeService interface {
	Init(app *core.Vega, tradeStore datastore.TradeStore)
	Stop()

	GetByMarket(market string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error)
	GetByParty(party string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error)
	GetByOrderId(orderId string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error)
	GetByMarketAndId(market string, id string) (trade *msg.Trade, err error)
	GetByPartyAndId(party string, id string) (trade *msg.Trade, err error)

	GetPositionsByParty(ctx context.Context, party string) (positions []*msg.MarketPosition, err error)
	ObservePositions(ctx context.Context, party string) (positions <-chan msg.MarketPosition, ref uint64)
	ObserveTrades(ctx context.Context, market *string, party *string) (orders <-chan []msg.Trade, ref uint64)
}

type tradeService struct {
	app        *core.Vega
	tradeStore datastore.TradeStore
}

func NewTradeService() TradeService {
	return &tradeService{}
}

func (t *tradeService) Init(app *core.Vega, tradeStore datastore.TradeStore) {
	t.app = app
	t.tradeStore = tradeStore
}

func (t *tradeService) Stop() {
	t.tradeStore.Close()
}

func (t *tradeService) GetByMarket(market string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error) {
	trades, err = t.tradeStore.GetByMarket(market, filters)
	if err != nil {
		return nil, err
	}
	return trades, err
}

func (t *tradeService) GetByParty(party string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error) {
	trades, err = t.tradeStore.GetByParty(party, filters)
	if err != nil {
		return nil, err
	}
	return trades, err
}

func (t *tradeService) GetByMarketAndId(market string, id string) (trade *msg.Trade, err error) {
	trade, err = t.tradeStore.GetByMarketAndId(market, id)
	if err != nil {
		return &msg.Trade{}, err
	}
	return trade, err
}

func (t *tradeService) GetByPartyAndId(party string, id string) (trade *msg.Trade, err error) {
	trade, err = t.tradeStore.GetByPartyAndId(party, id)
	if err != nil {
		return &msg.Trade{}, err
	}
	return trade, err
}

func (t *tradeService) GetByOrderId(orderId string, filters *filters.TradeQueryFilters) (trades []*msg.Trade, err error) {
	trades, err = t.tradeStore.GetByOrderId(orderId, filters)
	if err != nil {
		return nil, err
	}
	return trades, err
}

func (t *tradeService) ObserveTrades(ctx context.Context, market *string, party *string) (<-chan []msg.Trade, uint64) {
	trades := make(chan []msg.Trade)
	internal := make(chan []msg.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []msg.Trade, ctx context.Context) {
		ip := ipAddressFromContext(ctx)
		<-ctx.Done()
		log.Debugf("TradeService -> Subscriber closed connection: %d [%s]", id, ip)
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing when context.Done() on TradeService for subscriber %d [%s]: %s", id, ip, err)
		}
	}(ref, internal, ctx)

	go func(id uint64, ctx context.Context) {
		ip := ipAddressFromContext(ctx)
		for v := range internal {
			
			validatedTrades := make([]msg.Trade, 0)
			for _, item := range v {
				if market != nil && item.Market != *market {
					continue
				}
				if party != nil && (item.Seller != *party && item.Buyer != *party) {
					continue
				}
				validatedTrades = append(validatedTrades, item)
			}
			
			if len(validatedTrades) > 0 {
				select {
					case trades <- validatedTrades:
						log.Debugf("TradeService -> Trades for subscriber %d [%s] sent successfully", ref, ip)
					default:
						log.Debugf("TradeService -> Trades for subscriber %d [%s] not sent", ref, ip)
				}
			}
		}
		log.Debugf("TradeService -> Channel for subscriber %d [%s] has been closed", ref, ip)
	}(ref, ctx)

	return trades, ref
}

func (t *tradeService) ObservePositions(ctx context.Context, party string) (<-chan msg.MarketPosition, uint64) {
	positions := make(chan msg.MarketPosition)
	internal := make(chan []msg.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []msg.Trade, ctx context.Context) {
		ip := ipAddressFromContext(ctx)
		<-ctx.Done()
		log.Debugf("TradeService -> Positions subscriber closed connection: % [%s]", id, ip)
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing positions when context.Done() on TradeService for subscriber %d [%s]: %s", id, ip, err)
		}
	}(ref, internal, ctx)

	go func(id uint64, ctx context.Context) {
		ip := ipAddressFromContext(ctx)
		for range internal {
			mapOfMarketPositions, err := t.GetPositionsByParty(ctx, party)
			if err != nil {
				log.Errorf("Error getting positions by party on TradeService for subscriber %d [%s]: %s", id, ip, err)
			}
			for _, marketPositions := range mapOfMarketPositions {
				select {
					case positions <- *marketPositions:
						log.Debugf("TradeService -> Positions for subscriber %d [%s] sent successfully", ref, ip)
					default:
						log.Debugf("TradeService -> Positions for subscriber %d [%s] not sent", ref, ip)
				}
			}
		}
		log.Debugf("TradeService -> Channel for positions subscriber %d [%s] has been closed", ref, ip)
	}(ref, ctx)

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

	log.Debugf("Total market buckets = %d", len(marketBuckets))

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
	
	log.Debugf("Positions calculated: %d", len(positions))

	return positions, nil
}

func (t *tradeService) getRiskFactorByMarketAndPositionSign(ctx context.Context, market string, openVolumeSign int8) float64 {
	riskFactorLong, riskFactorShort, err := t.app.GetRiskFactors(market)
	if err != nil {
		log.Errorf("failed to obtain risk factors from risk engine for market: %s", market)
	}

	log.Debugf("Risk Factors = %v/%v", riskFactorLong, riskFactorShort)

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
