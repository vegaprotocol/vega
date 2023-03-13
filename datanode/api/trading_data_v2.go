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
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	v1 "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
var snapshotPageSize = 50

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

// ListAccounts lists accounts matching the request.
func (t *tradingDataServiceV2) ListAccounts(ctx context.Context, req *v2.ListAccountsRequest) (*v2.ListAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAccountsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	filter, err := entities.AccountFilterFromProto(req.Filter)
	if err != nil {
		return nil, formatE(ErrInvalidFilter, err)
	}

	accountBalances, pageInfo, err := t.accountService.QueryBalances(ctx, filter, pagination)
	if err != nil {
		return nil, formatE(ErrAccountServiceListAccounts, err)
	}

	edges, err := makeEdges[*v2.AccountEdge](accountBalances)
	if err != nil {
		return nil, formatE(err)
	}

	accountsConnection := &v2.AccountsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListAccountsResponse{
		Accounts: accountsConnection,
	}, nil
}

// ObserveAccounts streams account balances matching the request.
func (t *tradingDataServiceV2) ObserveAccounts(req *v2.ObserveAccountsRequest, srv v2.TradingDataService_ObserveAccountsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	// First get the 'initial image' of accounts matching the request and send those
	if err := t.sendAccountsSnapshot(ctx, req, srv); err != nil {
		return formatE(ErrFailedToSendSnapshot, err)
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
		batches := batch(protos, snapshotPageSize)

		for _, batch := range batches {
			updates := &v2.AccountUpdates{Accounts: batch}
			responseUpdates := &v2.ObserveAccountsResponse_Updates{Updates: updates}
			response := &v2.ObserveAccountsResponse{Response: responseUpdates}
			if err := srv.Send(response); err != nil {
				return errors.Wrap(err, "sending accounts updates")
			}
		}

		return nil
	})
}

func (t *tradingDataServiceV2) sendAccountsSnapshot(ctx context.Context, req *v2.ObserveAccountsRequest, srv v2.TradingDataService_ObserveAccountsServer) error {
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
		return errors.New("initial image spans multiple pages")
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

// Info returns the version and commit hash of the trading data service.
func (t *tradingDataServiceV2) Info(_ context.Context, _ *v2.InfoRequest) (*v2.InfoResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("InfoV2")()

	return &v2.InfoResponse{
		Version:    version.Get(),
		CommitHash: version.GetCommitHash(),
	}, nil
}

// ListLedgerEntries returns a list of ledger entries matching the request.
func (t *tradingDataServiceV2) ListLedgerEntries(ctx context.Context, req *v2.ListLedgerEntriesRequest) (*v2.ListLedgerEntriesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListLedgerEntriesV2")()

	leFilter, err := entities.LedgerEntryFilterFromProto(req.Filter)
	if err != nil {
		return nil, formatE(ErrInvalidFilter, err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	entries, pageInfo, err := t.ledgerService.Query(ctx, leFilter, dateRange, pagination)
	if err != nil {
		return nil, formatE(ErrLedgerServiceGet, err)
	}

	edges, err := makeEdges[*v2.AggregatedLedgerEntriesEdge](*entries)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.ListLedgerEntriesResponse{
		LedgerEntries: &v2.AggregatedLedgerEntriesConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}

// ExportLedgerEntries returns a list of ledger entries matching the request.
func (t *tradingDataServiceV2) ExportLedgerEntries(ctx context.Context, req *v2.ExportLedgerEntriesRequest) (*v2.ExportLedgerEntriesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ExportLedgerEntriesV2")()

	dateRange := entities.DateRangeFromProto(req.DateRange)
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	raw, pageInfo, err := t.ledgerService.Export(ctx, req.PartyId, req.AssetId, dateRange, pagination)
	if err != nil {
		return nil, formatE(ErrLedgerServiceExport, err)
	}

	header := metadata.New(map[string]string{
		"Content-Type":        "text/csv",
		"Content-Disposition": fmt.Sprintf("attachment;filename=%s", "ledger_entries_export.csv"),
	})

	if err = grpc.SendHeader(ctx, header); err != nil {
		return nil, formatE(ErrSendingGRPCHeader, err)
	}

	return &v2.ExportLedgerEntriesResponse{
		Data:     raw,
		PageInfo: pageInfo.ToProto(),
	}, nil
}

// ListBalanceChanges returns a list of balance changes matching the request.
func (t *tradingDataServiceV2) ListBalanceChanges(ctx context.Context, req *v2.ListBalanceChangesRequest) (*v2.ListBalanceChangesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListBalanceChangesV2")()

	filter, err := entities.AccountFilterFromProto(req.Filter)
	if err != nil {
		return nil, formatE(ErrInvalidFilter, err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	balances, pageInfo, err := t.accountService.QueryAggregatedBalances(ctx, filter, dateRange, pagination)
	if err != nil {
		return nil, formatE(ErrAccountServiceGetBalances, err)
	}

	edges, err := makeEdges[*v2.AggregatedBalanceEdge](*balances)
	if err != nil {
		return nil, formatE(err)
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
		return nil, err
	}

	return &v2.MarketDataConnection{
		Edges: edges,
	}, nil
}

// ObserveMarketsDepth subscribes to market depth updates.
func (t *tradingDataServiceV2) ObserveMarketsDepth(req *v2.ObserveMarketsDepthRequest, srv v2.TradingDataService_ObserveMarketsDepthServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	for _, marketID := range req.MarketIds {
		if !t.marketExistsForID(ctx, marketID) {
			return formatE(errors.Wrapf(ErrMalformedRequest, "no market found for id: %s", marketID))
		}
	}

	depthChan, ref := t.marketDepthService.ObserveDepth(ctx, t.config.StreamRetries, req.MarketIds)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Depth subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "MarketDepth", depthChan, ref, func(tr []*vega.MarketDepth) error {
		return srv.Send(&v2.ObserveMarketsDepthResponse{
			MarketDepth: tr,
		})
	})
}

// ObserveMarketsDepthUpdates subscribes to market depth updates.
func (t *tradingDataServiceV2) ObserveMarketsDepthUpdates(req *v2.ObserveMarketsDepthUpdatesRequest, srv v2.TradingDataService_ObserveMarketsDepthUpdatesServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	for _, marketID := range req.MarketIds {
		if !t.marketExistsForID(ctx, marketID) {
			return formatE(errors.Wrapf(ErrMalformedRequest, "no market found for id: %s", marketID))
		}
	}

	depthChan, ref := t.marketDepthService.ObserveDepthUpdates(ctx, t.config.StreamRetries, req.MarketIds)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Depth updates subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "MarketDepthUpdate", depthChan, ref, func(tr []*vega.MarketDepthUpdate) error {
		return srv.Send(&v2.ObserveMarketsDepthUpdatesResponse{
			Update: tr,
		})
	})
}

// ObserveMarketsData subscribes to market data updates.
func (t *tradingDataServiceV2) ObserveMarketsData(req *v2.ObserveMarketsDataRequest, srv v2.TradingDataService_ObserveMarketsDataServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	for _, marketID := range req.MarketIds {
		if !t.marketExistsForID(ctx, marketID) {
			return formatE(errors.Wrapf(ErrMalformedRequest, "no market found for id: %s", marketID))
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
		return nil, formatE(ErrMarketServiceGetMarketData, err)
	}

	return &v2.GetLatestMarketDataResponse{
		MarketData: md.ToProto(),
	}, nil
}

// ListLatestMarketData returns the latest market data for every market.
func (t *tradingDataServiceV2) ListLatestMarketData(ctx context.Context, _ *v2.ListLatestMarketDataRequest) (*v2.ListLatestMarketDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListLatestMarketData")()

	mds, err := t.marketDataService.GetMarketsData(ctx)
	if err != nil {
		return nil, formatE(ErrMarketServiceGetMarketData, err)
	}

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

	lastOne := entities.OffsetPagination{Skip: 0, Limit: 1, Descending: true}
	ts, err := t.tradeService.GetByMarket(ctx, req.MarketId, lastOne)
	if err != nil {
		return nil, formatE(ErrTradeServiceGetByMarket, err)
	}

	var lastTrade *vega.Trade
	if len(ts) > 0 {
		lastTrade = ts[0].ToProto()
	}

	depth := t.marketDepthService.GetMarketDepth(req.MarketId, ptr.UnBox(req.MaxDepth))
	// Build market depth response, including last trade (if available)
	return &v2.GetLatestMarketDepthResponse{
		Buy:            depth.Buy,
		MarketId:       depth.MarketId,
		Sell:           depth.Sell,
		SequenceNumber: depth.SequenceNumber,
		LastTrade:      lastTrade,
	}, nil
}

// GetMarketDataHistoryByID returns the market data history for a given market.
func (t *tradingDataServiceV2) GetMarketDataHistoryByID(ctx context.Context, req *v2.GetMarketDataHistoryByIDRequest) (*v2.GetMarketDataHistoryByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMarketDataHistoryV2")()

	startTime := vegatime.Unix(0, ptr.UnBox(req.StartTimestamp))
	endTime := vegatime.Unix(0, ptr.UnBox(req.EndTimestamp))

	if req.OffsetPagination != nil {
		// TODO: This has been deprecated in the GraphQL API, but needs to be supported until it is removed.
		marketData, err := t.handleGetMarketDataHistoryWithOffsetPagination(ctx, req, startTime, endTime)
		if err != nil {
			return marketData, formatE(ErrMarketServiceGetMarketDataHistory, err)
		}
		return marketData, nil
	}
	marketData, err := t.handleGetMarketDataHistoryWithCursorPagination(ctx, req, startTime, endTime)
	if err != nil {
		return marketData, formatE(ErrMarketServiceGetMarketDataHistory, err)
	}
	return marketData, nil
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
		return nil, errors.Wrap(ErrInvalidPagination, err.Error())
	}

	history, pageInfo, err := t.marketDataService.GetBetweenDatesByID(ctx, req.MarketId, startTime, endTime, pagination)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve historic market data")
	}

	edges, err := makeEdges[*v2.MarketDataEdge](history)
	if err != nil {
		return nil, err
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
	return &v2.GetMarketDataHistoryByIDResponse{
		MarketData: marketData,
	}, errors.Wrap(err, "could not parse market data results")
}

func (t *tradingDataServiceV2) getMarketDataHistoryByID(ctx context.Context, id string, start, end time.Time, pagination entities.OffsetPagination) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, _, err := t.marketDataService.GetBetweenDatesByID(ctx, id, start, end, pagination)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve market data history for market id: %s", id)
	}
	return parseMarketDataResults(results)
}

func (t *tradingDataServiceV2) getMarketDataByID(ctx context.Context, id string) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, err := t.marketDataService.GetMarketDataByID(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve market data for market id: %s", id)
	}
	return parseMarketDataResults([]entities.MarketData{results})
}

func (t *tradingDataServiceV2) getMarketDataHistoryFromDateByID(ctx context.Context, id string, start time.Time, pagination entities.OffsetPagination) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, _, err := t.marketDataService.GetFromDateByID(ctx, id, start, pagination)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve market data from date %s for market id: %s", start, id)
	}
	return parseMarketDataResults(results)
}

