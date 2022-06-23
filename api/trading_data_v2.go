// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package api

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/data-node/candlesv2"
	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/service"
	"code.vegaprotocol.io/data-node/vegatime"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

var defaultPaginationV2 = entities.OffsetPagination{
	Skip:       0,
	Limit:      1000,
	Descending: true,
}

type tradingDataServiceV2 struct {
	v2.UnimplementedTradingDataServiceServer
	v2ApiEnabled         bool
	config               Config
	log                  *logging.Logger
	orderService         *service.Order
	networkLimitsService *service.NetworkLimits
	marketDataService    *service.MarketData
	tradeService         *service.Trade
	multiSigService      *service.MultiSig
	notaryService        *service.Notary
	assetService         *service.Asset
	candleService        *candlesv2.Svc
	marketsService       *service.Markets
	partyService         *service.Party
	riskService          *service.Risk
	accountService       *service.Account
	rewardService        *service.Reward
	depositService       *service.Deposit
	withdrawalService    *service.Withdrawal
	oracleSpecService    *service.OracleSpec
	oracleDataService    *service.OracleData
}

func (t *tradingDataServiceV2) checkV2ApiEnabled() error {
	if !t.v2ApiEnabled {
		return fmt.Errorf("this API requires V2 datanode to be enabled")
	}

	return nil
}

func (t *tradingDataServiceV2) GetBalanceHistory(ctx context.Context, req *v2.GetBalanceHistoryRequest) (*v2.GetBalanceHistoryResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if t.accountService == nil {
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

	balances, err := t.accountService.QueryAggregatedBalances(filter, groupBy)
	if err != nil {
		return nil, fmt.Errorf("querying balances: %w", err)
	}

	pbBalances := make([]*v2.AggregatedBalance, len(*balances))
	for i, balance := range *balances {
		pbBalance := balance.ToProto()
		pbBalances[i] = &pbBalance
	}

	return &v2.GetBalanceHistoryResponse{Balances: pbBalances}, nil
}

func (t *tradingDataServiceV2) GetOrdersByMarket(ctx context.Context, req *v2.GetOrdersByMarketRequest) (*v2.GetOrdersByMarketResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if t.orderService == nil {
		return nil, errors.New("sql order store not available")
	}

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = entities.OffsetPaginationFromProto(req.Pagination)
	}

	orders, err := t.orderService.GetByMarket(ctx, req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByParty, err)
	}

	pbOrders := make([]*vega.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = order.ToProto()
	}

	return &v2.GetOrdersByMarketResponse{
		Orders: pbOrders,
	}, nil
}

func entityMarketDataListToProtoList(list []entities.MarketData) *v2.MarketDataConnection {
	if len(list) == 0 {
		return nil
	}

	results := make([]*vega.MarketData, 0, len(list))

	for _, item := range list {
		results = append(results, item.ToProto())
	}

	connection := v2.MarketDataConnection{
		Edges: makeEdges[*v2.MarketDataEdge](list),
	}

	return &connection
}

func (t *tradingDataServiceV2) GetMarketDataHistoryByID(ctx context.Context, req *v2.GetMarketDataHistoryByIDRequest) (*v2.GetMarketDataHistoryByIDResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if t.marketDataService == nil {
		return nil, errors.New("sql market data service not available")
	}

	var startTime, endTime time.Time

	if req.StartTimestamp != nil {
		startTime = time.Unix(0, *req.StartTimestamp)
	}

	if req.EndTimestamp != nil {
		endTime = time.Unix(0, *req.EndTimestamp)
	}

	if req.OffsetPagination != nil {
		// TODO: This has been deprecated in the GraphQL API, but needs to be supported until it is removed.
		return t.handleGetMarketDataHistoryWithOffsetPagination(ctx, req, startTime, endTime)
	}

	return t.handleGetMarketDataHistoryWithCursorPagination(ctx, req, startTime, endTime)
}

