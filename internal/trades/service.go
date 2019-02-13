package trades

import (
	"context"
	"math"

	"vega/internal/filtering"
	"vega/internal/logging"
	"vega/internal/storage"

	types "vega/proto"
)

type Service interface {
	GetByMarket(market string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error)
	GetByParty(party string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error)
	GetByOrderId(orderId string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error)
	GetByMarketAndId(market string, id string) (trade *types.Trade, err error)
	GetByPartyAndId(party string, id string) (trade *types.Trade, err error)
	GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error)
	ObservePositions(ctx context.Context, party string) (positions <-chan types.MarketPosition, ref uint64)
	ObserveTrades(ctx context.Context, market *string, party *string) (orders <-chan []types.Trade, ref uint64)
}

type tradeService struct {
	*Config
	tradeStore storage.TradeStore
	riskStore  storage.RiskStore
}

func NewTradeService(config *Config, tradeStore storage.TradeStore, riskStore storage.RiskStore) (Service, error) {
	return &tradeService{
		Config:     config,
		tradeStore: tradeStore,
		riskStore:  riskStore,
	}, nil
}

func (t *tradeService) GetByMarket(market string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error) {
	trades, err = t.tradeStore.GetByMarket(market, filters)
	if err != nil {
		return nil, err
	}
	return trades, err
}

func (t *tradeService) GetByParty(party string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error) {
	trades, err = t.tradeStore.GetByParty(party, filters)
	if err != nil {
		return nil, err
	}
	return trades, err
}

func (t *tradeService) GetByMarketAndId(market string, id string) (trade *types.Trade, err error) {
	trade, err = t.tradeStore.GetByMarketAndId(market, id)
	if err != nil {
		return &types.Trade{}, err
	}
	return trade, err
}

func (t *tradeService) GetByPartyAndId(party string, id string) (trade *types.Trade, err error) {
	trade, err = t.tradeStore.GetByPartyAndId(party, id)
	if err != nil {
		return &types.Trade{}, err
	}
	return trade, err
}

func (t *tradeService) GetByOrderId(orderId string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error) {
	trades, err = t.tradeStore.GetByOrderId(orderId, filters)
	if err != nil {
		return nil, err
	}
	return trades, err
}

func (t *tradeService) ObserveTrades(ctx context.Context, market *string, party *string) (<-chan []types.Trade, uint64) {
	trades := make(chan []types.Trade)
	internal := make(chan []types.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []types.Trade, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		<-ctx.Done()
		t.log.Debugf("TradeService -> Subscriber closed connection: %d [%s]", id, ip)
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			t.log.Errorf("Error un-subscribing when context.Done() on TradeService for subscriber %d [%s]: %s", id, ip, err)
		}
	}(ref, internal, ctx)

	go func(id uint64, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		for v := range internal {

			validatedTrades := make([]types.Trade, 0)
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
					t.log.Debugf("TradeService -> Trades for subscriber %d [%s] sent successfully", ref, ip)
				default:
					t.log.Debugf("TradeService -> Trades for subscriber %d [%s] not sent", ref, ip)
				}
			}
		}
		t.log.Debugf("TradeService -> Channel for subscriber %d [%s] has been closed", ref, ip)
	}(ref, ctx)

	return trades, ref
}

func (t *tradeService) ObservePositions(ctx context.Context, party string) (<-chan types.MarketPosition, uint64) {
	positions := make(chan types.MarketPosition)
	internal := make(chan []types.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []types.Trade, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		<-ctx.Done()
		t.log.Debugf("TradeService -> Positions subscriber closed connection: % [%s]", id, ip)
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			t.log.Errorf("Error un-subscribing positions when context.Done() on TradeService for subscriber %d [%s]: %s", id, ip, err)
		}
	}(ref, internal, ctx)

	go func(id uint64, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		for range internal {
			mapOfMarketPositions, err := t.GetPositionsByParty(ctx, party)
			if err != nil {
				t.log.Errorf("Error getting positions by party on TradeService for subscriber %d [%s]: %s", id, ip, err)
			}
			for _, marketPositions := range mapOfMarketPositions {
				select {
				case positions <- *marketPositions:
					t.log.Debugf("TradeService -> Positions for subscriber %d [%s] sent successfully", ref, ip)
				default:
					t.log.Debugf("TradeService -> Positions for subscriber %d [%s] not sent", ref, ip)
				}
			}
		}
		t.log.Debugf("TradeService -> Channel for positions subscriber %d [%s] has been closed", ref, ip)
	}(ref, ctx)

	return positions, ref
}

func (t *tradeService) GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error) {
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

	t.log.Debugf("Total market buckets = %d", len(marketBuckets))

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

		marketPositions := &types.MarketPosition{}
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

	t.log.Debugf("Positions calculated: %d", len(positions))

	return positions, nil
}

func (t *tradeService) getRiskFactorByMarketAndPositionSign(ctx context.Context, market string, openVolumeSign int8) float64 {
	rf, err := t.riskStore.GetByMarket(market)
	if err != nil {
		t.log.Errorf("failed to obtain risk factors from risk engine for market: %s", market)
	}

	t.log.Debugf("Risk Factors = %v/%v", rf.Long, rf.Short)

	var riskFactor float64
	if openVolumeSign == 1 {
		riskFactor = rf.Long
	}

	if openVolumeSign == 0 {
		riskFactor = 0
	}

	if openVolumeSign == -1 {
		riskFactor = rf.Short
	}

	return riskFactor
}

func (t *tradeService) calculateVolumeEntryPriceWeightedAveragesForLong(marketBucket *storage.MarketBucket,
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

func (t *tradeService) calculateVolumeEntryPriceWeightedAveragesForNet(marketBucket *storage.MarketBucket,
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

func (t *tradeService) calculateVolumeEntryPriceWeightedAveragesForShort(marketBucket *storage.MarketBucket,
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