func (t *tradingDataServiceV2) getMarketDataHistoryToDateByID(ctx context.Context, id string, end time.Time, pagination entities.OffsetPagination) (*v2.GetMarketDataHistoryByIDResponse, error) {
	results, _, err := t.marketDataService.GetToDateByID(ctx, id, end, pagination)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve market data to date %s for market id: %s", end, id)
	}
	return parseMarketDataResults(results)
}

// GetNetworkLimits returns the latest network limits.
func (t *tradingDataServiceV2) GetNetworkLimits(ctx context.Context, _ *v2.GetNetworkLimitsRequest) (*v2.GetNetworkLimitsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkLimitsV2")()

	limits, err := t.networkLimitsService.GetLatest(ctx)
	if err != nil {
		return nil, formatE(ErrGetNetworkLimits, err)
	}

	return &v2.GetNetworkLimitsResponse{
		Limits: limits.ToProto(),
	}, nil
}

// ListCandleData for a given market, time range and interval.  Interval must be a valid postgres interval value.
func (t *tradingDataServiceV2) ListCandleData(ctx context.Context, req *v2.ListCandleDataRequest) (*v2.ListCandleDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListCandleDataV2")()

	var from, to *time.Time
	if req.FromTimestamp != 0 {
		from = ptr.From(vegatime.UnixNano(req.FromTimestamp))
	}

	if req.ToTimestamp != 0 {
		to = ptr.From(vegatime.UnixNano(req.ToTimestamp))
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	if len(req.CandleId) == 0 {
		return nil, formatE(ErrMissingCandleID)
	}

	candles, pageInfo, err := t.candleService.GetCandleDataForTimeSpan(ctx, req.CandleId, from, to, pagination)
	if err != nil {
		return nil, formatE(ErrCandleServiceGetCandleData, err)
	}

	edges, err := makeEdges[*v2.CandleEdge](candles)
	if err != nil {
		return nil, formatE(err)
	}

	connection := v2.CandleDataConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListCandleDataResponse{
		Candles: &connection,
	}, nil
}

// ObserveCandleData subscribes to candle updates for a given market and interval.  Interval must be a valid postgres interval value.
func (t *tradingDataServiceV2) ObserveCandleData(req *v2.ObserveCandleDataRequest, srv v2.TradingDataService_ObserveCandleDataServer) error {
	defer metrics.StartActiveSubscriptionCountGRPC("Candle")()

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	subscriptionID, candlesChan, err := t.candleService.Subscribe(ctx, req.CandleId)
	defer t.candleService.Unsubscribe(subscriptionID)
	if err != nil {
		return formatE(ErrCandleServiceSubscribeToCandles, err)
	}

	publishedEventStatTicker := time.NewTicker(time.Second)
	defer publishedEventStatTicker.Stop()

	var publishedEvents int64
	for {
		select {
		case <-publishedEventStatTicker.C:
			metrics.PublishedEventsAdd("Candle", float64(publishedEvents))
			publishedEvents = 0
		case candle, ok := <-candlesChan:
			if !ok {
				return formatE(ErrChannelClosed)
			}

			resp := &v2.ObserveCandleDataResponse{
				Candle: candle.ToV2CandleProto(),
			}
			if err = srv.Send(resp); err != nil {
				return formatE(ErrCandleServiceSubscribeToCandles, err)
			}
			publishedEvents++
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil {
				return formatE(ErrCandleServiceSubscribeToCandles, err)
			}
			return nil
		}
	}
}

// ListCandleIntervals gets all available intervals for a given market along with the corresponding candle id.
func (t *tradingDataServiceV2) ListCandleIntervals(ctx context.Context, req *v2.ListCandleIntervalsRequest) (*v2.ListCandleIntervalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListCandleIntervals")()

	mappings, err := t.candleService.GetCandlesForMarket(ctx, req.MarketId)
	if err != nil {
		return nil, formatE(ErrCandleServiceGetCandlesForMarket, err)
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

// ListERC20MultiSigSignerAddedBundles returns the signature bundles needed to add a new validator to the multisig control ERC20 contract.
func (t *tradingDataServiceV2) ListERC20MultiSigSignerAddedBundles(ctx context.Context, req *v2.ListERC20MultiSigSignerAddedBundlesRequest) (*v2.ListERC20MultiSigSignerAddedBundlesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20MultiSigSignerAddedBundlesV2")()

	var epochID *int64
	if len(req.EpochSeq) > 0 {
		e, err := strconv.ParseInt(req.EpochSeq, 10, 64)
		if err != nil {
			return nil, formatE(ErrEpochIDParse, errors.Wrapf(err, "epochSql: %s", req.EpochSeq))
		}
		epochID = &e
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	res, pageInfo, err := t.multiSigService.GetAddedEvents(ctx, req.GetNodeId(), req.GetSubmitter(), epochID, pagination)
	if err != nil {
		return nil, formatE(ErrMultiSigServiceGetAdded, err)
	}

	// find bundle for this nodeID, might be multiple if it's added, then removed, then added again??
	edges := make([]*v2.ERC20MultiSigSignerAddedBundleEdge, len(res))
	for i, b := range res {
		// it doesn't really make sense to paginate this, so we'll just pass it an empty pagination object and get all available results
		resID := b.ID.String()
		signatures, _, err := t.notaryService.GetByResourceID(ctx, resID, entities.CursorPagination{})
		if err != nil {
			return nil, formatE(ErrNotaryServiceGetByResourceID, errors.Wrapf(err, "resourceID: %s", resID))
		}

		edges[i] = &v2.ERC20MultiSigSignerAddedBundleEdge{
			Node: &v2.ERC20MultiSigSignerAddedBundle{
				NewSigner:  b.SignerChange.String(),
				Submitter:  b.Submitter.String(),
				Nonce:      b.Nonce,
				Timestamp:  b.VegaTime.UnixNano(),
				Signatures: entities.PackNodeSignatures(signatures),
				EpochSeq:   strconv.FormatInt(b.EpochID, 10),
			},
			Cursor: b.Cursor().Encode(),
		}
	}

	connection := &v2.ERC20MultiSigSignerAddedConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListERC20MultiSigSignerAddedBundlesResponse{
		Bundles: connection,
	}, nil
}

// ListERC20MultiSigSignerRemovedBundles returns the signature bundles needed to add a new validator to the multisig control ERC20 contract.
func (t *tradingDataServiceV2) ListERC20MultiSigSignerRemovedBundles(ctx context.Context, req *v2.ListERC20MultiSigSignerRemovedBundlesRequest) (*v2.ListERC20MultiSigSignerRemovedBundlesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20MultiSigSignerRemovedBundlesV2")()

	var epochID *int64
	if len(req.EpochSeq) > 0 {
		e, err := strconv.ParseInt(req.EpochSeq, 10, 64)
		if err != nil {
			return nil, formatE(ErrEpochIDParse, errors.Wrapf(err, "epochSql: %s", req.EpochSeq))
		}
		epochID = &e
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	res, pageInfo, err := t.multiSigService.GetRemovedEvents(ctx, req.GetNodeId(), req.GetSubmitter(), epochID, pagination)
	if err != nil {
		return nil, formatE(ErrMultiSigServiceGetRemoved, err)
	}

	// find bundle for this nodeID, might be multiple if it's added, then, removed them added again??
	edges := make([]*v2.ERC20MultiSigSignerRemovedBundleEdge, len(res))
	for i, b := range res {
		// it doesn't really make sense to paginate this, so we'll just pass it an empty pagination object and get all available results
		resID := b.ID.String()
		signatures, _, err := t.notaryService.GetByResourceID(ctx, resID, entities.CursorPagination{})
		if err != nil {
			return nil, formatE(ErrNotaryServiceGetByResourceID, errors.Wrapf(err, "resourceID: %s", resID))
		}

		edges[i] = &v2.ERC20MultiSigSignerRemovedBundleEdge{
			Node: &v2.ERC20MultiSigSignerRemovedBundle{
				OldSigner:  b.SignerChange.String(),
				Submitter:  b.Submitter.String(),
				Nonce:      b.Nonce,
				Timestamp:  b.VegaTime.UnixNano(),
				Signatures: entities.PackNodeSignatures(signatures),
				EpochSeq:   strconv.FormatInt(b.EpochID, 10),
			},
			Cursor: b.Cursor().Encode(),
		}
	}

	connection := &v2.ERC20MultiSigSignerRemovedConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListERC20MultiSigSignerRemovedBundlesResponse{
		Bundles: connection,
	}, nil
}

// GetERC20SetAssetLimitsBundle returns the signature bundle needed to update the asset limits on the ERC20 contract.
func (t *tradingDataServiceV2) GetERC20SetAssetLimitsBundle(ctx context.Context, req *v2.GetERC20SetAssetLimitsBundleRequest) (*v2.GetERC20SetAssetLimitsBundleResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20SetAssetLimitsBundleV2")()

	if len(req.ProposalId) == 0 {
		return nil, formatE(ErrMissingProposalID)
	}

	proposal, err := t.governanceService.GetProposalByID(ctx, req.ProposalId)
	if err != nil {
		return nil, formatE(ErrGovernanceServiceGet, err)
	}

	if proposal.Terms.GetUpdateAsset() == nil {
		return nil, formatE(errors.New("not an update asset proposal"))
	}

	if proposal.Terms.GetUpdateAsset().GetChanges().GetErc20() == nil {
		return nil, formatE(errors.New("not an update erc20 asset proposal"))
	}

	signatures, _, err := t.notaryService.GetByResourceID(ctx, req.ProposalId, entities.CursorPagination{})
	if err != nil {
		return nil, formatE(ErrNotaryServiceGetByResourceID, errors.Wrapf(err, "proposalID: %s", req.ProposalId))
	}

	asset, err := t.assetService.GetByID(ctx, proposal.Terms.GetUpdateAsset().AssetId)
	if err != nil {
		return nil, formatE(ErrAssetServiceGetByID, err)
	}

	if len(asset.ERC20Contract) == 0 {
		return nil, formatE(ErrERC20InvalidTokenContractAddress)
	}

	nonce, err := num.UintFromHex("0x" + strings.TrimLeft(req.ProposalId, "0"))
	if err != nil {
		return nil, formatE(ErrInvalidProposalID, errors.Wrapf(err, "proposalID: %s", req.ProposalId))
	}

	return &v2.GetERC20SetAssetLimitsBundleResponse{
		AssetSource:   asset.ERC20Contract,
		Nonce:         nonce.String(),
		VegaAssetId:   asset.ID.String(),
		Signatures:    entities.PackNodeSignatures(signatures),
		LifetimeLimit: proposal.Terms.GetUpdateAsset().GetChanges().GetErc20().LifetimeLimit,
		Threshold:     proposal.Terms.GetUpdateAsset().GetChanges().GetErc20().WithdrawThreshold,
	}, nil
}

func (t *tradingDataServiceV2) GetERC20ListAssetBundle(ctx context.Context, req *v2.GetERC20ListAssetBundleRequest) (*v2.GetERC20ListAssetBundleResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20ListAssetBundleV2")()

	if len(req.AssetId) == 0 {
		return nil, formatE(ErrMissingAssetID)
	}

	asset, err := t.assetService.GetByID(ctx, req.AssetId)
	if err != nil {
		return nil, formatE(ErrAssetServiceGetByID, err)
	}

	signatures, _, err := t.notaryService.GetByResourceID(ctx, req.AssetId, entities.CursorPagination{})
	if err != nil {
		return nil, formatE(ErrNotaryServiceGetByResourceID, errors.Wrapf(err, "assetID: %s", req.AssetId))
	}

	if len(asset.ERC20Contract) == 0 {
		return nil, formatE(ErrERC20InvalidTokenContractAddress, err)
	}

	nonce, err := num.UintFromHex("0x" + strings.TrimLeft(req.AssetId, "0"))
	if err != nil {
		return nil, formatE(ErrorInvalidAssetID, errors.Wrapf(err, "assetID: %s", req.AssetId))
	}

	return &v2.GetERC20ListAssetBundleResponse{
		AssetSource: asset.ERC20Contract,
		Nonce:       nonce.String(),
		VegaAssetId: asset.ID.String(),
		Signatures:  entities.PackNodeSignatures(signatures),
	}, nil
}

// GetERC20WithdrawalApproval returns the signature bundle needed to approve a withdrawal on the ERC20 contract.
func (t *tradingDataServiceV2) GetERC20WithdrawalApproval(ctx context.Context, req *v2.GetERC20WithdrawalApprovalRequest) (*v2.GetERC20WithdrawalApprovalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20WithdrawalApprovalV2")()

	if len(req.WithdrawalId) == 0 {
		return nil, formatE(ErrMissingWithdrawalID)
	}

	w, err := t.withdrawalService.GetByID(ctx, req.WithdrawalId)
	if err != nil {
		return nil, formatE(ErrWithdrawalServiceGet, err)
	}

	signatures, _, err := t.notaryService.GetByResourceID(ctx, req.WithdrawalId, entities.CursorPagination{})
	if err != nil {
		return nil, formatE(ErrNotaryServiceGetByResourceID, errors.Wrapf(err, "withdrawalID: %s", req.WithdrawalId))
	}

	assets, err := t.assetService.GetAll(ctx)
	if err != nil {
		return nil, formatE(ErrAssetServiceGetAll, err)
	}

	var address string
	for _, v := range assets {
		if v.ID == w.Asset {
			address = v.ERC20Contract
			break
		}
	}
	if len(address) == 0 {
		return nil, formatE(ErrERC20InvalidTokenContractAddress)
	}

	return &v2.GetERC20WithdrawalApprovalResponse{
		AssetSource:   address,
		Amount:        fmt.Sprintf("%v", w.Amount),
		Nonce:         w.Ref,
		TargetAddress: w.Ext.GetErc20().ReceiverAddress,
		Signatures:    entities.PackNodeSignatures(signatures),
		// timestamps is unix nano, contract needs unix. So load if first, and cut nanos
		Creation: w.CreatedTimestamp.Unix(),
	}, nil
}

// GetLastTrade returns the last trade for a given market.
func (t *tradingDataServiceV2) GetLastTrade(ctx context.Context, req *v2.GetLastTradeRequest) (*v2.GetLastTradeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLastTradeV2")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	pagination := entities.OffsetPagination{
		Skip:       0,
		Limit:      1,
		Descending: true,
	}

	trades, err := t.tradeService.GetByMarket(ctx, req.MarketId, pagination)
	if err != nil {
		return nil, formatE(ErrTradeServiceGetByMarket, err)
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

// ListTrades lists trades by using a cursor based pagination model.
func (t *tradingDataServiceV2) ListTrades(ctx context.Context, req *v2.ListTradesRequest) (*v2.ListTradesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTradesV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	trades, pageInfo, err := t.tradeService.List(ctx,
		entities.MarketID(req.GetMarketId()),
		entities.PartyID(req.GetPartyId()),
		entities.OrderID(req.GetOrderId()),
		pagination,
		dateRange)
	if err != nil {
		return nil, formatE(ErrTradeServiceList, err)
	}

	edges, err := makeEdges[*v2.TradeEdge](trades)
	if err != nil {
		return nil, formatE(err)
	}

	tradesConnection := &v2.TradeConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListTradesResponse{
		Trades: tradesConnection,
	}, nil
}

// ObserveTrades opens a subscription to the Trades service.
func (t *tradingDataServiceV2) ObserveTrades(req *v2.ObserveTradesRequest, srv v2.TradingDataService_ObserveTradesServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	tradesChan, ref := t.tradeService.Observe(ctx, t.config.StreamRetries, req.MarketId, req.PartyId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Trades subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Trade", tradesChan, ref, func(trades []*entities.Trade) error {
		protos := make([]*vega.Trade, 0, len(trades))
		for _, v := range trades {
			protos = append(protos, v.ToProto())
		}

		batches := batch(protos, snapshotPageSize)

		for _, batch := range batches {
			response := &v2.ObserveTradesResponse{Trades: batch}
			if err := srv.Send(response); err != nil {
				return errors.Wrap(err, "sending trades updates")
			}
		}
		return nil
	})
}

/****************************** Markets **************************************/

// GetMarket returns a market by its ID.
func (t *tradingDataServiceV2) GetMarket(ctx context.Context, req *v2.GetMarketRequest) (*v2.GetMarketResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketByID_SQL")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	market, err := t.marketService.GetByID(ctx, req.MarketId)
	if err != nil {
		return nil, formatE(ErrMarketServiceGetByID, err)
	}

	return &v2.GetMarketResponse{
		Market: market.ToProto(),
	}, nil
}

// ListMarkets lists all markets using a cursor based pagination model.
func (t *tradingDataServiceV2) ListMarkets(ctx context.Context, req *v2.ListMarketsRequest) (*v2.ListMarketsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListMarketsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	includeSettled := true
	if req.IncludeSettled != nil {
		includeSettled = *req.IncludeSettled
	}

	markets, pageInfo, err := t.marketsService.GetAllPaged(ctx, "", pagination, includeSettled)
	if err != nil {
		return nil, formatE(ErrMarketServiceGetAllPaged, err)
	}

	edges, err := makeEdges[*v2.MarketEdge](markets)
	if err != nil {
		return nil, formatE(err)
	}

	marketsConnection := &v2.MarketConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListMarketsResponse{
		Markets: marketsConnection,
	}, nil
}

// List all Positions using a cursor based pagination model.
//
// Deprecated: Use ListAllPositions instead.
func (t *tradingDataServiceV2) ListPositions(ctx context.Context, req *v2.ListPositionsRequest) (*v2.ListPositionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListPositionsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	parties := []entities.PartyID{entities.PartyID(req.PartyId)}
	markets := []entities.MarketID{entities.MarketID(req.MarketId)}

	positions, pageInfo, err := t.positionService.GetByPartyConnection(ctx, parties, markets, pagination)
	if err != nil {
		return nil, formatE(ErrPositionServiceGetByParty, err)
	}

	edges, err := makeEdges[*v2.PositionEdge](positions)
	if err != nil {
		return nil, formatE(err)
	}

	PositionsConnection := &v2.PositionConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListPositionsResponse{
		Positions: PositionsConnection,
	}, nil
}

// ListAllPositions lists all positions using a cursor based pagination model.
func (t *tradingDataServiceV2) ListAllPositions(ctx context.Context, req *v2.ListAllPositionsRequest) (*v2.ListAllPositionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAllPositions")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		parties []entities.PartyID
		markets []entities.MarketID
	)
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
		return nil, formatE(ErrPositionServiceGetByParty, err)
	}

	edges, err := makeEdges[*v2.PositionEdge](positions)
	if err != nil {
		return nil, formatE(err)
	}

	PositionsConnection := &v2.PositionConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListAllPositionsResponse{
		Positions: PositionsConnection,
	}, nil
}

// ObservePositions subscribes to a stream of Positions.
func (t *tradingDataServiceV2) ObservePositions(req *v2.ObservePositionsRequest, srv v2.TradingDataService_ObservePositionsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	if err := t.sendPositionsSnapshot(ctx, req, srv); err != nil {
		return formatE(ErrPositionServiceSendSnapshot, err)
	}

	positionsChan, ref := t.positionService.Observe(ctx, t.config.StreamRetries, ptr.UnBox(req.PartyId), ptr.UnBox(req.MarketId))

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Positions subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Position", positionsChan, ref, func(positions []entities.Position) error {
		protos := make([]*vega.Position, len(positions))
		for i := 0; i < len(positions); i++ {
			protos[i] = positions[i].ToProto()
		}
		batches := batch(protos, snapshotPageSize)
		for _, batch := range batches {
			updates := &v2.PositionUpdates{Positions: batch}
			responseUpdates := &v2.ObservePositionsResponse_Updates{Updates: updates}
			response := &v2.ObservePositionsResponse{Response: responseUpdates}
			if err := srv.Send(response); err != nil {
				return errors.Wrap(err, "sending initial positions")
			}
		}

		return nil
	})
}

func (t *tradingDataServiceV2) sendPositionsSnapshot(ctx context.Context, req *v2.ObservePositionsRequest, srv v2.TradingDataService_ObservePositionsServer) error {
	var (
		positions []entities.Position
		err       error
	)
	// TODO: better use a filter struct instead of having 4 different cases here.
	// By market and party.
	if req.PartyId != nil && req.MarketId != nil {
		position, err := t.positionService.GetByMarketAndParty(ctx, *req.MarketId, *req.PartyId)
		if err != nil {
			return errors.Wrap(err, "getting initial positions by market+party")
		}
		positions = append(positions, position)
	}

	// By market.
	if req.PartyId == nil && req.MarketId != nil {
		positions, err = t.positionService.GetByMarket(ctx, *req.MarketId)
		if err != nil {
			return errors.Wrap(err, "getting initial positions by market")
		}
	}

	// By party.
	if req.PartyId != nil && req.MarketId == nil {
		positions, err = t.positionService.GetByParty(ctx, entities.PartyID(*req.PartyId))
		if err != nil {
			return errors.Wrap(err, "getting initial positions by party")
		}
	}

	// All the positions.
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
			return errors.Wrap(err, "sending initial positions")
		}
	}
	return nil
}