func (t *tradingDataServiceV2) handleGetMarketDataHistoryWithOffsetPagination(ctx context.Context, req *v2.GetMarketDataHistoryByIDRequest, startTime, endTime time.Time) (*v2.GetMarketDataHistoryByIDResponse, error) {
	pagination := defaultPaginationV2
	if req.OffsetPagination != nil {
		pagination = entities.OffsetPaginationFromProto(req.OffsetPagination)
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

func (t *tradingDataServiceV2) handleGetMarketDataHistoryWithCursorPagination(ctx context.Context, req *v2.GetMarketDataHistoryByIDRequest, startTime, endTime time.Time) (*v2.GetMarketDataHistoryByIDResponse, error) {
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, fmt.Errorf("could not parse cursor pagination information: %w", err)
	}
	history, pageInfo, err := t.marketDataService.GetBetweenDatesByID(ctx, req.MarketId, startTime, endTime, pagination)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve historic market data: %w", err)
	}

	connection := v2.MarketDataConnection{
		TotalCount: 0,
		Edges:      makeEdges[*v2.MarketDataEdge](history),
		PageInfo:   pageInfo.ToProto(),
	}

	return &v2.GetMarketDataHistoryByIDResponse{
		MarketData: &connection,
	}, nil
}

func parseMarketDataResults(results []entities.MarketData) (*v2.GetMarketDataHistoryByIDResponse, error) {
	response := v2.GetMarketDataHistoryByIDResponse{
		MarketData: entityMarketDataListToProtoList(results),
	}

	return &response, nil
}

func (t *tradingDataServiceV2) getMarketDataHistoryByID(ctx context.Context, id string, start, end time.Time, pagination entities.OffsetPagination) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, _, err := t.marketDataService.GetBetweenDatesByID(ctx, id, start, end, pagination)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve market data history for market id: %w", err)
	}

	return parseMarketDataResults(results)
}

func (t *tradingDataServiceV2) getMarketDataByID(ctx context.Context, id string) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, err := t.marketDataService.GetMarketDataByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve market data history for market id: %w", err)
	}

	return parseMarketDataResults([]entities.MarketData{results})
}

func (t *tradingDataServiceV2) getMarketDataHistoryFromDateByID(ctx context.Context, id string, start time.Time, pagination entities.OffsetPagination) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, _, err := t.marketDataService.GetFromDateByID(ctx, id, start, pagination)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve market data history for market id: %w", err)
	}

	return parseMarketDataResults(results)
}

func (t *tradingDataServiceV2) getMarketDataHistoryToDateByID(ctx context.Context, id string, end time.Time, pagination entities.OffsetPagination) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, _, err := t.marketDataService.GetToDateByID(ctx, id, end, pagination)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve market data history for market id: %w", err)
	}

	return parseMarketDataResults(results)
}

// MarketsDataSubscribe opens a subscription to market data provided by the markets service.
func (t *tradingDataServiceV2) MarketsDataSubscribe(req *v2.MarketsDataSubscribeRequest,
	srv v2.TradingDataService_MarketsDataSubscribeServer,
) error {
	if err := t.checkV2ApiEnabled(); err != nil {
		return apiError(codes.Unavailable, err)
	}

	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	ch, ref := t.marketDataService.ObserveMarketData(ctx, t.config.StreamRetries, req.MarketId)

	return observeBatch(ctx, t.log, "MarketsData", ch, ref, func(orders []*entities.MarketData) error {
		out := make([]*vega.MarketData, 0, len(orders))
		for _, v := range orders {
			out = append(out, v.ToProto())
		}
		return srv.Send(&v2.MarketsDataSubscribeResponse{MarketData: out})
	})
}

func (t *tradingDataServiceV2) GetNetworkLimits(ctx context.Context, req *v2.GetNetworkLimitsRequest) (*v2.GetNetworkLimitsResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if t.networkLimitsService == nil {
		return nil, errors.New("sql network limits store is not available")
	}

	limits, err := t.networkLimitsService.GetLatest(ctx)
	if err != nil {
		return nil, apiError(codes.Unknown, ErrGetNetworkLimits, err)
	}

	return &v2.GetNetworkLimitsResponse{Limits: limits.ToProto()}, nil
}

// GetCandleData for a given market, time range and interval.  Interval must be a valid postgres interval value
func (t *tradingDataServiceV2) GetCandleData(ctx context.Context, req *v2.GetCandleDataRequest) (*v2.GetCandleDataResponse, error) {
	var err error
	if err = t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if t.candleService == nil {
		return nil, errors.New("sql candle service not available")
	}

	from := vegatime.UnixNano(req.FromTimestamp)
	to := vegatime.UnixNano(req.ToTimestamp)

	pagination := entities.CursorPagination{}
	if req.Pagination != nil {
		pagination, err = entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, fmt.Errorf("could not parse cursor pagination information: %w", err)
		}
	}

	candles, pageInfo, err := t.candleService.GetCandleDataForTimeSpan(ctx, req.CandleId, &from, &to, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCandleServiceGetCandleData, err)
	}

	connection := v2.CandleDataConnection{
		TotalCount: 0,
		Edges:      makeEdges[*v2.CandleEdge](candles),
		PageInfo:   pageInfo.ToProto(),
	}

	return &v2.GetCandleDataResponse{Candles: &connection}, nil
}

