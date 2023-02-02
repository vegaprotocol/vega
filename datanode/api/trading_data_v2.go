//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/datanode/networkhistory"

	"code.vegaprotocol.io/vega/datanode/networkhistory/store"

	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/version"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

var defaultPaginationV2 = entities.OffsetPagination{
	Skip:       0,
	Limit:      1000,
	Descending: true,
}

// When returning an 'initial image' snapshot, how many updates to batch into each page.
var snapshotPageSize = 500

type tradingDataServiceV2 struct {
	v2.UnimplementedTradingDataServiceServer
	config                     Config
	log                        *logging.Logger
	eventService               EventService
	orderService               *service.Order
	networkLimitsService       *service.NetworkLimits
	marketDataService          *service.MarketData
	tradeService               *service.Trade
	multiSigService            *service.MultiSig
	notaryService              *service.Notary
	assetService               *service.Asset
	candleService              *candlesv2.Svc
	marketsService             *service.Markets
	partyService               *service.Party
	riskService                *service.Risk
	positionService            *service.Position
	accountService             *service.Account
	rewardService              *service.Reward
	depositService             *service.Deposit
	withdrawalService          *service.Withdrawal
	oracleSpecService          *service.OracleSpec
	oracleDataService          *service.OracleData
	liquidityProvisionService  *service.LiquidityProvision
	governanceService          *service.Governance
	transfersService           *service.Transfer
	delegationService          *service.Delegation
	marketService              *service.Markets
	marketDepthService         *service.MarketDepth
	nodeService                *service.Node
	epochService               *service.Epoch
	riskFactorService          *service.RiskFactor
	networkParameterService    *service.NetworkParameter
	checkpointService          *service.Checkpoint
	stakeLinkingService        *service.StakeLinking
	ledgerService              *service.Ledger
	keyRotationService         *service.KeyRotations
	ethereumKeyRotationService *service.EthereumKeyRotation
	blockService               BlockService
	protocolUpgradeService     *service.ProtocolUpgrade
	networkHistoryService      NetworkHistoryService
	coreSnapshotService        *service.SnapshotData
}

func (t *tradingDataServiceV2) ListAccounts(ctx context.Context, req *v2.ListAccountsRequest) (*v2.ListAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAccountsV2")()
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

	// First get the 'initial image' of accounts matching the request and send those
	if err := t.sendAccountsSnapshot(ctx, req, srv); err != nil {
		return err
	}
	accountsChan, ref := t.accountService.ObserveAccountBalances(
		ctx, t.config.StreamRetries, req.MarketId, req.PartyId, req.Asset, req.Type)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Accounts subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Accounts", accountsChan, ref, func(accounts []entities.AccountBalance) error {
		protos := make([]*v2.AccountBalance, len(accounts))
		for i := 0; i < len(accounts); i++ {
			protos[i] = accounts[i].ToProto()
		}
		updates := &v2.AccountUpdates{Accounts: protos}
		responseUpdates := &v2.ObserveAccountsResponse_Updates{Updates: updates}
		response := &v2.ObserveAccountsResponse{Response: responseUpdates}
		return srv.Send(response)
	})
}

func (t *tradingDataServiceV2) sendAccountsSnapshot(ctx context.Context, req *v2.ObserveAccountsRequest,
	srv v2.TradingDataService_ObserveAccountsServer,
) error {
	filter := entities.AccountFilter{}
	if req.Asset != "" {
		filter.AssetID = entities.AssetID(req.Asset)
	}
	if req.PartyId != "" {
		filter.PartyIDs = append(filter.PartyIDs, entities.PartyID(req.PartyId))
	}
	if req.MarketId != "" {
		filter.MarketIDs = append(filter.MarketIDs, entities.MarketID(req.MarketId))
	}
	if req.Type != vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		filter.AccountTypes = append(filter.AccountTypes, req.Type)
	}

	accounts, pageInfo, err := t.accountService.QueryBalances(ctx, filter, entities.CursorPagination{})
	if err != nil {
		return errors.Wrap(err, "fetching account balance initial image")
	}

	if pageInfo.HasNextPage {
		return fmt.Errorf("initial image spans multiple pages")
	}

	protos := make([]*v2.AccountBalance, len(accounts))
	for i := 0; i < len(accounts); i++ {
		protos[i] = accounts[i].ToProto()
	}

	batches := batch(protos, snapshotPageSize)
	for i, batch := range batches {
		isLast := i == len(batches)-1
		page := &v2.AccountSnapshotPage{Accounts: batch, LastPage: isLast}
		snapshot := &v2.ObserveAccountsResponse_Snapshot{Snapshot: page}
		response := &v2.ObserveAccountsResponse{Response: snapshot}
		if err := srv.Send(response); err != nil {
			return errors.Wrap(err, "sending account balance initial image")
		}
	}
	return nil
}

func (t *tradingDataServiceV2) Info(ctx context.Context, _ *v2.InfoRequest) (*v2.InfoResponse, error) {
	return &v2.InfoResponse{
		Version:    version.Get(),
		CommitHash: version.GetCommitHash(),
	}, nil
}

func (t *tradingDataServiceV2) ListLedgerEntries(ctx context.Context, req *v2.ListLedgerEntriesRequest) (*v2.ListLedgerEntriesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListLedgerEntriesV2")()
	if t.accountService == nil {
		return nil, apiError(codes.Internal, ErrAccountServiceSQLStoreNotAvailable, nil)
	}

	leFilter, err := entities.LedgerEntryFilterFromProto(req.Filter)
	if err != nil {
		return nil, fmt.Errorf("could not parse ledger entry filter: %w", err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid cursor: %w", err))
	}

	entries, pageInfo, err := t.ledgerService.Query(ctx, leFilter, dateRange, pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("could not query ledger entries: %w", err))
	}

	edges, err := makeEdges[*v2.AggregatedLedgerEntriesEdge](*entries)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceListAccounts, err)
	}

	return &v2.ListLedgerEntriesResponse{
		LedgerEntries: &v2.AggregatedLedgerEntriesConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}

func (t *tradingDataServiceV2) ExportLedgerEntries(ctx context.Context, req *v2.ExportLedgerEntriesRequest) (*v2.ExportLedgerEntriesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ExportLedgerEntriesV2")()
	if t.accountService == nil {
		return nil, apiError(codes.Internal, ErrAccountServiceSQLStoreNotAvailable, nil)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid cursor: %w", err))
	}

	raw, pageInfo, err := t.ledgerService.Export(ctx, req.PartyId, req.AssetId, dateRange, pagination)
	if err != nil {
		apiError(codes.Aborted, err)
	}

	header := metadata.New(map[string]string{
		"Content-Type":       "text/csv",
		"Content-diposition": fmt.Sprintf("attachment;filename=%s", "ledger_entries_export.csv"),
	})

	if err := grpc.SendHeader(ctx, header); err != nil {
		return nil, apiError(codes.Internal, fmt.Errorf("unable to send 'x-response-id' header"))
	}

	return &v2.ExportLedgerEntriesResponse{
		Data:     raw,
		PageInfo: pageInfo.ToProto(),
	}, nil
}

func (t *tradingDataServiceV2) ListBalanceChanges(ctx context.Context, req *v2.ListBalanceChangesRequest) (*v2.ListBalanceChangesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListBalanceChangesV2")()
	if t.accountService == nil {
		return nil, fmt.Errorf("sql balance store not available")
	}

	filter, err := entities.AccountFilterFromProto(req.Filter)
	if err != nil {
		return nil, fmt.Errorf("parsing filter: %w", err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid cursor: %w", err))
	}

	balances, pageInfo, err := t.accountService.QueryAggregatedBalances(ctx, filter, dateRange, pagination)
	if err != nil {
		return nil, fmt.Errorf("querying balances: %w", err)
	}

	edges, err := makeEdges[*v2.AggregatedBalanceEdge](*balances)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceListAccounts, err)
	}

	return &v2.ListBalanceChangesResponse{
		Balances: &v2.AggregatedBalanceConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}