// GetParty returns a Party by ID.
func (t *tradingDataServiceV2) GetParty(ctx context.Context, req *v2.GetPartyRequest) (*v2.GetPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetParty")()

	party, err := t.partyService.GetByID(ctx, req.PartyId)
	if err != nil {
		return nil, formatE(ErrPartyServiceGetByID, err)
	}

	return &v2.GetPartyResponse{
		Party: party.ToProto(),
	}, nil
}

// ListParties lists Parties using a cursor based pagination model.
func (t *tradingDataServiceV2) ListParties(ctx context.Context, req *v2.ListPartiesRequest) (*v2.ListPartiesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListPartiesV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	parties, pageInfo, err := t.partyService.GetAllPaged(ctx, req.PartyId, pagination)
	if err != nil {
		return nil, formatE(ErrPartyServiceGetAll, err)
	}

	edges, err := makeEdges[*v2.PartyEdge](parties)
	if err != nil {
		return nil, formatE(err)
	}

	partyConnection := &v2.PartyConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListPartiesResponse{
		Parties: partyConnection,
	}, nil
}

// ListMarginLevels lists MarginLevels using a cursor based pagination model.
func (t *tradingDataServiceV2) ListMarginLevels(ctx context.Context, req *v2.ListMarginLevelsRequest) (*v2.ListMarginLevelsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListMarginLevelsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	marginLevels, pageInfo, err := t.riskService.GetMarginLevelsByIDWithCursorPagination(ctx, req.PartyId, req.MarketId, pagination)
	if err != nil {
		return nil, formatE(ErrRiskServiceGetMarginLevelsByID, err)
	}

	edges, err := makeEdges[*v2.MarginEdge](marginLevels, ctx, t.accountService)
	if err != nil {
		return nil, formatE(err)
	}

	marginLevelsConnection := &v2.MarginConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListMarginLevelsResponse{
		MarginLevels: marginLevelsConnection,
	}, nil
}