// SubscribeToCandleData subscribes to candle updates for a given market and interval.  Interval must be a valid postgres interval value
func (t *tradingDataServiceV2) SubscribeToCandleData(req *v2.SubscribeToCandleDataRequest, srv v2.TradingDataService_SubscribeToCandleDataServer) error {
	if err := t.checkV2ApiEnabled(); err != nil {
		return err
	}

	defer metrics.StartActiveSubscriptionCountGRPC("Candle")()

	if t.candleService == nil {
		return errors.New("sql candle service not available")
	}

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	subscriptionId, candlesChan, err := t.candleService.Subscribe(ctx, req.CandleId)
	defer t.candleService.Unsubscribe(subscriptionId)

	if err != nil {
		return apiError(codes.Internal, ErrCandleServiceSubscribeToCandles, err)
	}

	publishedEventStatTicker := time.NewTicker(time.Second)
	var publishedEvents int64

	for {
		select {
		case <-publishedEventStatTicker.C:
			metrics.PublishedEventsAdd("Candle", float64(publishedEvents))
			publishedEvents = 0
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
			publishedEvents++
		case <-ctx.Done():
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
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if t.candleService == nil {
		return nil, errors.New("sql candle service not available")
	}

	mappings, err := t.candleService.GetCandlesForMarket(ctx, req.MarketId)
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

// GetERC20MutlsigSignerAddedBundles return the signature bundles needed to add a new validator to the multisig control ERC20 contract
func (t *tradingDataServiceV2) GetERC20MultiSigSignerAddedBundles(ctx context.Context, req *v2.GetERC20MultiSigSignerAddedBundlesRequest) (*v2.GetERC20MultiSigSignerAddedBundlesResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if t.notaryService == nil {
		return nil, errors.New("sql notary service not available")
	}

	if t.multiSigService == nil {
		return nil, errors.New("sql multisig event store not available")
	}

	nodeID := req.GetNodeId()
	if len(nodeID) == 0 {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("node id must be supplied"))
	}

	var epochID *int64
	if len(req.EpochSeq) != 0 {
		e, err := strconv.ParseInt(req.EpochSeq, 10, 64)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("epochID is not a valid integer"))
		}
		epochID = &e
	}

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = entities.OffsetPaginationFromProto(req.Pagination)
	}

	res, err := t.multiSigService.GetAddedEvents(ctx, nodeID, epochID, p)
	if err != nil {
		c := codes.Internal
		if errors.Is(err, entities.ErrInvalidID) {
			c = codes.InvalidArgument
		}
		return nil, apiError(c, err)

	}

	// find bundle for this nodeID, might be multiple if its added, then removed then added again??
	bundles := []*v2.ERC20MultiSigSignerAddedBundle{}
	for _, b := range res {

		signatures, err := t.notaryService.GetByResourceID(ctx, b.ID.String())
		if err != nil {
			return nil, apiError(codes.Internal, err)
		}

		pack := "0x"
		for _, v := range signatures {
			pack = fmt.Sprintf("%v%v", pack, hex.EncodeToString(v.Sig))
		}

		bundles = append(bundles,
			&v2.ERC20MultiSigSignerAddedBundle{
				NewSigner:  b.SignerChange.String(),
				Submitter:  b.Submitter.String(),
				Nonce:      b.Nonce,
				Timestamp:  b.VegaTime.UnixNano(),
				Signatures: pack,
				EpochSeq:   strconv.FormatInt(b.EpochID, 10),
			},
		)
	}

	return &v2.GetERC20MultiSigSignerAddedBundlesResponse{
		Bundles: bundles,
	}, nil
}