func entityMarketDataListToProtoList(list []entities.MarketData) (*v2.MarketDataConnection, error) {
	if len(list) == 0 {
		return nil, nil
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

	for _, marketID := range req.MarketIds {
		if !t.marketExistsForID(ctx, marketID) {
			return apiError(codes.InvalidArgument, ErrMalformedRequest, fmt.Errorf("no market found for id:%s", marketID))
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

	for _, marketID := range req.MarketIds {
		if !t.marketExistsForID(ctx, marketID) {
			return apiError(codes.InvalidArgument, ErrMalformedRequest, fmt.Errorf("no market found for id:%s", marketID))
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

	for _, marketID := range req.MarketIds {
		if !t.marketExistsForID(ctx, marketID) {
			return apiError(codes.InvalidArgument, ErrMalformedRequest, fmt.Errorf("no market found for id:%s", marketID))
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

// GetLatestMarketData returns the latest market data for a given market.
func (t *tradingDataServiceV2) GetLatestMarketData(ctx context.Context, req *v2.GetLatestMarketDataRequest) (*v2.GetLatestMarketDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLatestMarketData")()

	md, err := t.marketDataService.GetMarketDataByID(ctx, req.MarketId)
	if err != nil {
		return nil, t.formatE(err)
	}
	return &v2.GetLatestMarketDataResponse{
		MarketData: md.ToProto(),
	}, nil
}

// ListLatestMarketData returns the latest market data for every market.
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

// GetLatestMarketDepth returns the latest market depth for a given market.
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
		return nil, t.formatE(err)
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
		return nil, t.formatE(err)
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

// ListCandleData for a given market, time range and interval.  Interval must be a valid postgres interval value.
func (t *tradingDataServiceV2) ListCandleData(ctx context.Context, req *v2.ListCandleDataRequest) (*v2.ListCandleDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListCandleDataV2")()
	var err error
	if t.candleService == nil {
		return nil, errors.New("sql candle service not available")
	}

	var from, to *time.Time
	if req.FromTimestamp != 0 {
		from = ptr.From(vegatime.UnixNano(req.FromTimestamp))
	}

	if req.ToTimestamp != 0 {
		to = ptr.From(vegatime.UnixNano(req.ToTimestamp))
	}

	pagination := entities.CursorPagination{}
	if req.Pagination != nil {
		pagination, err = entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, fmt.Errorf("could not parse cursor pagination information: %w", err)
		}
	}

	if req.CandleId == "" {
		return nil, apiError(codes.InvalidArgument, ErrMissingCandleID)
	}

	candles, pageInfo, err := t.candleService.GetCandleDataForTimeSpan(ctx, req.CandleId, from, to, pagination)
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

// ObserveCandleData subscribes to candle updates for a given market and interval.  Interval must be a valid postgres interval value.
func (t *tradingDataServiceV2) ObserveCandleData(req *v2.ObserveCandleDataRequest, srv v2.TradingDataService_ObserveCandleDataServer) error {
	defer metrics.StartActiveSubscriptionCountGRPC("Candle")()

	if t.candleService == nil {
		return errors.New("sql candle service not available")
	}

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	subscriptionID, candlesChan, err := t.candleService.Subscribe(ctx, req.CandleId)
	defer t.candleService.Unsubscribe(subscriptionID)

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

// ListCandleIntervals gets all available intervals for a given market along with the corresponding candle id.
func (t *tradingDataServiceV2) ListCandleIntervals(ctx context.Context, req *v2.ListCandleIntervalsRequest) (*v2.ListCandleIntervalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListCandleIntervals")()
	if t.candleService == nil {
		return nil, errors.New("sql candle service not available")
	}

	mappings, err := t.candleService.GetCandlesForMarket(ctx, req.MarketId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCandleServiceGetCandlesForMarket, err)
	}

	intervalToCandleIds := make([]*v2.IntervalToCandleId, 0, len(mappings))
	for interval, candleID := range mappings {
		intervalToCandleIds = append(intervalToCandleIds, &v2.IntervalToCandleId{
			Interval: interval,
			CandleId: candleID,
		})
	}

	return &v2.ListCandleIntervalsResponse{
		IntervalToCandleId: intervalToCandleIds,
	}, nil
}

// ListERC20MutlsigSignerAddedBundles return the signature bundles needed to add a new validator to the multisig control ERC20 contract.
func (t *tradingDataServiceV2) ListERC20MultiSigSignerAddedBundles(ctx context.Context, req *v2.ListERC20MultiSigSignerAddedBundlesRequest) (*v2.ListERC20MultiSigSignerAddedBundlesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20MultiSigSignerAddedBundlesV2")()
	if t.notaryService == nil {
		return nil, errors.New("sql notary service not available")
	}

	if t.multiSigService == nil {
		return nil, errors.New("sql multisig event store not available")
	}

	var epochID *int64
	if len(req.EpochSeq) != 0 {
		e, err := strconv.ParseInt(req.EpochSeq, 10, 64)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("epochID is not a valid integer"))
		}
		epochID = &e
	}

	p := entities.CursorPagination{}
	var err error
	if req.Pagination != nil {
		p, err = entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err))
		}
	}

	res, pageInfo, err := t.multiSigService.GetAddedEvents(ctx, req.GetNodeId(), req.GetSubmitter(), epochID, p)
	if err != nil {
		c := codes.Internal
		if errors.Is(err, entities.ErrInvalidID) {
			c = codes.InvalidArgument
		}
		return nil, apiError(c, err)
	}

	// find bundle for this nodeID, might be multiple if its added, then removed then added again??
	edges := []*v2.ERC20MultiSigSignerAddedBundleEdge{}
	for _, b := range res {
		// it doesn't really make sense to paginate this, so we'll just pass it an empty pagination object and get all available results
		signatures, _, err := t.notaryService.GetByResourceID(ctx, b.ID.String(), entities.CursorPagination{})
		if err != nil {
			return nil, apiError(codes.Internal, err)
		}

		edges = append(edges,
			&v2.ERC20MultiSigSignerAddedBundleEdge{
				Node: &v2.ERC20MultiSigSignerAddedBundle{
					NewSigner:  b.SignerChange.String(),
					Submitter:  b.Submitter.String(),
					Nonce:      b.Nonce,
					Timestamp:  b.VegaTime.UnixNano(),
					Signatures: packNodeSignatures(signatures),
					EpochSeq:   strconv.FormatInt(b.EpochID, 10),
				},
				Cursor: b.Cursor().Encode(),
			},
		)
	}

	connection := &v2.ERC20MultiSigSignerAddedConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListERC20MultiSigSignerAddedBundlesResponse{
		Bundles: connection,
	}, nil
}

// ListERC20MutlsigSignerAddedBundles return the signature bundles needed to add a new validator to the multisig control ERC20 contract.
func (t *tradingDataServiceV2) ListERC20MultiSigSignerRemovedBundles(ctx context.Context, req *v2.ListERC20MultiSigSignerRemovedBundlesRequest) (*v2.ListERC20MultiSigSignerRemovedBundlesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20MultiSigSignerRemovedBundlesV2")()
	if t.notaryService == nil {
		return nil, errors.New("sql notary store not available")
	}

	if t.multiSigService == nil {
		return nil, errors.New("sql multisig event store not available")
	}

	var epochID *int64
	if len(req.EpochSeq) != 0 {
		e, err := strconv.ParseInt(req.EpochSeq, 10, 64)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("epochID is not a valid integer"))
		}
		epochID = &e
	}

	p := entities.CursorPagination{}
	var err error
	if req.Pagination != nil {
		p, err = entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err))
		}
	}

	res, pageInfo, err := t.multiSigService.GetRemovedEvents(ctx, req.GetNodeId(), req.GetSubmitter(), epochID, p)
	if err != nil {
		c := codes.Internal
		if errors.Is(err, entities.ErrInvalidID) {
			c = codes.InvalidArgument
		}
		return nil, apiError(c, err)
	}

	// find bundle for this nodeID, might be multiple if its added, then removed then added again??
	edges := []*v2.ERC20MultiSigSignerRemovedBundleEdge{}
	for _, b := range res {
		signatures, _, err := t.notaryService.GetByResourceID(ctx, b.ID.String(), entities.CursorPagination{})
		if err != nil {
			return nil, apiError(codes.Internal, err)
		}

		edges = append(edges, &v2.ERC20MultiSigSignerRemovedBundleEdge{
			Node: &v2.ERC20MultiSigSignerRemovedBundle{
				OldSigner:  b.SignerChange.String(),
				Submitter:  b.Submitter.String(),
				Nonce:      b.Nonce,
				Timestamp:  b.VegaTime.UnixNano(),
				Signatures: packNodeSignatures(signatures),
				EpochSeq:   strconv.FormatInt(b.EpochID, 10),
			},
			Cursor: b.Cursor().Encode(),
		})
	}

	connection := &v2.ERC20MultiSigSignerRemovedConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListERC20MultiSigSignerRemovedBundlesResponse{
		Bundles: connection,
	}, nil
}

