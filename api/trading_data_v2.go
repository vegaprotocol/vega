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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/data-node/candlesv2"
	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/service"
	"code.vegaprotocol.io/data-node/vegatime"
	"code.vegaprotocol.io/data-node/version"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
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
	config                    Config
	log                       *logging.Logger
	orderService              *service.Order
	networkLimitsService      *service.NetworkLimits
	marketDataService         *service.MarketData
	tradeService              *service.Trade
	multiSigService           *service.MultiSig
	notaryService             *service.Notary
	assetService              *service.Asset
	candleService             *candlesv2.Svc
	marketsService            *service.Markets
	partyService              *service.Party
	riskService               *service.Risk
	positionService           *service.Position
	accountService            *service.Account
	rewardService             *service.Reward
	depositService            *service.Deposit
	withdrawalService         *service.Withdrawal
	oracleSpecService         *service.OracleSpec
	oracleDataService         *service.OracleData
	liquidityProvisionService *service.LiquidityProvision
	governanceService         *service.Governance
	transfersService          *service.Transfer
	delegationService         *service.Delegation
}

func (t *tradingDataServiceV2) ListAccounts(ctx context.Context, req *v2.ListAccountsRequest) (*v2.ListAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAccountsV2")()
	if t.accountService == nil {
		return nil, apiError(codes.Internal, fmt.Errorf("Account service not available"))
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	filter, err := entities.AccountFilterFromProto(req.Filter)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	accountBalances, pageInfo, err := t.accountService.QueryBalances(ctx, filter, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceListAccounts, err)
	}

	edges, err := makeEdges[*v2.AccountEdge](accountBalances)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceListAccounts, err)
	}

	accountsConnection := &v2.AccountsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListAccountsResponse{
		Accounts: accountsConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) ObserveAccounts(req *v2.ObserveAccountsRequest,
	srv v2.TradingDataService_ObserveAccountsServer,
) error {

	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	accountsChan, ref := t.accountService.ObserveAccountBalances(
		ctx, t.config.StreamRetries, req.MarketId, req.PartyId, req.Asset, req.Type)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Accounts subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "Accounts", accountsChan, ref, func(account entities.AccountBalance) error {
		return srv.Send(&v2.ObserveAccountsResponse{
			Account: account.ToProto(),
		})
	})
}

func (t *tradingDataServiceV2) Info(ctx context.Context, _ *v2.InfoRequest) (*v2.InfoResponse, error) {
	return &v2.InfoResponse{
		Version:    version.Get(),
		CommitHash: version.GetCommitHash(),
	}, nil
}