// ObserveMarginLevels subscribes to a stream of Margin Levels.
func (t *tradingDataServiceV2) ObserveMarginLevels(req *v2.ObserveMarginLevelsRequest, srv v2.TradingDataService_ObserveMarginLevelsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	marginLevelsChan, ref := t.riskService.ObserveMarginLevels(ctx, t.config.StreamRetries, req.PartyId, ptr.UnBox(req.MarketId))

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Margin levels subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "MarginLevel", marginLevelsChan, ref, func(ml entities.MarginLevels) error {
		protoMl, err := ml.ToProto(ctx, t.accountService)
		if err != nil {
			return errors.Wrap(err, "converting margin levels to proto")
		}

		return srv.Send(&v2.ObserveMarginLevelsResponse{
			MarginLevels: protoMl,
		})
	})
}

// ListRewards lists Rewards using a cursor based pagination model.
func (t *tradingDataServiceV2) ListRewards(ctx context.Context, req *v2.ListRewardsRequest) (*v2.ListRewardsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListRewardsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	rewards, pageInfo, err := t.rewardService.GetByCursor(ctx, &req.PartyId, req.AssetId, req.FromEpoch, req.ToEpoch, pagination)
	if err != nil {
		return nil, formatE(ErrGetRewards, err)
	}

	edges, err := makeEdges[*v2.RewardEdge](rewards)
	if err != nil {
		return nil, formatE(err)
	}

	rewardsConnection := &v2.RewardsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListRewardsResponse{
		Rewards: rewardsConnection,
	}, nil
}

// ListRewardSummaries gets reward summaries.
func (t *tradingDataServiceV2) ListRewardSummaries(ctx context.Context, req *v2.ListRewardSummariesRequest) (*v2.ListRewardSummariesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListRewardSummariesV2")()

	summaries, err := t.rewardService.GetSummaries(ctx, req.PartyId, req.AssetId)
	if err != nil {
		return nil, formatE(ErrSummaryServiceGet, err)
	}

	summaryProtos := make([]*vega.RewardSummary, len(summaries))

	for i, summary := range summaries {
		summaryProtos[i] = summary.ToProto()
	}

	return &v2.ListRewardSummariesResponse{
		Summaries: summaryProtos,
	}, nil
}

// ListEpochRewardSummaries gets reward summaries for epoch range.
func (t *tradingDataServiceV2) ListEpochRewardSummaries(ctx context.Context, req *v2.ListEpochRewardSummariesRequest) (*v2.ListEpochRewardSummariesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListEpochRewardSummaries")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	filter := entities.RewardSummaryFilterFromProto(req.Filter)
	summaries, pageInfo, err := t.rewardService.GetEpochRewardSummaries(ctx, filter, pagination)
	if err != nil {
		return nil, formatE(ErrSummaryServiceGet, err)
	}

	edges, err := makeEdges[*v2.EpochRewardSummaryEdge](summaries)
	if err != nil {
		return nil, formatE(err)
	}

	connection := v2.EpochRewardSummaryConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListEpochRewardSummariesResponse{
		Summaries: &connection,
	}, nil
}

// ObserveRewards subscribes to a stream of rewards.
func (t *tradingDataServiceV2) ObserveRewards(req *v2.ObserveRewardsRequest, srv v2.TradingDataService_ObserveRewardsServer) error {
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming reward updates")
	}

	ch, ref := t.rewardService.Observe(ctx, t.config.StreamRetries, ptr.UnBox(req.AssetId), ptr.UnBox(req.PartyId))

	return observe(ctx, t.log, "Reward", ch, ref, func(reward entities.Reward) error {
		return srv.Send(&v2.ObserveRewardsResponse{
			Reward: reward.ToProto(),
		})
	})
}

// GetDeposit gets a deposit by ID.
func (t *tradingDataServiceV2) GetDeposit(ctx context.Context, req *v2.GetDepositRequest) (*v2.GetDepositResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetDepositV2")()

	if len(req.Id) == 0 {
		return nil, formatE(ErrMissingDepositID)
	}

	deposit, err := t.depositService.GetByID(ctx, req.Id)
	if err != nil {
		return nil, formatE(ErrDepositServiceGet, err)
	}

	return &v2.GetDepositResponse{
		Deposit: deposit.ToProto(),
	}, nil
}

// ListDeposits gets deposits for a party.
func (t *tradingDataServiceV2) ListDeposits(ctx context.Context, req *v2.ListDepositsRequest) (*v2.ListDepositsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListDepositsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)

	deposits, pageInfo, err := t.depositService.GetByParty(ctx, req.PartyId, false, pagination, dateRange)
	if err != nil {
		return nil, formatE(ErrDepositServiceGet, err)
	}

	edges, err := makeEdges[*v2.DepositEdge](deposits)
	if err != nil {
		return nil, formatE(err)
	}

	depositConnection := &v2.DepositsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListDepositsResponse{
		Deposits: depositConnection,
	}, nil
}

func makeEdges[T proto.Message, V entities.PagedEntity[T]](inputs []V, args ...any) (edges []T, err error) {
	if len(inputs) == 0 {
		return
	}
	edges = make([]T, len(inputs))
	for i, input := range inputs {
		edges[i], err = input.ToProtoEdge(args...)
		if err != nil {
			err = errors.Wrapf(err, "failed to make edge for %v", input)
			return
		}
	}
	return
}

// GetWithdrawal gets a withdrawal by ID.
func (t *tradingDataServiceV2) GetWithdrawal(ctx context.Context, req *v2.GetWithdrawalRequest) (*v2.GetWithdrawalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetWithdrawalV2")()

	if len(req.Id) == 0 {
		return nil, formatE(ErrMissingWithdrawalID)
	}

	withdrawal, err := t.withdrawalService.GetByID(ctx, req.Id)
	if err != nil {
		return nil, formatE(ErrWithdrawalServiceGet, err)
	}

	return &v2.GetWithdrawalResponse{
		Withdrawal: withdrawal.ToProto(),
	}, nil
}

// ListWithdrawals gets withdrawals for a party.
func (t *tradingDataServiceV2) ListWithdrawals(ctx context.Context, req *v2.ListWithdrawalsRequest) (*v2.ListWithdrawalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListWithdrawalsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	withdrawals, pageInfo, err := t.withdrawalService.GetByParty(ctx, req.PartyId, false, pagination, dateRange)
	if err != nil {
		return nil, formatE(ErrWithdrawalServiceGet, err)
	}

	edges, err := makeEdges[*v2.WithdrawalEdge](withdrawals)
	if err != nil {
		return nil, formatE(err)
	}

	depositConnection := &v2.WithdrawalsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListWithdrawalsResponse{
		Withdrawals: depositConnection,
	}, nil
}

// GetAsset gets an asset by ID.
func (t *tradingDataServiceV2) GetAsset(ctx context.Context, req *v2.GetAssetRequest) (*v2.GetAssetResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetAssetV2")()

	if len(req.AssetId) == 0 {
		return nil, formatE(ErrMissingAssetID)
	}

	asset, err := t.assetService.GetByID(ctx, req.AssetId)
	if err != nil {
		return nil, formatE(ErrAssetServiceGetByID, err)
	}

	return &v2.GetAssetResponse{
		Asset: asset.ToProto(),
	}, nil
}

// ListAssets gets all assets. If an asset ID is provided, it will return a single asset.
func (t *tradingDataServiceV2) ListAssets(ctx context.Context, req *v2.ListAssetsRequest) (*v2.ListAssetsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAssetsV2")()

	if assetId := ptr.UnBox(req.AssetId); assetId != "" {
		asset, err := t.getSingleAsset(ctx, assetId)
		if err != nil {
			return nil, formatE(ErrAssetServiceGetByID, err)
		}
		return asset, nil
	}

	assets, err := t.getAllAssets(ctx, req.Pagination)
	if err != nil {
		return nil, formatE(ErrAssetServiceGetAll, err)
	}
	return assets, nil
}

func (t *tradingDataServiceV2) getSingleAsset(ctx context.Context, assetID string) (*v2.ListAssetsResponse, error) {
	asset, err := t.assetService.GetByID(ctx, assetID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get asset by ID: %s", assetID)
	}

	edges, err := makeEdges[*v2.AssetEdge]([]entities.Asset{asset})
	if err != nil {
		return nil, err
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

	return &v2.ListAssetsResponse{
		Assets: connection,
	}, nil
}

func (t *tradingDataServiceV2) getAllAssets(ctx context.Context, p *v2.Pagination) (*v2.ListAssetsResponse, error) {
	pagination, err := entities.CursorPaginationFromProto(p)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidPagination, err.Error())
	}

	assets, pageInfo, err := t.assetService.GetAllWithCursorPagination(ctx, pagination)
	if err != nil {
		return nil, errors.Wrap(ErrAssetServiceGetAll, err.Error())
	}

	edges, err := makeEdges[*v2.AssetEdge](assets)
	if err != nil {
		return nil, err
	}

	connection := &v2.AssetsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListAssetsResponse{
		Assets: connection,
	}, nil
}

// GetOracleSpec gets an oracle spec by ID.
func (t *tradingDataServiceV2) GetOracleSpec(ctx context.Context, req *v2.GetOracleSpecRequest) (*v2.GetOracleSpecResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOracleSpecV2")()

	if len(req.OracleSpecId) == 0 {
		return nil, formatE(ErrMissingOracleSpecID)
	}

	spec, err := t.oracleSpecService.GetSpecByID(ctx, req.OracleSpecId)
	if err != nil {
		return nil, formatE(ErrOracleSpecServiceGet, errors.Wrapf(err, "OracleSpecId: %s", req.OracleSpecId))
	}

	return &v2.GetOracleSpecResponse{
		OracleSpec: &vega.OracleSpec{
			ExternalDataSourceSpec: &vega.ExternalDataSourceSpec{
				Spec: spec.ToProto().ExternalDataSourceSpec.Spec,
			},
		},
	}, nil
}