func (t *tradingDataServiceV2) GetERC20SetAssetLimitsBundle(ctx context.Context, req *v2.GetERC20SetAssetLimitsBundleRequest) (*v2.GetERC20SetAssetLimitsBundleResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20SetAssetLimitsBundleV2")()
	if len(req.ProposalId) <= 0 {
		return nil, t.formatE(entities.ErrInvalidID)
	}

	// first here we gonna get the proposal by its ID,
	proposal, err := t.governanceService.GetProposalByID(ctx, req.ProposalId)
	if err != nil {
		return nil, t.formatE(err)
	}

	if proposal.Terms.GetUpdateAsset() == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("not an update asset proposal"))
	}
	if proposal.Terms.GetUpdateAsset().GetChanges().GetErc20() == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("not an update erc20 asset proposal"))
	}

	// then we get the signature and pack them altogether
	signatures, _, err := t.notaryService.GetByResourceID(ctx, req.ProposalId, entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	// first here we gonna get the proposal by its ID,
	asset, err := t.assetService.GetByID(ctx, proposal.Terms.GetUpdateAsset().AssetId)
	if err != nil {
		return nil, t.formatE(err)
	}

	var address string
	if asset.ERC20Contract != "" {
		address = asset.ERC20Contract
	} else {
		return nil, apiError(codes.InvalidArgument, errors.New("invalid asset source"))
	}

	if len(address) <= 0 {
		return nil, apiError(codes.Internal, errors.New("invalid erc20 token contract address"))
	}

	nonce, err := num.UintFromHex("0x" + strings.TrimLeft(req.ProposalId, "0"))
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &v2.GetERC20SetAssetLimitsBundleResponse{
		AssetSource:   address,
		Nonce:         nonce.String(),
		VegaAssetId:   asset.ID.String(),
		Signatures:    packNodeSignatures(signatures),
		LifetimeLimit: proposal.Terms.GetUpdateAsset().GetChanges().GetErc20().LifetimeLimit,
		Threshold:     proposal.Terms.GetUpdateAsset().GetChanges().GetErc20().WithdrawThreshold,
	}, nil
}

// packNodeSignatures packs a list signatures into the form form:
// 0x + sig1 + sig2 + ... + sigN in hex encoded form
// If the list is empty, return an empty string instead.
func packNodeSignatures(signatures []entities.NodeSignature) string {
	pack := ""
	if len(signatures) > 0 {
		pack = "0x"
	}

	for _, v := range signatures {
		pack = fmt.Sprintf("%v%v", pack, hex.EncodeToString(v.Sig))
	}

	return pack
}

func (t *tradingDataServiceV2) GetERC20ListAssetBundle(ctx context.Context, req *v2.GetERC20ListAssetBundleRequest) (*v2.GetERC20ListAssetBundleResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20ListAssetBundleV2")()

	if len(req.AssetId) <= 0 {
		return nil, t.formatE(entities.ErrInvalidID)
	}

	// first here we gonna get the proposal by its ID,
	asset, err := t.assetService.GetByID(ctx, req.AssetId)
	if err != nil {
		return nil, t.formatE(err)
	}

	if t.notaryService == nil {
		return nil, errors.New("sql notary store not available")
	}

	// then we get the signature and pack them altogether
	signatures, _, err := t.notaryService.GetByResourceID(ctx, req.AssetId, entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	var address string
	if asset.ERC20Contract != "" {
		address = asset.ERC20Contract
	} else {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid asset source"))
	}

	if len(address) <= 0 {
		return nil, apiError(codes.Internal, fmt.Errorf("invalid erc20 token contract address"))
	}

	nonce, err := num.UintFromHex("0x" + strings.TrimLeft(req.AssetId, "0"))
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &v2.GetERC20ListAssetBundleResponse{
		AssetSource: address,
		Nonce:       nonce.String(),
		VegaAssetId: asset.ID.String(),
		Signatures:  packNodeSignatures(signatures),
	}, nil
}

func (t *tradingDataServiceV2) GetERC20WithdrawalApproval(ctx context.Context, req *v2.GetERC20WithdrawalApprovalRequest) (*v2.GetERC20WithdrawalApprovalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20WithdrawalApprovalV2")()
	if len(req.WithdrawalId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrMissingWithdrawalID)
	}

	// get withdrawal first
	w, err := t.withdrawalService.GetByID(ctx, req.WithdrawalId)
	if err != nil {
		return nil, t.formatE(err)
	}

	// get the signatures from  notaryService
	signatures, _, err := t.notaryService.GetByResourceID(ctx, req.WithdrawalId, entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	// some assets stuff
	assets, err := t.assetService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	var address string
	for _, v := range assets {
		if v.ID == w.Asset {
			address = v.ERC20Contract
			break // found the one we want
		}
	}
	if len(address) <= 0 {
		return nil, apiError(codes.Internal, fmt.Errorf("invalid erc20 token contract address"))
	}

	return &v2.GetERC20WithdrawalApprovalResponse{
		AssetSource:   address,
		Amount:        fmt.Sprintf("%v", w.Amount),
		Nonce:         w.Ref,
		TargetAddress: w.Ext.GetErc20().ReceiverAddress,
		Signatures:    packNodeSignatures(signatures),
		// timestamps is unix nano, contract needs unix. So load if first, and cut nanos
		Creation: w.CreatedTimestamp.Unix(),
	}, nil
}

// Get latest Trade.
func (t *tradingDataServiceV2) GetLastTrade(ctx context.Context, req *v2.GetLastTradeRequest) (*v2.GetLastTradeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLastTradeV2")()

	if len(req.MarketId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}

	p := entities.OffsetPagination{
		Skip:       0,
		Limit:      1,
		Descending: true,
	}

	trades, err := t.tradeService.GetByMarket(ctx, req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}

	protoTrades := tradesToProto(trades)

	if len(protoTrades) > 0 && protoTrades[0] != nil {
		return &v2.GetLastTradeResponse{Trade: protoTrades[0]}, nil
	}
	// No trades found on the market yet (and no errors)
	// this can happen at the beginning of a new market
	return &v2.GetLastTradeResponse{}, nil
}

func tradesToProto(trades []entities.Trade) []*vega.Trade {
	protoTrades := make([]*vega.Trade, len(trades))
	for i := range trades {
		protoTrades[i] = trades[i].ToProto()
	}
	return protoTrades
}

// Get trades by using a cursor based pagination model.
func (t *tradingDataServiceV2) ListTrades(ctx context.Context, in *v2.ListTradesRequest) (*v2.ListTradesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTradesV2")()
	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}
	dateRange := entities.DateRangeFromProto(in.DateRange)
	trades, pageInfo, err := t.tradeService.List(ctx,
		entities.MarketID(in.GetMarketId()),
		entities.PartyID(in.GetPartyId()),
		entities.OrderID(in.GetOrderId()),
		pagination,
		dateRange)
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

/****************************** Markets **************************************/

// GetMarket provides the given market.
func (t *tradingDataServiceV2) GetMarket(ctx context.Context, req *v2.GetMarketRequest) (*v2.GetMarketResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketByID_SQL")()

	if len(req.MarketId) == 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}

	market, err := t.marketService.GetByID(ctx, req.MarketId)
	if err != nil {
		// Show a relevant error here -> no such market exists.
		return nil, t.formatE(err)
	}

	return &v2.GetMarketResponse{
		Market: market.ToProto(),
	}, nil
}