func (t *tradingDataServiceV2) GetBalanceHistory(ctx context.Context, req *v2.GetBalanceHistoryRequest) (*v2.GetBalanceHistoryResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetBalanceHistoryV2")()
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

func entityMarketDataListToProtoList(list []entities.MarketData) (*v2.MarketDataConnection, error) {
	if len(list) == 0 {
		return nil, nil
	}

	results := make([]*vega.MarketData, 0, len(list))

	for _, item := range list {
		results = append(results, item.ToProto())
	}

	edges, err := makeEdges[*v2.MarketDataEdge](list)
	if err != nil {
		return nil, errors.Wrap(err, "making edges")
	}

	connection := v2.MarketDataConnection{
		Edges: edges,
	}

	return &connection, nil
}

func (t *tradingDataServiceV2) GetMarketDataHistoryByID(ctx context.Context, req *v2.GetMarketDataHistoryByIDRequest) (*v2.GetMarketDataHistoryByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMarketDataHistoryV2")()
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

	edges, err := makeEdges[*v2.MarketDataEdge](history)
	if err != nil {
		return nil, errors.Wrap(err, "making edges")
	}

	connection := v2.MarketDataConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.GetMarketDataHistoryByIDResponse{
		MarketData: &connection,
	}, nil
}

func parseMarketDataResults(results []entities.MarketData) (*v2.GetMarketDataHistoryByIDResponse, error) {
	marketData, err := entityMarketDataListToProtoList(results)
	if err != nil {
		return nil, err
	}

	response := v2.GetMarketDataHistoryByIDResponse{
		MarketData: marketData,
	}

	return &response, err
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
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkLimitsV2")()
	if t.networkLimitsService == nil {
		return nil, errors.New("sql network limits store is not available")
	}

	limits, err := t.networkLimitsService.GetLatest(ctx)
	if err != nil {
		return nil, apiError(codes.Unknown, ErrGetNetworkLimits, err)
	}

	return &v2.GetNetworkLimitsResponse{Limits: limits.ToProto()}, nil
}

// ListCandleData for a given market, time range and interval.  Interval must be a valid postgres interval value
func (t *tradingDataServiceV2) ListCandleData(ctx context.Context, req *v2.ListCandleDataRequest) (*v2.ListCandleDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetCandleDataV2")()
	var err error
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

	edges, err := makeEdges[*v2.CandleEdge](candles)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := v2.CandleDataConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListCandleDataResponse{Candles: &connection}, nil
}

// SubscribeToCandleData subscribes to candle updates for a given market and interval.  Interval must be a valid postgres interval value
func (t *tradingDataServiceV2) SubscribeToCandleData(req *v2.SubscribeToCandleDataRequest, srv v2.TradingDataService_SubscribeToCandleDataServer) error {

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
	defer metrics.StartAPIRequestAndTimeGRPC("GetCandlesForMarketV2")()

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
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20MultiSigSignerAddedBundlesV2")()
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
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20MultiSigSignerRemovedBundlesV2")()
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
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20ListAssetBundleV2")()

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

	nonce, err := num.UintFromHex("0x" + strings.TrimLeft(req.AssetId, "0"))
	if err != nil {
		return nil, err
	}

	return &v2.GetERC20ListAssetBundleResponse{
		AssetSource: address,
		Nonce:       nonce.String(),
		VegaAssetId: asset.ID.String(),
		Signatures:  pack,
	}, nil
}

// Get trades by market using a cursor based pagination model
func (t *tradingDataServiceV2) GetTradesByMarket(ctx context.Context, in *v2.GetTradesByMarketRequest) (*v2.GetTradesByMarketResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetTradesByMarketV2")()

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

	edges, err := makeEdges[*v2.TradeEdge](trades)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	tradesConnection := &v2.TradeConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.GetTradesByMarketResponse{
		Trades: tradesConnection,
	}

	return resp, nil
}

// Get trades by party using a cursor based pagination model
func (t *tradingDataServiceV2) GetTradesByParty(ctx context.Context, in *v2.GetTradesByPartyRequest) (*v2.GetTradesByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetTradesByPartyV2")()

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

	edges, err := makeEdges[*v2.TradeEdge](trades)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	tradesConnection := &v2.TradeConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.GetTradesByPartyResponse{
		Trades: tradesConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) GetTradesByOrderID(ctx context.Context, in *v2.GetTradesByOrderIDRequest) (*v2.GetTradesByOrderIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetTradesByOrderIDV2")()

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

	edges, err := makeEdges[*v2.TradeEdge](trades)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	tradesConnection := &v2.TradeConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.GetTradesByOrderIDResponse{
		Trades: tradesConnection,
	}

	return resp, nil
}

// List all markets using a cursor based pagination model
func (t *tradingDataServiceV2) ListMarkets(ctx context.Context, in *v2.ListMarketsRequest) (*v2.ListMarketsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMarketsV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}
	markets, pageInfo, err := t.marketsService.GetAllPaged(ctx, in.MarketId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.MarketEdge](markets)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	marketsConnection := &v2.MarketConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListMarketsResponse{
		Markets: marketsConnection,
	}

	return resp, nil
}

// List all Positions using a cursor based pagination model
func (t *tradingDataServiceV2) ListPositions(ctx context.Context, in *v2.ListPositionsRequest) (*v2.ListPositionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetPositionsByPartyConnection")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	positions, pageInfo, err := t.positionService.GetByPartyConnection(ctx, entities.NewPartyID(in.PartyId), entities.NewMarketID(in.MarketId), pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.PositionEdge](positions)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	PositionsConnection := &v2.PositionConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListPositionsResponse{
		Positions: PositionsConnection,
	}

	return resp, nil
}

// List Parties using a cursor based pagination model
func (t *tradingDataServiceV2) ListParties(ctx context.Context, in *v2.ListPartiesRequest) (*v2.ListPartiesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetPartiesV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}
	parties, pageInfo, err := t.partyService.GetAllPaged(ctx, in.PartyId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.PartyEdge](parties)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	partyConnection := &v2.PartyConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListPartiesResponse{
		Party: partyConnection,
	}
	return resp, nil
}

func (t *tradingDataServiceV2) ListMarginLevels(ctx context.Context, in *v2.ListMarginLevelsRequest) (*v2.ListMarginLevelsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMarginLevelsV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	marginLevels, pageInfo, err := t.riskService.GetMarginLevelsByIDWithCursorPagination(ctx, in.PartyId, in.MarketId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.MarginEdge](marginLevels, t.accountService)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	marginLevelsConnection := &v2.MarginConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListMarginLevelsResponse{
		MarginLevels: marginLevelsConnection,
	}

	return resp, nil
}

// List rewards
func (t *tradingDataServiceV2) ListRewards(ctx context.Context, in *v2.ListRewardsRequest) (*v2.ListRewardsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetRewardsV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	rewards, pageInfo, err := t.rewardService.GetByCursor(ctx, &in.PartyId, &in.AssetId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.RewardEdge](rewards)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	rewardsConnection := &v2.RewardsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := v2.ListRewardsResponse{Rewards: rewardsConnection}
	return &resp, nil
}