// ListOracleSpecs gets all oracle specs.
func (t *tradingDataServiceV2) ListOracleSpecs(ctx context.Context, req *v2.ListOracleSpecsRequest) (*v2.ListOracleSpecsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOracleSpecsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	specs, pageInfo, err := t.oracleSpecService.GetSpecsWithCursorPagination(ctx, "", pagination)
	if err != nil {
		return nil, formatE(ErrOracleSpecServiceGetAll, err)
	}

	edges, err := makeEdges[*v2.OracleSpecEdge](specs)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.OracleSpecsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListOracleSpecsResponse{
		OracleSpecs: connection,
	}, nil
}

// ListOracleData gets all oracle data.
func (t *tradingDataServiceV2) ListOracleData(ctx context.Context, req *v2.ListOracleDataRequest) (*v2.ListOracleDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOracleDataConnectionV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		data     []entities.OracleData
		pageInfo entities.PageInfo
	)

	if oracleSpecID := ptr.UnBox(req.OracleSpecId); oracleSpecID != "" {
		data, pageInfo, err = t.oracleDataService.GetOracleDataBySpecID(ctx, oracleSpecID, pagination)
	} else {
		data, pageInfo, err = t.oracleDataService.ListOracleData(ctx, pagination)
	}
	if err != nil {
		return nil, formatE(ErrOracleDataServiceGet, err)
	}

	edges, err := makeEdges[*v2.OracleDataEdge](data)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.OracleDataConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListOracleDataResponse{
		OracleData: connection,
	}, nil
}

// ListLiquidityProvisions gets all liquidity provisions.
func (t *tradingDataServiceV2) ListLiquidityProvisions(ctx context.Context, req *v2.ListLiquidityProvisionsRequest) (*v2.ListLiquidityProvisionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLiquidityProvisionsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	partyID := entities.PartyID(ptr.UnBox(req.PartyId))
	marketID := entities.MarketID(ptr.UnBox(req.MarketId))
	reference := ptr.UnBox(req.Reference)
	live := ptr.UnBox(req.Live)

	lps, pageInfo, err := t.liquidityProvisionService.Get(ctx, partyID, marketID, reference, live, pagination)
	if err != nil {
		return nil, formatE(ErrLiquidityProvisionServiceGet, errors.Wrapf(err,
			"partyID: %s, marketID: %s, reference: %s", partyID, marketID, reference))
	}

	edges, err := makeEdges[*v2.LiquidityProvisionsEdge](lps)
	if err != nil {
		return nil, formatE(err)
	}

	liquidityProvisionConnection := &v2.LiquidityProvisionsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListLiquidityProvisionsResponse{
		LiquidityProvisions: liquidityProvisionConnection,
	}, nil
}

// ObserveLiquidityProvisions subscribes to liquidity provisions.
func (t *tradingDataServiceV2) ObserveLiquidityProvisions(req *v2.ObserveLiquidityProvisionsRequest, srv v2.TradingDataService_ObserveLiquidityProvisionsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	lpCh, ref := t.liquidityProvisionService.ObserveLiquidityProvisions(ctx, t.config.StreamRetries, req.PartyId, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Liquidity Provisions subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "LiquidityProvision", lpCh, ref, func(lps []entities.LiquidityProvision) error {
		protos := make([]*vega.LiquidityProvision, 0, len(lps))
		for _, v := range lps {
			protos = append(protos, v.ToProto())
		}
		batches := batch(protos, snapshotPageSize)
		for _, batch := range batches {
			response := &v2.ObserveLiquidityProvisionsResponse{LiquidityProvisions: batch}
			if err := srv.Send(response); err != nil {
				return errors.Wrap(err, "sending liquidity provisions updates")
			}
		}
		return nil
	})
}

// GetGovernanceData gets governance data.
func (t *tradingDataServiceV2) GetGovernanceData(ctx context.Context, req *v2.GetGovernanceDataRequest) (*v2.GetGovernanceDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetGovernanceData")()

	var (
		proposal entities.Proposal
		err      error
	)
	if req.ProposalId != nil {
		proposal, err = t.governanceService.GetProposalByID(ctx, *req.ProposalId)
	} else if req.Reference != nil {
		proposal, err = t.governanceService.GetProposalByReference(ctx, *req.Reference)
	} else {
		return nil, formatE(ErrMissingProposalIDOrReference)
	}
	if err != nil {
		return nil, formatE(ErrGovernanceServiceGet,
			errors.Wrapf(err, "proposalID: %s, reference: %s", ptr.UnBox(req.ProposalId), ptr.UnBox(req.Reference)))
	}

	gd, err := t.proposalToGovernanceData(ctx, proposal)
	if err != nil {
		return nil, formatE(ErrNotMapped, err)
	}

	return &v2.GetGovernanceDataResponse{
		Data: gd,
	}, nil
}

// ListGovernanceData lists governance data using cursor pagination.
func (t *tradingDataServiceV2) ListGovernanceData(ctx context.Context, req *v2.ListGovernanceDataRequest) (*v2.ListGovernanceDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListGovernanceDataV2")()

	var state *entities.ProposalState
	if req.ProposalState != nil {
		state = ptr.From(entities.ProposalState(*req.ProposalState))
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	proposals, pageInfo, err := t.governanceService.GetProposals(
		ctx,
		state,
		req.ProposerPartyId,
		(*entities.ProposalType)(req.ProposalType),
		pagination,
	)
	if err != nil {
		return nil, formatE(ErrGovernanceServiceGetProposals, errors.Wrapf(err, "ProposerPartyId: %s", ptr.UnBox(req.ProposerPartyId)))
	}

	edges, err := makeEdges[*v2.GovernanceDataEdge](proposals)
	if err != nil {
		return nil, formatE(err)
	}

	for i := range edges {
		proposalID := edges[i].Node.Proposal.Id
		edges[i].Node.Yes, edges[i].Node.No, err = t.getVotesByProposal(ctx, proposalID)
		if err != nil {
			return nil, formatE(ErrGovernanceServiceGetVotes, errors.Wrapf(err, "proposalID: %s", proposalID))
		}
	}

	proposalsConnection := &v2.GovernanceDataConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListGovernanceDataResponse{
		Connection: proposalsConnection,
	}, nil
}

func (t *tradingDataServiceV2) getVotesByProposal(ctx context.Context, proposalID string) (yesVotes, noVotes []*vega.Vote, err error) {
	var votes []entities.Vote
	votes, err = t.governanceService.GetVotes(ctx, &proposalID, nil, nil)
	if err != nil {
		return
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

// ListVotes gets all Votes using a cursor based pagination model.
func (t *tradingDataServiceV2) ListVotes(ctx context.Context, req *v2.ListVotesRequest) (*v2.ListVotesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListVotesV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	if req.PartyId == nil && req.ProposalId == nil {
		return nil, formatE(ErrMissingProposalIDAndPartyID)
	}

	votes, pageInfo, err := t.governanceService.GetConnection(ctx, req.ProposalId, req.PartyId, pagination)
	if err != nil {
		return nil, formatE(ErrGovernanceServiceGetVotes, errors.Wrapf(err,
			"proposalID: %s, partyID: %s", ptr.UnBox(req.ProposalId), ptr.UnBox(req.PartyId)))
	}

	edges, err := makeEdges[*v2.VoteEdge](votes)
	if err != nil {
		return nil, formatE(err)
	}

	VotesConnection := &v2.VoteConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListVotesResponse{
		Votes: VotesConnection,
	}, nil
}

// ListTransfers lists transfers using cursor pagination. If a pubkey is provided, it will list transfers for that pubkey.
func (t *tradingDataServiceV2) ListTransfers(ctx context.Context, req *v2.ListTransfersRequest) (*v2.ListTransfersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTransfersV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		transfers []entities.Transfer
		pageInfo  entities.PageInfo
	)
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
			err = errors.Errorf("unknown transfer direction: %v", req.Direction)
		}
	}
	if err != nil {
		return nil, formatE(ErrTransferServiceGet, errors.Wrapf(err, "pubkey: %s", ptr.UnBox(req.Pubkey)))
	}

	edges, err := makeEdges[*v2.TransferEdge](transfers, ctx, t.accountService)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.ListTransfersResponse{Transfers: &v2.TransferConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}}, nil
}

// GetOrder gets an order by ID.
func (t *tradingDataServiceV2) GetOrder(ctx context.Context, req *v2.GetOrderRequest) (*v2.GetOrderResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOrderV2")()

	if len(req.OrderId) == 0 {
		return nil, formatE(ErrMissingOrderID)
	}

	order, err := t.orderService.GetOrder(ctx, req.OrderId, req.Version)
	if err != nil {
		return nil, formatE(ErrOrderNotFound, errors.Wrapf(err, "orderID: %s", req.OrderId))
	}

	return &v2.GetOrderResponse{
		Order: order.ToProto(),
	}, nil
}

// ListOrders lists orders using cursor pagination.
func (t *tradingDataServiceV2) ListOrders(ctx context.Context, req *v2.ListOrdersRequest) (*v2.ListOrdersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOrdersV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var filter entities.OrderFilter
	if req.Filter != nil {
		dateRange := entities.DateRangeFromProto(req.Filter.DateRange)
		filter = entities.OrderFilter{
			Statuses:         req.Filter.Statuses,
			Types:            req.Filter.Types,
			TimeInForces:     req.Filter.TimeInForces,
			Reference:        req.Filter.Reference,
			ExcludeLiquidity: req.Filter.ExcludeLiquidity,
			LiveOnly:         ptr.UnBox(req.Filter.LiveOnly),
			PartyIDs:         req.Filter.PartyIds,
			MarketIDs:        req.Filter.MarketIds,
			DateRange:        &entities.DateRange{Start: dateRange.Start, End: dateRange.End},
		}
	}

	orders, pageInfo, err := t.orderService.ListOrders(ctx, pagination, filter)
	if err != nil {
		return nil, formatE(ErrOrderServiceGetOrders, err)
	}

	edges, err := makeEdges[*v2.OrderEdge](orders)
	if err != nil {
		return nil, formatE(err)
	}

	ordersConnection := &v2.OrderConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListOrdersResponse{
		Orders: ordersConnection,
	}, nil
}

// ListOrderVersions lists order versions using cursor pagination.
func (t *tradingDataServiceV2) ListOrderVersions(ctx context.Context, req *v2.ListOrderVersionsRequest) (*v2.ListOrderVersionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOrderVersionsV2")()

	if len(req.OrderId) == 0 {
		return nil, formatE(ErrMissingOrderID)
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	orders, pageInfo, err := t.orderService.ListOrderVersions(ctx, req.OrderId, pagination)
	if err != nil {
		return nil, formatE(ErrOrderServiceGetVersions, errors.Wrapf(err, "orderID: %s", req.OrderId))
	}

	edges, err := makeEdges[*v2.OrderEdge](orders)
	if err != nil {
		return nil, formatE(err)
	}

	ordersConnection := &v2.OrderConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListOrderVersionsResponse{
		Orders: ordersConnection,
	}, nil
}

// ObserveOrders subscribes to a stream of orders.
func (t *tradingDataServiceV2) ObserveOrders(req *v2.ObserveOrdersRequest, srv v2.TradingDataService_ObserveOrdersServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	if err := t.sendOrdersSnapshot(ctx, req, srv); err != nil {
		return formatE(err)
	}
	ordersChan, ref := t.orderService.ObserveOrders(ctx, t.config.StreamRetries, req.MarketIds, req.PartyIds, ptr.UnBox(req.ExcludeLiquidity))

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Order", ordersChan, ref, func(orders []entities.Order) error {
		protos := make([]*vega.Order, 0, len(orders))
		for _, v := range orders {
			protos = append(protos, v.ToProto())
		}

		batches := batch(protos, snapshotPageSize)

		for _, batch := range batches {
			updates := &v2.OrderUpdates{Orders: batch}
			responseUpdates := &v2.ObserveOrdersResponse_Updates{Updates: updates}
			response := &v2.ObserveOrdersResponse{Response: responseUpdates}
			if err := srv.Send(response); err != nil {
				return errors.Wrap(err, "sending orders updates")
			}
		}
		return nil
	})
}

