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
	GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error)
	ObservePositions(ctx context.Context, retries int, party string) (positions <-chan *types.MarketPosition, ref uint64)
	ObserveTrades(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Trade, ref uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination newmocks/trade_store_mock.go -package newmocks code.vegaprotocol.io/vega/internal/trades TradeStore
type TradeStore interface {
	GetByMarket(ctx context.Context, market string, params *filtering.TradeQueryFilters) ([]*types.Trade, error)
	GetByMarketAndId(ctx context.Context, market string, id string) (*types.Trade, error)
	GetByParty(ctx context.Context, party string, params *filtering.TradeQueryFilters) ([]*types.Trade, error)
	GetByPartyAndId(ctx context.Context, party string, id string) (*types.Trade, error)
	GetByOrderId(ctx context.Context, orderId string, params *filtering.TradeQueryFilters) ([]*types.Trade, error)
	GetTradesBySideBuckets(ctx context.Context, party string) map[string]*storage.MarketBucket
	GetMarkPrice(ctx context.Context, market string) (uint64, error)
	Subscribe(trades chan<- []types.Trade) uint64
	Unsubscribe(id uint64) error
}

//go:generate go run github.com/golang/mock/mockgen -destination newmocks/risk_store_mock.go -package newmocks code.vegaprotocol.io/vega/internal/trades RiskStore
type RiskStore interface {
	GetByMarket(market string) (*types.RiskFactor, error)
}

type tradeService struct {
	*Config
	tradeStore TradeStore
	riskStore  RiskStore
}

func NewTradeService(config *Config, tradeStore TradeStore, riskStore RiskStore) (Service, error) {
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

func (t *tradeService) ObserveTrades(ctx context.Context, retries int, market *string, party *string) (<-chan []types.Trade, uint64) {
	trades := make(chan []types.Trade)
	internal := make(chan []types.Trade)
	ref := t.tradeStore.Subscribe(internal)
	retryCount := retries

	go func() {
		ip := logging.IPAddressFromContext(ctx)
		ctx, cfunc := context.WithCancel(ctx)
		defer cfunc()
		for {
			select {
			case <-ctx.Done():
				t.log.Debug(
					"Trades subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				if err := t.tradeStore.Unsubscribe(ref); err != nil {
					t.log.Error(
						"Failure un-subscribing trades subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(trades)
				return
			case v := <-internal:
				// max length of validated == length of data from channel
				validatedTrades := make([]types.Trade, 0, len(v))
				for _, item := range v {
					// if market is nil or matches item market and party was nil, or matches seller or buyer
					if (market == nil || item.Market == *market) && (party == nil || item.Seller == *party || item.Buyer == *party) {
						validatedTrades = append(validatedTrades, item)
					}
				}
				if len(validatedTrades) == 0 {
					continue
				}
				select {
				case trades <- validatedTrades:
					retryCount = retries
					t.log.Debug(
						"Trades for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
					)
				default:
					retryCount--
					if retryCount == 0 {
						t.log.Warn(
							"Trades subscriber has hit the retry limit",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
							logging.Int("retries", retries),
						)
						cfunc()
					}
					t.log.Debug(
						"Trades for subscriber not sent",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
					)
				}
			}
		}
	}()

	return trades, ref
}

func (t *tradeService) ObservePositions(ctx context.Context, retries int, party string) (<-chan *types.MarketPosition, uint64) {
	positions := make(chan *types.MarketPosition)
	internal := make(chan []types.Trade)
	ref := t.tradeStore.Subscribe(internal)
	retryCount := retries

	go func() {
		ip := logging.IPAddressFromContext(ctx)
		ctx, cfunc := context.WithCancel(ctx)
		defer cfunc()
		for {
			select {
			case <-ctx.Done():
				t.log.Debug(
					"Positions subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				if err := t.tradeStore.Unsubscribe(ref); err != nil {
					t.log.Error(
						"Failure un-subscribing positions subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(positions)
				return
			case <-internal: // again, we're using this channel to detect state changes, the data itself isn't relevant
				mapOfMarketPositions, err := t.GetPositionsByParty(ctx, party)
				if err != nil {
					t.log.Error(
						"Failed to get positions for subscriber (getPositionsByParty)",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
					continue
				}
				for _, marketPositions := range mapOfMarketPositions {
					marketPositions := marketPositions
					select {
					case positions <- marketPositions:
						retryCount = retries
						t.log.Debug(
							"Positions for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
					default:
						retryCount--
						if retryCount == 0 {
							t.log.Warn(
								"Positions subscriber has hit the retry limit",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip),
								logging.Int("retries", retries),
							)
							cfunc()
						}
						t.log.Debug(
							"Positions for subscriber not sent",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
					}
				}
			}
		}
	}()

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