// GetERC20MutlsigSignerAddedBundles return the signature bundles needed to add a new validator to the multisig control ERC20 contract
func (t *tradingDataServiceV2) GetERC20MultiSigSignerRemovedBundles(ctx context.Context, req *v2.GetERC20MultiSigSignerRemovedBundlesRequest) (*v2.GetERC20MultiSigSignerRemovedBundlesResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if t.notaryService == nil {
		return nil, errors.New("sql notary store not available")
	}

	if t.multiSigService == nil {
		return nil, errors.New("sql multisig event store not available")
	}

	nodeID := req.GetNodeId()
	submitter := req.GetSubmitter()

	if len(nodeID) == 0 || len(submitter) == 0 {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("nodeId and submitter must be supplied"))
	}

	var epochID *int64
	if len(req.EpochSeq) != 0 {
		e, err := strconv.ParseInt(req.EpochSeq, 10, 64)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("epochID is not a valid integer"))
		}
		epochID = &e
	}

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = entities.OffsetPaginationFromProto(req.Pagination)
	}

	res, err := t.multiSigService.GetRemovedEvents(ctx, nodeID, strings.TrimPrefix(submitter, "0x"), epochID, p)
	if err != nil {
		c := codes.Internal
		if errors.Is(err, entities.ErrInvalidID) {
			c = codes.InvalidArgument
		}
		return nil, apiError(c, err)
	}

	// find bundle for this nodeID, might be multiple if its added, then removed then added again??
	bundles := []*v2.ERC20MultiSigSignerRemovedBundle{}
	for _, b := range res {

		signatures, err := t.notaryService.GetByResourceID(ctx, b.ID.String())
		if err != nil {
			return nil, apiError(codes.Internal, err)
		}

		pack := "0x"
		for _, v := range signatures {
			pack = fmt.Sprintf("%v%v", pack, hex.EncodeToString(v.Sig))
		}

		bundles = append(bundles, &v2.ERC20MultiSigSignerRemovedBundle{
			OldSigner:  b.SignerChange.String(),
			Submitter:  b.Submitter.String(),
			Nonce:      b.Nonce,
			Timestamp:  b.VegaTime.UnixNano(),
			Signatures: pack,
			EpochSeq:   strconv.FormatInt(b.EpochID, 10),
		})
	}

	return &v2.GetERC20MultiSigSignerRemovedBundlesResponse{
		Bundles: bundles,
	}, nil
}

func (t *tradingDataServiceV2) GetERC20ListAssetBundle(ctx context.Context, req *v2.GetERC20ListAssetBundleRequest) (*v2.GetERC20ListAssetBundleResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if len(req.AssetId) <= 0 {
		return nil, ErrMissingAssetID
	}

	if t.assetService == nil {
		return nil, errors.New("sql asset store not available")
	}

	// first here we gonna get the proposal by its ID,
	asset, err := t.assetService.GetByID(ctx, req.AssetId)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	if t.notaryService == nil {
		return nil, errors.New("sql notary store not available")
	}

	// then we get the signature and pack them altogether
	signatures, err := t.notaryService.GetByResourceID(ctx, req.AssetId)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	// now we pack them
	pack := "0x"
	for _, v := range signatures {
		pack = fmt.Sprintf("%v%v", pack, hex.EncodeToString(v.Sig))
	}

	var address string
	if asset.ERC20Contract != "" {
		address = asset.ERC20Contract
	} else {
		return nil, fmt.Errorf("invalid asset source")
	}

	if len(address) <= 0 {
		return nil, fmt.Errorf("invalid erc20 token contract address")
	}

	return &v2.GetERC20ListAssetBundleResponse{
		AssetSource: address,
		Nonce:       req.AssetId,
		VegaAssetId: asset.ID.String(),
		Signatures:  pack,
	}, nil
}