func (t *tradingDataServiceV2) sendOrdersSnapshot(ctx context.Context, req *v2.ObserveOrdersRequest, srv v2.TradingDataService_ObserveOrdersServer) error {
	orders, pageInfo, err := t.orderService.ListOrders(ctx, entities.CursorPagination{NewestFirst: true}, entities.OrderFilter{
		MarketIDs:        req.MarketIds,
		PartyIDs:         req.PartyIds,
		ExcludeLiquidity: ptr.UnBox(req.ExcludeLiquidity),
	})
	if err != nil {
		return errors.Wrap(err, "fetching orders initial image")
	}

	if pageInfo.HasNextPage {
		return errors.New("orders initial image spans multiple pages")
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
			return errors.Wrap(err, "sending orders initial image")
		}
	}
	return nil
}

// ListDelegations returns a list of delegations using cursor pagination.
func (t *tradingDataServiceV2) ListDelegations(ctx context.Context, req *v2.ListDelegationsRequest) (*v2.ListDelegationsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListDelegationsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var epochID *int64
	if req.EpochId != nil {
		epochIDVal := *req.EpochId
		epoch, err := strconv.ParseInt(epochIDVal, 10, 64)
		if err != nil {
			return nil, formatE(ErrEpochIDParse, errors.Wrapf(err, "epochID: %s", epochIDVal))
		}
		epochID = &epoch
	}

	delegations, pageInfo, err := t.delegationService.Get(ctx, req.PartyId, req.NodeId, epochID, pagination)
	if err != nil {
		return nil, formatE(ErrDelegationServiceGet, errors.Wrapf(err, "partyID: %s, nodeID: %s, epochID: %d",
			ptr.UnBox(req.PartyId), ptr.UnBox(req.NodeId), epochID))
	}

	edges, err := makeEdges[*v2.DelegationEdge](delegations)
	if err != nil {
		return nil, formatE(err)
	}

	delegationsConnection := &v2.DelegationsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListDelegationsResponse{
		Delegations: delegationsConnection,
	}, nil
}

// ObserveDelegations subscribe to delegation events.
func (t *tradingDataServiceV2) ObserveDelegations(req *v2.ObserveDelegationsRequest, srv v2.TradingDataService_ObserveDelegationsServer) error {
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming delegation updates")
	}

	ch, ref := t.delegationService.Observe(ctx, t.config.StreamRetries, ptr.UnBox(req.PartyId), ptr.UnBox(req.NodeId))

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
		return nil, formatE(ErrGetEpoch, err)
	}

	// get the node-y bits
	networkData, err := t.nodeService.GetNodeData(ctx, uint64(epoch.ID))
	if err != nil {
		return nil, formatE(ErrNodeServiceGetNodeData, errors.Wrapf(err, "epochID: %d", epoch.ID))
	}

	// now use network parameters to calculate the maximum nodes allowed in each nodeSet
	key := "network.validators.tendermint.number"
	np, err := t.networkParameterService.GetByKey(ctx, key)
	if err != nil {
		return nil, formatE(ErrGetNetworkParameters, errors.Wrapf(err, "key: %s", key))
	}

	maxTendermint, err := strconv.ParseUint(np.Value, 10, 32)
	if err != nil {
		return nil, formatE(ErrGetNetworkParameters, errors.Wrapf(err, "value: %s", np.Value))
	}

	key = "network.validators.ersatz.multipleOfTendermintValidators"
	np, err = t.networkParameterService.GetByKey(ctx, key)
	if err != nil {
		return nil, formatE(ErrGetNetworkParameters, errors.Wrapf(err, "key: %s", key))
	}

	ersatzFactor, err := strconv.ParseFloat(np.Value, 32)
	if err != nil {
		return nil, formatE(ErrGetNetworkParameters, errors.Wrapf(err, "value: %s", np.Value))
	}

	data := networkData.ToProto()
	data.TendermintNodes.Maximum = ptr.From(uint32(maxTendermint))
	data.ErsatzNodes.Maximum = ptr.From(uint32(float64(maxTendermint) * ersatzFactor))

	return &v2.GetNetworkDataResponse{
		NodeData: data,
	}, nil
}

// GetNode retrieves information about a given node.
func (t *tradingDataServiceV2) GetNode(ctx context.Context, req *v2.GetNodeRequest) (*v2.GetNodeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodeV2")()

	if len(req.Id) == 0 {
		return nil, formatE(ErrMissingNodeID)
	}

	epoch, err := t.epochService.GetCurrent(ctx)
	if err != nil {
		return nil, formatE(ErrGetEpoch, err)
	}

	node, err := t.nodeService.GetNodeByID(ctx, req.Id, uint64(epoch.ID))
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.GetNodeResponse{
		Node: node.ToProto(),
	}, nil
}

