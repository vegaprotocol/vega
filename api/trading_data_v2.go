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
	"code.vegaprotocol.io/data-node/service"
	"code.vegaprotocol.io/data-node/vegatime"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"google.golang.org/grpc/codes"
)

var defaultPaginationV2 = entities.OffsetPagination{
	Skip:       0,
	Limit:      1000,
	Descending: true,
}

type tradingDataServiceV2 struct {
	v2.UnimplementedTradingDataServiceServer
	v2ApiEnabled         bool
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
		Edges: makeMarketDataHistoryEdges(list),
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
		Edges:      makeMarketDataHistoryEdges(history),
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
		Edges:      makeCandleDataEdges(candles),
		PageInfo:   pageInfo.ToProto(),
	}

	return &v2.GetCandleDataResponse{Candles: &connection}, nil
}

func makeCandleDataEdges(candles []entities.Candle) []*v2.CandleEdge {
	edges := make([]*v2.CandleEdge, len(candles))
	for i, candle := range candles {
		edges[i] = &v2.CandleEdge{
			Node:   candle.ToV2CandleProto(),
			Cursor: candle.Cursor().Encode(),
		}
	}
	return edges
}

// SubscribeToCandleData subscribes to candle updates for a given market and interval.  Interval must be a valid postgres interval value
func (t *tradingDataServiceV2) SubscribeToCandleData(req *v2.SubscribeToCandleDataRequest, srv v2.TradingDataService_SubscribeToCandleDataServer) error {
	if err := t.checkV2ApiEnabled(); err != nil {
		return err
	}

	if t.candleService == nil {
		return errors.New("sql candle service not available")
	}

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	subscriptionId, candlesChan, err := t.candleService.Subscribe(ctx, req.CandleId)
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
			err := t.candleService.Unsubscribe(subscriptionId)
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
		Edges:      makeTradeEdges(trades),
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
		Edges:      makeTradeEdges(trades),
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
		Edges:      makeTradeEdges(trades),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetTradesByOrderIDResponse{
		Trades: tradesConnection,
	}

	return resp, nil
}

func makeTradeEdges(trades []entities.Trade) []*v2.TradeEdge {
	edges := make([]*v2.TradeEdge, len(trades))
	for i, t := range trades {
		edges[i] = &v2.TradeEdge{
			Node:   t.ToProto(),
			Cursor: t.Cursor().Encode(),
		}
	}
	return edges
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
		Edges:      makeMarketEdges(markets),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetMarketsResponse{
		Markets: marketsConnection,
	}

	return resp, nil
}

func makeMarketEdges(markets []entities.Market) []*v2.MarketEdge {
	edges := make([]*v2.MarketEdge, len(markets))
	for i, m := range markets {
		marketProto, err := m.ToProto()
		if err != nil {
			continue
		}
		edges[i] = &v2.MarketEdge{
			Node:   marketProto,
			Cursor: m.Cursor().Encode(),
		}
	}
	return edges
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
		Edges:      makePartyEdges(parties),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetPartiesResponse{
		Party: partyConnection,
	}
	return resp, nil
}

func makePartyEdges(parties []entities.Party) []*v2.PartyEdge {
	edges := make([]*v2.PartyEdge, len(parties))
	for i, p := range parties {
		edges[i] = &v2.PartyEdge{
			Node:   p.ToProto(),
			Cursor: p.Cursor().Encode(),
		}
	}
	return edges
}

func (t *tradingDataServiceV2) GetOrdersByMarketPaged(ctx context.Context, in *v2.GetOrdersByMarketPagedRequest) (*v2.GetOrdersByMarketPagedResponse, error) {
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
		Edges:      makeOrderEdges(orders),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetOrdersByMarketPagedResponse{
		Orders: ordersConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) GetOrderVersionsByIDPaged(ctx context.Context, in *v2.GetOrderVersionsByIDPagedRequest) (*v2.GetOrderVersionsByIDPagedResponse, error) {
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
		Edges:      makeOrderEdges(orders),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetOrderVersionsByIDPagedResponse{
		Orders: ordersConnection,
	}
	return resp, nil
}

func (t *tradingDataServiceV2) GetOrdersByPartyPaged(ctx context.Context, in *v2.GetOrdersByPartyPagedRequest) (*v2.GetOrdersByPartyPagedResponse, error) {
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
		Edges:      makeOrderEdges(orders),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetOrdersByPartyPagedResponse{
		Orders: ordersConnection,
	}

	return resp, nil
}

func makeOrderEdges(orders []entities.Order) []*v2.OrderEdge {
	edges := make([]*v2.OrderEdge, len(orders))
	for i, o := range orders {
		edges[i] = &v2.OrderEdge{
			Node:   o.ToProto(),
			Cursor: o.Cursor().Encode(),
		}
	}
	return edges
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
		Edges:      makeMarginLevelEdges(t.accountService, marginLevels),
		PageInfo:   pageInfo.ToProto(),
	}

	resp := &v2.GetMarginLevelsResponse{
		MarginLevels: marginLevelsConnection,
	}

	return resp, nil
}

func makeMarginLevelEdges(accountService *service.Account, marginLevels []entities.MarginLevels) []*v2.MarginEdge {
	edges := make([]*v2.MarginEdge, len(marginLevels))
	for i, ml := range marginLevels {
		mlProto, err := ml.ToProto(accountService)
		if err != nil {
			continue
		}
		edges[i] = &v2.MarginEdge{
			Node:   mlProto,
			Cursor: ml.Cursor().Encode(),
		}
	}
	return edges
}

func makeMarketDataHistoryEdges(history []entities.MarketData) []*v2.MarketDataEdge {
	edges := make([]*v2.MarketDataEdge, len(history))
	for i, md := range history {
		edges[i] = &v2.MarketDataEdge{
			Node:   md.ToProto(),
			Cursor: md.Cursor().Encode(),
		}
	}
	return edges
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
		Edges:      makeRewardEdges(rewards),
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

func makeRewardEdges(rewards []entities.Reward) []*v2.RewardEdge {
	edges := make([]*v2.RewardEdge, len(rewards))
	for i, r := range rewards {
		edges[i] = &v2.RewardEdge{
			Node:   r.ToProto(),
			Cursor: r.Cursor().Encode(),
		}
	}
	return edges
}