// Get reward summaries
func (t *tradingDataServiceV2) ListRewardSummaries(ctx context.Context, in *v2.ListRewardSummariesRequest) (*v2.ListRewardSummariesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetRewardSummariesV2")()

	summaries, err := t.rewardService.GetSummaries(ctx, &in.PartyId, &in.AssetId)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	summaryProtos := make([]*vega.RewardSummary, len(summaries))

	for i, summary := range summaries {
		summaryProtos[i] = summary.ToProto()
	}

	resp := v2.ListRewardSummariesResponse{Summaries: summaryProtos}
	return &resp, nil
}

// -- Deposits --
func (t *tradingDataServiceV2) ListDeposits(ctx context.Context, req *v2.ListDepositsRequest) (*v2.ListDepositsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetDepositsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	deposits, pageInfo, err := t.depositService.GetByParty(ctx, req.PartyId, false, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.DepositEdge](deposits)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	depositConnection := &v2.DepositsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := v2.ListDepositsResponse{Deposits: depositConnection}

	return &resp, nil
}

func makeEdges[T proto.Message, V entities.PagedEntity[T]](inputs []V, args ...any) ([]T, error) {
	edges := make([]T, 0, len(inputs))
	for _, input := range inputs {
		edge, err := input.ToProtoEdge(args...)
		if err != nil {
			return nil, fmt.Errorf("failed to make edge for %v: %w", input, err)
		}

		edges = append(edges, edge)
	}
	return edges, nil
}

// -- Withdrawals --
func (t *tradingDataServiceV2) ListWithdrawals(ctx context.Context, req *v2.ListWithdrawalsRequest) (*v2.ListWithdrawalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetWithdrawalsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	withdrawals, pageInfo, err := t.withdrawalService.GetByParty(ctx, req.PartyId, false, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.WithdrawalEdge](withdrawals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	depositConnection := &v2.WithdrawalsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := v2.ListWithdrawalsResponse{Withdrawals: depositConnection}

	return &resp, nil
}

// -- Assets --
func (t *tradingDataServiceV2) ListAssets(ctx context.Context, req *v2.ListAssetsRequest) (*v2.ListAssetsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetAssetsV2")()

	if req.AssetId != "" {
		return t.getSingleAsset(ctx, req.AssetId)
	}

	return t.getAllAssets(ctx, req.Pagination)
}

func (t *tradingDataServiceV2) getSingleAsset(ctx context.Context, assetID string) (*v2.ListAssetsResponse, error) {
	asset, err := t.assetService.GetByID(ctx, assetID)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.AssetEdge]([]entities.Asset{asset})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := &v2.AssetsConnection{
		Edges: edges,
		PageInfo: &v2.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     asset.Cursor().Encode(),
			EndCursor:       asset.Cursor().Encode(),
		},
	}

	return &v2.ListAssetsResponse{Assets: connection}, nil
}

func (t *tradingDataServiceV2) getAllAssets(ctx context.Context, p *v2.Pagination) (*v2.ListAssetsResponse, error) {
	pagination, err := entities.CursorPaginationFromProto(p)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	assets, pageInfo, err := t.assetService.GetAllWithCursorPagination(ctx, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.AssetEdge](assets)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := &v2.AssetsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := v2.ListAssetsResponse{Assets: connection}
	return &resp, nil
}

func (t *tradingDataServiceV2) GetOracleSpecsConnection(ctx context.Context, req *v2.GetOracleSpecsConnectionRequest) (*v2.GetOracleSpecsConnectionResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOracleSpecsConnectionV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	specs, pageInfo, err := t.oracleSpecService.GetSpecsWithCursorPagination(ctx, req.SpecId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.OracleSpecEdge](specs)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := &v2.OracleSpecsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := v2.GetOracleSpecsConnectionResponse{
		OracleSpecs: connection,
	}

	return &resp, nil
}

func (t *tradingDataServiceV2) GetOracleDataConnection(ctx context.Context, req *v2.GetOracleDataConnectionRequest) (*v2.GetOracleDataConnectionResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOracleDataConnectionV2")()

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

	edges, err := makeEdges[*v2.OracleDataEdge](data)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := &v2.OracleDataConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := v2.GetOracleDataConnectionResponse{
		OracleData: connection,
	}

	return &resp, nil
}