// ListNodes returns information about the nodes on the network.
func (t *tradingDataServiceV2) ListNodes(ctx context.Context, req *v2.ListNodesRequest) (*v2.ListNodesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListNodesV2")()

	var (
		epoch entities.Epoch
		err   error
	)
	if req.EpochSeq == nil || *req.EpochSeq > math.MaxInt64 {
		epoch, err = t.epochService.GetCurrent(ctx)
	} else {
		epochSeq := int64(*req.EpochSeq)
		epoch, err = t.epochService.Get(ctx, epochSeq)
	}
	if err != nil {
		return nil, formatE(ErrGetEpoch, err)
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	nodes, pageInfo, err := t.nodeService.GetNodes(ctx, uint64(epoch.ID), pagination)
	if err != nil {
		return nil, formatE(ErrNodeServiceGetNodes, err)
	}

	edges, err := makeEdges[*v2.NodeEdge](nodes)
	if err != nil {
		return nil, formatE(err)
	}

	nodesConnection := &v2.NodesConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListNodesResponse{
		Nodes: nodesConnection,
	}, nil
}

// ListNodeSignatures returns the signatures for a given node.
func (t *tradingDataServiceV2) ListNodeSignatures(ctx context.Context, req *v2.ListNodeSignaturesRequest) (*v2.ListNodeSignaturesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListNodeSignatures")()

	if len(req.Id) == 0 {
		return nil, formatE(ErrMissingResourceID)
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	sigs, pageInfo, err := t.notaryService.GetByResourceID(ctx, req.Id, pagination)
	if err != nil {
		return nil, formatE(ErrNotaryServiceGetByResourceID, errors.Wrapf(err, "resourceID: %s", req.Id))
	}

	edges, err := makeEdges[*v2.NodeSignatureEdge](sigs)
	if err != nil {
		return nil, formatE(err)
	}

	nodeSignatureConnection := &v2.NodeSignaturesConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListNodeSignaturesResponse{
		Signatures: nodeSignatureConnection,
	}, nil
}

// GetEpoch retrieves data for a specific epoch, if id omitted it gets the current epoch.
func (t *tradingDataServiceV2) GetEpoch(ctx context.Context, req *v2.GetEpochRequest) (*v2.GetEpochResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetEpochV2")()

	var (
		epoch entities.Epoch
		err   error
	)
	if req.GetId() == 0 {
		epoch, err = t.epochService.GetCurrent(ctx)
	} else {
		epoch, err = t.epochService.Get(ctx, int64(req.GetId()))
	}
	if err != nil {
		return nil, formatE(ErrGetEpoch, err)
	}

	delegations, _, err := t.delegationService.Get(ctx, nil, nil, &epoch.ID, nil)
	if err != nil {
		return nil, formatE(ErrDelegationServiceGet, err)
	}

	protoEpoch := epoch.ToProto()
	protoEpoch.Delegations = make([]*vega.Delegation, len(delegations))
	for i, delegation := range delegations {
		protoEpoch.Delegations[i] = delegation.ToProto()
	}

	nodes, _, err := t.nodeService.GetNodes(ctx, uint64(epoch.ID), entities.CursorPagination{})
	if err != nil {
		return nil, formatE(ErrNodeServiceGetNodes, errors.Wrapf(err, "epochID: %d", epoch.ID))
	}

	protoEpoch.Validators = make([]*vega.Node, len(nodes))
	for i, node := range nodes {
		protoEpoch.Validators[i] = node.ToProto()
	}

	return &v2.GetEpochResponse{
		Epoch: protoEpoch,
	}, nil
}

// EstimateFee estimates the fee for a given market, price and size.
func (t *tradingDataServiceV2) EstimateFee(ctx context.Context, req *v2.EstimateFeeRequest) (*v2.EstimateFeeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateFee SQL")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	if len(req.Price) == 0 {
		return nil, formatE(ErrMissingPrice)
	}

	fee, err := t.estimateFee(ctx, req.MarketId, req.Price, req.Size)
	if err != nil {
		return nil, formatE(ErrEstimateFee, errors.Wrapf(err,
			"marketID: %s, price: %s, size: %d", req.MarketId, req.Price, req.Size))
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
		return nil, errors.Wrap(err, "getting asset from market")
	}

	asset, err := t.assetService.GetByID(ctx, assetID)
	if err != nil {
		return nil, errors.Wrapf(ErrAssetServiceGetByID, "assetID: %s", assetID)
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
		return nil, errors.Wrap(ErrMarketServiceGetByID, err.Error())
	}

	price, overflowed := num.UintFromString(priceS, 10)
	if overflowed {
		return nil, errors.Wrapf(ErrInvalidOrderPrice, "overflowed: %s", priceS)
	}

	price, err = t.scaleFromMarketToAssetPrice(ctx, mkt, price)
	if err != nil {
		return nil, errors.Wrap(ErrScalingPriceFromMarketToAsset, err.Error())
	}

	mdpd := num.DecimalFromFloat(10).
		Pow(num.DecimalFromInt64(int64(mkt.PositionDecimalPlaces)))

	base := num.DecimalFromUint(price.Mul(price, num.NewUint(size))).Div(mdpd)
	maker, infra, liquidity, err := t.feeFactors(mkt)
	if err != nil {
		return nil, errors.Wrap(err, "getting fee factors")
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
	liquidity, err = strconv.ParseFloat(mkt.Fees.Factors.LiquidityFee, 64)
	return
}

// EstimateMargin estimates the margin required for a given order.
func (t *tradingDataServiceV2) EstimateMargin(ctx context.Context, req *v2.EstimateMarginRequest) (*v2.EstimateMarginResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateMargin SQL")()

	margin, err := t.estimateMargin(
		ctx, req.Side, req.Type, req.MarketId, req.PartyId, req.Price, req.Size)
	if err != nil {
		return nil, formatE(ErrEstimateMargin, errors.Wrapf(err,
			"marketID: %s, partyID: %s, price: %s, size: %d", req.MarketId, req.PartyId, req.Price, req.Size))
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
		return nil, errors.Wrapf(err, "getting risk factors: %s", rMarket)
	}

	mkt, err := t.marketService.GetByID(ctx, rMarket)
	if err != nil {
		return nil, errors.Wrapf(err, "getting market: %s", rMarket)
	}

	mktData, err := t.marketDataService.GetMarketDataByID(ctx, rMarket)
	if err != nil {
		return nil, errors.Wrapf(err, "getting market data: %s", rMarket)
	}

	f, err := num.DecimalFromString(rf.Short.String())
	if err != nil {
		return nil, errors.Wrapf(err, "parsing risk factor short: %s", rf.Short.String())
	}
	if rSide == vega.Side_SIDE_BUY {
		f, err = num.DecimalFromString(rf.Long.String())
		if err != nil {
			return nil, errors.Wrapf(err, "parsing risk factor long: %s", rf.Long.String())
		}
	}

	mktProto := mkt.ToProto()

	asset, err := mktProto.GetAsset()
	if err != nil {
		return nil, errors.Wrap(err, "getting asset from market")
	}

	// now calculate margin maintenance
	priceD, err := num.DecimalFromString(mktData.MarkPrice.String())
	if err != nil {
		return nil, errors.Wrapf(err, "parsing mark price: %s", mktData.MarkPrice.String())
	}

	// if the order is a limit order, use the limit price to calculate the margin maintenance
	if rType == vega.Order_TYPE_LIMIT {
		priceD, err = num.DecimalFromString(rPrice)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing limit price: %s", rPrice)
		}
	}

	price, _ := num.UintFromDecimal(priceD)
	price, err = t.scaleFromMarketToAssetPrice(ctx, mkt, price)
	if err != nil {
		return nil, errors.Wrap(ErrScalingPriceFromMarketToAsset, err.Error())
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

// ListNetworkParameters returns a list of network parameters.
func (t *tradingDataServiceV2) ListNetworkParameters(ctx context.Context, req *v2.ListNetworkParametersRequest) (*v2.ListNetworkParametersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListNetworkParametersV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	nps, pageInfo, err := t.networkParameterService.GetAll(ctx, pagination)
	if err != nil {
		return nil, formatE(ErrGetNetworkParameters, err)
	}

	edges, err := makeEdges[*v2.NetworkParameterEdge](nps)
	if err != nil {
		return nil, formatE(err)
	}

	networkParametersConnection := &v2.NetworkParameterConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListNetworkParametersResponse{
		NetworkParameters: networkParametersConnection,
	}, nil
}

// GetNetworkParameter returns a network parameter by key.
func (t *tradingDataServiceV2) GetNetworkParameter(ctx context.Context, req *v2.GetNetworkParameterRequest) (*v2.GetNetworkParameterResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkParameter")()

	nps, _, err := t.networkParameterService.GetAll(ctx, entities.CursorPagination{})
	if err != nil {
		return nil, formatE(ErrGetNetworkParameters, err)
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

// ListCheckpoints returns a list of checkpoints.
func (t *tradingDataServiceV2) ListCheckpoints(ctx context.Context, req *v2.ListCheckpointsRequest) (*v2.ListCheckpointsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("NetworkParametersV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	checkpoints, pageInfo, err := t.checkpointService.GetAll(ctx, pagination)
	if err != nil {
		return nil, formatE(ErrCheckpointServiceGet, err)
	}

	edges, err := makeEdges[*v2.CheckpointEdge](checkpoints)
	if err != nil {
		return nil, formatE(err)
	}

	checkpointsConnection := &v2.CheckpointsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListCheckpointsResponse{
		Checkpoints: checkpointsConnection,
	}, nil
}

// GetStake returns the stake for a party and the linkings to that stake.
func (t *tradingDataServiceV2) GetStake(ctx context.Context, req *v2.GetStakeRequest) (*v2.GetStakeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetStake")()

	if len(req.PartyId) == 0 {
		return nil, formatE(ErrMissingPartyID)
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	stake, stakeLinkings, pageInfo, err := t.stakeLinkingService.GetStake(ctx, entities.PartyID(req.PartyId), pagination)
	if err != nil {
		return nil, formatE(ErrStakeLinkingServiceGet, err)
	}

	edges, err := makeEdges[*v2.StakeLinkingEdge](stakeLinkings)
	if err != nil {
		return nil, formatE(err)
	}

	stakesConnection := &v2.StakesConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.GetStakeResponse{
		CurrentStakeAvailable: num.UintToString(stake),
		StakeLinkings:         stakesConnection,
	}, nil
}

// GetRiskFactors returns the risk factors for a given market.
func (t *tradingDataServiceV2) GetRiskFactors(ctx context.Context, req *v2.GetRiskFactorsRequest) (*v2.GetRiskFactorsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetRiskFactors SQL")()

	rfs, err := t.riskFactorService.GetMarketRiskFactors(ctx, req.MarketId)
	if err != nil {
		return nil, formatE(ErrRiskFactorServiceGet, errors.Wrapf(err, "marketID: %s", req.MarketId))
	}

	return &v2.GetRiskFactorsResponse{
		RiskFactor: rfs.ToProto(),
	}, nil
}

// ObserveGovernance streams governance updates to the client.
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
			return errors.Wrapf(err, "converting proposal to governance data for proposalID: %s", proposal.ID.String())
		}
		return stream.Send(&v2.ObserveGovernanceResponse{
			Data: gd,
		})
	})
}

func (t *tradingDataServiceV2) proposalToGovernanceData(ctx context.Context, proposal entities.Proposal) (*vega.GovernanceData, error) {
	yesVotes, err := t.governanceService.GetYesVotesForProposal(ctx, proposal.ID.String())
	if err != nil {
		return nil, errors.Wrap(err, "getting yes votes for proposal")
	}

	noVotes, err := t.governanceService.GetNoVotesForProposal(ctx, proposal.ID.String())
	if err != nil {
		return nil, errors.Wrap(err, "getting no votes for proposal")
	}

	return &vega.GovernanceData{
		Proposal: proposal.ToProto(),
		Yes:      voteListToProto(yesVotes),
		No:       voteListToProto(noVotes),
	}, nil
}

func voteListToProto(votes []entities.Vote) []*vega.Vote {
	protoVotes := make([]*vega.Vote, len(votes))
	for i, vote := range votes {
		protoVotes[i] = vote.ToProto()
	}
	return protoVotes
}

// ObserveVotes streams votes for a given party or proposal.
func (t *tradingDataServiceV2) ObserveVotes(req *v2.ObserveVotesRequest, stream v2.TradingDataService_ObserveVotesServer) error {
	if partyID := ptr.UnBox(req.PartyId); partyID != "" {
		return t.observePartyVotes(partyID, stream)
	}

	if proposalID := ptr.UnBox(req.ProposalId); proposalID != "" {
		return t.observeProposalVotes(proposalID, stream)
	}

	return formatE(ErrMissingProposalIDOrPartyID)
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

// GetProtocolUpgradeStatus returns the status of the protocol upgrade process.
func (t *tradingDataServiceV2) GetProtocolUpgradeStatus(context.Context, *v2.GetProtocolUpgradeStatusRequest) (*v2.GetProtocolUpgradeStatusResponse, error) {
	ready := t.protocolUpgradeService.GetProtocolUpgradeStarted()
	return &v2.GetProtocolUpgradeStatusResponse{
		Ready: ready,
	}, nil
}

// ListProtocolUpgradeProposals returns a list of protocol upgrade proposals.
func (t *tradingDataServiceV2) ListProtocolUpgradeProposals(ctx context.Context, req *v2.ListProtocolUpgradeProposalsRequest) (*v2.ListProtocolUpgradeProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListProtocolUpgradeProposals")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
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
		return nil, formatE(ErrProtocolUpgradeServiceListProposals, err)
	}

	edges, err := makeEdges[*v2.ProtocolUpgradeProposalEdge](pups)
	if err != nil {
		return nil, formatE(err)
	}

	connection := v2.ProtocolUpgradeProposalConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListProtocolUpgradeProposalsResponse{
		ProtocolUpgradeProposals: &connection,
	}, nil
}

// ListCoreSnapshots returns a list of core snapshots.
func (t *tradingDataServiceV2) ListCoreSnapshots(ctx context.Context, req *v2.ListCoreSnapshotsRequest) (*v2.ListCoreSnapshotsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListCoreSnapshots")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	snaps, pageInfo, err := t.coreSnapshotService.ListSnapshots(ctx, pagination)
	if err != nil {
		return nil, formatE(ErrCoreSnapshotServiceListSnapshots, err)
	}

	edges, err := makeEdges[*v2.CoreSnapshotEdge](snaps)
	if err != nil {
		return nil, formatE(err)
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

// RecvMsg receives a message from the stream.
func (t tradingDataEventBusServerV2) RecvMsg(m interface{}) error {
	return t.stream.RecvMsg(m)
}

// Context gets the context from the stream.
func (t tradingDataEventBusServerV2) Context() context.Context {
	return t.stream.Context()
}

// Send sends a message to the stream.
func (t tradingDataEventBusServerV2) Send(data []*eventspb.BusEvent) error {
	return t.stream.Send(&v2.ObserveEventBusResponse{
		Events: data,
	})
}

// ObserveEventBus subscribes to a stream of events.
func (t *tradingDataServiceV2) ObserveEventBus(stream v2.TradingDataService_ObserveEventBusServer) error {
	return observeEventBus(t.log, t.config, tradingDataEventBusServerV2{stream}, t.eventService)
}

// ObserveLedgerMovements subscribes to a stream of ledger movements.
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

// ListKeyRotations returns a list of key rotations for a given node.
func (t *tradingDataServiceV2) ListKeyRotations(ctx context.Context, req *v2.ListKeyRotationsRequest) (*v2.ListKeyRotationsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListKeyRotations")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	if nodeID := ptr.UnBox(req.NodeId); nodeID != "" {
		rotations, err := t.getNodeKeyRotations(ctx, nodeID, pagination)
		if err != nil {
			return nil, formatE(ErrKeyRotationServiceGetPerNode, errors.Wrapf(err, "nodeID: %s", nodeID))
		}
		return rotations, nil
	}

	rotations, err := t.getAllKeyRotations(ctx, pagination)
	if err != nil {
		return nil, formatE(ErrKeyRotationServiceGetAll, err)
	}
	return rotations, nil
}

func (t *tradingDataServiceV2) getAllKeyRotations(ctx context.Context, pagination entities.CursorPagination) (*v2.ListKeyRotationsResponse, error) {
	return makeKeyRotationResponse(
		t.keyRotationService.GetAllPubKeyRotations(ctx, pagination),
	)
}

func (t *tradingDataServiceV2) getNodeKeyRotations(ctx context.Context, nodeID string, pagination entities.CursorPagination) (*v2.ListKeyRotationsResponse, error) {
	return makeKeyRotationResponse(
		t.keyRotationService.GetPubKeyRotationsPerNode(ctx, nodeID, pagination),
	)
}

func makeKeyRotationResponse(rotations []entities.KeyRotation, pageInfo entities.PageInfo, err error) (*v2.ListKeyRotationsResponse, error) {
	if err != nil {
		return nil, err
	}

	edges, err := makeEdges[*v2.KeyRotationEdge](rotations)
	if err != nil {
		return nil, err
	}

	keyRotationConnection := &v2.KeyRotationConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListKeyRotationsResponse{
		Rotations: keyRotationConnection,
	}, nil
}

// ListEthereumKeyRotations returns a list of Ethereum key rotations.
func (t *tradingDataServiceV2) ListEthereumKeyRotations(ctx context.Context, req *v2.ListEthereumKeyRotationsRequest) (*v2.ListEthereumKeyRotationsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListEthereumKeyRotationsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	rotations, pageInfo, err := t.ethereumKeyRotationService.List(ctx, entities.NodeID(req.GetNodeId()), pagination)
	if err != nil {
		return nil, formatE(ErrEthereumKeyRotationServiceGetPerNode, errors.Wrapf(err, "nodeID: %s", req.GetNodeId()))
	}

	edges, err := makeEdges[*v2.EthereumKeyRotationEdge](rotations)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.EthereumKeyRotationsConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListEthereumKeyRotationsResponse{
		KeyRotations: connection,
	}, nil
}

// GetVegaTime returns the current vega time.
func (t *tradingDataServiceV2) GetVegaTime(ctx context.Context, _ *v2.GetVegaTimeRequest) (*v2.GetVegaTimeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVegaTimeV2")()

	b, err := t.blockService.GetLastBlock(ctx)
	if err != nil {
		return nil, formatE(ErrBlockServiceGetLast, err)
	}

	return &v2.GetVegaTimeResponse{
		Timestamp: b.VegaTime.UnixNano(),
	}, nil
}

// -- NetworkHistory --.

// GetMostRecentNetworkHistorySegment returns the most recent network history segment.
func (t *tradingDataServiceV2) GetMostRecentNetworkHistorySegment(context.Context, *v2.GetMostRecentNetworkHistorySegmentRequest) (*v2.GetMostRecentNetworkHistorySegmentResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMostRecentNetworkHistorySegment")()

	segment, err := t.networkHistoryService.GetHighestBlockHeightHistorySegment()
	if err != nil {
		if errors.Is(err, store.ErrSegmentNotFound) {
			return &v2.GetMostRecentNetworkHistorySegmentResponse{
				Segment: nil,
			}, nil
		}
		return nil, formatE(ErrGetMostRecentHistorySegment, err)
	}

	return &v2.GetMostRecentNetworkHistorySegmentResponse{
		Segment:      toHistorySegment(segment),
		SwarmKeySeed: t.networkHistoryService.GetSwarmKeySeed(),
	}, nil
}

// ListAllNetworkHistorySegments returns all network history segments.
func (t *tradingDataServiceV2) ListAllNetworkHistorySegments(context.Context, *v2.ListAllNetworkHistorySegmentsRequest) (*v2.ListAllNetworkHistorySegmentsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAllNetworkHistorySegments")()

	segments, err := t.networkHistoryService.ListAllHistorySegments()
	if err != nil {
		return nil, formatE(ErrListAllNetworkHistorySegment, err)
	}

	historySegments := make([]*v2.HistorySegment, 0, len(segments))
	for _, segment := range segments {
		historySegments = append(historySegments, toHistorySegment(segment))
	}

	// Newest first
	sort.Slice(historySegments, func(i, j int) bool {
		return historySegments[i].ToHeight > historySegments[j].ToHeight
	})

	return &v2.ListAllNetworkHistorySegmentsResponse{
		Segments: historySegments,
	}, nil
}

func toHistorySegment(segment networkhistory.Segment) *v2.HistorySegment {
	return &v2.HistorySegment{
		FromHeight:               segment.GetFromHeight(),
		ToHeight:                 segment.GetToHeight(),
		HistorySegmentId:         segment.GetHistorySegmentId(),
		PreviousHistorySegmentId: segment.GetPreviousHistorySegmentId(),
	}
}

// GetActiveNetworkHistoryPeerAddresses returns the active network history peer addresses.
func (t *tradingDataServiceV2) GetActiveNetworkHistoryPeerAddresses(context.Context, *v2.GetActiveNetworkHistoryPeerAddressesRequest) (*v2.GetActiveNetworkHistoryPeerAddressesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMostRecentHistorySegmentFromPeers")()
	return &v2.GetActiveNetworkHistoryPeerAddressesResponse{
		IpAddresses: t.networkHistoryService.GetActivePeerIPAddresses(),
	}, nil
}

// NetworkHistoryStatus returns the network history status.
func (t *tradingDataServiceV2) NetworkHistoryStatus(context.Context, *v2.NetworkHistoryStatusRequest) (*v2.NetworkHistoryStatusResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("NetworkHistoryStatus")()

	connectedPeerAddresses, err := t.networkHistoryService.GetConnectedPeerAddresses()
	if err != nil {
		return nil, formatE(ErrGetConnectedPeerAddresses, err)
	}

	// A subset of the connected peer addresses are likely to be copied to form another nodes peer set, randomise the list
	// to minimise the chance that the same sub set are copied each time.
	rand.Shuffle(len(connectedPeerAddresses), func(i, j int) {
		connectedPeerAddresses[i], connectedPeerAddresses[j] = connectedPeerAddresses[j], connectedPeerAddresses[i]
	})

	ipfsAddress, err := t.networkHistoryService.GetIpfsAddress()
	if err != nil {
		return nil, formatE(ErrGetIpfsAddress, err)
	}

	return &v2.NetworkHistoryStatusResponse{
		IpfsAddress:    ipfsAddress,
		SwarmKey:       t.networkHistoryService.GetSwarmKey(),
		SwarmKeySeed:   t.networkHistoryService.GetSwarmKeySeed(),
		ConnectedPeers: connectedPeerAddresses,
	}, nil
}

// NetworkHistoryBootstrapPeers returns the network history bootstrap peers.
func (t *tradingDataServiceV2) NetworkHistoryBootstrapPeers(context.Context, *v2.NetworkHistoryBootstrapPeersRequest) (*v2.NetworkHistoryBootstrapPeersResponse, error) {
	return &v2.NetworkHistoryBootstrapPeersResponse{BootstrapPeers: t.networkHistoryService.GetBootstrapPeers()}, nil
}

// Ping returns a ping response.
func (t *tradingDataServiceV2) Ping(context.Context, *v2.PingRequest) (*v2.PingResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Ping")()
	return &v2.PingResponse{}, nil
}

func (t *tradingDataServiceV2) ListTransactionEntities(ctx context.Context, req *v2.ListTransactionEntitiesRequest) (*v2.ListTransactionEntitiesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTransactionEntities")()

	txHash := entities.TxHash(req.GetTransactionHash())

	accounts, err := t.accountService.GetByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceListAccounts, err)
	}

	balances, err := t.accountService.GetBalancesByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceListAccounts, err)
	}

	orders, err := t.orderService.GetByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrOrderServiceGetByTxHash, err)
	}

	positions, err := t.positionService.GetByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPostitionsGetByTxHash, err)
	}

	ledgerEntries, err := t.ledgerService.GetByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrLedgerEntriesGetByTxHash, err)
	}

	ledgerEntriesProtos, err := mapSlice(ledgerEntries,
		func(item entities.LedgerEntry) (*vega.LedgerEntry, error) {
			return item.ToProto(ctx, t.accountService)
		},
	)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	transfers, err := t.transfersService.GetByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTransfersGetByTxHash)
	}

	transfersProtos, err := mapSlice(transfers,
		func(item entities.Transfer) (*v1.Transfer, error) {
			return item.ToProto(ctx, t.accountService)
		},
	)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	votes, err := t.governanceService.GetVotesByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrVotesGetByTxHash)
	}

	addedEvents, err := t.multiSigService.GetAddedByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrERC20MultiSigSignerAddedEventGetByTxHash)
	}

	addedEventsProtos, err := mapSlice(addedEvents,
		func(item entities.ERC20MultiSigSignerAddedEvent) (*v2.ERC20MultiSigSignerAddedBundle, error) {
			return item.ToDataNodeApiV2Proto(ctx, t.notaryService)
		},
	)
	if err != nil {
		return nil, err
	}

	removedEvents, err := t.multiSigService.GetRemovedByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrERC20MultiSigSignerRemovedEventGetByTxHash)
	}

	removedEventsProtos, err := mapSlice(removedEvents,
		func(item entities.ERC20MultiSigSignerRemovedEvent) (*v2.ERC20MultiSigSignerRemovedBundle, error) {
			return item.ToDataNodeApiV2Proto(ctx, t.notaryService)
		},
	)
	if err != nil {
		return nil, err
	}

	trades, err := t.tradeService.GetByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByTxHash)
	}

	oracleSpecs, err := t.oracleSpecService.GetByTxHash(ctx, txHash)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByTxHash) // TODO return proper error
	}

	return &v2.ListTransactionEntitiesResponse{
		Accounts:                          toProtos[*vega.Account](accounts),
		Orders:                            toProtos[*vega.Order](orders),
		Positions:                         toProtos[*vega.Position](positions),
		LedgerEntries:                     ledgerEntriesProtos,
		BalanceChanges:                    toProtos[*v2.AccountBalance](balances),
		Transfers:                         transfersProtos,
		Votes:                             toProtos[*vega.Vote](votes),
		Erc20MultiSigSignerAddedBundles:   addedEventsProtos,
		Erc20MultiSigSignerRemovedBundles: removedEventsProtos,
		Trades:                            toProtos[*vega.Trade](trades),
		OracleSpecs:                       toProtos[*vega.OracleSpec](oracleSpecs),
		// OracleData: ,
	}, nil
}

func toProtos[T proto.Message, V entities.ProtoEntity[T]](inputs []V) []T {
	protos := make([]T, 0, len(inputs))
	for _, input := range inputs {
		proto := input.ToProto()
		protos = append(protos, proto)
	}
	return protos
}

func mapSlice[T proto.Message, V any](inputs []V, toProto func(V) (T, error)) ([]T, error) {
	protos := make([]T, 0, len(inputs))
	for _, input := range inputs {
		proto, err := toProto(input)
		if err != nil {
			return nil, fmt.Errorf("failed to convert to proto: %w", err)
		}
		protos = append(protos, proto)
	}
	return protos, nil
}

func batch[T any](in []T, batchSize int) [][]T {
	batches := make([][]T, 0, (len(in)+batchSize-1)/batchSize)
	for batchSize < len(in) {
		in, batches = in[batchSize:], append(batches, in[0:batchSize:batchSize])
	}
	batches = append(batches, in)
	return batches
}
