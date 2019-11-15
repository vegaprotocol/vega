package trades

import (
	"context"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"

	"github.com/pkg/errors"
)

// TradeStore represents an abstraction over a trade storage
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_store_mock.go -package mocks code.vegaprotocol.io/vega/trades TradeStore
type TradeStore interface {
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) ([]*types.Trade, error)
	GetByMarketAndID(ctx context.Context, market string, id string) (*types.Trade, error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, market *string) ([]*types.Trade, error)
	GetByPartyAndID(ctx context.Context, party string, id string) (*types.Trade, error)
	GetByOrderID(ctx context.Context, orderID string, skip, limit uint64, descending bool, market *string) ([]*types.Trade, error)
	GetTradesBySideBuckets(ctx context.Context, party string) map[string]*storage.MarketBucket
	GetMarkPrice(ctx context.Context, market string) (uint64, error)
	Subscribe(trades chan<- []types.Trade) uint64
	Unsubscribe(id uint64) error
}

// RiskStore represents an abstraction over a Risk storage
//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_store_mock.go -package mocks code.vegaprotocol.io/vega/trades RiskStore
type RiskStore interface {
	GetByMarket(market string) (*types.RiskFactor, error)
}

// Svc is the service handling trades
type Svc struct {
	Config
	log                     *logging.Logger
	tradeStore              TradeStore
	riskStore               RiskStore
	positionsSubscribersCnt int32
	tradeSubscribersCnt     int32
}

// NewService instanciate a new Trades service
func NewService(log *logging.Logger, config Config, tradeStore TradeStore, riskStore RiskStore) (*Svc, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		log:        log,
		Config:     config,
		tradeStore: tradeStore,
		riskStore:  riskStore,
	}, nil
}

// ReloadConf update the internal configuration of the service
func (s *Svc) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.Config = cfg
}

func (s *Svc) checkPagination(limit *uint64) error {
	if *limit == 0 {
		*limit = s.Config.PageSizeDefault
		// Do not return yet. The default may have been set to a number greater
		// than the maximum.
	}

	if *limit > s.Config.PageSizeMaximum {
		return fmt.Errorf("invalid pagination limit: %d is greater than %d", *limit, s.Config.PageSizeMaximum)
	}

	return nil
}

//GetByMarket returns a list of trades for a given market
func (s *Svc) GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) (trades []*types.Trade, err error) {
	if err = s.checkPagination(&limit); err != nil {
		return nil, err
	}

	trades, err = s.tradeStore.GetByMarket(ctx, market, skip, limit, descending)
	if err != nil {
		return nil, err
	}
	return trades, err
}

// GetByParty returns a list of trade for a given party
func (s *Svc) GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, market *string) (trades []*types.Trade, err error) {
	if err = s.checkPagination(&limit); err != nil {
		return nil, err
	}

	trades, err = s.tradeStore.GetByParty(ctx, party, skip, limit, descending, market)
	if err != nil {
		return nil, err
	}
	return trades, err
}

// GetByMarketAndID return a single trade per its ID and the market it was created in
func (s *Svc) GetByMarketAndID(ctx context.Context, market string, id string) (trade *types.Trade, err error) {
	trade, err = s.tradeStore.GetByMarketAndID(ctx, market, id)
	if err != nil {
		return &types.Trade{}, err
	}
	return trade, err
}

// GetByPartyAndID returns a single trade, filter through a party ID and the trade ID
func (s *Svc) GetByPartyAndID(ctx context.Context, party string, id string) (trade *types.Trade, err error) {
	trade, err = s.tradeStore.GetByPartyAndID(ctx, party, id)
	if err != nil {
		return &types.Trade{}, err
	}
	return trade, err
}

// GetByOrderID return a list of trades filter by order ID (even the buy or sell side of the trade)
func (s *Svc) GetByOrderID(ctx context.Context, orderID string) (trades []*types.Trade, err error) {
	trades, err = s.tradeStore.GetByOrderID(ctx, orderID, 0, 0, false, nil)
	if err != nil {
		return nil, err
	}
	return trades, err
}

// GetTradeSubscribersCount return the count of subscribers to the Trades updates
func (s *Svc) GetTradeSubscribersCount() int32 {
	return atomic.LoadInt32(&s.tradeSubscribersCnt)
}