// List all markets using a cursor based pagination model.
func (t *tradingDataServiceV2) ListMarkets(ctx context.Context, in *v2.ListMarketsRequest) (*v2.ListMarketsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListMarketsV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	includeSettled := true
	if in.IncludeSettled != nil {
		includeSettled = *in.IncludeSettled
	}

	markets, pageInfo, err := t.marketsService.GetAllPaged(ctx, "", pagination, includeSettled)
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

// List all Positions using a cursor based pagination model.
//
// Deprecated: Use ListAllPositions instead.
func (t *tradingDataServiceV2) ListPositions(ctx context.Context, in *v2.ListPositionsRequest) (*v2.ListPositionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListPositionsV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	parties := []entities.PartyID{entities.PartyID(in.PartyId)}
	markets := []entities.MarketID{entities.MarketID(in.MarketId)}

	positions, pageInfo, err := t.positionService.GetByPartyConnection(ctx, parties, markets, pagination)
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

func (t *tradingDataServiceV2) ListAllPositions(ctx context.Context, req *v2.ListAllPositionsRequest) (*v2.ListAllPositionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAllPositions")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	var parties []entities.PartyID
	var markets []entities.MarketID

	if req.Filter != nil {
		parties = make([]entities.PartyID, len(req.Filter.PartyIds))
		markets = make([]entities.MarketID, len(req.Filter.MarketIds))

		for i, party := range req.Filter.PartyIds {
			parties[i] = entities.PartyID(party)
		}

		for i, market := range req.Filter.MarketIds {
			markets[i] = entities.MarketID(market)
		}
	}

	positions, pageInfo, err := t.positionService.GetByPartyConnection(ctx, parties, markets, pagination)
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

	resp := &v2.ListAllPositionsResponse{
		Positions: PositionsConnection,
	}

	return resp, nil
}

// Subscribe to a stream of Positions.
func (t *tradingDataServiceV2) ObservePositions(req *v2.ObservePositionsRequest, srv v2.TradingDataService_ObservePositionsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	t.sendPositionsSnapshot(ctx, req, srv)

	var partyID, marketID string
	if req.PartyId != nil {
		partyID = *req.PartyId
	}

	if req.MarketId != nil {
		marketID = *req.MarketId
	}

	positionsChan, ref := t.positionService.Observe(ctx, t.config.StreamRetries, partyID, marketID)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Positions subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Position", positionsChan, ref, func(positions []entities.Position) error {
		protos := make([]*vega.Position, len(positions))
		for i := 0; i < len(positions); i++ {
			protos[i] = positions[i].ToProto()
		}
		updates := &v2.PositionUpdates{Positions: protos}
		responseUpdates := &v2.ObservePositionsResponse_Updates{Updates: updates}
		response := &v2.ObservePositionsResponse{Response: responseUpdates}
		return srv.Send(response)
	})
}

func (t *tradingDataServiceV2) sendPositionsSnapshot(ctx context.Context, req *v2.ObservePositionsRequest, srv v2.TradingDataService_ObservePositionsServer) error {
	var positions []entities.Position
	var err error

	// By market and party
	if req.PartyId != nil && req.MarketId != nil {
		position, err := t.positionService.GetByMarketAndParty(ctx, *req.MarketId, *req.PartyId)
		if err != nil {
			return errors.Wrap(err, "getting initial positions by market+party")
		}
		positions = append(positions, position)
	}

	// By market
	if req.PartyId == nil && req.MarketId != nil {
		positions, err = t.positionService.GetByMarket(ctx, *req.MarketId)
		if err != nil {
			return errors.Wrap(err, "getting initial positions by market")
		}
	}

	// By party
	if req.PartyId != nil && req.MarketId == nil {
		positions, err = t.positionService.GetByParty(ctx, entities.PartyID(*req.PartyId))
		if err != nil {
			return errors.Wrap(err, "getting initial positions by party")
		}
	}

	// All the positions
	if req.PartyId == nil && req.MarketId == nil {
		positions, err = t.positionService.GetAll(ctx)
		if err != nil {
			return errors.Wrap(err, "getting initial positions by party")
		}
	}

	protos := make([]*vega.Position, len(positions))
	for i := 0; i < len(positions); i++ {
		protos[i] = positions[i].ToProto()
	}

	batches := batch(protos, snapshotPageSize)
	for i, batch := range batches {
		isLast := i == len(batches)-1
		positionList := &v2.PositionSnapshotPage{Positions: batch, LastPage: isLast}
		snapshot := &v2.ObservePositionsResponse_Snapshot{Snapshot: positionList}
		response := &v2.ObservePositionsResponse{Response: snapshot}
		if err := srv.Send(response); err != nil {
			return err
		}
	}
	return nil
}

func (t *tradingDataServiceV2) GetParty(ctx context.Context, req *v2.GetPartyRequest) (*v2.GetPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetParty")()

	party, err := t.partyService.GetByID(ctx, req.PartyId)
	if err != nil {
		return nil, t.formatE(err)
	}

	return &v2.GetPartyResponse{
		Party: party.ToProto(),
	}, nil
}

// List Parties using a cursor based pagination model.
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
		Parties: partyConnection,
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

	edges, err := makeEdges[*v2.MarginEdge](marginLevels, ctx, t.accountService)
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

// Subscribe to a stream of Margin Levels.
func (t *tradingDataServiceV2) ObserveMarginLevels(req *v2.ObserveMarginLevelsRequest, srv v2.TradingDataService_ObserveMarginLevelsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var marketID string
	if req.MarketId != nil {
		marketID = *req.MarketId
	}

	marginLevelsChan, ref := t.riskService.ObserveMarginLevels(ctx, t.config.StreamRetries, req.PartyId, marketID)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Margin levels subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "MarginLevel", marginLevelsChan, ref, func(ml entities.MarginLevels) error {
		protoMl, err := ml.ToProto(ctx, t.accountService)
		if err != nil {
			return apiError(codes.Internal, err)
		}

		return srv.Send(&v2.ObserveMarginLevelsResponse{
			MarginLevels: protoMl,
		})
	})
}

// List rewards.
func (t *tradingDataServiceV2) ListRewards(ctx context.Context, in *v2.ListRewardsRequest) (*v2.ListRewardsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListRewardsV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	rewards, pageInfo, err := t.rewardService.GetByCursor(ctx, &in.PartyId, in.AssetId, in.FromEpoch, in.ToEpoch, pagination)
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

// Get reward summaries.
func (t *tradingDataServiceV2) ListRewardSummaries(ctx context.Context, in *v2.ListRewardSummariesRequest) (*v2.ListRewardSummariesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListRewardSummariesV2")()

	summaries, err := t.rewardService.GetSummaries(ctx, in.PartyId, in.AssetId)
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

// Get reward summaries for epoch range.
func (t *tradingDataServiceV2) ListEpochRewardSummaries(ctx context.Context, in *v2.ListEpochRewardSummariesRequest) (*v2.ListEpochRewardSummariesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListEpochRewardSummaries")()

	if in == nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("empty request"))
	}

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	summaries, pageInfo, err := t.rewardService.GetEpochRewardSummaries(ctx, in.FromEpoch, in.ToEpoch, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.EpochRewardSummaryEdge](summaries)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := v2.EpochRewardSummaryConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListEpochRewardSummariesResponse{
		Summaries: &connection,
	}, nil
}

// subscribe to rewards.
func (t *tradingDataServiceV2) ObserveRewards(req *v2.ObserveRewardsRequest, srv v2.TradingDataService_ObserveRewardsServer) error {
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming reward updates")
	}
	var assetID, partyID string
	if req.AssetId != nil {
		assetID = *req.AssetId
	}

	if req.PartyId != nil {
		partyID = *req.PartyId
	}
	ch, ref := t.rewardService.Observe(ctx, t.config.StreamRetries, assetID, partyID)

	return observe(ctx, t.log, "Reward", ch, ref, func(reward entities.Reward) error {
		return srv.Send(&v2.ObserveRewardsResponse{
			Reward: reward.ToProto(),
		})
	})
}

// -- Deposits --.
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

	dateRange := entities.DateRangeFromProto(req.DateRange)

	deposits, pageInfo, err := t.depositService.GetByParty(ctx, req.PartyId, false, pagination, dateRange)
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

