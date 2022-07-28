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
	"math"
	"strconv"
	"strings"
	"time"

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
	"github.com/pkg/errors"
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
	marketService             *service.Markets
	marketDepthService        *service.MarketDepth
	nodeService               *service.Node
	epochService              *service.Epoch
	riskFactorService         *service.RiskFactor
	networkParameterService   *service.NetworkParameter
	checkpointService         *service.Checkpoint
	stakeLinkingService       *service.StakeLinking
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

func (t *tradingDataServiceV2) ObserveMarketsDepth(req *v2.ObserveMarketsDepthRequest, srv v2.TradingDataService_ObserveMarketsDepthServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	for _, marketId := range req.MarketIds {
		if !t.marketExistsForId(ctx, marketId) {
			return apiError(codes.InvalidArgument, ErrMalformedRequest, fmt.Errorf("no market found for id:%s", marketId))
		}
	}

	depthChan, ref := t.marketDepthService.ObserveDepth(
		ctx, t.config.StreamRetries, req.MarketIds)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Depth subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "MarketDepth", depthChan, ref, func(tr []*vega.MarketDepth) error {
		return srv.Send(&v2.ObserveMarketsDepthResponse{
			MarketDepth: tr,
		})
	})

}

func (t *tradingDataServiceV2) ObserveMarketsDepthUpdates(req *v2.ObserveMarketsDepthUpdatesRequest, srv v2.TradingDataService_ObserveMarketsDepthUpdatesServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	for _, marketId := range req.MarketIds {
		if !t.marketExistsForId(ctx, marketId) {
			return apiError(codes.InvalidArgument, ErrMalformedRequest, fmt.Errorf("no market found for id:%s", marketId))
		}
	}
	depthChan, ref := t.marketDepthService.ObserveDepthUpdates(
		ctx, t.config.StreamRetries, req.MarketIds)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Depth updates subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "MarketDepthUpdate", depthChan, ref, func(tr []*vega.MarketDepthUpdate) error {
		return srv.Send(&v2.ObserveMarketsDepthUpdatesResponse{
			Update: tr,
		})
	})
}

func (t *tradingDataServiceV2) ObserveMarketsData(req *v2.ObserveMarketsDataRequest, srv v2.TradingDataService_ObserveMarketsDataServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	for _, marketId := range req.MarketIds {
		if !t.marketExistsForId(ctx, marketId) {
			return apiError(codes.InvalidArgument, ErrMalformedRequest, fmt.Errorf("no market found for id:%s", marketId))
		}
	}

	ch, ref := t.marketDataService.ObserveMarketData(ctx, t.config.StreamRetries, req.MarketIds)

	return observeBatch(ctx, t.log, "MarketsData", ch, ref, func(marketData []*entities.MarketData) error {
		out := make([]*vega.MarketData, 0, len(marketData))
		for _, v := range marketData {
			out = append(out, v.ToProto())
		}
		return srv.Send(&v2.ObserveMarketsDataResponse{MarketData: out})
	})

}

// GetLatestMarketData returns the latest market data for a given market
func (t *tradingDataServiceV2) GetLatestMarketData(ctx context.Context, req *v2.GetLatestMarketDataRequest) (*v2.GetLatestMarketDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLatestMarketData")()

	if !t.marketExistsForId(ctx, req.MarketId) {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, fmt.Errorf("no market found for id:%s", req.MarketId))
	}

	md, err := t.marketDataService.GetMarketDataByID(ctx, req.MarketId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetMarketData, err)
	}
	return &v2.GetLatestMarketDataResponse{
		MarketData: md.ToProto(),
	}, nil

}

// ListLatestMarketData returns the latest market data for every market
func (t *tradingDataServiceV2) ListLatestMarketData(ctx context.Context, req *v2.ListLatestMarketDataRequest) (*v2.ListLatestMarketDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListLatestMarketData")()
	mds, _ := t.marketDataService.GetMarketsData(ctx)

	mdptrs := make([]*vega.MarketData, 0, len(mds))
	for _, v := range mds {
		mdptrs = append(mdptrs, v.ToProto())
	}

	return &v2.ListLatestMarketDataResponse{
		MarketsData: mdptrs,
	}, nil

}

