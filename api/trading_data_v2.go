package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/data-node/candlesv2"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/vegatime"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"google.golang.org/grpc/codes"
)

var defaultPaginationV2 = entities.Pagination{
	Skip:       0,
	Limit:      1000,
	Descending: true,
}

type tradingDataServiceV2 struct {
	v2.UnimplementedTradingDataServiceServer
	log                *logging.Logger
	balanceStore       *sqlstore.Balances
	orderStore         *sqlstore.Orders
	networkLimitsStore *sqlstore.NetworkLimits
	marketDataStore    *sqlstore.MarketData
	tradeStore         *sqlstore.Trades
	candleServiceV2    *candlesv2.Svc
}

func (t *tradingDataServiceV2) QueryBalanceHistory(ctx context.Context, req *v2.QueryBalanceHistoryRequest) (*v2.QueryBalanceHistoryResponse, error) {
	if t.balanceStore == nil {
		return nil, fmt.Errorf("sql balance store not available")
	}

	filter, err := entities.AccountFilterFromProto(req.Filter)
	if err != nil {
		return nil, fmt.Errorf("parsing filter: %w", err)
	}

	groupBy := []entities.AccountField{}
	for _, field := range req.GroupBy {
		field, err := entities.AccountFieldFromProto(field)
		if err != nil {
			return nil, fmt.Errorf("parsing group by list: %w", err)
		}
		groupBy = append(groupBy, field)
	}

	balances, err := t.balanceStore.Query(filter, groupBy)
	if err != nil {
		return nil, fmt.Errorf("querying balances: %w", err)
	}

	pbBalances := make([]*v2.AggregatedBalance, len(*balances))
	for i, balance := range *balances {
		pbBalance := balance.ToProto()
		pbBalances[i] = &pbBalance
	}

	return &v2.QueryBalanceHistoryResponse{Balances: pbBalances}, nil
}

func (t *tradingDataServiceV2) OrdersByMarket(ctx context.Context, req *v2.OrdersByMarketRequest) (*v2.OrdersByMarketResponse, error) {
	if t.orderStore == nil {
		return nil, errors.New("sql order store not available")
	}

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = entities.PaginationFromProto(req.Pagination)
	}

	orders, err := t.orderStore.GetByMarket(ctx, req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByParty, err)
	}

	pbOrders := make([]*vega.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = order.ToProto()
	}

	return &v2.OrdersByMarketResponse{
		Orders: pbOrders,
	}, nil
}

func entityMarketDataListToProtoList(list []entities.MarketData) []*vega.MarketData {
	if len(list) == 0 {
		return nil
	}

	results := make([]*vega.MarketData, 0, len(list))

	for _, item := range list {
		results = append(results, item.ToProto())
	}

	return results
}

func (t *tradingDataServiceV2) GetMarketDataHistoryByID(ctx context.Context, req *v2.GetMarketDataHistoryByIDRequest) (*v2.GetMarketDataHistoryByIDResponse, error) {
	if t.marketDataStore == nil {
		return nil, errors.New("sql market data store not available")
	}

	var startTime, endTime time.Time

	if req.StartTimestamp != nil {
		startTime = time.Unix(0, *req.StartTimestamp)
	}

	if req.EndTimestamp != nil {
		endTime = time.Unix(0, *req.EndTimestamp)
	}

	pagination := defaultPaginationV2
	if req.Pagination != nil {
		pagination = entities.PaginationFromProto(req.Pagination)
	}

	if req.StartTimestamp != nil && req.EndTimestamp != nil {
		return t.getMarketDataHistoryByID(ctx, req.MarketId, startTime, endTime, pagination)
	}

	if req.StartTimestamp != nil {
		return t.getMarketDataHistoryFromDateByID(ctx, req.MarketId, startTime, pagination)
	}

	if req.EndTimestamp != nil {
		return t.getMarketDataHistoryToDateByID(ctx, req.MarketId, endTime, pagination)
	}

	return t.getMarketDataByID(ctx, req.MarketId)
}

func parseMarketDataResults(results []entities.MarketData) (*v2.GetMarketDataHistoryByIDResponse, error) {
	response := v2.GetMarketDataHistoryByIDResponse{
		MarketData: entityMarketDataListToProtoList(results),
	}

	return &response, nil
}

func (t *tradingDataServiceV2) getMarketDataHistoryByID(ctx context.Context, id string, start, end time.Time, pagination entities.Pagination) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, err := t.marketDataStore.GetBetweenDatesByID(ctx, id, start, end, pagination)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve market data history for market id: %w", err)
	}

	return parseMarketDataResults(results)
}