func (t *tradingDataServiceV2) ListLiquidityProvisions(ctx context.Context, req *v2.ListLiquidityProvisionsRequest) (*v2.ListLiquidityProvisionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLiquidityProvisionsV2")()
	if req == nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("request is nil"))
	}

	var partyID entities.PartyID
	var marketID entities.MarketID
	var reference string

	if req.PartyId != nil {
		partyID = entities.NewPartyID(*req.PartyId)
	}

	if req.MarketId != nil {
		marketID = entities.NewMarketID(*req.MarketId)
	}

	if req.Reference != nil {
		reference = *req.Reference
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	lps, pageInfo, err := t.liquidityProvisionService.Get(ctx, partyID, marketID, reference, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.LiquidityProvisionsEdge](lps)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	liquidityProvisionConnection := &v2.LiquidityProvisionsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListLiquidityProvisionsResponse{LiquidityProvisions: liquidityProvisionConnection}, nil
}

func (t *tradingDataServiceV2) ListGovernanceData(ctx context.Context, req *v2.ListGovernanceDataRequest) (*v2.ListGovernanceDataResponse, error) {
	var state *entities.ProposalState
	var proposalType *entities.ProposalType

	if req.ProposalState != nil {
		s := entities.ProposalState(*req.ProposalState)
		state = &s
	}

	if req.ProposalType != nil {
		t := entities.ProposalType(*req.ProposalType)
		proposalType = &t
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	proposals, pageInfo, err := t.governanceService.GetProposals(
		ctx,
		state,
		req.ProposerPartyId,
		proposalType,
		pagination,
	)

	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.GovernanceDataEdge](proposals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	proposalsConnection := &v2.GovernanceDataConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListGovernanceDataResponse{Connection: proposalsConnection}, nil
}

// Get all Votes using a cursor based pagination model
func (t *tradingDataServiceV2) ListVotes(ctx context.Context, in *v2.ListVotesRequest) (*v2.ListVotesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListVotesV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	votes, pageInfo, err := t.governanceService.GetByPartyConnection(ctx, in.PartyId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.VoteEdge](votes)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	VotesConnection := &v2.VoteConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListVotesResponse{
		Votes: VotesConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) ListTransfers(ctx context.Context, req *v2.ListTransfersRequest) (*v2.ListTransfersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTransfersV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	var transfers []entities.Transfer
	var pageInfo entities.PageInfo
	if req.Pubkey == nil {
		transfers, pageInfo, err = t.transfersService.GetAll(ctx, pagination)
	} else {
		switch req.Direction {
		case v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_FROM:
			transfers, pageInfo, err = t.transfersService.GetTransfersFromParty(ctx, entities.NewPartyID(*req.Pubkey), pagination)
		case v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO:
			transfers, pageInfo, err = t.transfersService.GetTransfersToParty(ctx, entities.NewPartyID(*req.Pubkey), pagination)
		case v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO_OR_FROM:
			transfers, pageInfo, err = t.transfersService.GetTransfersToOrFromParty(ctx, entities.NewPartyID(*req.Pubkey), pagination)
		default:
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("transfer direction not supported:%v", req.Direction))
		}
	}

	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.TransferEdge](transfers)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &v2.ListTransfersResponse{Transfers: &v2.TransferConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}}, nil
}

func (t *tradingDataServiceV2) GetOrder(ctx context.Context, req *v2.GetOrderRequest) (*v2.GetOrderResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOrderV2")()

	order, err := t.orderService.GetOrder(ctx, req.OrderId, req.Version)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &v2.GetOrderResponse{Order: order.ToProto()}, nil
}

func (t *tradingDataServiceV2) ListOrders(ctx context.Context, in *v2.ListOrdersRequest) (*v2.ListOrdersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOrdersV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}
	orders, pageInfo, err := t.orderService.ListOrders(ctx, in.PartyId, in.MarketId, in.Reference, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.OrderEdge](orders)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	ordersConnection := &v2.OrderConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListOrdersResponse{
		Orders: ordersConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) ListOrderVersions(ctx context.Context, in *v2.ListOrderVersionsRequest) (*v2.ListOrderVersionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOrderVersionsV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}
	orders, pageInfo, err := t.orderService.ListOrderVersions(ctx, in.OrderId, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.OrderEdge](orders)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	ordersConnection := &v2.OrderConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListOrderVersionsResponse{
		Orders: ordersConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) ListDelegations(ctx context.Context, in *v2.ListDelegationsRequest) (*v2.ListDelegationsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListDelegationsV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)

	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	var epochID *int64

	if in.EpochId != nil {
		epoch, err := strconv.ParseInt(*in.EpochId, 10, 64)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid epoch id: %w", err))
		}

		epochID = &epoch
	}

	delegations, pageInfo, err := t.delegationService.Get(ctx, in.PartyId, in.NodeId, epochID, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.DelegationEdge](delegations)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	delegationsConnection := &v2.DelegationsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListDelegationsResponse{
		Delegations: delegationsConnection,
	}

	return resp, nil
}