// -- Withdrawals --.
func (t *tradingDataServiceV2) GetWithdrawal(ctx context.Context, req *v2.GetWithdrawalRequest) (*v2.GetWithdrawalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetWithdrawalV2")()

	if req == nil || req.Id == "" {
		return nil, apiError(codes.InvalidArgument, ErrMissingWithdrawalID)
	}
	withdrawal, err := t.withdrawalService.GetByID(ctx, req.Id)
	if err != nil {
		return nil, t.formatE(err)
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
	dateRange := entities.DateRangeFromProto(req.DateRange)
	withdrawals, pageInfo, err := t.withdrawalService.GetByParty(ctx, req.PartyId, false, pagination, dateRange)
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

// -- Assets --.
func (t *tradingDataServiceV2) GetAsset(ctx context.Context, req *v2.GetAssetRequest) (*v2.GetAssetResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetAssetV2")()
	if req == nil || req.AssetId == "" {
		return nil, apiError(codes.InvalidArgument, ErrMissingAssetID)
	}

	asset, err := t.assetService.GetByID(ctx, req.AssetId)
	if err != nil {
		return nil, t.formatE(err)
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

	if req.AssetId != nil && *req.AssetId != "" {
		return t.getSingleAsset(ctx, *req.AssetId)
	}

	return t.getAllAssets(ctx, req.Pagination)
}

func (t *tradingDataServiceV2) getSingleAsset(ctx context.Context, assetID string) (*v2.ListAssetsResponse, error) {
	asset, err := t.assetService.GetByID(ctx, assetID)
	if err != nil {
		return nil, t.formatE(err)
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
		return nil, apiError(codes.InvalidArgument, ErrMissingOracleSpecID)
	}

	spec, err := t.oracleSpecService.GetSpecByID(ctx, req.OracleSpecId)
	if err != nil {
		return nil, t.formatE(err)
	}

	return &v2.GetOracleSpecResponse{
		OracleSpec: &vega.OracleSpec{
			ExternalDataSourceSpec: &vega.ExternalDataSourceSpec{
				Spec: spec.ToProto().ExternalDataSourceSpec.Spec,
			},
		},
	}, nil
}

func (t *tradingDataServiceV2) ListOracleSpecs(ctx context.Context, req *v2.ListOracleSpecsRequest) (*v2.ListOracleSpecsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOracleSpecsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
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
		return nil, apiError(codes.InvalidArgument, err)
	}

	var data []entities.OracleData
	var pageInfo entities.PageInfo

	if req != nil && req.OracleSpecId != nil && *req.OracleSpecId != "" {
		data, pageInfo, err = t.oracleDataService.GetOracleDataBySpecID(ctx, *req.OracleSpecId, pagination)
	} else {
		data, pageInfo, err = t.oracleDataService.ListOracleData(ctx, pagination)
	}

	if err != nil {
		return nil, apiError(codes.Internal, fmt.Errorf("could not retrieve data for OracleSpecID: %s %w", *req.OracleSpecId, err))
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

	var partyID entities.PartyID
	var marketID entities.MarketID
	var reference string

	if req.PartyId != nil {
		partyID = entities.PartyID(*req.PartyId)
	}

	if req.MarketId != nil {
		marketID = entities.MarketID(*req.MarketId)
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

func (t *tradingDataServiceV2) ObserveLiquidityProvisions(request *v2.ObserveLiquidityProvisionsRequest, srv v2.TradingDataService_ObserveLiquidityProvisionsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	lpCh, ref := t.liquidityProvisionService.ObserveLiquidityProvisions(ctx, t.config.StreamRetries, request.PartyId, request.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Order", lpCh, ref, func(lps []entities.LiquidityProvision) error {
		protos := make([]*vega.LiquidityProvision, 0, len(lps))
		for _, v := range lps {
			protos = append(protos, v.ToProto())
		}
		response := &v2.ObserveLiquidityProvisionsResponse{LiquidityProvisions: protos}
		return srv.Send(response)
	})
}

func (t *tradingDataServiceV2) GetGovernanceData(ctx context.Context, req *v2.GetGovernanceDataRequest) (*v2.GetGovernanceDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetGovernanceData")

	var (
		proposal entities.Proposal
		err      error
	)
	if req.ProposalId != nil {
		proposal, err = t.governanceService.GetProposalByID(ctx, *req.ProposalId)
	} else if req.Reference != nil {
		proposal, err = t.governanceService.GetProposalByReference(ctx, *req.Reference)
	} else {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("proposal id or reference required"))
	}

	if err != nil {
		return nil, t.formatE(err)
	}

	gd, err := t.proposalToGovernanceData(ctx, proposal)
	if err != nil {
		return nil, apiError(codes.Internal, ErrNotMapped, err)
	}

	return &v2.GetGovernanceDataResponse{Data: gd}, nil
}

func (t *tradingDataServiceV2) ListGovernanceData(ctx context.Context, req *v2.ListGovernanceDataRequest) (*v2.ListGovernanceDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListGovernanceDataV2")()

	var state *entities.ProposalState
	if req.ProposalState != nil {
		state = ptr.From(entities.ProposalState(*req.ProposalState))
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	proposals, pageInfo, err := t.governanceService.GetProposals(
		ctx,
		state,
		req.ProposerPartyId,
		(*entities.ProposalType)(req.ProposalType),
		pagination,
	)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.GovernanceDataEdge](proposals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	for i := range edges {
		edges[i].Node.Yes, edges[i].Node.No, err = t.getVotesByProposal(ctx, edges[i].Node.Proposal.Id)
		if err != nil {
			return nil, apiError(codes.Internal, err)
		}
	}

	proposalsConnection := &v2.GovernanceDataConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListGovernanceDataResponse{Connection: proposalsConnection}, nil
}

func (t *tradingDataServiceV2) getVotesByProposal(ctx context.Context, proposalID string) (yesVotes, noVotes []*vega.Vote, err error) {
	votes, err := t.governanceService.GetVotes(ctx, &proposalID, nil, nil)
	if err != nil {
		return nil, nil, apiError(codes.Internal, err)
	}

	for _, vote := range votes {
		switch vote.Value {
		case entities.VoteValueYes:
			yesVotes = append(yesVotes, vote.ToProto())
		case entities.VoteValueNo:
			noVotes = append(noVotes, vote.ToProto())
		}
	}
	return
}

// Get all Votes using a cursor based pagination model.
func (t *tradingDataServiceV2) ListVotes(ctx context.Context, in *v2.ListVotesRequest) (*v2.ListVotesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListVotesV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	if in.PartyId == nil && in.ProposalId == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("missing party or proposal id"))
	}

	votes, pageInfo, err := t.governanceService.GetConnection(ctx, in.ProposalId, in.PartyId, pagination)
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
			transfers, pageInfo, err = t.transfersService.GetTransfersFromParty(ctx, entities.PartyID(*req.Pubkey), pagination)
		case v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO:
			transfers, pageInfo, err = t.transfersService.GetTransfersToParty(ctx, entities.PartyID(*req.Pubkey), pagination)
		case v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO_OR_FROM:
			transfers, pageInfo, err = t.transfersService.GetTransfersToOrFromParty(ctx, entities.PartyID(*req.Pubkey), pagination)
		default:
			return nil, apiError(codes.InvalidArgument, fmt.Errorf("transfer direction not supported:%v", req.Direction))
		}
	}

	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.TransferEdge](transfers, ctx, t.accountService)
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
		return nil, t.formatE(err)
	}

	return &v2.GetOrderResponse{Order: order.ToProto()}, nil
}

func (t *tradingDataServiceV2) formatE(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, entities.ErrNotFound):
		return apiError(codes.NotFound, err)
	case errors.Is(err, entities.ErrInvalidID):
		return apiError(codes.InvalidArgument, err)
	default:
		// could handle more errors like context cancelled,
		// deadling exceeded, but let's see later
		return apiError(codes.Internal, err)
	}
}

func (t *tradingDataServiceV2) ListOrders(ctx context.Context, in *v2.ListOrdersRequest) (*v2.ListOrdersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOrdersV2")()

	pagination, err := entities.CursorPaginationFromProto(in.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	liveOnly := false
	if in.LiveOnly != nil {
		liveOnly = *in.LiveOnly
	}
	dateRange := entities.DateRangeFromProto(in.DateRange)
	var filter entities.OrderFilter

	if in.Filter != nil {
		filter = entities.OrderFilter{
			Statuses:         in.Filter.Statuses,
			Types:            in.Filter.Types,
			TimeInForces:     in.Filter.TimeInForces,
			ExcludeLiquidity: in.Filter.ExcludeLiquidity,
		}
	}

	orders, pageInfo, err := t.orderService.ListOrders(ctx, in.PartyId, in.MarketId, in.Reference, liveOnly,
		pagination, dateRange, filter)
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

// Subscribe to a stream of Orders.
func (t *tradingDataServiceV2) ObserveOrders(req *v2.ObserveOrdersRequest, srv v2.TradingDataService_ObserveOrdersServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	excludeLiquidity := false
	if req.ExcludeLiquidity != nil {
		excludeLiquidity = *req.ExcludeLiquidity
	}

	if err := t.sendOrdersSnapshot(ctx, req, srv); err != nil {
		return t.formatE(err)
	}
	ordersChan, ref := t.orderService.ObserveOrders(ctx, t.config.StreamRetries, req.MarketId, req.PartyId, excludeLiquidity)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Order", ordersChan, ref, func(orders []entities.Order) error {
		protos := make([]*vega.Order, 0, len(orders))
		for _, v := range orders {
			protos = append(protos, v.ToProto())
		}
		updates := &v2.OrderUpdates{Orders: protos}
		responseUpdates := &v2.ObserveOrdersResponse_Updates{Updates: updates}
		response := &v2.ObserveOrdersResponse{Response: responseUpdates}
		return srv.Send(response)
	})
}