// Get trades by market using a cursor based pagination model
func (t *tradingDataServiceV2) GetTradesByMarket(ctx context.Context, in *v2.GetTradesByMarketRequest) (*v2.GetTradesByMarketResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	market := in.GetMarketId()
	if len(market) == 0 {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("marketId must be supplied"))
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	trades, pageInfo, err := t.tradeService.GetByMarketWithCursor(ctx, market, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	tradesConnection := &v2.TradeConnection{
		TotalCount: 0,
		Edges:      makeEdges[*v2.TradeEdge](trades),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetTradesByMarketResponse{
		Trades: tradesConnection,
	}

	return resp, nil
}

// Get trades by party using a cursor based pagination model
func (t *tradingDataServiceV2) GetTradesByParty(ctx context.Context, in *v2.GetTradesByPartyRequest) (*v2.GetTradesByPartyResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	party := in.GetPartyId()
	if len(party) == 0 {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("partyId must be supplied"))
	}
	var market *string
	if len(in.GetMarketId()) > 0 {
		market = &in.MarketId
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	trades, pageInfo, err := t.tradeService.GetByPartyWithCursor(ctx, party, market, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	tradesConnection := &v2.TradeConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.TradeEdge](trades),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetTradesByPartyResponse{
		Trades: tradesConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) GetTradesByOrderID(ctx context.Context, in *v2.GetTradesByOrderIDRequest) (*v2.GetTradesByOrderIDResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	orderID := in.GetOrderId()
	if len(orderID) == 0 {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("orderId must be supplied"))
	}
	var market *string
	if len(in.GetMarketId()) > 0 {
		market = &in.MarketId
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	trades, pageInfo, err := t.tradeService.GetByOrderIDWithCursor(ctx, orderID, market, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	tradesConnection := &v2.TradeConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.TradeEdge](trades),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetTradesByOrderIDResponse{
		Trades: tradesConnection,
	}

	return resp, nil
}

// Get all markets using a cursor based pagination model
func (t *tradingDataServiceV2) GetMarkets(ctx context.Context, in *v2.GetMarketsRequest) (*v2.GetMarketsResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}
	markets, pageInfo, err := t.marketsService.GetAllPaged(ctx, in.MarketId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	marketsConnection := &v2.MarketConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.MarketEdge](markets),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetMarketsResponse{
		Markets: marketsConnection,
	}

	return resp, nil
}

// Get Parties using a cursor based pagination model
func (t *tradingDataServiceV2) GetParties(ctx context.Context, in *v2.GetPartiesRequest) (*v2.GetPartiesResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}
	parties, pageInfo, err := t.partyService.GetAllPaged(ctx, in.PartyId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	partyConnection := &v2.PartyConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.PartyEdge](parties),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetPartiesResponse{
		Party: partyConnection,
	}
	return resp, nil
}

func (t *tradingDataServiceV2) GetOrdersByMarketConnection(ctx context.Context, in *v2.GetOrdersByMarketConnectionRequest) (*v2.GetOrdersByMarketConnectionResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}
	orders, pageInfo, err := t.orderService.GetByMarketPaged(ctx, in.MarketId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	ordersConnection := &v2.OrderConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.OrderEdge](orders),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetOrdersByMarketConnectionResponse{
		Orders: ordersConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) GetOrderVersionsByIDConnection(ctx context.Context, in *v2.GetOrderVersionsByIDConnectionRequest) (*v2.GetOrderVersionsByIDConnectionResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	orders, pageInfo, err := t.orderService.GetOrderVersionsByIDPaged(ctx, in.OrderId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	ordersConnection := &v2.OrderConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.OrderEdge](orders),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetOrderVersionsByIDConnectionResponse{
		Orders: ordersConnection,
	}
	return resp, nil
}

func (t *tradingDataServiceV2) GetOrdersByPartyConnection(ctx context.Context, in *v2.GetOrdersByPartyConnectionRequest) (*v2.GetOrdersByPartyConnectionResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	orders, pageInfo, err := t.orderService.GetByPartyPaged(ctx, in.PartyId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	ordersConnection := &v2.OrderConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.OrderEdge](orders),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetOrdersByPartyConnectionResponse{
		Orders: ordersConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) GetMarginLevels(ctx context.Context, in *v2.GetMarginLevelsRequest) (*v2.GetMarginLevelsResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	marginLevels, pageInfo, err := t.riskService.GetMarginLevelsByIDWithCursorPagination(ctx, in.PartyId, in.MarketId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	marginLevelsConnection := &v2.MarginConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.MarginEdge](marginLevels, t.accountService),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetMarginLevelsResponse{
		MarginLevels: marginLevelsConnection,
	}

	return resp, nil
}

// Get rewards
func (t *tradingDataServiceV2) GetRewards(ctx context.Context, in *v2.GetRewardsRequest) (*v2.GetRewardsResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	rewards, pageInfo, err := t.rewardService.GetByCursor(ctx, &in.PartyId, &in.AssetId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	rewardsConnection := &v2.RewardsConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.RewardEdge](rewards),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := v2.GetRewardsResponse{Rewards: rewardsConnection}
	return &resp, nil
}

// Get reward summaries
func (t *tradingDataServiceV2) GetRewardSummaries(ctx context.Context, in *v2.GetRewardSummariesRequest) (*v2.GetRewardSummariesResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	summaries, err := t.rewardService.GetSummaries(ctx, &in.PartyId, &in.AssetId)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	summaryProtos := make([]*vega.RewardSummary, len(summaries))

	for i, summary := range summaries {
		summaryProtos[i] = summary.ToProto()
	}

	resp := v2.GetRewardSummariesResponse{Summaries: summaryProtos}
	return &resp, nil
}

// -- Deposits --
func (t *tradingDataServiceV2) GetDeposits(ctx context.Context, req *v2.GetDepositsRequest) (*v2.GetDepositsResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	deposits, pageInfo, err := t.depositService.GetByParty(ctx, req.PartyId, false, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	depositConnection := &v2.DepositsConnection{
		TotalCount: 0, // TODO: implement total count
		//Edges:      makeDepositEdges(deposits),
		Edges:    makeEdges[*v2.DepositEdge](deposits),
		PageInfo: pageInfo.ToProto(),
	}

	resp := v2.GetDepositsResponse{Deposits: depositConnection}

	return &resp, nil
}

func makeEdges[T proto.Message, V entities.PagedEntity[T]](inputs []V, args ...any) []T {
	edges := make([]T, 0, len(inputs))
	for _, input := range inputs {
		edges = append(edges, input.ToProtoEdge(args))
	}
	return edges
}

// -- Withdrawals --
func (t *tradingDataServiceV2) GetWithdrawals(ctx context.Context, req *v2.GetWithdrawalsRequest) (*v2.GetWithdrawalsResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	withdrawals, pageInfo, err := t.withdrawalService.GetByParty(ctx, req.PartyId, false, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	depositConnection := &v2.WithdrawalsConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.WithdrawalEdge](withdrawals),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := v2.GetWithdrawalsResponse{Withdrawals: depositConnection}

	return &resp, nil
}

// -- Assets --
func (t *tradingDataServiceV2) GetAssets(ctx context.Context, req *v2.GetAssetsRequest) (*v2.GetAssetsResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	if req.AssetId != "" {
		return t.getSingleAsset(ctx, req.AssetId)
	}

	return t.getAllAssets(ctx, req.Pagination)
}

func (t *tradingDataServiceV2) getSingleAsset(ctx context.Context, assetID string) (*v2.GetAssetsResponse, error) {
	asset, err := t.assetService.GetByID(ctx, assetID)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := &v2.AssetsConnection{
		TotalCount: 1,
		Edges:      makeEdges[*v2.AssetEdge]([]entities.Asset{asset}),
		PageInfo: &v2.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     asset.Cursor().Encode(),
			EndCursor:       asset.Cursor().Encode(),
		},
	}

	return &v2.GetAssetsResponse{Assets: connection}, nil
}

func (t *tradingDataServiceV2) getAllAssets(ctx context.Context, p *v2.Pagination) (*v2.GetAssetsResponse, error) {
	pagination, err := entities.CursorPaginationFromProto(p)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	assets, pageInfo, err := t.assetService.GetAllWithCursorPagination(ctx, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := &v2.AssetsConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.AssetEdge](assets),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := v2.GetAssetsResponse{Assets: connection}
	return &resp, nil
}

func (t *tradingDataServiceV2) GetOracleSpecsConnection(ctx context.Context, req *v2.GetOracleSpecsConnectionRequest) (*v2.GetOracleSpecsConnectionResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	specs, pageInfo, err := t.oracleSpecService.GetSpecsWithCursorPagination(ctx, req.SpecId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := &v2.OracleSpecsConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.OracleSpecEdge](specs),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := v2.GetOracleSpecsConnectionResponse{
		OracleSpecs: connection,
	}

	return &resp, nil
}

func (t *tradingDataServiceV2) GetOracleDataConnection(ctx context.Context, req *v2.GetOracleDataConnectionRequest) (*v2.GetOracleDataConnectionResponse, error) {
	if err := t.checkV2ApiEnabled(); err != nil {
		return nil, err
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	var data []entities.OracleData
	var pageInfo entities.PageInfo

	if req.SpecId != "" {
		data, pageInfo, err = t.oracleDataService.GetOracleDataBySpecID(ctx, req.SpecId, pagination)
	} else {
		data, pageInfo, err = t.oracleDataService.ListOracleData(ctx, pagination)
	}

	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := &v2.OracleDataConnection{
		TotalCount: 0, // TODO: implement total count
		Edges:      makeEdges[*v2.OracleDataEdge](data),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := v2.GetOracleDataConnectionResponse{
		OracleData: connection,
	}

	return &resp, nil
}
