package trades

import (
	"context"
	"fmt"
	"math"

	"code.vegaprotocol.io/vega/internal/filtering"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

type Service interface {
	GetByMarket(ctx context.Context, market string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error)
	GetByParty(ctx context.Context, party string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error)
	GetByOrderId(ctx context.Context, orderId string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (trade *types.Trade, err error)
	GetByPartyAndId(ctx context.Context, party string, id string) (trade *types.Trade, err error)
	GetLastTrade(ctx context.Context) (trade *types.Trade)
	GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error)
	ObservePositions(ctx context.Context, party string) (positions <-chan *types.MarketPosition, ref uint64)
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

func (t *tradeService) GetByMarket(ctx context.Context, market string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error) {
	trades, err = t.tradeStore.GetByMarket(ctx, market, filters)
	if err != nil {
		return nil, err
	}
	return trades, err
}

func (t *tradeService) GetByParty(ctx context.Context, party string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error) {
	trades, err = t.tradeStore.GetByParty(ctx, party, filters)
	if err != nil {
		return nil, err
	}
	return trades, err
}

func (t *tradeService) GetByMarketAndId(ctx context.Context, market string, id string) (trade *types.Trade, err error) {
	trade, err = t.tradeStore.GetByMarketAndId(ctx, market, id)
	if err != nil {
		return &types.Trade{}, err
	}
	return trade, err
}

func (t *tradeService) GetByPartyAndId(ctx context.Context, party string, id string) (trade *types.Trade, err error) {
	trade, err = t.tradeStore.GetByPartyAndId(ctx, party, id)
	if err != nil {
		return &types.Trade{}, err
	}
	return trade, err
}

func (t *tradeService) GetByOrderId(ctx context.Context, orderId string, filters *filtering.TradeQueryFilters) (trades []*types.Trade, err error) {
	trades, err = t.tradeStore.GetByOrderId(ctx, orderId, filters)
	if err != nil {
		return nil, err
	}
	return trades, err
}

func (t *tradeService) GetLastTrade(ctx context.Context) (trade *types.Trade) {
	return t.tradeStore.GetLastTrade(ctx)
}

func (t *tradeService) ObserveTrades(ctx context.Context, market *string, party *string) (<-chan []types.Trade, uint64) {
	trades := make(chan []types.Trade)
	internal := make(chan []types.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []types.Trade, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		<-ctx.Done()
		t.log.Debug("Trades subscriber closed connection",
			logging.Uint64("id", id),
			logging.String("ip-address", ip))
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			t.log.Error("Failure un-subscribing trades subscriber when context.Done()",
				logging.Uint64("id", id),
				logging.String("ip-address", ip),
				logging.Error(err))
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
					t.log.Debug("Trades for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				default:
					t.log.Debug("Trades for subscriber not sent",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				}
			}
		}
		t.log.Debug("Trades subscriber channel has been closed",
			logging.Uint64("ref", ref),
			logging.String("ip-address", ip))
	}(ref, ctx)

	return trades, ref
}

func (t *tradeService) ObservePositions(ctx context.Context, party string) (<-chan *types.MarketPosition, uint64) {
	positions := make(chan *types.MarketPosition)
	internal := make(chan []types.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []types.Trade, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		<-ctx.Done()
		t.log.Debug("Positions subscriber closed connection",
			logging.Uint64("id", id),
			logging.String("ip-address", ip))
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			t.log.Error("Failure un-subscribing positions subscriber when context.Done()",
				logging.Uint64("id", id),
				logging.String("ip-address", ip),
				logging.Error(err))
		}
	}(ref, internal, ctx)

	go func(id uint64, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		for range internal {
			mapOfMarketPositions, err := t.GetPositionsByParty(ctx, party)
			if err != nil {
				t.log.Error("Failed to get positions for subscriber (getPositionsByParty)",
					logging.Uint64("id", id),
					logging.Uint64("ref", ref),
					logging.String("ip-address", ip),
					logging.Error(err))
			}
			for _, marketPositions := range mapOfMarketPositions {
				marketPositions := marketPositions
				select {
				case positions <- marketPositions:
					t.log.Debug("Positions for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				default:
					t.log.Debug("Positions for subscriber not sent",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				}
			}
		}
		t.log.Debug("Positions subscriber channel has been closed",
			logging.Uint64("ref", ref),
			logging.String("ip-address", ip))
	}(ref, ctx)

	return positions, ref
}

func (t *tradeService) GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error) {

	t.log.Debug("Calculate positions for party",
		logging.String("party-id", party))

	marketBuckets := t.tradeStore.GetTradesBySideBuckets(ctx, party)

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

	t.log.Debug("Loaded market buckets for party",
		logging.String("party-id", party),
		logging.Int("total-buckets", len(marketBuckets)))

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

		markPrice, _ = t.tradeStore.GetMarkPrice(ctx, market)
		if markPrice == 0 {
			continue
		}

		riskFactor, err = t.getRiskFactorByMarketAndPositionSign(ctx, market, OpenVolumeSign)
		if err != nil {
			return nil, err
		}

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

	t.log.Debug("Positions for party calculated",
		logging.String("party-id", party),
		logging.Int("total-buckets", len(positions)))

	return positions, nil
}

func (t *tradeService) getRiskFactorByMarketAndPositionSign(ctx context.Context, market string, openVolumeSign int8) (float64, error) {
	rf, err := t.riskStore.GetByMarket(market)
	if err != nil {
		t.log.Error("Failed to obtain risk factors from risk engine",
			logging.String("market-id", market))
		return -1, errors.Wrap(err, fmt.Sprintf("Failed to obtain risk factors from risk engine for market: %s", market))
	}

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

	return riskFactor, nil
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