func (t *tradingDataServiceV2) sendOrdersSnapshot(ctx context.Context, req *v2.ObserveOrdersRequest, srv v2.TradingDataService_ObserveOrdersServer) error {
	orders, pageInfo, err := t.orderService.ListOrders(ctx, req.PartyId, req.MarketId, nil, true, entities.CursorPagination{},
		entities.DateRange{}, entities.OrderFilter{})
	if err != nil {
		return errors.Wrap(err, "fetching orders initial image")
	}

	if pageInfo.HasNextPage {
		return fmt.Errorf("orders initial image spans multiple pages")
	}

	protos := make([]*vega.Order, len(orders))
	for i := 0; i < len(orders); i++ {
		protos[i] = orders[i].ToProto()
	}

	batches := batch(protos, snapshotPageSize)
	for i, batch := range batches {
		isLast := i == len(batches)-1
		positionList := &v2.OrderSnapshotPage{Orders: batch, LastPage: isLast}
		responseSnapshot := &v2.ObserveOrdersResponse_Snapshot{Snapshot: positionList}
		response := &v2.ObserveOrdersResponse{Response: responseSnapshot}
		if err := srv.Send(response); err != nil {
			return errors.Wrap(err, "sending account balance initial image")
		}
	}
	return nil
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

// subscribe to delegation events.
func (t *tradingDataServiceV2) ObserveDelegations(req *v2.ObserveDelegationsRequest, srv v2.TradingDataService_ObserveDelegationsServer) error {
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming delegation updates")
	}

	var partyID, nodeID string

	if req.PartyId != nil {
		partyID = *req.PartyId
	}

	if req.NodeId != nil {
		nodeID = *req.NodeId
	}

	ch, ref := t.delegationService.Observe(ctx, t.config.StreamRetries, partyID, nodeID)

	return observe(ctx, t.log, "Delegations", ch, ref, func(delegation entities.Delegation) error {
		return srv.Send(&v2.ObserveDelegationsResponse{
			Delegation: delegation.ToProto(),
		})
	})
}

func (t *tradingDataServiceV2) marketExistsForID(ctx context.Context, marketID string) bool {
	_, err := t.marketsService.GetByID(ctx, marketID)
	return err == nil
}

// GetNetworkData retrieve network data regarding the nodes of the network.
func (t *tradingDataServiceV2) GetNetworkData(ctx context.Context, _ *v2.GetNetworkDataRequest) (*v2.GetNetworkDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkDataV2")()

	epoch, err := t.epochService.GetCurrent(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	// get the node-y bits
	networkData, err := t.nodeService.GetNodeData(ctx, uint64(epoch.ID))
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	data := networkData.ToProto()

	// now use network parameters to calculate the maximum nodes allowed in each nodeSet
	np, err := t.networkParameterService.GetByKey(ctx, "network.validators.tendermint.number")
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	maxTendermint, err := strconv.ParseUint(np.Value, 10, 32)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	np, err = t.networkParameterService.GetByKey(ctx, "network.validators.ersatz.multipleOfTendermintValidators")
	if err != nil {
		return nil, t.formatE(err)
	}

	ersatzFactor, err := strconv.ParseFloat(np.Value, 32)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	data.TendermintNodes.Maximum = ptr.From(uint32(maxTendermint))
	data.ErsatzNodes.Maximum = ptr.From(uint32(float64(maxTendermint) * ersatzFactor))

	// we're done
	return &v2.GetNetworkDataResponse{
		NodeData: data,
	}, nil
}

// GetNode retrieves information about a given node.
func (t *tradingDataServiceV2) GetNode(ctx context.Context, req *v2.GetNodeRequest) (*v2.GetNodeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodeV2")()

	if req.GetId() == "" {
		return nil, apiError(codes.InvalidArgument, ErrMissingNodeID)
	}

	epoch, err := t.epochService.GetCurrent(ctx)
	if err != nil {
		return nil, t.formatE(err)
	}

	node, err := t.nodeService.GetNodeByID(ctx, req.GetId(), uint64(epoch.ID))
	if err != nil {
		return nil, t.formatE(err)
	}

	return &v2.GetNodeResponse{
		Node: node.ToProto(),
	}, nil
}

// ListNodes returns information about the nodes on the network.
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
		return nil, t.formatE(err)
	}

	pagination, err = entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("invalid pagination: %w", err))
	}

	nodes, pageInfo, err := t.nodeService.GetNodes(ctx, uint64(epoch.ID), pagination)
	if err != nil {
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

func (t *tradingDataServiceV2) ListNodeSignatures(ctx context.Context, req *v2.ListNodeSignaturesRequest) (*v2.ListNodeSignaturesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListNodeSignatures")()
	if req == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("request is nil"))
	}

	if len(req.Id) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing ID"))
	}

	var pagination entities.CursorPagination
	var err error

	if req != nil {
		pagination, err = entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, err)
		}
	}

	sigs, pageInfo, err := t.notaryService.GetByResourceID(ctx, req.Id, pagination)
	if err != nil {
		return nil, t.formatE(err)
	}

	edges, err := makeEdges[*v2.NodeSignatureEdge](sigs)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	nodeSignatureConnection := &v2.NodeSignaturesConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := &v2.ListNodeSignaturesResponse{
		Signatures: nodeSignatureConnection,
	}

	return resp, nil
}

// GetEpoch retrieves data for a specific epoch, if id omitted it gets the current epoch.
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
		return nil, t.formatE(err)
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

	if req == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("nil request"))
	}

	if len(req.MarketId) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing market id"))
	}

	if len(req.Price) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing price"))
	}

	fee, err := t.estimateFee(ctx, req.MarketId, req.Price, req.Size)
	if err != nil {
		return nil, err
	}

	return &v2.EstimateFeeResponse{
		Fee: fee,
	}, nil
}

func (t *tradingDataServiceV2) scaleFromMarketToAssetPrice(
	ctx context.Context,
	mkt entities.Market,
	price *num.Uint,
) (*num.Uint, error) {
	assetID, err := mkt.ToProto().GetAsset()
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	asset, err := t.assetService.GetByID(ctx, assetID)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	// scale the price if needed
	// price is expected in market decimal
	if exp := asset.Decimals - mkt.DecimalPlaces; exp != 0 {
		priceFactor := num.NewUint(1)
		priceFactor.Exp(num.NewUint(10), num.NewUint(uint64(exp)))
		price.Mul(price, priceFactor)
	}

	return price, nil
}

func (t *tradingDataServiceV2) estimateFee(
	ctx context.Context,
	market, priceS string,
	size uint64,
) (*vega.Fee, error) {
	mkt, err := t.marketService.GetByID(ctx, market)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	price, overflowed := num.UintFromString(priceS, 10)
	if overflowed {
		return nil, apiError(codes.InvalidArgument, errors.New("invalid order price"))
	}

	price, err = t.scaleFromMarketToAssetPrice(ctx, mkt, price)
	if err != nil {
		return nil, err
	}

	mdpd := num.DecimalFromFloat(10).
		Pow(num.DecimalFromInt64(int64(mkt.PositionDecimalPlaces)))

	base := num.DecimalFromUint(price.Mul(price, num.NewUint(size))).Div(mdpd)
	maker, infra, liquidity, err := t.feeFactors(mkt)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &vega.Fee{
		MakerFee:          base.Mul(num.NewDecimalFromFloat(maker)).Round(0).String(),
		InfrastructureFee: base.Mul(num.NewDecimalFromFloat(infra)).Round(0).String(),
		LiquidityFee:      base.Mul(num.NewDecimalFromFloat(liquidity)).Round(0).String(),
	}, nil
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

	margin, err := t.estimateMargin(
		ctx, req.Side, req.Type, req.MarketId, req.PartyId, req.Price, req.Size)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &v2.EstimateMarginResponse{
		MarginLevels: margin,
	}, nil
}