// ObserveTrades return a channel to the caller through which it will receive notification
// on all trades happening in the system.
func (s *Svc) ObserveTrades(ctx context.Context, retries int, market *string, party *string) (<-chan []types.Trade, uint64) {
	trades := make(chan []types.Trade)
	internal := make(chan []types.Trade)
	ref := s.tradeStore.Subscribe(internal)

	go func() {
		atomic.AddInt32(&s.tradeSubscribersCnt, 1)
		defer atomic.AddInt32(&s.tradeSubscribersCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		ctx2, cfunc := context.WithCancel(ctx)
		defer cfunc()
		for {
			select {
			case <-ctx2.Done():
				s.log.Debug(
					"Trades subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				if err := s.tradeStore.Unsubscribe(ref); err != nil {
					s.log.Error(
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
					if (market == nil || item.MarketID == *market) && (party == nil || item.Seller == *party || item.Buyer == *party) {
						validatedTrades = append(validatedTrades, item)
					}
				}
				if len(validatedTrades) == 0 {
					continue
				}
				retryCount := retries
				success := false
				for !success && retryCount >= 0 {
					select {
					case trades <- validatedTrades:
						s.log.Debug(
							"Trades for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
						success = true
					default:
						retryCount--
						if retryCount >= 0 {
							s.log.Debug(
								"Trades for subscriber not sent",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip),
							)
							time.Sleep(time.Duration(10) * time.Millisecond)
						}
					}
				}
				if !success && retryCount <= 0 {
					s.log.Warn(
						"Trades subscriber has hit the retry limit",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Int("retries", retries),
					)
					cfunc()
					break
				}

			}
		}
	}()

	return trades, ref
}

// GetPositionsSubscribersCount return the number of subscriber to the positions observer
func (s *Svc) GetPositionsSubscribersCount() int32 {
	return atomic.LoadInt32(&s.positionsSubscribersCnt)
}

// ObservePositions return a channel through which all positions are streamed to the caller
// when they get updated
func (s *Svc) ObservePositions(ctx context.Context, retries int, party string) (<-chan *types.MarketPosition, uint64) {
	positions := make(chan *types.MarketPosition)
	internal := make(chan []types.Trade)
	ref := s.tradeStore.Subscribe(internal)

	go func() {
		atomic.AddInt32(&s.positionsSubscribersCnt, 1)
		defer atomic.AddInt32(&s.positionsSubscribersCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		ctx2, cfunc := context.WithCancel(ctx)
		defer cfunc()
		for {
			select {
			case <-ctx2.Done():
				s.log.Debug(
					"Positions subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				if err := s.tradeStore.Unsubscribe(ref); err != nil {
					s.log.Error(
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
				mapOfMarketPositions, err := s.GetPositionsByParty(ctx2, party)
				if err != nil {
					s.log.Error(
						"Failed to get positions for subscriber (getPositionsByParty)",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
					continue
				}
				for _, marketPositions := range mapOfMarketPositions {
					marketPositions := marketPositions
					retryCount := retries
					success := false
					for !success && retryCount > 0 {
						select {
						case positions <- marketPositions:
							s.log.Debug(
								"Positions for subscriber sent successfully",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip),
							)
							success = true
						default:
							retryCount--
							if retryCount > 0 {
								s.log.Debug(
									"Positions for subscriber not sent",
									logging.Uint64("ref", ref),
									logging.String("ip-address", ip),
								)
								time.Sleep(time.Duration(10) * time.Millisecond)
							}
						}
					}
					if retryCount <= 0 {
						s.log.Warn(
							"Positions subscriber has hit the retry limit",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
							logging.Int("retries", retries),
						)
						cfunc()
						break
					}

				}
			}
		}
	}()

	return positions, ref
}

// GetPositionsByParty returns a list of positions for a given party
func (s *Svc) GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error) {

	s.log.Debug("Calculate positions for party",
		logging.String("party-id", party))

	marketBuckets := s.tradeStore.GetTradesBySideBuckets(ctx, party)

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

	s.log.Debug("Loaded market buckets for party",
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
				s.calculateVolumeEntryPriceWeightedAveragesForLong(marketBucket, OpenContracts, ClosedContracts)
		}

		// net
		if OpenVolumeSign == 0 {
			//// calculate avg entry price for closed and open contracts when position is net
			deltaAverageEntryPrice, avgEntryPriceForOpenContracts =
				s.calculateVolumeEntryPriceWeightedAveragesForNet(marketBucket, OpenContracts, ClosedContracts)
		}

		// short
		if OpenVolumeSign == -1 {
			//// calculate avg entry price for closed and open contracts when position is short
			deltaAverageEntryPrice, avgEntryPriceForOpenContracts =
				s.calculateVolumeEntryPriceWeightedAveragesForShort(marketBucket, OpenContracts, ClosedContracts)
		}

		markPrice, _ = s.tradeStore.GetMarkPrice(ctx, market)
		if markPrice == 0 {
			continue
		}

		riskFactor, err = s.getRiskFactorByMarketAndPositionSign(ctx, market, OpenVolumeSign)
		if err != nil {
			return nil, err
		}

		marketPositions := &types.MarketPosition{}
		marketPositions.MarketID = market
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

	s.log.Debug("Positions for party calculated",
		logging.String("party-id", party),
		logging.Int("total-buckets", len(positions)))

	return positions, nil
}

func (s *Svc) getRiskFactorByMarketAndPositionSign(ctx context.Context, market string, openVolumeSign int8) (float64, error) {
	rf, err := s.riskStore.GetByMarket(market)
	if err != nil {
		s.log.Error("Failed to obtain risk factors from risk engine",
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

func (s *Svc) calculateVolumeEntryPriceWeightedAveragesForLong(marketBucket *storage.MarketBucket,
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

func (s *Svc) calculateVolumeEntryPriceWeightedAveragesForNet(marketBucket *storage.MarketBucket,
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

func (s *Svc) calculateVolumeEntryPriceWeightedAveragesForShort(marketBucket *storage.MarketBucket,
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