func (t *tradingDataServiceV2) getMarketDataByID(ctx context.Context, id string) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, err := t.marketDataStore.GetMarketDataByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve market data history for market id: %w", err)
	}

	return parseMarketDataResults([]entities.MarketData{results})
}

func (t *tradingDataServiceV2) getMarketDataHistoryFromDateByID(ctx context.Context, id string, start time.Time, pagination entities.Pagination) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, err := t.marketDataStore.GetFromDateByID(ctx, id, start, pagination)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve market data history for market id: %w", err)
	}

	return parseMarketDataResults(results)
}

func (t *tradingDataServiceV2) getMarketDataHistoryToDateByID(ctx context.Context, id string, end time.Time, pagination entities.Pagination) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, err := t.marketDataStore.GetToDateByID(ctx, id, end, pagination)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve market data history for market id: %w", err)
	}

	return parseMarketDataResults(results)
}

func (t *tradingDataServiceV2) GetNetworkLimits(ctx context.Context, req *v2.GetNetworkLimitsRequest) (*v2.GetNetworkLimitsResponse, error) {
	if t.networkLimitsStore == nil {
		return nil, errors.New("sql network limits store is not available")
	}

	limits, err := t.networkLimitsStore.GetLatest(ctx)
	if err != nil {
		return nil, apiError(codes.Unknown, ErrGetNetworkLimits, err)
	}

	return &v2.GetNetworkLimitsResponse{Limits: limits.ToProto()}, nil
}

// GetCandleData for a given market, time range and interval.  Interval must be a valid postgres interval value
func (t *tradingDataServiceV2) GetCandleData(ctx context.Context, req *v2.GetCandleDataRequest) (*v2.GetCandleDataResponse, error) {
	from := vegatime.UnixNano(req.FromTimestamp)
	to := vegatime.UnixNano(req.ToTimestamp)

	pagination := defaultPaginationV2
	if req.Pagination != nil {
		pagination = entities.PaginationFromProto(req.Pagination)
	}

	candles, err := t.candleServiceV2.GetCandleDataForTimeSpan(ctx, req.CandleId, &from, &to, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCandleServiceGetCandleData, err)
	}

	protoCandles := make([]*v2.Candle, len(candles))
	for _, candle := range candles {
		protoCandles = append(protoCandles, candle.ToV2CandleProto())
	}

	return &v2.GetCandleDataResponse{Candles: protoCandles}, nil
}

// SubscribeToCandleData subscribes to candle updates for a given market and interval.  Interval must be a valid postgres interval value
func (t *tradingDataServiceV2) SubscribeToCandleData(req *v2.SubscribeToCandleDataRequest, srv v2.TradingDataService_SubscribeToCandleDataServer) error {
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	subscriptionId, candlesChan, err := t.candleServiceV2.Subscribe(ctx, req.CandleId)
	if err != nil {
		return apiError(codes.Internal, ErrCandleServiceSubscribeToCandles, err)
	}

	for {
		select {
		case candle, ok := <-candlesChan:
			if !ok {
				return apiError(codes.Internal, ErrCandleServiceSubscribeToCandles, fmt.Errorf("channel closed"))
			}

			resp := &v2.SubscribeToCandleDataResponse{
				Candle: candle.ToV2CandleProto(),
			}
			if err = srv.Send(resp); err != nil {
				return apiError(codes.Internal, ErrCandleServiceSubscribeToCandles,
					fmt.Errorf("sending candles:%w", err))
			}
		case <-ctx.Done():
			err := t.candleServiceV2.Unsubscribe(subscriptionId)
			if err != nil {
				t.log.Errorf("failed to unsubscribe from candle updates:%s", err)
			}

			err = ctx.Err()
			if err != nil {
				return apiError(codes.Internal, ErrCandleServiceSubscribeToCandles, err)
			}
			return nil
		}
	}
}

// GetCandlesForMarket gets all available intervals for a given market along with the corresponding candle id
func (t *tradingDataServiceV2) GetCandlesForMarket(ctx context.Context, req *v2.GetCandlesForMarketRequest) (*v2.GetCandlesForMarketResponse, error) {
	mappings, err := t.candleServiceV2.GetCandlesForMarket(ctx, req.MarketId)

	if err != nil {
		return nil, apiError(codes.Internal, ErrCandleServiceGetCandlesForMarket, err)
	}

	var intervalToCandleIds []*v2.IntervalToCandleId
	for interval, candleId := range mappings {
		intervalToCandleIds = append(intervalToCandleIds, &v2.IntervalToCandleId{
			Interval: interval,
			CandleId: candleId,
		})
	}

	return &v2.GetCandlesForMarketResponse{
		IntervalToCandleId: intervalToCandleIds,
	}, nil
}