func (t *tradingDataServiceV2) estimateMargin(
	ctx context.Context,
	rSide vega.Side,
	rType vega.Order_Type,
	rMarket, rParty, rPrice string,
	rSize uint64,
) (*vega.MarginLevels, error) {
	if rSide == vega.Side_SIDE_UNSPECIFIED {
		return nil, ErrInvalidOrderSide
	}

	// first get the risk factors and market data (marketdata->markprice)
	rf, err := t.riskFactorService.GetMarketRiskFactors(ctx, rMarket)
	if err != nil {
		return nil, err
	}
	mkt, err := t.marketService.GetByID(ctx, rMarket)
	if err != nil {
		return nil, err
	}

	mktProto := mkt.ToProto()

	mktData, err := t.marketDataService.GetMarketDataByID(ctx, rMarket)
	if err != nil {
		return nil, err
	}

	f, err := num.DecimalFromString(rf.Short.String())
	if err != nil {
		return nil, err
	}
	if rSide == vega.Side_SIDE_BUY {
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
	priceD, _ := num.DecimalFromString(mktData.MarkPrice.String())

	// if the order is a limit order, use the limit price to calculate the margin maintenance
	if rType == vega.Order_TYPE_LIMIT {
		priceD, _ = num.DecimalFromString(rPrice)
	}

	price, _ := num.UintFromDecimal(priceD)
	price, err = t.scaleFromMarketToAssetPrice(ctx, mkt, price)
	if err != nil {
		return nil, err
	}

	priceD = price.ToDecimal()

	mdpd := num.DecimalFromFloat(10).
		Pow(num.DecimalFromInt64(int64(mkt.PositionDecimalPlaces)))

	maintenanceMargin := num.DecimalFromFloat(float64(rSize)).
		Mul(f).Mul(priceD).Div(mdpd)
	// now we use the risk factors
	return &vega.MarginLevels{
		PartyId:                rParty,
		MarketId:               mktProto.GetId(),
		Asset:                  asset,
		Timestamp:              0,
		MaintenanceMargin:      maintenanceMargin.Round(0).String(),
		SearchLevel:            maintenanceMargin.Mul(num.DecimalFromFloat(mkt.TradableInstrument.MarginCalculator.ScalingFactors.SearchLevel)).Round(0).String(),
		InitialMargin:          maintenanceMargin.Mul(num.DecimalFromFloat(mkt.TradableInstrument.MarginCalculator.ScalingFactors.InitialMargin)).Round(0).String(),
		CollateralReleaseLevel: maintenanceMargin.Mul(num.DecimalFromFloat(mkt.TradableInstrument.MarginCalculator.ScalingFactors.CollateralRelease)).Round(0).String(),
	}, nil
}

func (t *tradingDataServiceV2) ListNetworkParameters(ctx context.Context, req *v2.ListNetworkParametersRequest) (*v2.ListNetworkParametersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListNetworkParametersV2")()

	var pagination entities.CursorPagination
	var err error
	if req.Pagination != nil {
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

	return &v2.ListNetworkParametersResponse{
		NetworkParameters: networkParametersConnection,
	}, nil
}

func (t *tradingDataServiceV2) GetNetworkParameter(ctx context.Context, req *v2.GetNetworkParameterRequest) (*v2.GetNetworkParameterResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkParameter")()
	nps, _, err := t.networkParameterService.GetAll(ctx, entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	var np *vega.NetworkParameter
	for _, v := range nps {
		if req.Key == v.Key {
			np = v.ToProto()
			break
		}
	}

	return &v2.GetNetworkParameterResponse{
		NetworkParameter: np,
	}, nil
}

func (t *tradingDataServiceV2) ListCheckpoints(ctx context.Context, req *v2.ListCheckpointsRequest) (*v2.ListCheckpointsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("NetworkParametersV2")()
	var pagination entities.CursorPagination
	var err error
	if req.Pagination != nil {
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
	if req == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("nil request"))
	}

	if len(req.PartyId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrMissingPartyID)
	}

	var pagination entities.CursorPagination

	partyID := entities.PartyID(req.PartyId)

	if req.Pagination != nil {
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
		return nil, t.formatE(err)
	}

	return &v2.GetRiskFactorsResponse{
		RiskFactor: rfs.ToProto(),
	}, nil
}

func (t *tradingDataServiceV2) ObserveGovernance(req *v2.ObserveGovernanceRequest, stream v2.TradingDataService_ObserveGovernanceServer) error {
	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming governance updates")
	}
	ch, ref := t.governanceService.ObserveProposals(ctx, t.config.StreamRetries, req.PartyId)

	return observe(ctx, t.log, "Governance", ch, ref, func(proposal entities.Proposal) error {
		gd, err := t.proposalToGovernanceData(ctx, proposal)
		if err != nil {
			return t.formatE(err)
		}
		return stream.Send(&v2.ObserveGovernanceResponse{
			Data: gd,
		})
	})
}

func (t *tradingDataServiceV2) proposalToGovernanceData(ctx context.Context, proposal entities.Proposal) (*vega.GovernanceData, error) {
	yesVotes, err := t.governanceService.GetYesVotesForProposal(ctx, proposal.ID.String())
	if err != nil {
		return nil, err
	}
	protoYesVotes := voteListToProto(yesVotes)

	noVotes, err := t.governanceService.GetNoVotesForProposal(ctx, proposal.ID.String())
	if err != nil {
		return nil, err
	}
	protoNoVotes := voteListToProto(noVotes)

	gd := vega.GovernanceData{
		Proposal: proposal.ToProto(),
		Yes:      protoYesVotes,
		No:       protoNoVotes,
	}
	return &gd, nil
}

func voteListToProto(votes []entities.Vote) []*vega.Vote {
	protoVotes := make([]*vega.Vote, len(votes))
	for j, vote := range votes {
		protoVotes[j] = vote.ToProto()
	}
	return protoVotes
}

func (t *tradingDataServiceV2) ObserveVotes(req *v2.ObserveVotesRequest, stream v2.TradingDataService_ObserveVotesServer) error {
	if req.PartyId != nil && *req.PartyId != "" {
		return t.observePartyVotes(*req.PartyId, stream)
	}

	if req.ProposalId != nil && *req.ProposalId != "" {
		return t.observeProposalVotes(*req.ProposalId, stream)
	}

	return apiError(codes.InvalidArgument, errors.New("party id or proposal id required"))
}

func (t *tradingDataServiceV2) observePartyVotes(partyID string, stream v2.TradingDataService_ObserveVotesServer) error {
	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming party votes")
	}
	ch, ref := t.governanceService.ObservePartyVotes(ctx, t.config.StreamRetries, partyID)

	return observe(ctx, t.log, "PartyVote", ch, ref, func(vote entities.Vote) error {
		return stream.Send(&v2.ObserveVotesResponse{
			Vote: vote.ToProto(),
		})
	})
}

func (t *tradingDataServiceV2) observeProposalVotes(proposalID string, stream v2.TradingDataService_ObserveVotesServer) error {
	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming proposal votes")
	}
	ch, ref := t.governanceService.ObserveProposalVotes(ctx, t.config.StreamRetries, proposalID)

	return observe(ctx, t.log, "ProposalVote", ch, ref, func(p entities.Vote) error {
		return stream.Send(&v2.ObserveVotesResponse{
			Vote: p.ToProto(),
		})
	})
}

func (t *tradingDataServiceV2) GetProtocolUpgradeStatus(context.Context, *v2.GetProtocolUpgradeStatusRequest) (*v2.GetProtocolUpgradeStatusResponse, error) {
	ready := t.protocolUpgradeService.GetProtocolUpgradeStarted()
	return &v2.GetProtocolUpgradeStatusResponse{
		Ready: ready,
	}, nil
}

func (t *tradingDataServiceV2) ListProtocolUpgradeProposals(ctx context.Context, req *v2.ListProtocolUpgradeProposalsRequest) (*v2.ListProtocolUpgradeProposalsResponse, error) {
	if req == nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("empty request"))
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	var status *entities.ProtocolUpgradeProposalStatus
	if req.Status != nil {
		status = ptr.From(entities.ProtocolUpgradeProposalStatus(*req.Status))
	}

	pups, pageInfo, err := t.protocolUpgradeService.ListProposals(
		ctx,
		status,
		req.ApprovedBy,
		pagination,
	)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.ProtocolUpgradeProposalEdge](pups)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := v2.ProtocolUpgradeProposalConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListProtocolUpgradeProposalsResponse{
		ProtocolUpgradeProposals: &connection,
	}, nil
}