// GetLatestMarketDepth returns the latest market depth for a given market
func (t *tradingDataServiceV2) GetLatestMarketDepth(ctx context.Context, req *v2.GetLatestMarketDepthRequest) (*v2.GetLatestMarketDepthResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLatestMarketDepth")()

	var maxDepth uint64
	if req.MaxDepth != nil {
		maxDepth = *req.MaxDepth
	}

	depth := t.marketDepthService.GetMarketDepth(req.MarketId, maxDepth)

	lastOne := entities.OffsetPagination{Skip: 0, Limit: 1, Descending: true}
	ts, err := t.tradeService.GetByMarket(ctx, req.MarketId, lastOne)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}

	// Build market depth response, including last trade (if available)
	resp := &v2.GetLatestMarketDepthResponse{
		Buy:            depth.Buy,
		MarketId:       depth.MarketId,
		Sell:           depth.Sell,
		SequenceNumber: depth.SequenceNumber,
	}
	if len(ts) > 0 {
		resp.LastTrade = ts[0].ToProto()
	}
	return resp, nil
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
	defer metrics.StartAPIRequestAndTimeGRPC("ListCandleDataV2")()
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

// ObserveCandleData subscribes to candle updates for a given market and interval.  Interval must be a valid postgres interval value
func (t *tradingDataServiceV2) ObserveCandleData(req *v2.ObserveCandleDataRequest, srv v2.TradingDataService_ObserveCandleDataServer) error {

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

			resp := &v2.ObserveCandleDataResponse{
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

// ListCandleIntervals gets all available intervals for a given market along with the corresponding candle id
func (t *tradingDataServiceV2) ListCandleIntervals(ctx context.Context, req *v2.ListCandleIntervalsRequest) (*v2.ListCandleIntervalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListCandleIntervals")()
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

	return &v2.ListCandleIntervalsResponse{
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

func (t *tradingDataServiceV2) GetERC20SetAssetLimitsBundle(ctx context.Context, req *v2.GetERC20SetAssetLimitsBundleRequest) (*v2.GetERC20SetAssetLimitsBundleResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20SetAssetLimitsBundleV2")()
	if len(req.ProposalId) <= 0 {
		return nil, ErrMissingAssetID
	}

	if t.governanceService == nil {
		return nil, errors.New("sql asset store not available")
	}

	// first here we gonna get the proposal by its ID,
	proposal, err := t.governanceService.GetProposalByID(ctx, req.ProposalId)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	if proposal.Terms.GetUpdateAsset() == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("not an update asset proposal"))
	}
	if proposal.Terms.GetUpdateAsset().GetChanges().GetErc20() == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("not an update erc20 asset proposal"))

	}

	if t.notaryService == nil {
		return nil, errors.New("sql notary store not available")
	}

	// then we get the signature and pack them altogether
	signatures, err := t.notaryService.GetByResourceID(ctx, req.ProposalId)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	// now we pack them
	pack := "0x"
	for _, v := range signatures {
		pack = fmt.Sprintf("%v%v", pack, hex.EncodeToString(v.Sig))
	}

	if t.assetService == nil {
		return nil, errors.New("sql asset store not available")
	}

	// first here we gonna get the proposal by its ID,
	asset, err := t.assetService.GetByID(ctx, proposal.Terms.GetUpdateAsset().AssetId)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
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

	nonce, err := num.UintFromHex("0x" + strings.TrimLeft(req.ProposalId, "0"))
	if err != nil {
		return nil, err
	}

	return &v2.GetERC20SetAssetLimitsBundleResponse{
		AssetSource:   address,
		Nonce:         nonce.String(),
		VegaAssetId:   asset.ID.String(),
		Signatures:    pack,
		LifetimeLimit: proposal.Terms.GetUpdateAsset().GetChanges().GetErc20().LifetimeLimit,
		Threshold:     proposal.Terms.GetUpdateAsset().GetChanges().GetErc20().WithdrawThreshold,
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

func (t *tradingDataServiceV2) GetERC20WithdrawalApproval(ctx context.Context, req *v2.GetERC20WithdrawalApprovalRequest) (*v2.GetERC20WithdrawalApprovalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20WithdrawalApprovalV2")()
	if len(req.WithdrawalId) <= 0 {
		return nil, ErrMissingDepositID
	}

	// get withdrawal first
	w, err := t.withdrawalService.GetByID(ctx, req.WithdrawalId)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	// get the signatures from  notaryService
	signatures, err := t.notaryService.GetByResourceID(ctx, req.WithdrawalId)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	// some assets stuff
	assets, err := t.assetService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	// get the signature into the form form:
	// 0x + sig1 + sig2 + ... + sigN in hex encoded form
	pack := "0x"
	for _, v := range signatures {
		pack = fmt.Sprintf("%v%v", pack, hex.EncodeToString(v.Sig))
	}

	var address string
	for _, v := range assets {
		if v.ID == w.Asset {
			address = v.ERC20Contract
			break // found the one we want
		}
	}
	if len(address) <= 0 {
		return nil, fmt.Errorf("invalid erc20 token contract address")
	}

	return &v2.GetERC20WithdrawalApprovalResponse{
		AssetSource:   address,
		Amount:        fmt.Sprintf("%v", w.Amount),
		Expiry:        w.Expiry.UnixMicro(),
		Nonce:         w.Ref,
		TargetAddress: w.Ext.GetErc20().ReceiverAddress,
		Signatures:    pack,
		// timestamps is unix nano, contract needs unix. So load if first, and cut nanos
		Creation: w.CreatedTimestamp.Unix(),
	}, nil
}

// Get trades by using a cursor based pagination model
func (t *tradingDataServiceV2) ListTrades(ctx context.Context, in *v2.ListTradesRequest) (*v2.ListTradesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTradesV2")()
	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	trades, pageInfo, err := t.tradeService.List(ctx,
		entities.NewMarketID(in.GetMarketId()),
		entities.NewPartyID(in.GetPartyId()),
		entities.NewOrderID(in.GetOrderId()),
		pagination)
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

	resp := &v2.ListTradesResponse{
		Trades: tradesConnection,
	}

	return resp, nil
}

// ObserveTrades opens a subscription to the Trades service.
func (t *tradingDataServiceV2) ObserveTrades(req *v2.ObserveTradesRequest,
	srv v2.TradingDataService_ObserveTradesServer,
) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	tradesChan, ref := t.tradeService.Observe(ctx, t.config.StreamRetries, req.MarketId, req.PartyId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Trades subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Trade", tradesChan, ref, func(trades []*entities.Trade) error {
		out := make([]*vega.Trade, 0, len(trades))
		for _, v := range trades {
			out = append(out, v.ToProto())
		}
		return srv.Send(&v2.ObserveTradesResponse{Trades: out})
	})
}