func (t *tradingDataServiceV2) ListCoreSnapshots(ctx context.Context, req *v2.ListCoreSnapshotsRequest) (*v2.ListCoreSnapshotsResponse, error) {
	if req == nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("empty request"))
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	snaps, pageInfo, err := t.coreSnapshotService.ListSnapshots(ctx, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.CoreSnapshotEdge](snaps)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := v2.CoreSnapshotConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListCoreSnapshotsResponse{
		CoreSnapshots: &connection,
	}, nil
}

type tradingDataEventBusServerV2 struct {
	stream v2.TradingDataService_ObserveEventBusServer
}

func (t tradingDataEventBusServerV2) RecvMsg(m interface{}) error {
	return t.stream.RecvMsg(m)
}

func (t tradingDataEventBusServerV2) Context() context.Context {
	return t.stream.Context()
}

func (t tradingDataEventBusServerV2) Send(data []*eventspb.BusEvent) error {
	resp := &v2.ObserveEventBusResponse{
		Events: data,
	}
	return t.stream.Send(resp)
}

func (t *tradingDataServiceV2) ObserveEventBus(stream v2.TradingDataService_ObserveEventBusServer) error {
	server := tradingDataEventBusServerV2{stream}
	eventService := t.eventService

	return observeEventBus(t.log, t.config, server, eventService)
}

// Subscribe to a stream of Transfer Responses.
func (t *tradingDataServiceV2) ObserveLedgerMovements(_ *v2.ObserveLedgerMovementsRequest, srv v2.TradingDataService_ObserveLedgerMovementsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan in error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	transferResponsesChan, ref := t.ledgerService.Observe(ctx, t.config.StreamRetries)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("TransferResponses subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "TransferResponse", transferResponsesChan, ref, func(tr *vega.LedgerMovement) error {
		return srv.Send(&v2.ObserveLedgerMovementsResponse{
			LedgerMovement: tr,
		})
	})
}

// -- Key Rotations --.
func (t *tradingDataServiceV2) ListKeyRotations(ctx context.Context, req *v2.ListKeyRotationsRequest) (*v2.ListKeyRotationsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListKeyRotations")()
	var (
		pagination entities.CursorPagination
		err        error
	)

	if req != nil {
		pagination, err = entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, t.formatE(err)
		}
	}
	if req.NodeId == nil || *req.NodeId == "" {
		return t.getAllKeyRotations(ctx, pagination)
	}

	return t.getNodeKeyRotations(ctx, *req.NodeId, pagination)
}

func (t *tradingDataServiceV2) getAllKeyRotations(ctx context.Context, pagination entities.CursorPagination) (*v2.ListKeyRotationsResponse, error) {
	rotations, pageInfo, err := t.keyRotationService.GetAllPubKeyRotations(ctx, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	return makeKeyRotationResponse(rotations, pageInfo)
}

func (t *tradingDataServiceV2) getNodeKeyRotations(ctx context.Context, nodeID string, pagination entities.CursorPagination) (*v2.ListKeyRotationsResponse, error) {
	rotations, pageInfo, err := t.keyRotationService.GetPubKeyRotationsPerNode(ctx, nodeID, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	return makeKeyRotationResponse(rotations, pageInfo)
}

func makeKeyRotationResponse(rotations []entities.KeyRotation, pageInfo entities.PageInfo) (*v2.ListKeyRotationsResponse, error) {
	edges, err := makeEdges[*v2.KeyRotationEdge](rotations)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	keyRotationConnection := &v2.KeyRotationConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListKeyRotationsResponse{
		Rotations: keyRotationConnection,
	}, nil
}

// -- Ethereum Key Rotations --.
func (t *tradingDataServiceV2) ListEthereumKeyRotations(ctx context.Context, req *v2.ListEthereumKeyRotationsRequest) (*v2.ListEthereumKeyRotationsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListEthereumKeyRotationsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, err)
	}

	rotations, pageInfo, err := t.ethereumKeyRotationService.List(ctx, entities.NodeID(req.GetNodeId()), pagination)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	edges, err := makeEdges[*v2.EthereumKeyRotationEdge](rotations)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	connection := &v2.EthereumKeyRotationsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	resp := v2.ListEthereumKeyRotationsResponse{KeyRotations: connection}
	return &resp, nil
}

// Get Time.
func (t *tradingDataServiceV2) GetVegaTime(ctx context.Context, req *v2.GetVegaTimeRequest) (*v2.GetVegaTimeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVegaTimeV2")()
	b, err := t.blockService.GetLastBlock(ctx)
	if err != nil {
		return nil, t.formatE(err)
	}

	return &v2.GetVegaTimeResponse{
		Timestamp: b.VegaTime.UnixNano(),
	}, nil
}

// -- NetworkHistory --.

func (t *tradingDataServiceV2) GetMostRecentNetworkHistorySegment(context.Context, *v2.GetMostRecentNetworkHistorySegmentRequest) (*v2.GetMostRecentNetworkHistorySegmentResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMostRecentNetworkHistorySegment")()

	segment, err := t.networkHistoryService.GetHighestBlockHeightHistorySegment()
	if err != nil {
		if errors.Is(err, store.ErrSegmentNotFound) {
			return &v2.GetMostRecentNetworkHistorySegmentResponse{
				Segment: nil,
			}, nil
		}

		return nil, apiError(codes.Internal, ErrGetMostRecentHistorySegment, err)
	}

	return &v2.GetMostRecentNetworkHistorySegmentResponse{
		Segment:  toHistorySegment(segment),
		SwarmKey: t.networkHistoryService.GetSwarmKey(),
	}, nil
}

func (t *tradingDataServiceV2) ListAllNetworkHistorySegments(context.Context, *v2.ListAllNetworkHistorySegmentsRequest) (*v2.ListAllNetworkHistorySegmentsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAllNetworkHistorySegments")()
	if t.networkHistoryService == nil {
		return nil, apiError(codes.Internal, ErrNetworkHistoryNotEnabled, fmt.Errorf("network history is not enabled"))
	}
	segments, err := t.networkHistoryService.ListAllHistorySegments()
	if err != nil {
		return nil, apiError(codes.Internal, ErrListAllNetworkHistorySegment, err)
	}

	historySegments := make([]*v2.HistorySegment, 0, len(segments))
	for _, segment := range segments {
		historySegments = append(historySegments, toHistorySegment(segment))
	}

	return &v2.ListAllNetworkHistorySegmentsResponse{
		Segments: historySegments,
	}, nil
}

func (t *tradingDataServiceV2) Ping(context.Context, *v2.PingRequest) (*v2.PingResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Ping")()
	return &v2.PingResponse{}, nil
}

func toHistorySegment(segment networkhistory.Segment) *v2.HistorySegment {
	return &v2.HistorySegment{
		FromHeight:               segment.GetFromHeight(),
		ToHeight:                 segment.GetToHeight(),
		HistorySegmentId:         segment.GetHistorySegmentId(),
		PreviousHistorySegmentId: segment.GetPreviousHistorySegmentId(),
	}
}

func (t *tradingDataServiceV2) GetActiveNetworkHistoryPeerAddresses(_ context.Context, _ *v2.GetActiveNetworkHistoryPeerAddressesRequest) (*v2.GetActiveNetworkHistoryPeerAddressesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMostRecentHistorySegmentFromPeers")()
	if t.networkHistoryService == nil {
		return nil, apiError(codes.Internal, ErrNetworkHistoryNotEnabled, fmt.Errorf("network history is not enabled"))
	}
	addresses := t.networkHistoryService.GetActivePeerAddresses()

	return &v2.GetActiveNetworkHistoryPeerAddressesResponse{
		IpAddresses: addresses,
	}, nil
}

func batch[T any](in []T, batchSize int) [][]T {
	batches := make([][]T, 0, (len(in)+batchSize-1)/batchSize)
	for batchSize < len(in) {
		in, batches = in[batchSize:], append(batches, in[0:batchSize:batchSize])
	}
	batches = append(batches, in)
	return batches
}