// List all markets using a cursor based pagination model
func (t *tradingDataServiceV2) ListMarkets(ctx context.Context, in *v2.ListMarketsRequest) (*v2.ListMarketsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListMarketsV2")()

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
	defer metrics.StartAPIRequestAndTimeGRPC("ListPositionsV2")()

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
	defer metrics.StartAPIRequestAndTimeGRPC("ListPartiesV2")()

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
	defer metrics.StartAPIRequestAndTimeGRPC("ListMarginLevelsV2")()

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
	defer metrics.StartAPIRequestAndTimeGRPC("ListRewardsV2")()

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
	defer metrics.StartAPIRequestAndTimeGRPC("ListRewardSummariesV2")()

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
func (t *tradingDataServiceV2) GetDeposit(ctx context.Context, req *v2.GetDepositRequest) (*v2.GetDepositResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetDepositV2")()

	if req == nil || req.Id == "" {
		return nil, apiError(codes.InvalidArgument, errors.New("deposit id is required"))
	}
	deposit, err := t.depositService.GetByID(ctx, req.Id)

	if err != nil {
		return nil, apiError(codes.Internal, fmt.Errorf("retrieving deposit: %w", err))
	}

	return &v2.GetDepositResponse{
		Deposit: deposit.ToProto(),
	}, nil
}

func (t *tradingDataServiceV2) ListDeposits(ctx context.Context, req *v2.ListDepositsRequest) (*v2.ListDepositsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListDepositsV2")()

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
func (t *tradingDataServiceV2) GetWithdrawal(ctx context.Context, req *v2.GetWithdrawalRequest) (*v2.GetWithdrawalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetWithdrawalV2")()

	if req == nil || req.Id == "" {
		return nil, apiError(codes.InvalidArgument, errors.New("withdrawal id is required"))
	}
	withdrawal, err := t.withdrawalService.GetByID(ctx, req.Id)

	if err != nil {
		return nil, apiError(codes.Internal, fmt.Errorf("retrieving withdrawal: %w", err))
	}

	return &v2.GetWithdrawalResponse{
		Withdrawal: withdrawal.ToProto(),
	}, nil
}

func (t *tradingDataServiceV2) ListWithdrawals(ctx context.Context, req *v2.ListWithdrawalsRequest) (*v2.ListWithdrawalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListWithdrawalsV2")()

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
func (t *tradingDataServiceV2) GetAsset(ctx context.Context, req *v2.GetAssetRequest) (*v2.GetAssetResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetAssetV2")()
	if req == nil || req.AssetId == "" {
		return nil, apiError(codes.InvalidArgument, errors.New("asset Id is required"))
	}

	asset, err := t.assetService.GetByID(ctx, req.AssetId)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &v2.GetAssetResponse{
		Asset: asset.ToProto(),
	}, nil
}

func (t *tradingDataServiceV2) ListAssets(ctx context.Context, req *v2.ListAssetsRequest) (*v2.ListAssetsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAssetsV2")()

	if req == nil {
		req = &v2.ListAssetsRequest{}
	}

	if req != nil && *req.AssetId != "" {
		return t.getSingleAsset(ctx, *req.AssetId)
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

func (t *tradingDataServiceV2) GetOracleSpec(ctx context.Context, req *v2.GetOracleSpecRequest) (*v2.GetOracleSpecResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOracleSpecV2")()
	if req == nil || req.OracleSpecId == "" {
		return nil, apiError(codes.InvalidArgument, errors.New("oracle spec id is required"))
	}

	spec, err := t.oracleSpecService.GetSpecByID(ctx, req.OracleSpecId)
	if err != nil {
		return nil, apiError(codes.Internal, fmt.Errorf("retrieving oracle data for spec id: %w", err))
	}

	return &v2.GetOracleSpecResponse{
		OracleSpec: spec.ToProto(),
	}, nil
}

func (t *tradingDataServiceV2) ListOracleSpecs(ctx context.Context, req *v2.ListOracleSpecsRequest) (*v2.ListOracleSpecsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOracleSpecsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	specs, pageInfo, err := t.oracleSpecService.GetSpecsWithCursorPagination(ctx, "", pagination)
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

	resp := v2.ListOracleSpecsResponse{
		OracleSpecs: connection,
	}

	return &resp, nil
}

func (t *tradingDataServiceV2) ListOracleData(ctx context.Context, req *v2.ListOracleDataRequest) (*v2.ListOracleDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOracleDataConnectionV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	var data []entities.OracleData
	var pageInfo entities.PageInfo

	if req != nil && req.OracleSpecId != nil && *req.OracleSpecId != "" {
		data, pageInfo, err = t.oracleDataService.GetOracleDataBySpecID(ctx, *req.OracleSpecId, pagination)
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

	resp := v2.ListOracleDataResponse{
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

func (t *tradingDataServiceV2) marketExistsForId(ctx context.Context, marketID string) bool {
	_, err := t.marketsService.GetByID(ctx, marketID)
	return err == nil
}

// GetNetworkData retrieve network data regarding the nodes of the network
func (t *tradingDataServiceV2) GetNetworkData(ctx context.Context, _ *v2.GetNetworkDataRequest) (*v2.GetNetworkDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkDataV2")()

	nodeData, err := t.nodeService.GetNodeData(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &v2.GetNetworkDataResponse{
		NodeData: nodeData.ToProto(),
	}, nil
}

// GetNode retrieves information about a given node
func (t *tradingDataServiceV2) GetNode(ctx context.Context, req *v2.GetNodeRequest) (*v2.GetNodeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodeV2")()

	if req.GetId() == "" {
		return nil, apiError(codes.InvalidArgument, errors.New("missing node ID parameter"))
	}

	epoch, err := t.epochService.GetCurrent(ctx)
	if err != nil {
		fmt.Printf("%v", err)
		return nil, apiError(codes.Internal, err)
	}

	node, err := t.nodeService.GetNodeByID(ctx, req.GetId(), uint64(epoch.ID))
	if err != nil {
		fmt.Printf("%v", err)
		return nil, apiError(codes.NotFound, err)
	}

	return &v2.GetNodeResponse{
		Node: node.ToProto(),
	}, nil
}

// ListNodes returns information about the nodes on the network
func (t *tradingDataServiceV2) ListNodes(ctx context.Context, req *v2.ListNodesRequest) (*v2.ListNodesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListNodesV2")()
	var epoch entities.Epoch
	var pagination entities.CursorPagination
	var err error

	if req == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("request is nil"))
	}

	if req.EpochSeq == nil || *req.EpochSeq > math.MaxInt64 {
		epoch, err = t.epochService.GetCurrent(ctx)
	} else {
		epochSeq := int64(*req.EpochSeq)
		epoch, err = t.epochService.Get(ctx, epochSeq)
	}

	if err != nil {
		fmt.Printf("%v", err)
		return nil, apiError(codes.Internal, err)
	}

	pagination, err = entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err))
	}

	nodes, pageInfo, err := t.nodeService.GetNodes(ctx, uint64(epoch.ID), pagination)
	if err != nil {
		fmt.Printf("%v", err)
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.NodeEdge](nodes)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	nodesConnection := &v2.NodesConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListNodesResponse{
		Nodes: nodesConnection,
	}

	return resp, nil
}

// GetEpoch retrieves data for a specific epoch, if id omitted it gets the current epoch
func (t *tradingDataServiceV2) GetEpoch(ctx context.Context, req *v2.GetEpochRequest) (*v2.GetEpochResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetEpochV2")()

	var epoch entities.Epoch
	var err error

	if req.GetId() == 0 {
		epoch, err = t.epochService.GetCurrent(ctx)
	} else {
		epoch, err = t.epochService.Get(ctx, int64(req.GetId()))
	}

	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoEpoch := epoch.ToProto()

	delegations, _, err := t.delegationService.Get(ctx, nil, nil, &epoch.ID, nil)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoDelegations := make([]*vega.Delegation, len(delegations))
	for i, delegation := range delegations {
		protoDelegations[i] = delegation.ToProto()
	}
	protoEpoch.Delegations = protoDelegations

	nodes, _, err := t.nodeService.GetNodes(ctx, uint64(epoch.ID), entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoNodes := make([]*vega.Node, len(nodes))
	for i, node := range nodes {
		protoNodes[i] = node.ToProto()
	}

	protoEpoch.Validators = protoNodes

	return &v2.GetEpochResponse{
		Epoch: protoEpoch,
	}, nil
}

func (t *tradingDataServiceV2) EstimateFee(ctx context.Context, req *v2.EstimateFeeRequest) (*v2.EstimateFeeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateFee SQL")()
	if req.Order == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("missing order"))
	}

	fee, err := t.estimateFee(ctx, req.Order)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &v2.EstimateFeeResponse{
		Fee: fee,
	}, nil
}

func (t *tradingDataServiceV2) estimateFee(ctx context.Context, order *vega.Order) (*vega.Fee, error) {
	mkt, err := t.marketService.GetByID(ctx, order.MarketId)
	if err != nil {
		return nil, err
	}
	price, overflowed := num.UintFromString(order.Price, 10)
	if overflowed {
		return nil, errors.New("invalid order price")
	}
	if order.PeggedOrder != nil {
		return &vega.Fee{
			MakerFee:          "0",
			InfrastructureFee: "0",
			LiquidityFee:      "0",
		}, nil
	}

	base := num.DecimalFromUint(price.Mul(price, num.NewUint(order.Size)))
	maker, infra, liquidity, err := t.feeFactors(mkt)
	if err != nil {
		return nil, err
	}

	fee := &vega.Fee{
		MakerFee:          base.Mul(num.NewDecimalFromFloat(maker)).String(),
		InfrastructureFee: base.Mul(num.NewDecimalFromFloat(infra)).String(),
		LiquidityFee:      base.Mul(num.NewDecimalFromFloat(liquidity)).String(),
	}

	return fee, nil
}

func (t *tradingDataServiceV2) feeFactors(mkt entities.Market) (maker, infra, liquidity float64, err error) {
	if maker, err = strconv.ParseFloat(mkt.Fees.Factors.MakerFee, 64); err != nil {
		return
	}
	if infra, err = strconv.ParseFloat(mkt.Fees.Factors.InfrastructureFee, 64); err != nil {
		return
	}
	if liquidity, err = strconv.ParseFloat(mkt.Fees.Factors.LiquidityFee, 64); err != nil {
		return
	}

	return
}

func (t *tradingDataServiceV2) EstimateMargin(ctx context.Context, req *v2.EstimateMarginRequest) (*v2.EstimateMarginResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateMargin SQL")()
	if req.Order == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("missing order"))
	}

	margin, err := t.estimateMargin(ctx, req.Order)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &v2.EstimateMarginResponse{
		MarginLevels: margin,
	}, nil
}

func (t *tradingDataServiceV2) estimateMargin(ctx context.Context, order *vega.Order) (*vega.MarginLevels, error) {
	if order.Side == vega.Side_SIDE_UNSPECIFIED {
		return nil, ErrInvalidOrderSide
	}

	// first get the risk factors and market data (marketdata->markprice)
	rf, err := t.riskFactorService.GetMarketRiskFactors(ctx, order.MarketId)
	if err != nil {
		return nil, err
	}
	mkt, err := t.marketService.GetByID(ctx, order.MarketId)
	if err != nil {
		return nil, err
	}

	mktProto := mkt.ToProto()

	mktData, err := t.marketDataService.GetMarketDataByID(ctx, order.MarketId)
	if err != nil {
		return nil, err
	}

	f, err := num.DecimalFromString(rf.Short.String())
	if err != nil {
		return nil, err
	}
	if order.Side == vega.Side_SIDE_BUY {
		f, err = num.DecimalFromString(rf.Long.String())
		if err != nil {
			return nil, err
		}
	}

	asset, err := mktProto.GetAsset()
	if err != nil {
		return nil, err
	}

	// now calculate margin maintenance
	markPrice, _ := num.DecimalFromString(mktData.MarkPrice.String())

	// if the order is a limit order, use the limit price to calculate the margin maintenance
	if order.Type == vega.Order_TYPE_LIMIT {
		markPrice, _ = num.DecimalFromString(order.Price)
	}

	maintenanceMargin := num.DecimalFromFloat(float64(order.Size)).Mul(f).Mul(markPrice)
	// now we use the risk factors
	return &vega.MarginLevels{
		PartyId:                order.PartyId,
		MarketId:               mktProto.GetId(),
		Asset:                  asset,
		Timestamp:              0,
		MaintenanceMargin:      maintenanceMargin.String(),
		SearchLevel:            maintenanceMargin.Mul(num.DecimalFromFloat(mkt.TradableInstrument.MarginCalculator.ScalingFactors.SearchLevel)).String(),
		InitialMargin:          maintenanceMargin.Mul(num.DecimalFromFloat(mkt.TradableInstrument.MarginCalculator.ScalingFactors.InitialMargin)).String(),
		CollateralReleaseLevel: maintenanceMargin.Mul(num.DecimalFromFloat(mkt.TradableInstrument.MarginCalculator.ScalingFactors.CollateralRelease)).String(),
	}, nil
}

func (t *tradingDataServiceV2) ListNetworkParameters(ctx context.Context, req *v2.ListNetworkParametersRequest) (*v2.ListNetworkParametersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("NetworkParametersV2")()
	var pagination entities.CursorPagination
	var err error
	if req != nil {
		pagination, err = entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err))
		}
	}
	nps, pageInfo, err := t.networkParameterService.GetAll(ctx, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.NetworkParameterEdge](nps)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	networkParametersConnection := &v2.NetworkParameterConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListNetworkParametersResponse{
		NetworkParameters: networkParametersConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) ListCheckpoints(ctx context.Context, req *v2.ListCheckpointsRequest) (*v2.ListCheckpointsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("NetworkParametersV2")()
	var pagination entities.CursorPagination
	var err error
	if req != nil {
		pagination, err = entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err))
		}
	}

	checkpoints, pageInfo, err := t.checkpointService.GetAll(ctx, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.CheckpointEdge](checkpoints)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	checkpointsConnection := &v2.CheckpointsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListCheckpointsResponse{
		Checkpoints: checkpointsConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) GetStake(ctx context.Context, req *v2.GetStakeRequest) (*v2.GetStakeResponse, error) {
	if req == nil || len(req.Party) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing party id"))
	}

	var pagination entities.CursorPagination

	partyID := entities.NewPartyID(req.Party)

	if req != nil && req.Pagination != nil {
		var err error
		pagination, err = entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, apiError(codes.Internal, fmt.Errorf("invalid pagination: %w", err))
		}
	}

	stake, stakeLinkings, pageInfo, err := t.stakeLinkingService.GetStake(ctx, partyID, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, fmt.Errorf("fetching party stake linkings: %w", err))
	}

	edges, err := makeEdges[*v2.StakeLinkingEdge](stakeLinkings)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	stakesConnection := &v2.StakesConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.GetStakeResponse{
		CurrentStakeAvailable: num.UintToString(stake),
		StakeLinkings:         stakesConnection,
	}

	return resp, nil
}

func (t *tradingDataServiceV2) GetRiskFactors(ctx context.Context, req *v2.GetRiskFactorsRequest) (*v2.GetRiskFactorsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetRiskFactors SQL")()

	rfs, err := t.riskFactorService.GetMarketRiskFactors(ctx, req.MarketId)
	if err != nil {
		return nil, nil
	}

	return &v2.GetRiskFactorsResponse{
		RiskFactor: rfs.ToProto(),
	}, nil
}
