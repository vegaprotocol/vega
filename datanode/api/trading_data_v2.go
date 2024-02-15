// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/datanode/networkhistory/fsutil"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	cmdsV1 "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	v1 "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/version"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const (
	networkPartyID = "network"
)

// When returning an 'initial image' snapshot, how many updates to batch into each page.
var snapshotPageSize = 50

// When sending files in chunks, how much data to send per stream message.
var httpBodyChunkSize = 1024 * 1024

type TradingDataServiceV2 struct {
	v2.UnimplementedTradingDataServiceServer
	config                        Config
	log                           *logging.Logger
	eventService                  EventService
	orderService                  *service.Order
	networkLimitsService          *service.NetworkLimits
	MarketDataService             MarketDataService
	tradeService                  *service.Trade
	multiSigService               *service.MultiSig
	notaryService                 *service.Notary
	AssetService                  AssetService
	candleService                 *candlesv2.Svc
	MarketsService                MarketsService
	partyService                  *service.Party
	riskService                   *service.Risk
	positionService               *service.Position
	accountService                *service.Account
	rewardService                 *service.Reward
	depositService                *service.Deposit
	withdrawalService             *service.Withdrawal
	oracleSpecService             *service.OracleSpec
	oracleDataService             *service.OracleData
	liquidityProvisionService     *service.LiquidityProvision
	governanceService             *service.Governance
	transfersService              *service.Transfer
	delegationService             *service.Delegation
	marketDepthService            *service.MarketDepth
	nodeService                   *service.Node
	epochService                  *service.Epoch
	RiskFactorService             RiskFactorService
	networkParameterService       *service.NetworkParameter
	checkpointService             *service.Checkpoint
	stakeLinkingService           *service.StakeLinking
	ledgerService                 *service.Ledger
	keyRotationService            *service.KeyRotations
	ethereumKeyRotationService    *service.EthereumKeyRotation
	blockService                  BlockService
	protocolUpgradeService        *service.ProtocolUpgrade
	NetworkHistoryService         NetworkHistoryService
	coreSnapshotService           *service.SnapshotData
	stopOrderService              *service.StopOrders
	fundingPeriodService          *service.FundingPeriods
	partyActivityStreak           *service.PartyActivityStreak
	fundingPaymentService         *service.FundingPayment
	referralProgramService        *service.ReferralPrograms
	referralSetsService           *service.ReferralSets
	teamsService                  *service.Teams
	feesStatsService              *service.FeesStats
	volumeDiscountStatsService    *service.VolumeDiscountStats
	volumeDiscountProgramService  *service.VolumeDiscountPrograms
	paidLiquidityFeesStatsService *service.PaidLiquidityFeesStats
	partyLockedBalances           *service.PartyLockedBalances
	partyVestingBalances          *service.PartyVestingBalances
	vestingStats                  *service.VestingStats
	transactionResults            *service.TransactionResults
	gamesService                  *service.Games
	marginModesService            *service.MarginModes
}

func (t *TradingDataServiceV2) GetPartyVestingStats(
	ctx context.Context,
	req *v2.GetPartyVestingStatsRequest,
) (*v2.GetPartyVestingStatsResponse, error) {
	if !crypto.IsValidVegaPubKey(req.PartyId) {
		return nil, formatE(ErrInvalidPartyID)
	}

	stats, err := t.vestingStats.GetByPartyID(ctx, req.PartyId)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.GetPartyVestingStatsResponse{
		PartyId:               stats.PartyID.String(),
		EpochSeq:              stats.AtEpoch,
		RewardBonusMultiplier: stats.RewardBonusMultiplier.String(),
		QuantumBalance:        stats.QuantumBalance.String(),
	}, nil
}

func (t *TradingDataServiceV2) GetVestingBalancesSummary(
	ctx context.Context,
	req *v2.GetVestingBalancesSummaryRequest,
) (*v2.GetVestingBalancesSummaryResponse, error) {
	if !crypto.IsValidVegaPubKey(req.PartyId) {
		return nil, formatE(ErrInvalidPartyID)
	}

	var assetId *entities.AssetID
	if req.AssetId != nil {
		if !crypto.IsValidVegaID(*req.AssetId) {
			return nil, formatE(ErrInvalidAssetID)
		}
		assetId = ptr.From(entities.AssetID(*req.AssetId))
	}

	// then get separately both things
	vestingBalances, err := t.partyVestingBalances.Get(
		ctx, ptr.From(entities.PartyID(req.PartyId)), assetId,
	)
	if err != nil {
		return nil, formatE(err)
	}

	lockedBalances, err := t.partyLockedBalances.Get(
		ctx, ptr.From(entities.PartyID(req.PartyId)), assetId,
	)
	if err != nil {
		return nil, formatE(err)
	}

	var (
		outVesting = make([]*eventspb.PartyVestingBalance, 0, len(vestingBalances))
		outLocked  = make([]*eventspb.PartyLockedBalance, 0, len(lockedBalances))
		epoch      *uint64
		setEpoch   = func(e uint64) {
			if epoch == nil {
				epoch = ptr.From(e)
				return
			}
			if *epoch < e {
				epoch = ptr.From(e)
			}
		}
	)

	for _, v := range vestingBalances {
		setEpoch(v.AtEpoch)
		outVesting = append(
			outVesting, &eventspb.PartyVestingBalance{
				Asset:   v.AssetID.String(),
				Balance: v.Balance.String(),
			},
		)
	}

	for _, v := range lockedBalances {
		setEpoch(v.AtEpoch)
		outLocked = append(
			outLocked, &eventspb.PartyLockedBalance{
				Asset:      v.AssetID.String(),
				UntilEpoch: v.UntilEpoch,
				Balance:    v.Balance.String(),
			},
		)
	}

	return &v2.GetVestingBalancesSummaryResponse{
		PartyId:         req.PartyId,
		EpochSeq:        epoch,
		VestingBalances: outVesting,
		LockedBalances:  outLocked,
	}, nil
}

func (t *TradingDataServiceV2) GetPartyActivityStreak(ctx context.Context, req *v2.GetPartyActivityStreakRequest) (*v2.GetPartyActivityStreakResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetPartyActivityStreak")()

	if !crypto.IsValidVegaPubKey(req.PartyId) {
		return nil, formatE(ErrInvalidPartyID)
	}

	activityStreak, err := t.partyActivityStreak.Get(ctx, entities.PartyID(req.PartyId), req.Epoch)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.GetPartyActivityStreakResponse{
		ActivityStreak: activityStreak.ToProto(),
	}, nil
}

func (t *TradingDataServiceV2) ListFundingPayments(ctx context.Context, req *v2.ListFundingPaymentsRequest) (*v2.ListFundingPaymentsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListFundingPayments")()

	if !crypto.IsValidVegaPubKey(req.PartyId) {
		return nil, formatE(ErrInvalidPartyID)
	}

	var marketID *entities.MarketID
	if req.MarketId != nil {
		if !crypto.IsValidVegaPubKey(*req.MarketId) {
			return nil, formatE(ErrInvalidMarketID)
		}
		marketID = ptr.From(entities.MarketID(*req.MarketId))
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	fundingPayments, pageInfo, err := t.fundingPaymentService.List(
		ctx, entities.PartyID(req.PartyId), marketID, pagination)
	if err != nil {
		return nil, formatE(err)
	}

	edges, err := makeEdges[*v2.FundingPaymentEdge](fundingPayments)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.FundingPaymentConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListFundingPaymentsResponse{
		FundingPayments: connection,
	}, nil
}

// ListAccounts lists accounts matching the request.
func (t *TradingDataServiceV2) ListAccounts(ctx context.Context, req *v2.ListAccountsRequest) (*v2.ListAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAccountsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	if req.Filter != nil {
		if err := VegaIDsSlice(req.Filter.MarketIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more market id is invalid"))
		}

		if err := VegaIDsSlice(req.Filter.PartyIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more party id is invalid"))
		}
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
func (t *TradingDataServiceV2) ObserveAccounts(req *v2.ObserveAccountsRequest, srv v2.TradingDataService_ObserveAccountsServer) error {
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

func (t *TradingDataServiceV2) sendAccountsSnapshot(ctx context.Context, req *v2.ObserveAccountsRequest, srv v2.TradingDataService_ObserveAccountsServer) error {
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
func (t *TradingDataServiceV2) Info(_ context.Context, _ *v2.InfoRequest) (*v2.InfoResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("InfoV2")()

	return &v2.InfoResponse{
		Version:    version.Get(),
		CommitHash: version.GetCommitHash(),
	}, nil
}

// ListLedgerEntries returns a list of ledger entries matching the request.
func (t *TradingDataServiceV2) ListLedgerEntries(ctx context.Context, req *v2.ListLedgerEntriesRequest) (*v2.ListLedgerEntriesResponse, error) {
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
		if errors.Is(err, sqlstore.ErrLedgerEntryFilterForParty) {
			return nil, formatE(ErrInvalidFilter, err)
		}
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
func (t *TradingDataServiceV2) ExportLedgerEntries(req *v2.ExportLedgerEntriesRequest, stream v2.TradingDataService_ExportLedgerEntriesServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ExportLedgerEntriesV2")()

	if len(req.PartyId) <= 0 {
		return formatE(ErrMissingPartyID)
	}
	if !crypto.IsValidVegaID(req.PartyId) {
		return formatE(ErrInvalidPartyID)
	}

	if req.AssetId != nil && !crypto.IsValidVegaID(*req.AssetId) {
		return formatE(ErrInvalidAssetID)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	timeFormat := strings.ReplaceAll(time.RFC3339, ":", "_")

	var startDateStr, endDateStr string
	if dateRange.Start != nil {
		startDateStr = "_" + dateRange.Start.Format(timeFormat)
	}
	if dateRange.End != nil {
		endDateStr = "-" + dateRange.End.Format(timeFormat)
	}

	assetStr := ""
	if req.AssetId != nil {
		assetStr = fmt.Sprintf("_%s", *req.AssetId)
	}
	header := metadata.Pairs(
		"Content-Disposition",
		fmt.Sprintf("attachment;filename=ledger_entries_%s%s_%s%s.csv",
			req.PartyId, assetStr, startDateStr, endDateStr))
	if err := stream.SendHeader(header); err != nil {
		return formatE(ErrSendingGRPCHeader, err)
	}

	httpWriter := &httpBodyWriter{chunkSize: httpBodyChunkSize, contentType: "text/csv", buf: &bytes.Buffer{}, stream: stream}
	defer httpWriter.Close()

	if err := t.ledgerService.Export(stream.Context(), req.PartyId, req.AssetId, dateRange, httpWriter); err != nil {
		return formatE(ErrLedgerServiceExport, err)
	}

	return nil
}

// ListBalanceChanges returns a list of balance changes matching the request.
func (t *TradingDataServiceV2) ListBalanceChanges(ctx context.Context, req *v2.ListBalanceChangesRequest) (*v2.ListBalanceChangesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListBalanceChangesV2")()

	if req.Filter != nil {
		if err := VegaIDsSlice(req.Filter.MarketIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more market id is invalid"))
		}

		if err := VegaIDsSlice(req.Filter.PartyIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more party id is invalid"))
		}
	}

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

// ObserveMarketsDepth subscribes to market depth updates.
func (t *TradingDataServiceV2) ObserveMarketsDepth(req *v2.ObserveMarketsDepthRequest, srv v2.TradingDataService_ObserveMarketsDepthServer) error {
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
func (t *TradingDataServiceV2) ObserveMarketsDepthUpdates(req *v2.ObserveMarketsDepthUpdatesRequest, srv v2.TradingDataService_ObserveMarketsDepthUpdatesServer) error {
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
func (t *TradingDataServiceV2) ObserveMarketsData(req *v2.ObserveMarketsDataRequest, srv v2.TradingDataService_ObserveMarketsDataServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	for _, marketID := range req.MarketIds {
		if !t.marketExistsForID(ctx, marketID) {
			return formatE(errors.Wrapf(ErrMalformedRequest, "no market found for id: %s", marketID))
		}
	}

	ch, ref := t.MarketDataService.ObserveMarketData(ctx, t.config.StreamRetries, req.MarketIds)

	return observeBatch(ctx, t.log, "MarketsData", ch, ref, func(marketData []*entities.MarketData) error {
		out := make([]*vega.MarketData, 0, len(marketData))
		for _, v := range marketData {
			out = append(out, v.ToProto())
		}
		return srv.Send(&v2.ObserveMarketsDataResponse{MarketData: out})
	})
}

// GetLatestMarketData returns the latest market data for a given market.
func (t *TradingDataServiceV2) GetLatestMarketData(ctx context.Context, req *v2.GetLatestMarketDataRequest) (*v2.GetLatestMarketDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLatestMarketData")()

	md, err := t.MarketDataService.GetMarketDataByID(ctx, req.MarketId)
	if err != nil {
		return nil, formatE(ErrMarketServiceGetMarketData, err)
	}

	return &v2.GetLatestMarketDataResponse{
		MarketData: md.ToProto(),
	}, nil
}

// ListLatestMarketData returns the latest market data for every market.
func (t *TradingDataServiceV2) ListLatestMarketData(ctx context.Context, _ *v2.ListLatestMarketDataRequest) (*v2.ListLatestMarketDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListLatestMarketData")()

	mds, err := t.MarketDataService.GetMarketsData(ctx)
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
func (t *TradingDataServiceV2) GetLatestMarketDepth(ctx context.Context, req *v2.GetLatestMarketDepthRequest) (*v2.GetLatestMarketDepthResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLatestMarketDepth")()

	ts, err := t.tradeService.GetLastTradeByMarket(ctx, req.MarketId)
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
func (t *TradingDataServiceV2) GetMarketDataHistoryByID(ctx context.Context, req *v2.GetMarketDataHistoryByIDRequest) (*v2.GetMarketDataHistoryByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMarketDataHistoryV2")()

	marketData, err := t.handleGetMarketDataHistoryWithCursorPagination(ctx, req)
	if err != nil {
		return marketData, formatE(ErrMarketServiceGetMarketDataHistory, err)
	}
	return marketData, nil
}

func (t *TradingDataServiceV2) handleGetMarketDataHistoryWithCursorPagination(ctx context.Context, req *v2.GetMarketDataHistoryByIDRequest) (*v2.GetMarketDataHistoryByIDResponse, error) {
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidPagination, err.Error())
	}

	var startTime, endTime *time.Time
	if req.StartTimestamp != nil {
		startTime = ptr.From(time.Unix(0, *req.StartTimestamp))
	}
	if req.EndTimestamp != nil {
		endTime = ptr.From(time.Unix(0, *req.EndTimestamp))
	}

	history, pageInfo, err := t.MarketDataService.GetHistoricMarketData(ctx, req.MarketId, startTime, endTime, pagination)
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

// GetNetworkLimits returns the latest network limits.
func (t *TradingDataServiceV2) GetNetworkLimits(ctx context.Context, _ *v2.GetNetworkLimitsRequest) (*v2.GetNetworkLimitsResponse, error) {
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
func (t *TradingDataServiceV2) ListCandleData(ctx context.Context, req *v2.ListCandleDataRequest) (*v2.ListCandleDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListCandleDataV2")()

	var from, to *time.Time
	if req.FromTimestamp != 0 {
		from = ptr.From(vegatime.UnixNano(req.FromTimestamp))
	}

	if req.ToTimestamp != 0 {
		to = ptr.From(vegatime.UnixNano(req.ToTimestamp))
	}

	if to != nil {
		if from != nil && to.Before(*from) {
			return nil, formatE(ErrInvalidCandleTimestampsRange)
		}
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
func (t *TradingDataServiceV2) ObserveCandleData(req *v2.ObserveCandleDataRequest, srv v2.TradingDataService_ObserveCandleDataServer) error {
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
func (t *TradingDataServiceV2) ListCandleIntervals(ctx context.Context, req *v2.ListCandleIntervalsRequest) (*v2.ListCandleIntervalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListCandleIntervals")()

	if len(req.MarketId) <= 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	if !crypto.IsValidVegaID(req.MarketId) {
		return nil, formatE(ErrInvalidMarketID)
	}

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
func (t *TradingDataServiceV2) ListERC20MultiSigSignerAddedBundles(ctx context.Context, req *v2.ListERC20MultiSigSignerAddedBundlesRequest) (*v2.ListERC20MultiSigSignerAddedBundlesResponse, error) {
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
func (t *TradingDataServiceV2) ListERC20MultiSigSignerRemovedBundles(ctx context.Context, req *v2.ListERC20MultiSigSignerRemovedBundlesRequest) (*v2.ListERC20MultiSigSignerRemovedBundlesResponse, error) {
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
func (t *TradingDataServiceV2) GetERC20SetAssetLimitsBundle(ctx context.Context, req *v2.GetERC20SetAssetLimitsBundleRequest) (*v2.GetERC20SetAssetLimitsBundleResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20SetAssetLimitsBundleV2")()

	if len(req.ProposalId) == 0 {
		return nil, formatE(ErrMissingProposalID)
	}

	if !crypto.IsValidVegaID(req.ProposalId) {
		return nil, formatE(ErrInvalidProposalID)
	}

	proposal, err := t.governanceService.GetProposalByIDWithoutBatch(ctx, req.ProposalId)
	if err != nil {
		return nil, formatE(ErrGovernanceServiceGet, err)
	}

	if proposal.IsBatch() {
		return nil, formatE(errors.New("can not get bundle for batch proposal"))
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

	asset, err := t.AssetService.GetByID(ctx, proposal.Terms.GetUpdateAsset().AssetId)
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

func (t *TradingDataServiceV2) GetERC20ListAssetBundle(ctx context.Context, req *v2.GetERC20ListAssetBundleRequest) (*v2.GetERC20ListAssetBundleResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20ListAssetBundleV2")()

	if len(req.AssetId) == 0 {
		return nil, formatE(ErrMissingAssetID)
	}

	if !crypto.IsValidVegaID(req.AssetId) {
		return nil, formatE(ErrInvalidAssetID)
	}

	asset, err := t.AssetService.GetByID(ctx, req.AssetId)
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
		return nil, formatE(ErrInvalidAssetID, errors.Wrapf(err, "assetID: %s", req.AssetId))
	}

	return &v2.GetERC20ListAssetBundleResponse{
		AssetSource: asset.ERC20Contract,
		Nonce:       nonce.String(),
		VegaAssetId: asset.ID.String(),
		Signatures:  entities.PackNodeSignatures(signatures),
	}, nil
}

// GetERC20WithdrawalApproval returns the signature bundle needed to approve a withdrawal on the ERC20 contract.
func (t *TradingDataServiceV2) GetERC20WithdrawalApproval(ctx context.Context, req *v2.GetERC20WithdrawalApprovalRequest) (*v2.GetERC20WithdrawalApprovalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetERC20WithdrawalApprovalV2")()

	if len(req.WithdrawalId) == 0 {
		return nil, formatE(ErrMissingWithdrawalID)
	}

	if !crypto.IsValidVegaID(req.WithdrawalId) {
		return nil, formatE(ErrInvalidWithdrawalID)
	}

	w, err := t.withdrawalService.GetByID(ctx, req.WithdrawalId)
	if err != nil {
		return nil, formatE(ErrWithdrawalServiceGet, err)
	}

	signatures, _, err := t.notaryService.GetByResourceID(ctx, req.WithdrawalId, entities.CursorPagination{})
	if err != nil {
		return nil, formatE(ErrNotaryServiceGetByResourceID, errors.Wrapf(err, "withdrawalID: %s", req.WithdrawalId))
	}

	assets, err := t.AssetService.GetAll(ctx)
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
func (t *TradingDataServiceV2) GetLastTrade(ctx context.Context, req *v2.GetLastTradeRequest) (*v2.GetLastTradeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLastTradeV2")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	if !crypto.IsValidVegaID(req.MarketId) {
		return nil, formatE(ErrInvalidMarketID)
	}

	trades, err := t.tradeService.GetLastTradeByMarket(ctx, req.MarketId)
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

type filterableIDs interface {
	entities.MarketID | entities.PartyID | entities.OrderID
}

func toEntityIDs[T filterableIDs](ids []string) []T {
	entityIDs := make([]T, len(ids))
	for i := range ids {
		entityIDs[i] = T(ids[i])
	}
	return entityIDs
}

// ListTrades lists trades by using a cursor based pagination model.
func (t *TradingDataServiceV2) ListTrades(ctx context.Context, req *v2.ListTradesRequest) (*v2.ListTradesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTradesV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	trades, pageInfo, err := t.tradeService.List(ctx,
		toEntityIDs[entities.MarketID](req.GetMarketIds()),
		toEntityIDs[entities.PartyID](req.GetPartyIds()),
		toEntityIDs[entities.OrderID](req.GetOrderIds()),
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
func (t *TradingDataServiceV2) ObserveTrades(req *v2.ObserveTradesRequest, srv v2.TradingDataService_ObserveTradesServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	tradesChan, ref := t.tradeService.Observe(ctx, t.config.StreamRetries, req.MarketIds, req.PartyIds)

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
func (t *TradingDataServiceV2) GetMarket(ctx context.Context, req *v2.GetMarketRequest) (*v2.GetMarketResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketByID_SQL")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	if !crypto.IsValidVegaID(req.MarketId) {
		return nil, formatE(ErrInvalidMarketID)
	}

	market, err := t.MarketsService.GetByID(ctx, req.MarketId)
	if err != nil {
		return nil, formatE(ErrMarketServiceGetByID, err)
	}

	return &v2.GetMarketResponse{
		Market: market.ToProto(),
	}, nil
}

// ListMarkets lists all markets.
func (t *TradingDataServiceV2) ListMarkets(ctx context.Context, req *v2.ListMarketsRequest) (*v2.ListMarketsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListMarketsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	includeSettled := true
	if req.IncludeSettled != nil {
		includeSettled = *req.IncludeSettled
	}

	markets, pageInfo, err := t.MarketsService.GetAllPaged(ctx, "", pagination, includeSettled)
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

// ListSuccessorMarkets returns the successor chain for a given market.
func (t *TradingDataServiceV2) ListSuccessorMarkets(ctx context.Context, req *v2.ListSuccessorMarketsRequest) (*v2.ListSuccessorMarketsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListSuccessorMarkets")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	if !crypto.IsValidVegaID(req.MarketId) {
		return nil, formatE(ErrInvalidMarketID)
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	markets, pageInfo, err := t.MarketsService.ListSuccessorMarkets(ctx, req.MarketId, req.IncludeFullHistory, pagination)
	if err != nil {
		return nil, formatE(ErrMarketServiceGetAllPaged, err)
	}

	edges, err := makeEdges[*v2.SuccessorMarketEdge](markets)
	if err != nil {
		return nil, formatE(err)
	}

	for i := range edges {
		for j := range edges[i].Node.Proposals {
			proposalID := edges[i].Node.Proposals[j].Proposal.Id
			node := edges[i].Node.Proposals[j]
			node.Yes, node.No, err = t.getVotesByProposal(ctx, proposalID)
			if err != nil {
				return nil, formatE(ErrGovernanceServiceGetVotes, errors.Wrapf(err, "proposalID: %s", proposalID))
			}
		}
	}

	marketsConnection := &v2.SuccessorMarketConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListSuccessorMarketsResponse{
		SuccessorMarkets: marketsConnection,
	}, nil
}

// List all Positions.
//
// Deprecated: Use ListAllPositions instead.
func (t *TradingDataServiceV2) ListPositions(ctx context.Context, req *v2.ListPositionsRequest) (*v2.ListPositionsResponse, error) {
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

// ListAllPositions lists all positions.
func (t *TradingDataServiceV2) ListAllPositions(ctx context.Context, req *v2.ListAllPositionsRequest) (*v2.ListAllPositionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAllPositions")()

	if req.Filter != nil {
		if err := VegaIDsSlice(req.Filter.MarketIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more market id is invalid"))
		}

		if err := VegaIDsSlice(req.Filter.PartyIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more party id is invalid"))
		}
	}

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
func (t *TradingDataServiceV2) ObservePositions(req *v2.ObservePositionsRequest, srv v2.TradingDataService_ObservePositionsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	if err := t.sendPositionsSnapshot(ctx, req, srv); err != nil {
		if !errors.Is(err, entities.ErrNotFound) {
			return formatE(ErrPositionServiceSendSnapshot, err)
		}
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

func (t *TradingDataServiceV2) sendPositionsSnapshot(ctx context.Context, req *v2.ObservePositionsRequest, srv v2.TradingDataService_ObservePositionsServer) error {
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
func (t *TradingDataServiceV2) GetParty(ctx context.Context, req *v2.GetPartyRequest) (*v2.GetPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetParty")()

	if len(req.PartyId) == 0 {
		return nil, formatE(ErrMissingPartyID)
	}

	if req.PartyId != networkPartyID && !crypto.IsValidVegaID(req.PartyId) {
		return nil, formatE(ErrInvalidPartyID)
	}

	party, err := t.partyService.GetByID(ctx, req.PartyId)
	if err != nil {
		return nil, formatE(ErrPartyServiceGetByID, err)
	}

	return &v2.GetPartyResponse{
		Party: party.ToProto(),
	}, nil
}

// ListParties lists Parties.
func (t *TradingDataServiceV2) ListParties(ctx context.Context, req *v2.ListPartiesRequest) (*v2.ListPartiesResponse, error) {
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

func (t *TradingDataServiceV2) ListPartiesProfiles(ctx context.Context, req *v2.ListPartiesProfilesRequest) (*v2.ListPartiesProfilesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListPartiesProfiles")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	parties, pageInfo, err := t.partyService.ListProfiles(ctx, req.Parties, pagination)
	if err != nil {
		return nil, formatE(ErrPartyServiceListProfiles, err)
	}

	edges, err := makeEdges[*v2.PartyProfileEdge](parties)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.PartiesProfilesConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListPartiesProfilesResponse{
		Profiles: connection,
	}, nil
}

// ListMarginLevels lists MarginLevels.
func (t *TradingDataServiceV2) ListMarginLevels(ctx context.Context, req *v2.ListMarginLevelsRequest) (*v2.ListMarginLevelsResponse, error) {
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
func (t *TradingDataServiceV2) ObserveMarginLevels(req *v2.ObserveMarginLevelsRequest, srv v2.TradingDataService_ObserveMarginLevelsServer) error {
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

// ListRewards lists Rewards.
func (t *TradingDataServiceV2) ListRewards(ctx context.Context, req *v2.ListRewardsRequest) (*v2.ListRewardsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListRewardsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	rewards, pageInfo, err := t.rewardService.GetByCursor(ctx, &req.PartyId, req.AssetId, req.FromEpoch, req.ToEpoch, pagination, req.TeamId, req.GameId)
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
func (t *TradingDataServiceV2) ListRewardSummaries(ctx context.Context, req *v2.ListRewardSummariesRequest) (*v2.ListRewardSummariesResponse, error) {
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
func (t *TradingDataServiceV2) ListEpochRewardSummaries(ctx context.Context, req *v2.ListEpochRewardSummariesRequest) (*v2.ListEpochRewardSummariesResponse, error) {
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

// GetDeposit gets a deposit by ID.
func (t *TradingDataServiceV2) GetDeposit(ctx context.Context, req *v2.GetDepositRequest) (*v2.GetDepositResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetDepositV2")()

	if len(req.Id) == 0 {
		return nil, formatE(ErrMissingDepositID)
	}

	if !crypto.IsValidVegaPubKey(req.Id) {
		return nil, formatE(ErrNotAValidVegaID)
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
func (t *TradingDataServiceV2) ListDeposits(ctx context.Context, req *v2.ListDepositsRequest) (*v2.ListDepositsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListDepositsV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	if len(req.PartyId) > 0 && req.PartyId != networkPartyID && !crypto.IsValidVegaPubKey(req.PartyId) {
		return nil, formatE(ErrInvalidPartyID)
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
func (t *TradingDataServiceV2) GetWithdrawal(ctx context.Context, req *v2.GetWithdrawalRequest) (*v2.GetWithdrawalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetWithdrawalV2")()

	if len(req.Id) == 0 {
		return nil, formatE(ErrMissingWithdrawalID)
	}

	if !crypto.IsValidVegaPubKey(req.Id) {
		return nil, formatE(ErrInvalidWithdrawalID)
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
func (t *TradingDataServiceV2) ListWithdrawals(ctx context.Context, req *v2.ListWithdrawalsRequest) (*v2.ListWithdrawalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListWithdrawalsV2")()

	if len(req.PartyId) > 0 && req.PartyId != networkPartyID && !crypto.IsValidVegaPubKey(req.PartyId) {
		return nil, formatE(ErrInvalidPartyID)
	}

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
func (t *TradingDataServiceV2) GetAsset(ctx context.Context, req *v2.GetAssetRequest) (*v2.GetAssetResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetAssetV2")()

	if len(req.AssetId) == 0 {
		return nil, formatE(ErrMissingAssetID)
	}

	// TODO: VOTE is a special case used for system tests. Remove this once the system tests are updated to remove the VOTE asset.
	if req.AssetId != "VOTE" && !crypto.IsValidVegaPubKey(req.AssetId) {
		return nil, formatE(ErrInvalidAssetID)
	}

	asset, err := t.AssetService.GetByID(ctx, req.AssetId)
	if err != nil {
		return nil, formatE(ErrAssetServiceGetByID, err)
	}

	return &v2.GetAssetResponse{
		Asset: asset.ToProto(),
	}, nil
}

// ListAssets gets all assets. If an asset ID is provided, it will return a single asset.
func (t *TradingDataServiceV2) ListAssets(ctx context.Context, req *v2.ListAssetsRequest) (*v2.ListAssetsResponse, error) {
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

func (t *TradingDataServiceV2) getSingleAsset(ctx context.Context, assetID string) (*v2.ListAssetsResponse, error) {
	asset, err := t.AssetService.GetByID(ctx, assetID)
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

func (t *TradingDataServiceV2) getAllAssets(ctx context.Context, p *v2.Pagination) (*v2.ListAssetsResponse, error) {
	pagination, err := entities.CursorPaginationFromProto(p)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidPagination, err.Error())
	}

	assets, pageInfo, err := t.AssetService.GetAllWithCursorPagination(ctx, pagination)
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
func (t *TradingDataServiceV2) GetOracleSpec(ctx context.Context, req *v2.GetOracleSpecRequest) (*v2.GetOracleSpecResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOracleSpecV2")()

	if len(req.OracleSpecId) == 0 {
		return nil, formatE(ErrMissingOracleSpecID)
	}

	if !crypto.IsValidVegaPubKey(req.OracleSpecId) {
		return nil, formatE(ErrInvalidOracleSpecID)
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
func (t *TradingDataServiceV2) ListOracleSpecs(ctx context.Context, req *v2.ListOracleSpecsRequest) (*v2.ListOracleSpecsResponse, error) {
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
func (t *TradingDataServiceV2) ListOracleData(ctx context.Context, req *v2.ListOracleDataRequest) (*v2.ListOracleDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOracleDataConnectionV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		data     []entities.OracleData
		pageInfo entities.PageInfo
	)

	oracleSpecID := ptr.UnBox(req.OracleSpecId)
	data, pageInfo, err = t.oracleDataService.ListOracleData(ctx, oracleSpecID, pagination)

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
func (t *TradingDataServiceV2) ListLiquidityProvisions(ctx context.Context, req *v2.ListLiquidityProvisionsRequest) (*v2.ListLiquidityProvisionsResponse, error) {
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

	provisions := make([]entities.LiquidityProvision, len(lps))
	for i, lp := range lps {
		provisions[i] = entities.LiquidityProvision{
			ID:               lp.ID,
			PartyID:          lp.PartyID,
			CreatedAt:        lp.CreatedAt,
			UpdatedAt:        lp.UpdatedAt,
			MarketID:         lp.MarketID,
			CommitmentAmount: lp.CommitmentAmount,
			Fee:              lp.Fee,
			Sells:            lp.Sells,
			Buys:             lp.Buys,
			Version:          lp.Version,
			Status:           lp.Status,
			Reference:        lp.Reference,
			TxHash:           lp.TxHash,
			VegaTime:         lp.VegaTime,
		}
	}

	edges, err := makeEdges[*v2.LiquidityProvisionsEdge](provisions)
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

// ListAllLiquidityProvisions gets a list of liquidity provisions for a given market. This is similar to the list liquidity provisions API
// but returns a current and pending liquidity provision in the event a provision has been updated by the provider
// but the updated provision will not be active until the next epoch.
func (t *TradingDataServiceV2) ListAllLiquidityProvisions(ctx context.Context, req *v2.ListAllLiquidityProvisionsRequest) (*v2.ListAllLiquidityProvisionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetLiquidityProvisionsWithPending")()

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

	edges, err := makeEdges[*v2.LiquidityProvisionWithPendingEdge](lps)
	if err != nil {
		return nil, formatE(err)
	}

	liquidityProvisionConnection := &v2.LiquidityProvisionsWithPendingConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListAllLiquidityProvisionsResponse{
		LiquidityProvisions: liquidityProvisionConnection,
	}, nil
}

// ObserveLiquidityProvisions subscribes to liquidity provisions.
func (t *TradingDataServiceV2) ObserveLiquidityProvisions(req *v2.ObserveLiquidityProvisionsRequest, srv v2.TradingDataService_ObserveLiquidityProvisionsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	lpCh, ref := t.liquidityProvisionService.ObserveLiquidityProvisions(ctx, t.config.StreamRetries, req.MarketId, req.PartyId)

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

func (t *TradingDataServiceV2) ListLiquidityProviders(ctx context.Context, req *v2.ListLiquidityProvidersRequest) (
	*v2.ListLiquidityProvidersResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListLiquidityProviders")

	if req.MarketId == nil && req.PartyId == nil {
		return nil, formatE(ErrMalformedRequest, errors.New("marketId or partyId or both must be provided"))
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		marketID *entities.MarketID
		partyID  *entities.PartyID
	)

	if req.MarketId != nil {
		marketID = ptr.From(entities.MarketID(ptr.UnBox(req.MarketId)))
	}

	if req.PartyId != nil {
		partyID = ptr.From(entities.PartyID(ptr.UnBox(req.PartyId)))
	}

	providers, pageInfo, err := t.liquidityProvisionService.ListProviders(ctx, partyID, marketID, pagination)
	if err != nil {
		return nil, formatE(ErrLiquidityProvisionServiceGetProviders, errors.Wrapf(err,
			"marketID: %s", marketID))
	}

	if len(providers) == 0 {
		return &v2.ListLiquidityProvidersResponse{
			LiquidityProviders: &v2.LiquidityProviderConnection{
				Edges:    []*v2.LiquidityProviderEdge{},
				PageInfo: pageInfo.ToProto(),
			},
		}, nil
	}

	edges, err := makeEdges[*v2.LiquidityProviderEdge](providers)
	if err != nil {
		return nil, formatE(err)
	}

	conn := &v2.LiquidityProviderConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListLiquidityProvidersResponse{
		LiquidityProviders: conn,
	}, nil
}

func (t *TradingDataServiceV2) ListPaidLiquidityFees(ctx context.Context, req *v2.ListPaidLiquidityFeesRequest) (
	*v2.ListPaidLiquidityFeesResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListPaidLiquidityFees")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var marketID *entities.MarketID
	if req.MarketId != nil {
		marketID = ptr.From(entities.MarketID(*req.MarketId))
	}

	var assetID *entities.AssetID
	if req.AssetId != nil {
		assetID = ptr.From(entities.AssetID(*req.AssetId))
	}

	stats, pageInfo, err := t.paidLiquidityFeesStatsService.List(ctx, marketID, assetID, req.EpochSeq, req.PartyIds, pagination)
	if err != nil {
		return nil, formatE(ErrListPaidLiquidityFees, err)
	}

	edges, err := makeEdges[*v2.PaidLiquidityFeesEdge](stats)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.ListPaidLiquidityFeesResponse{
		PaidLiquidityFees: &v2.PaidLiquidityFeesConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}

// GetGovernanceData gets governance data.
func (t *TradingDataServiceV2) GetGovernanceData(ctx context.Context, req *v2.GetGovernanceDataRequest) (*v2.GetGovernanceDataResponse, error) {
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
func (t *TradingDataServiceV2) ListGovernanceData(ctx context.Context, req *v2.ListGovernanceDataRequest) (*v2.ListGovernanceDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListGovernanceDataV2")()

	var state *entities.ProposalState
	if req.ProposalState != nil {
		state = ptr.From(entities.ProposalState(*req.ProposalState))
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		proposal  entities.Proposal
		proposals []entities.Proposal
		pageInfo  entities.PageInfo
	)

	if req.ProposalReference != nil {
		proposal, err = t.governanceService.GetProposalByReference(ctx, *req.ProposalReference)
		if err != nil {
			return nil, formatE(ErrGovernanceServiceGet,
				errors.Wrapf(err, "proposalReference: %s", ptr.UnBox(req.ProposalReference)))
		}
		proposals = []entities.Proposal{proposal}
		pageInfo.StartCursor = proposal.Cursor().Encode()
		pageInfo.EndCursor = proposal.Cursor().Encode()
	} else {
		proposals, pageInfo, err = t.governanceService.GetProposals(
			ctx,
			state,
			req.ProposerPartyId,
			(*entities.ProposalType)(req.ProposalType),
			pagination,
		)
		if err != nil {
			return nil, formatE(ErrGovernanceServiceGetProposals, errors.Wrapf(err, "ProposerPartyId: %s", ptr.UnBox(req.ProposerPartyId)))
		}
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

		if len(edges[i].Node.Proposals) > 0 {
			edges[i].Node.ProposalType = vega.GovernanceData_TYPE_BATCH
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

func (t *TradingDataServiceV2) getVotesByProposal(ctx context.Context, proposalID string) (yesVotes, noVotes []*vega.Vote, err error) {
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

// ListVotes gets all Votes.
func (t *TradingDataServiceV2) ListVotes(ctx context.Context, req *v2.ListVotesRequest) (*v2.ListVotesResponse, error) {
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

func (t *TradingDataServiceV2) GetTransfer(ctx context.Context, req *v2.GetTransferRequest) (*v2.GetTransferResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetTransferV2")()
	if len(req.TransferId) == 0 {
		return nil, formatE(ErrMissingTransferID)
	}
	transfer, err := t.transfersService.GetByID(ctx, req.TransferId)
	if err != nil {
		return nil, formatE(err)
	}
	tp, err := transfer.ToProto(ctx, t.accountService)
	if err != nil {
		return nil, formatE(err)
	}
	fees := make([]*eventspb.TransferFees, 0, len(transfer.Fees))
	for _, f := range transfer.Fees {
		fees = append(fees, f.ToProto())
	}
	return &v2.GetTransferResponse{
		TransferNode: &v2.TransferNode{
			Transfer: tp,
			Fees:     fees,
		},
	}, nil
}

// ListTransfers lists transfers using cursor pagination. If a pubkey is provided, it will list transfers for that pubkey.
func (t *TradingDataServiceV2) ListTransfers(ctx context.Context, req *v2.ListTransfersRequest) (*v2.ListTransfersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTransfersV2")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		transfers []entities.TransferDetails
		pageInfo  entities.PageInfo
		isReward  bool
	)
	if req.IsReward != nil {
		isReward = *req.IsReward
	}

	filters := sqlstore.ListTransfersFilters{
		FromEpoch: req.FromEpoch,
		ToEpoch:   req.ToEpoch,
	}
	if req.Status != nil {
		filters.Status = ptr.From(entities.TransferStatus(*req.Status))
	}
	if req.Scope != nil {
		filters.Scope = ptr.From(entities.TransferScope(*req.Scope))
	}
	if req.GameId != nil {
		filters.GameID = ptr.From(entities.GameID(*req.GameId))
	}

	if req.Pubkey == nil {
		if !isReward {
			transfers, pageInfo, err = t.transfersService.GetAll(ctx, pagination, filters)
		} else {
			transfers, pageInfo, err = t.transfersService.GetAllRewards(ctx, pagination, filters)
		}
	} else {
		if isReward && req.Direction != v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_FROM {
			err = errors.Errorf("invalid transfer direction for reward transfers: %v", req.Direction)
			return nil, formatE(ErrTransferServiceGet, errors.Wrapf(err, "pubkey: %s", ptr.UnBox(req.Pubkey)))
		}
		switch req.Direction {
		case v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_FROM:
			if isReward {
				transfers, pageInfo, err = t.transfersService.GetRewardTransfersFromParty(ctx, pagination, filters, entities.PartyID(*req.Pubkey))
			} else {
				transfers, pageInfo, err = t.transfersService.GetTransfersFromParty(ctx, pagination, filters, entities.PartyID(*req.Pubkey))
			}
		case v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO:
			transfers, pageInfo, err = t.transfersService.GetTransfersToParty(ctx, pagination, filters, entities.PartyID(*req.Pubkey))
		case v2.TransferDirection_TRANSFER_DIRECTION_TRANSFER_TO_OR_FROM:
			transfers, pageInfo, err = t.transfersService.GetTransfersToOrFromParty(ctx, pagination, filters, entities.PartyID(*req.Pubkey))
		default:
			err = errors.Errorf("unknown transfer direction: %v", req.Direction)
		}
	}
	if err != nil {
		t.log.Error("Something went wrong listing transfers", logging.Error(err))
		return nil, formatE(ErrTransferServiceGet, errors.Wrapf(err, "pubkey: %s", ptr.UnBox(req.Pubkey)))
	}

	edges, err := makeEdges[*v2.TransferEdge](transfers, ctx, t.accountService)
	if err != nil {
		t.log.Error("Something went wrong making transfer edges", logging.Error(err))
		return nil, formatE(err)
	}

	return &v2.ListTransfersResponse{Transfers: &v2.TransferConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}}, nil
}

// GetOrder gets an order by ID.
func (t *TradingDataServiceV2) GetOrder(ctx context.Context, req *v2.GetOrderRequest) (*v2.GetOrderResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetOrderV2")()

	if len(req.OrderId) == 0 {
		return nil, formatE(ErrMissingOrderID)
	}

	if !crypto.IsValidVegaID(req.OrderId) {
		return nil, formatE(ErrInvalidOrderID)
	}

	if req.Version != nil && *req.Version <= 0 {
		return nil, formatE(ErrNegativeOrderVersion)
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
func (t *TradingDataServiceV2) ListOrders(ctx context.Context, req *v2.ListOrdersRequest) (*v2.ListOrdersResponse, error) {
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
		if err := VegaIDsSlice(req.Filter.MarketIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more market id is invalid"))
		}

		if err := VegaIDsSlice(req.Filter.PartyIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more party id is invalid"))
		}
	}

	orders, pageInfo, err := t.orderService.ListOrders(ctx, pagination, filter)
	if err != nil {
		if errors.Is(err, sqlstore.ErrLastPaginationNotSupported) {
			return nil, formatE(ErrLastPaginationNotSupported, err)
		}

		return nil, formatE(err)
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
func (t *TradingDataServiceV2) ListOrderVersions(ctx context.Context, req *v2.ListOrderVersionsRequest) (*v2.ListOrderVersionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOrderVersionsV2")()

	if len(req.OrderId) == 0 {
		return nil, formatE(ErrMissingOrderID)
	}

	if !crypto.IsValidVegaID(req.OrderId) {
		return nil, formatE(ErrInvalidOrderID)
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

type VegaIDsSlice []string

func (s VegaIDsSlice) Ensure() error {
	for _, v := range s {
		if v != networkPartyID && !crypto.IsValidVegaPubKey(v) {
			return ErrInvalidPartyID
		}
	}

	return nil
}

// ObserveOrders subscribes to a stream of orders.
func (t *TradingDataServiceV2) ObserveOrders(req *v2.ObserveOrdersRequest, srv v2.TradingDataService_ObserveOrdersServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	if err := VegaIDsSlice(req.MarketIds).Ensure(); err != nil {
		return formatE(err, errors.New("one or more market id is invalid"))
	}

	if err := VegaIDsSlice(req.PartyIds).Ensure(); err != nil {
		return formatE(err, errors.New("one or more party id is invalid"))
	}

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

func (t *TradingDataServiceV2) sendOrdersSnapshot(ctx context.Context, req *v2.ObserveOrdersRequest, srv v2.TradingDataService_ObserveOrdersServer) error {
	orders, pageInfo, err := t.orderService.ListOrders(ctx, entities.CursorPagination{NewestFirst: true}, entities.OrderFilter{
		MarketIDs:        req.MarketIds,
		PartyIDs:         req.PartyIds,
		ExcludeLiquidity: ptr.UnBox(req.ExcludeLiquidity),
		LiveOnly:         true,
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
func (t *TradingDataServiceV2) ListDelegations(ctx context.Context, req *v2.ListDelegationsRequest) (*v2.ListDelegationsResponse, error) {
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

func (t *TradingDataServiceV2) marketExistsForID(ctx context.Context, marketID string) bool {
	_, err := t.MarketsService.GetByID(ctx, marketID)
	return err == nil
}

// GetNetworkData retrieve network data regarding the nodes of the network.
func (t *TradingDataServiceV2) GetNetworkData(ctx context.Context, _ *v2.GetNetworkDataRequest) (*v2.GetNetworkDataResponse, error) {
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
func (t *TradingDataServiceV2) GetNode(ctx context.Context, req *v2.GetNodeRequest) (*v2.GetNodeResponse, error) {
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
func (t *TradingDataServiceV2) ListNodes(ctx context.Context, req *v2.ListNodesRequest) (*v2.ListNodesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListNodesV2")()

	var (
		epoch entities.Epoch
		err   error
	)
	if req.EpochSeq == nil || *req.EpochSeq > math.MaxInt64 {
		epoch, err = t.epochService.GetCurrent(ctx)
	} else {
		epoch, err = t.epochService.Get(ctx, *req.EpochSeq)
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
func (t *TradingDataServiceV2) ListNodeSignatures(ctx context.Context, req *v2.ListNodeSignaturesRequest) (*v2.ListNodeSignaturesResponse, error) {
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
func (t *TradingDataServiceV2) GetEpoch(ctx context.Context, req *v2.GetEpochRequest) (*v2.GetEpochResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetEpochV2")()

	var (
		epoch entities.Epoch
		err   error
	)
	if req.GetId() > 0 {
		epoch, err = t.epochService.Get(ctx, req.GetId())
	} else if req.GetBlock() > 0 {
		epoch, err = t.epochService.GetByBlock(ctx, req.GetBlock())
	} else {
		epoch, err = t.epochService.GetCurrent(ctx)
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
func (t *TradingDataServiceV2) EstimateFee(ctx context.Context, req *v2.EstimateFeeRequest) (*v2.EstimateFeeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateFee SQL")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	if !crypto.IsValidVegaID(req.MarketId) {
		return nil, formatE(ErrInvalidMarketID)
	}

	if len(req.Price) == 0 {
		return nil, formatE(ErrMissingPrice)
	}

	fee, err := t.estimateFee(ctx, req.MarketId, req.Price, req.Size)
	if err != nil {
		return nil, formatE(ErrEstimateFee, err)
	}

	return &v2.EstimateFeeResponse{
		Fee: fee,
	}, nil
}

func (t *TradingDataServiceV2) scaleFromMarketToAssetPrice(
	ctx context.Context,
	mkt entities.Market,
	price *num.Uint,
) (*num.Uint, error) {
	priceFactor, err := t.getMarketPriceFactor(ctx, mkt)
	if err != nil {
		return nil, err
	}

	return price.Mul(price, priceFactor), nil
}

func (t *TradingDataServiceV2) scaleDecimalFromMarketToAssetPrice(
	price, priceFactor num.Decimal,
) num.Decimal {
	return price.Mul(priceFactor)
}

func (t *TradingDataServiceV2) scaleDecimalFromAssetToMarketPrice(
	price, priceFactor num.Decimal,
) num.Decimal {
	return price.Div(priceFactor)
}

func (t *TradingDataServiceV2) getMarketPriceFactor(
	ctx context.Context,
	mkt entities.Market,
) (*num.Uint, error) {
	assetID, err := mkt.ToProto().GetAsset()
	if err != nil {
		return nil, errors.Wrap(err, "getting asset from market")
	}

	asset, err := t.AssetService.GetByID(ctx, assetID)
	if err != nil {
		return nil, errors.Wrapf(ErrAssetServiceGetByID, "assetID: %s", assetID)
	}

	// scale the price if needed
	// price is expected in market decimal
	priceFactor := num.NewUint(1)
	if exp := asset.Decimals - mkt.DecimalPlaces; exp != 0 {
		priceFactor.Exp(num.NewUint(10), num.NewUint(uint64(exp)))
	}
	return priceFactor, nil
}

func (t *TradingDataServiceV2) estimateFee(
	ctx context.Context,
	market, priceS string,
	size uint64,
) (*vega.Fee, error) {
	mkt, err := t.MarketsService.GetByID(ctx, market)
	if err != nil {
		return nil, err
	}

	price, overflowed := num.UintFromString(priceS, 10)
	if overflowed {
		return nil, ErrInvalidOrderPrice
	}

	if price.IsNegative() || price.IsZero() {
		return nil, ErrInvalidOrderPrice
	}

	if size <= 0 {
		return nil, ErrInvalidOrderSize
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

func (t *TradingDataServiceV2) feeFactors(mkt entities.Market) (maker, infra, liquidity float64, err error) {
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
func (t *TradingDataServiceV2) EstimateMargin(ctx context.Context, req *v2.EstimateMarginRequest) (*v2.EstimateMarginResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateMargin SQL")()

	margin, err := t.estimateMargin(
		ctx, req.Side, req.Type, req.MarketId, req.PartyId, req.Price, req.Size)
	if err != nil {
		return nil, formatE(ErrEstimateMargin, err)
	}

	return &v2.EstimateMarginResponse{
		MarginLevels: margin,
	}, nil
}

func (t *TradingDataServiceV2) estimateMargin(
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
	rf, err := t.RiskFactorService.GetMarketRiskFactors(ctx, rMarket)
	if err != nil {
		return nil, err
	}

	mkt, err := t.MarketsService.GetByID(ctx, rMarket)
	if err != nil {
		return nil, err
	}

	mktData, err := t.MarketDataService.GetMarketDataByID(ctx, rMarket)
	if err != nil {
		return nil, err
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
	if price.IsNegative() || price.IsZero() {
		return nil, ErrInvalidOrderPrice
	}
	price, err = t.scaleFromMarketToAssetPrice(ctx, mkt, price)
	if err != nil {
		return nil, errors.Wrap(ErrScalingPriceFromMarketToAsset, err.Error())
	}

	if rSize <= 0 {
		return nil, ErrInvalidOrderSize
	}

	priceD = price.ToDecimal()

	mdpd := num.DecimalFromFloat(10).
		Pow(num.DecimalFromInt64(int64(mkt.PositionDecimalPlaces)))

	maintenanceMargin := num.DecimalFromFloat(float64(rSize)).
		Mul(f).Mul(priceD).Div(mdpd)

	return implyMarginLevels(maintenanceMargin, num.DecimalZero(), num.DecimalZero(), mkt.TradableInstrument.MarginCalculator.ScalingFactors, rParty, rMarket, asset, false), nil
}

func implyMarginLevels(maintenanceMargin, orderMargin, marginFactor num.Decimal, scalingFactors *vega.ScalingFactors, partyId, marketId, asset string, isolatedMarginMode bool) *vega.MarginLevels {
	marginLevels := &vega.MarginLevels{
		PartyId:                partyId,
		MarketId:               marketId,
		Asset:                  asset,
		Timestamp:              0,
		MaintenanceMargin:      maintenanceMargin.Round(0).String(),
		SearchLevel:            maintenanceMargin.Mul(num.DecimalFromFloat(scalingFactors.SearchLevel)).Round(0).String(),
		InitialMargin:          maintenanceMargin.Mul(num.DecimalFromFloat(scalingFactors.InitialMargin)).Round(0).String(),
		CollateralReleaseLevel: maintenanceMargin.Mul(num.DecimalFromFloat(scalingFactors.CollateralRelease)).Round(0).String(),
		MarginMode:             types.MarginModeCrossMargin,
	}
	if isolatedMarginMode {
		marginLevels.OrderMargin = orderMargin.Round(0).String()
		marginLevels.SearchLevel = "0"
		marginLevels.CollateralReleaseLevel = "0"
		marginLevels.MarginMode = types.MarginModeIsolatedMargin
		marginLevels.MarginFactor = marginFactor.String()
	}
	return marginLevels
}

func (t *TradingDataServiceV2) EstimatePosition(ctx context.Context, req *v2.EstimatePositionRequest) (*v2.EstimatePositionResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimatePosition")()

	if req.MarketId == "" {
		return nil, ErrEmptyMissingMarketID
	}

	marginAccountBalance, err := num.DecimalFromString(req.MarginAccountBalance)
	if err != nil {
		return nil, formatE(ErrPositionsInvalidAccountBalance, err)
	}
	orderAccountBalance, err := num.DecimalFromString(req.OrderMarginAccountBalance)
	if err != nil {
		return nil, formatE(ErrPositionsInvalidAccountBalance, err)
	}
	mkt, err := t.MarketsService.GetByID(ctx, req.MarketId)
	if err != nil {
		return nil, formatE(ErrMarketServiceGetByID, err)
	}
	collateralAvailable := marginAccountBalance
	crossMarginMode := req.MarginMode == types.MarginModeCrossMargin
	if crossMarginMode {
		generalAccountBalance, err := num.DecimalFromString(req.GeneralAccountBalance)
		if err != nil {
			return nil, formatE(ErrPositionsInvalidAccountBalance, err)
		}
		collateralAvailable = collateralAvailable.Add(generalAccountBalance).Add(orderAccountBalance)
	}

	dMarginFactor := num.DecimalZero()
	isolatedMarginMode := req.MarginMode == types.MarginModeIsolatedMargin
	if isolatedMarginMode {
		if req.MarginFactor == nil {
			return nil, formatE(ErrMissingMarginFactor, errors.New("margin factor are required with isolated margin"))
		}
		dMarginFactor, err = num.DecimalFromString(*req.MarginFactor)
		if err != nil {
			return nil, formatE(ErrMissingMarginFactor, err)
		}
	}

	priceFactor, err := t.getMarketPriceFactor(ctx, mkt)
	if err != nil {
		return nil, err
	}

	dPriceFactor := priceFactor.ToDecimal()

	buyOrders := make([]*risk.OrderInfo, 0, len(req.Orders))
	sellOrders := make([]*risk.OrderInfo, 0, len(req.Orders))

	for _, o := range req.Orders {
		if o == nil {
			continue
		}
		var price num.Decimal
		p, err := num.DecimalFromString(o.Price)
		if err != nil {
			return nil, formatE(ErrInvalidOrderPrice, err)
		}

		if p.IsNegative() || !p.IsInteger() {
			return nil, ErrInvalidOrderPrice
		}

		price = t.scaleDecimalFromMarketToAssetPrice(p, dPriceFactor)

		switch o.Side {
		case types.SideBuy:
			buyOrders = append(buyOrders, &risk.OrderInfo{TrueRemaining: o.Remaining, Price: price, IsMarketOrder: o.IsMarketOrder})
		case types.SideSell:
			sellOrders = append(sellOrders, &risk.OrderInfo{TrueRemaining: o.Remaining, Price: price, IsMarketOrder: o.IsMarketOrder})
		default:
			return nil, ErrInvalidOrderSide
		}
	}

	rf, err := t.RiskFactorService.GetMarketRiskFactors(ctx, req.MarketId)
	if err != nil {
		return nil, formatE(ErrRiskFactorServiceGet, err)
	}

	mktData, err := t.MarketDataService.GetMarketDataByID(ctx, req.MarketId)
	if err != nil {
		return nil, formatE(ErrMarketServiceGetMarketData, err)
	}

	mktProto := mkt.ToProto()

	asset, err := mktProto.GetAsset()
	if err != nil {
		return nil, formatE(err)
	}

	marketObservable := t.scaleDecimalFromMarketToAssetPrice(mktData.MarkPrice, dPriceFactor)
	auctionPrice := t.scaleDecimalFromMarketToAssetPrice(mktData.IndicativePrice, dPriceFactor)

	auction := mktData.AuctionEnd > 0
	if auction && mktData.MarketTradingMode == types.MarketTradingModeOpeningAuction.String() {
		marketObservable = auctionPrice
	}

	positionFactor := num.DecimalFromFloat(10).
		Pow(num.DecimalFromInt64(int64(mkt.PositionDecimalPlaces)))

	linearSlippageFactor, err := num.DecimalFromString(mktProto.LinearSlippageFactor)
	if err != nil {
		return nil, formatE(fmt.Errorf("can't parse linear slippage factor: %s", mktProto.LinearSlippageFactor), err)
	}
	quadraticSlippageFactor, err := num.DecimalFromString(mktProto.QuadraticSlippageFactor)
	if err != nil {
		return nil, formatE(fmt.Errorf("can't parse quadratic slippage factor: %s", mktProto.QuadraticSlippageFactor), err)
	}

	avgEntryPrice, err := num.DecimalFromString(req.AverageEntryPrice)
	if err != nil {
		return nil, formatE(fmt.Errorf("can't parse average entry price: %s", req.AverageEntryPrice), err)
	}

	avgEntryPrice = t.scaleDecimalFromMarketToAssetPrice(avgEntryPrice, dPriceFactor)

	marginFactorScaledFundingPaymentPerUnitPosition := num.DecimalZero()

	if perp := mktProto.TradableInstrument.Instrument.GetPerpetual(); perp != nil {
		factor, err := num.DecimalFromString(perp.MarginFundingFactor)
		if err != nil {
			return nil, formatE(fmt.Errorf("can't parse margin funding factor: %s", perp.MarginFundingFactor), err)
		}
		if !factor.IsZero() && mktData.ProductData != nil { // for perps it's possible that the product data is not available in the market data
			if perpData := mktData.ProductData.GetPerpetualData(); perpData != nil {
				fundingPayment, err := num.DecimalFromString(perpData.FundingPayment)
				if err != nil {
					return nil, formatE(fmt.Errorf("can't parse funding payment from perpetual product data: %s", perpData.FundingPayment), err)
				}
				fundingRate, err := num.DecimalFromString(perpData.FundingRate)
				if err != nil {
					return nil, formatE(fmt.Errorf("can't parse funding rate from perpetual product data: %s", perpData.FundingRate), err)
				}
				if !fundingPayment.IsZero() && !fundingRate.IsZero() {
					marginFactorScaledFundingPaymentPerUnitPosition = factor.Mul(fundingPayment)
				}
			}
		}
	}

	wMaintenance, bMaintenance, orderMargin := t.computeMarginRange(
		req.OpenVolume,
		buyOrders,
		sellOrders,
		marketObservable,
		positionFactor,
		linearSlippageFactor,
		quadraticSlippageFactor,
		rf,
		marginFactorScaledFundingPaymentPerUnitPosition,
		auction,
		req.MarginMode,
		dMarginFactor,
		auctionPrice,
	)

	marginEstimate := &v2.MarginEstimate{
		WorstCase: implyMarginLevels(wMaintenance, orderMargin, dMarginFactor, mkt.TradableInstrument.MarginCalculator.ScalingFactors, "", req.MarketId, asset, isolatedMarginMode),
		BestCase:  implyMarginLevels(bMaintenance, orderMargin, dMarginFactor, mkt.TradableInstrument.MarginCalculator.ScalingFactors, "", req.MarketId, asset, isolatedMarginMode),
	}

	var wMarginDelta, bMarginDelta, posMarginDelta num.Decimal
	combinedMargin := marginAccountBalance.Add(orderAccountBalance)
	if isolatedMarginMode {
		requiredPositionMargin, requiredOrderMargin := risk.CalculateRequiredMarginInIsolatedMode(req.OpenVolume, avgEntryPrice, marketObservable, buyOrders, sellOrders, positionFactor, dMarginFactor)
		posMarginDelta = requiredPositionMargin.Sub(marginAccountBalance)
		wMarginDelta = requiredPositionMargin.Add(requiredOrderMargin).Sub(combinedMargin)
		bMarginDelta = wMarginDelta
	} else {
		wInitial, _ := num.DecimalFromString(marginEstimate.WorstCase.InitialMargin)
		bInitial, _ := num.DecimalFromString(marginEstimate.BestCase.InitialMargin)
		wRelease, _ := num.DecimalFromString(marginEstimate.WorstCase.CollateralReleaseLevel)
		bRelease, _ := num.DecimalFromString(marginEstimate.BestCase.CollateralReleaseLevel)
		wMarginDifference := wInitial.Sub(combinedMargin)
		bMarginDifference := bInitial.Sub(combinedMargin)
		if wMarginDifference.IsPositive() || combinedMargin.GreaterThan(wRelease) {
			wMarginDelta = wMarginDifference
		}
		if bMarginDifference.IsPositive() || combinedMargin.GreaterThan(bRelease) {
			bMarginDelta = bMarginDifference
		}
	}

	if isolatedMarginMode && ptr.UnBox(req.IncludeRequiredPositionMarginInAvailableCollateral) {
		collateralAvailable = collateralAvailable.Add(posMarginDelta)
	}

	bPositionOnly, bWithBuy, bWithSell := marketObservable, marketObservable, marketObservable
	wPositionOnly, wWithBuy, wWithSell := marketObservable, marketObservable, marketObservable

	// only calculate liquidation price if collateral available is above maintenance margin, otherwise return current market observable to signify that the theoretical position would be liquidated instantenously
	if collateralAvailable.GreaterThanOrEqual(bMaintenance) {
		bPositionOnly, bWithBuy, bWithSell, err = risk.CalculateLiquidationPriceWithSlippageFactors(req.OpenVolume, buyOrders, sellOrders, marketObservable, collateralAvailable, positionFactor, num.DecimalZero(), num.DecimalZero(), rf.Long, rf.Short, marginFactorScaledFundingPaymentPerUnitPosition, isolatedMarginMode, dMarginFactor)
		if err != nil {
			return nil, err
		}
	}
	if collateralAvailable.GreaterThanOrEqual(wMaintenance) {
		wPositionOnly, wWithBuy, wWithSell, err = risk.CalculateLiquidationPriceWithSlippageFactors(req.OpenVolume, buyOrders, sellOrders, marketObservable, collateralAvailable, positionFactor, linearSlippageFactor, quadraticSlippageFactor, rf.Long, rf.Short, marginFactorScaledFundingPaymentPerUnitPosition, isolatedMarginMode, dMarginFactor)
		if err != nil {
			return nil, err
		}
	}

	if ptr.UnBox(req.ScaleLiquidationPriceToMarketDecimals) {
		bPositionOnly = t.scaleDecimalFromAssetToMarketPrice(bPositionOnly, dPriceFactor)
		bWithBuy = t.scaleDecimalFromAssetToMarketPrice(bWithBuy, dPriceFactor)
		bWithSell = t.scaleDecimalFromAssetToMarketPrice(bWithSell, dPriceFactor)
		wPositionOnly = t.scaleDecimalFromAssetToMarketPrice(wPositionOnly, dPriceFactor)
		wWithBuy = t.scaleDecimalFromAssetToMarketPrice(wWithBuy, dPriceFactor)
		wWithSell = t.scaleDecimalFromAssetToMarketPrice(wWithSell, dPriceFactor)
	}

	liquidationEstimate := &v2.LiquidationEstimate{
		WorstCase: &v2.LiquidationPrice{
			OpenVolumeOnly:      wPositionOnly.Round(0).String(),
			IncludingBuyOrders:  wWithBuy.Round(0).String(),
			IncludingSellOrders: wWithSell.Round(0).String(),
		},
		BestCase: &v2.LiquidationPrice{
			OpenVolumeOnly:      bPositionOnly.Round(0).String(),
			IncludingBuyOrders:  bWithBuy.Round(0).String(),
			IncludingSellOrders: bWithSell.Round(0).String(),
		},
	}

	return &v2.EstimatePositionResponse{
		Margin: marginEstimate,
		CollateralIncreaseEstimate: &v2.CollateralIncreaseEstimate{
			WorstCase: wMarginDelta.Round(0).String(),
			BestCase:  bMarginDelta.Round(0).String(),
		},
		Liquidation: liquidationEstimate,
	}, nil
}

func (t *TradingDataServiceV2) computeMarginRange(
	openVolume int64,
	buyOrders, sellOrders []*risk.OrderInfo,
	marketObservable, positionFactor, linearSlippageFactor, quadraticSlippageFactor num.Decimal,
	riskFactors entities.RiskFactor,
	fundingPaymentPerUnitPosition num.Decimal,
	auction bool,
	marginMode vega.MarginMode,
	marginFactor, auctionPrice num.Decimal,
) (num.Decimal, num.Decimal, num.Decimal) {
	bOrders, sOrders := buyOrders, sellOrders
	orderMargin := num.DecimalZero()
	isolatedMarginMode := marginMode == vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN
	if isolatedMarginMode {
		bOrders = []*risk.OrderInfo{}
		bNonMarketOrders := []*risk.OrderInfo{}
		for _, o := range buyOrders {
			if o.IsMarketOrder {
				bOrders = append(bOrders, o)
			} else {
				bNonMarketOrders = append(bNonMarketOrders, o)
			}
		}
		sOrders = []*risk.OrderInfo{}
		sNonMarketOrders := []*risk.OrderInfo{}
		for _, o := range sellOrders {
			if o.IsMarketOrder {
				sOrders = append(sOrders, o)
			} else {
				sNonMarketOrders = append(sNonMarketOrders, o)
			}
		}
		orderMargin = risk.CalcOrderMarginIsolatedMode(openVolume, bNonMarketOrders, sNonMarketOrders, positionFactor, marginFactor, auctionPrice)
	}

	worst := risk.CalculateMaintenanceMarginWithSlippageFactors(openVolume, bOrders, sOrders, marketObservable, positionFactor, linearSlippageFactor, quadraticSlippageFactor, riskFactors.Long, riskFactors.Short, fundingPaymentPerUnitPosition, auction)
	best := risk.CalculateMaintenanceMarginWithSlippageFactors(openVolume, bOrders, sOrders, marketObservable, positionFactor, num.DecimalZero(), num.DecimalZero(), riskFactors.Long, riskFactors.Short, fundingPaymentPerUnitPosition, auction)

	return worst, best, orderMargin
}

// ListNetworkParameters returns a list of network parameters.
func (t *TradingDataServiceV2) ListNetworkParameters(ctx context.Context, req *v2.ListNetworkParametersRequest) (*v2.ListNetworkParametersResponse, error) {
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
func (t *TradingDataServiceV2) GetNetworkParameter(ctx context.Context, req *v2.GetNetworkParameterRequest) (*v2.GetNetworkParameterResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkParameter")()

	v, err := t.networkParameterService.GetByKey(ctx, req.Key)
	if err != nil {
		return nil, formatE(ErrGetNetworkParameters, err)
	}

	if req.Key != v.Key {
		return nil, formatE(ErrNetworkParameterNotFound, errors.Wrapf(err, "network parameter: %s", req.Key))
	}

	return &v2.GetNetworkParameterResponse{
		NetworkParameter: v.ToProto(),
	}, nil
}

// ListCheckpoints returns a list of checkpoints.
func (t *TradingDataServiceV2) ListCheckpoints(ctx context.Context, req *v2.ListCheckpointsRequest) (*v2.ListCheckpointsResponse, error) {
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
func (t *TradingDataServiceV2) GetStake(ctx context.Context, req *v2.GetStakeRequest) (*v2.GetStakeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetStake")()

	if len(req.PartyId) == 0 {
		return nil, formatE(ErrMissingPartyID)
	}

	if req.PartyId != networkPartyID && !crypto.IsValidVegaID(req.PartyId) {
		return nil, formatE(ErrInvalidPartyID)
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
func (t *TradingDataServiceV2) GetRiskFactors(ctx context.Context, req *v2.GetRiskFactorsRequest) (*v2.GetRiskFactorsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetRiskFactors SQL")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	if !crypto.IsValidVegaID(req.MarketId) {
		return nil, formatE(ErrInvalidMarketID)
	}

	rfs, err := t.RiskFactorService.GetMarketRiskFactors(ctx, req.MarketId)
	if err != nil {
		return nil, formatE(ErrRiskFactorServiceGet, errors.Wrapf(err, "marketID: %s", req.MarketId))
	}

	return &v2.GetRiskFactorsResponse{
		RiskFactor: rfs.ToProto(),
	}, nil
}

// ObserveGovernance streams governance updates to the client.
func (t *TradingDataServiceV2) ObserveGovernance(req *v2.ObserveGovernanceRequest, stream v2.TradingDataService_ObserveGovernanceServer) error {
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

func (t *TradingDataServiceV2) proposalToGovernanceData(ctx context.Context, proposal entities.Proposal) (*vega.GovernanceData, error) {
	yesVotes, err := t.governanceService.GetYesVotesForProposal(ctx, proposal.ID.String())
	if err != nil {
		return nil, errors.Wrap(err, "getting yes votes for proposal")
	}

	noVotes, err := t.governanceService.GetNoVotesForProposal(ctx, proposal.ID.String())
	if err != nil {
		return nil, errors.Wrap(err, "getting no votes for proposal")
	}

	var subProposals []*vega.Proposal
	if proposal.IsBatch() {
		for _, p := range proposal.Proposals {
			subProposals = append(subProposals, p.ToProto())
		}
	}

	proposalType := vega.GovernanceData_TYPE_SINGLE_OR_UNSPECIFIED
	if proposal.IsBatch() {
		proposalType = vega.GovernanceData_TYPE_BATCH
	}

	return &vega.GovernanceData{
		Proposal:     proposal.ToProto(),
		Yes:          voteListToProto(yesVotes),
		No:           voteListToProto(noVotes),
		ProposalType: proposalType,
		Proposals:    subProposals,
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
func (t *TradingDataServiceV2) ObserveVotes(req *v2.ObserveVotesRequest, stream v2.TradingDataService_ObserveVotesServer) error {
	if partyID := ptr.UnBox(req.PartyId); partyID != "" {
		return t.observePartyVotes(partyID, stream)
	}

	if proposalID := ptr.UnBox(req.ProposalId); proposalID != "" {
		return t.observeProposalVotes(proposalID, stream)
	}

	return formatE(ErrMissingProposalIDOrPartyID)
}

func (t *TradingDataServiceV2) observePartyVotes(partyID string, stream v2.TradingDataService_ObserveVotesServer) error {
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

func (t *TradingDataServiceV2) observeProposalVotes(proposalID string, stream v2.TradingDataService_ObserveVotesServer) error {
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
func (t *TradingDataServiceV2) GetProtocolUpgradeStatus(context.Context, *v2.GetProtocolUpgradeStatusRequest) (*v2.GetProtocolUpgradeStatusResponse, error) {
	ready := t.protocolUpgradeService.GetProtocolUpgradeStarted()
	return &v2.GetProtocolUpgradeStatusResponse{
		Ready: ready,
	}, nil
}

// ListProtocolUpgradeProposals returns a list of protocol upgrade proposals.
func (t *TradingDataServiceV2) ListProtocolUpgradeProposals(ctx context.Context, req *v2.ListProtocolUpgradeProposalsRequest) (*v2.ListProtocolUpgradeProposalsResponse, error) {
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
func (t *TradingDataServiceV2) ListCoreSnapshots(ctx context.Context, req *v2.ListCoreSnapshotsRequest) (*v2.ListCoreSnapshotsResponse, error) {
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
func (t *TradingDataServiceV2) ObserveEventBus(stream v2.TradingDataService_ObserveEventBusServer) error {
	return observeEventBus(t.log, t.config, tradingDataEventBusServerV2{stream}, t.eventService)
}

// ObserveLedgerMovements subscribes to a stream of ledger movements.
func (t *TradingDataServiceV2) ObserveLedgerMovements(_ *v2.ObserveLedgerMovementsRequest, srv v2.TradingDataService_ObserveLedgerMovementsServer) error {
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
func (t *TradingDataServiceV2) ListKeyRotations(ctx context.Context, req *v2.ListKeyRotationsRequest) (*v2.ListKeyRotationsResponse, error) {
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

func (t *TradingDataServiceV2) getAllKeyRotations(ctx context.Context, pagination entities.CursorPagination) (*v2.ListKeyRotationsResponse, error) {
	return makeKeyRotationResponse(
		t.keyRotationService.GetAllPubKeyRotations(ctx, pagination),
	)
}

func (t *TradingDataServiceV2) getNodeKeyRotations(ctx context.Context, nodeID string, pagination entities.CursorPagination) (*v2.ListKeyRotationsResponse, error) {
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
func (t *TradingDataServiceV2) ListEthereumKeyRotations(ctx context.Context, req *v2.ListEthereumKeyRotationsRequest) (*v2.ListEthereumKeyRotationsResponse, error) {
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
func (t *TradingDataServiceV2) GetVegaTime(ctx context.Context, _ *v2.GetVegaTimeRequest) (*v2.GetVegaTimeResponse, error) {
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
func (t *TradingDataServiceV2) ExportNetworkHistory(req *v2.ExportNetworkHistoryRequest, stream v2.TradingDataService_ExportNetworkHistoryServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ExportNetworkHistory")()

	if t.NetworkHistoryService == nil || reflect.ValueOf(t.NetworkHistoryService).IsNil() {
		return formatE(ErrNetworkHistoryServiceNotInitialised)
	}

	if req.Table == v2.Table_TABLE_UNSPECIFIED {
		return formatE(ErrNetworkHistoryNoTableName, errors.New("empty table name"))
	}

	tableName := strings.TrimPrefix(strings.ToLower(req.Table.String()), "table_")

	allSegments, err := t.NetworkHistoryService.ListAllHistorySegments()
	if err != nil {
		return formatE(ErrListAllNetworkHistorySegment, err)
	}

	ch, err := allSegments.ContiguousHistoryInRange(req.FromBlock, req.ToBlock)
	if err != nil || len(ch.Segments) == 0 {
		return formatE(ErrNetworkHistoryGetContiguousSegments, err)
	}
	chainID := ch.Segments[0].GetChainId()

	header := metadata.Pairs("Content-Disposition", fmt.Sprintf("attachment;filename=%s-%s-%06d-%06d.zip", chainID, tableName, ch.HeightFrom, ch.HeightTo))
	if err := stream.SendHeader(header); err != nil {
		return formatE(ErrSendingGRPCHeader, err)
	}

	grpcWriter := httpBodyWriter{chunkSize: httpBodyChunkSize, contentType: "application/zip", buf: &bytes.Buffer{}, stream: stream}
	zipWriter := zip.NewWriter(&grpcWriter)
	defer grpcWriter.Close()
	defer zipWriter.Close()

	partitionedSegments := partitionSegmentsByDBVersion(ch.Segments)

	for _, segments := range partitionedSegments {
		if len(segments) == 0 {
			continue
		}
		csvFileName := fmt.Sprintf("%s-%s-%03d-%06d-%06d.csv",
			segments[0].GetChainId(),
			tableName,
			segments[0].GetDatabaseVersion(),
			segments[0].GetFromHeight(),
			segments[len(segments)-1].GetToHeight())

		out, err := zipWriter.Create(csvFileName)
		if err != nil {
			return formatE(ErrNetworkHistoryCreatingZipFile, err)
		}

		for i, segment := range segments {
			segmentReader, size, err := t.NetworkHistoryService.GetHistorySegmentReader(stream.Context(), segment.GetHistorySegmentId())
			if err != nil {
				segmentReader.Close()
				return formatE(ErrNetworkHistoryOpeningSegment, err)
			}

			segmentData, err := fsutil.ReadNetworkHistorySegmentData(segmentReader, size, tableName)
			if err != nil {
				segmentReader.Close()
				return formatE(ErrNetworkHistoryExtractingSegment, err)
			}
			scanner := bufio.NewScanner(segmentData)

			// For all except first segment, skip the header.
			if i != 0 {
				scanner.Scan()
			}

			for scanner.Scan() {
				out.Write(scanner.Bytes())
				out.Write([]byte("\n"))
			}

			segmentReader.Close()
		}
	}
	return nil
}

func partitionSegmentsByDBVersion(segments []segment.Full) [][]segment.Full {
	partitioned := [][]segment.Full{}
	sliceStart := 0

	for i, segment := range segments {
		sliceVersion := segments[sliceStart].GetDatabaseVersion()
		if segment.GetDatabaseVersion() != sliceVersion {
			partitioned = append(partitioned, segments[sliceStart:i])
			sliceStart = i
		}
	}
	partitioned = append(partitioned, segments[sliceStart:])
	return partitioned
}

// GetMostRecentNetworkHistorySegment returns the most recent network history segment.
func (t *TradingDataServiceV2) GetMostRecentNetworkHistorySegment(context.Context, *v2.GetMostRecentNetworkHistorySegmentRequest) (*v2.GetMostRecentNetworkHistorySegmentResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetMostRecentNetworkHistorySegment")()

	if t.NetworkHistoryService == nil || reflect.ValueOf(t.NetworkHistoryService).IsNil() {
		return nil, formatE(ErrNetworkHistoryServiceNotInitialised)
	}

	segment, err := t.NetworkHistoryService.GetHighestBlockHeightHistorySegment()
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
		SwarmKeySeed: t.NetworkHistoryService.GetSwarmKeySeed(),
	}, nil
}

// ListAllNetworkHistorySegments returns all network history segments.
func (t *TradingDataServiceV2) ListAllNetworkHistorySegments(context.Context, *v2.ListAllNetworkHistorySegmentsRequest) (*v2.ListAllNetworkHistorySegmentsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListAllNetworkHistorySegments")()

	if t.NetworkHistoryService == nil || reflect.ValueOf(t.NetworkHistoryService).IsNil() {
		return nil, formatE(ErrNetworkHistoryServiceNotInitialised)
	}

	segments, err := t.NetworkHistoryService.ListAllHistorySegments()
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

func toHistorySegment(segment segment.Full) *v2.HistorySegment {
	return &v2.HistorySegment{
		FromHeight:               segment.GetFromHeight(),
		ToHeight:                 segment.GetToHeight(),
		ChainId:                  segment.GetChainId(),
		DatabaseVersion:          segment.GetDatabaseVersion(),
		HistorySegmentId:         segment.GetHistorySegmentId(),
		PreviousHistorySegmentId: segment.GetPreviousHistorySegmentId(),
	}
}

// GetActiveNetworkHistoryPeerAddresses returns the active network history peer addresses.
func (t *TradingDataServiceV2) GetActiveNetworkHistoryPeerAddresses(context.Context, *v2.GetActiveNetworkHistoryPeerAddressesRequest) (*v2.GetActiveNetworkHistoryPeerAddressesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetActiveNetworkHistoryPeerAddresses")()

	if t.NetworkHistoryService == nil || reflect.ValueOf(t.NetworkHistoryService).IsNil() {
		return nil, formatE(ErrNetworkHistoryServiceNotInitialised)
	}

	return &v2.GetActiveNetworkHistoryPeerAddressesResponse{
		IpAddresses: t.NetworkHistoryService.GetActivePeerIPAddresses(),
	}, nil
}

// NetworkHistoryStatus returns the network history status.
func (t *TradingDataServiceV2) GetNetworkHistoryStatus(context.Context, *v2.GetNetworkHistoryStatusRequest) (*v2.GetNetworkHistoryStatusResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkHistoryStatus")()

	if t.NetworkHistoryService == nil || reflect.ValueOf(t.NetworkHistoryService).IsNil() {
		return nil, formatE(ErrNetworkHistoryServiceNotInitialised)
	}

	connectedPeerAddresses, err := t.NetworkHistoryService.GetConnectedPeerAddresses()
	if err != nil {
		return nil, formatE(ErrGetConnectedPeerAddresses, err)
	}

	// A subset of the connected peer addresses are likely to be copied to form another nodes peer set, randomise the list
	// to minimise the chance that the same sub set are copied each time.
	rand.Shuffle(len(connectedPeerAddresses), func(i, j int) {
		connectedPeerAddresses[i], connectedPeerAddresses[j] = connectedPeerAddresses[j], connectedPeerAddresses[i]
	})

	ipfsAddress, err := t.NetworkHistoryService.GetIpfsAddress()
	if err != nil {
		return nil, formatE(ErrGetIpfsAddress, err)
	}

	return &v2.GetNetworkHistoryStatusResponse{
		IpfsAddress:    ipfsAddress,
		SwarmKey:       t.NetworkHistoryService.GetSwarmKey(),
		SwarmKeySeed:   t.NetworkHistoryService.GetSwarmKeySeed(),
		ConnectedPeers: connectedPeerAddresses,
	}, nil
}

// NetworkHistoryBootstrapPeers returns the network history bootstrap peers.
func (t *TradingDataServiceV2) GetNetworkHistoryBootstrapPeers(context.Context, *v2.GetNetworkHistoryBootstrapPeersRequest) (*v2.GetNetworkHistoryBootstrapPeersResponse, error) {
	if t.NetworkHistoryService == nil || reflect.ValueOf(t.NetworkHistoryService).IsNil() {
		return nil, formatE(ErrNetworkHistoryServiceNotInitialised)
	}

	return &v2.GetNetworkHistoryBootstrapPeersResponse{BootstrapPeers: t.NetworkHistoryService.GetBootstrapPeers()}, nil
}

// Ping returns a ping response.
func (t *TradingDataServiceV2) Ping(context.Context, *v2.PingRequest) (*v2.PingResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Ping")()
	return &v2.PingResponse{}, nil
}

func (t *TradingDataServiceV2) ListEntities(ctx context.Context, req *v2.ListEntitiesRequest) (*v2.ListEntitiesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListEntities")()

	if len(req.GetTransactionHash()) == 0 {
		return nil, formatE(ErrMissingEmptyTxHash)
	}

	if !crypto.IsValidVegaID(req.GetTransactionHash()) {
		return nil, formatE(ErrInvalidTxHash)
	}

	txHash := entities.TxHash(req.GetTransactionHash())
	eg, ctx := errgroup.WithContext(ctx)

	// query
	accounts := queryProtoEntities[*vega.Account](ctx, eg, txHash,
		t.accountService.GetByTxHash, ErrAccountServiceGetByTxHash)

	orders := queryProtoEntities[*vega.Order](ctx, eg, txHash,
		t.orderService.GetByTxHash, ErrOrderServiceGetByTxHash)

	positions := queryProtoEntities[*vega.Position](ctx, eg, txHash,
		t.positionService.GetByTxHash, ErrPositionsGetByTxHash)

	balances := queryProtoEntities[*v2.AccountBalance](ctx, eg, txHash,
		t.accountService.GetBalancesByTxHash, ErrAccountServiceGetBalancesByTxHash)

	votes := queryProtoEntities[*vega.Vote](ctx, eg, txHash,
		t.governanceService.GetVotesByTxHash, ErrVotesGetByTxHash)

	trades := queryProtoEntities[*vega.Trade](ctx, eg, txHash,
		t.tradeService.GetByTxHash, ErrTradeServiceGetByTxHash)

	oracleSpecs := queryProtoEntities[*vega.OracleSpec](ctx, eg, txHash,
		t.oracleSpecService.GetByTxHash, ErrOracleSpecGetByTxHash)

	oracleData := queryProtoEntities[*vega.OracleData](ctx, eg, txHash,
		t.oracleDataService.GetByTxHash, ErrOracleDataGetByTxHash)

	markets := queryProtoEntities[*vega.Market](ctx, eg, txHash,
		t.MarketsService.GetByTxHash, ErrMarketServiceGetByTxHash)

	parties := queryProtoEntities[*vega.Party](ctx, eg, txHash,
		t.partyService.GetByTxHash, ErrPartyServiceGetByTxHash)

	rewards := queryProtoEntities[*vega.Reward](ctx, eg, txHash,
		t.rewardService.GetByTxHash, ErrRewardsGetByTxHash)

	deposits := queryProtoEntities[*vega.Deposit](ctx, eg, txHash,
		t.depositService.GetByTxHash, ErrDepositsGetByTxHash)

	withdrawals := queryProtoEntities[*vega.Withdrawal](ctx, eg, txHash,
		t.withdrawalService.GetByTxHash, ErrWithdrawalsGetByTxHash)

	assets := queryProtoEntities[*vega.Asset](ctx, eg, txHash,
		t.AssetService.GetByTxHash, ErrAssetsGetByTxHash)

	lps := queryProtoEntities[*vega.LiquidityProvision](ctx, eg, txHash,
		t.liquidityProvisionService.GetByTxHash, ErrLiquidityProvisionGetByTxHash)

	proposals := queryProtoEntities[*vega.Proposal](ctx, eg, txHash,
		t.governanceService.GetProposalsByTxHash, ErrProposalsGetByTxHash)

	delegations := queryProtoEntities[*vega.Delegation](ctx, eg, txHash,
		t.delegationService.GetByTxHash, ErrDelegationsGetByTxHash)

	signatures := queryProtoEntities[*cmdsV1.NodeSignature](ctx, eg, txHash,
		t.notaryService.GetByTxHash, ErrSignaturesGetByTxHash)

	netParams := queryProtoEntities[*vega.NetworkParameter](ctx, eg, txHash,
		t.networkParameterService.GetByTxHash, ErrNetworkParametersGetByTxHash)

	keyRotations := queryProtoEntities[*v1.KeyRotation](ctx, eg, txHash,
		t.keyRotationService.GetByTxHash, ErrKeyRotationsGetByTxHash)

	ethKeyRotations := queryProtoEntities[*v1.EthereumKeyRotation](ctx, eg, txHash,
		t.ethereumKeyRotationService.GetByTxHash, ErrEthereumKeyRotationsGetByTxHash)

	pups := queryProtoEntities[*v1.ProtocolUpgradeEvent](ctx, eg, txHash,
		t.protocolUpgradeService.GetByTxHash, ErrEthereumKeyRotationsGetByTxHash)

	nodes := queryProtoEntities[*v2.NodeBasic](ctx, eg, txHash,
		t.nodeService.GetByTxHash, ErrNodeServiceGetByTxHash)

	// query and map
	ledgerEntries := queryAndMapEntities(ctx, eg, txHash,
		t.ledgerService.GetByTxHash,
		func(item entities.LedgerEntry) (*vega.LedgerEntry, error) {
			return item.ToProto(ctx, t.accountService)
		},
		ErrLedgerEntriesGetByTxHash,
	)

	transfers := queryAndMapEntities(ctx, eg, txHash,
		t.transfersService.GetByTxHash,
		func(item entities.Transfer) (*v1.Transfer, error) {
			return item.ToProto(ctx, t.accountService)
		},
		ErrTransfersGetByTxHash,
	)

	marginLevels := queryAndMapEntities(ctx, eg, txHash,
		t.riskService.GetByTxHash,
		func(item entities.MarginLevels) (*vega.MarginLevels, error) {
			return item.ToProto(ctx, t.accountService)
		},
		ErrMarginLevelsGetByTxHash,
	)

	addedEvents := queryAndMapEntities(ctx, eg, txHash,
		t.multiSigService.GetAddedByTxHash,
		func(item entities.ERC20MultiSigSignerAddedEvent) (*v2.ERC20MultiSigSignerAddedBundle, error) {
			return item.ToDataNodeApiV2Proto(ctx, t.notaryService)
		},
		ErrERC20MultiSigSignerAddedEventGetByTxHash,
	)

	removedEvents := queryAndMapEntities(ctx, eg, txHash,
		t.multiSigService.GetRemovedByTxHash,
		func(item entities.ERC20MultiSigSignerRemovedEvent) (*v2.ERC20MultiSigSignerRemovedBundle, error) {
			return item.ToDataNodeApiV2Proto(ctx, t.notaryService)
		},
		ErrERC20MultiSigSignerRemovedEventGetByTxHash,
	)

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return &v2.ListEntitiesResponse{
		Accounts:                          <-accounts,
		Orders:                            <-orders,
		Positions:                         <-positions,
		LedgerEntries:                     <-ledgerEntries,
		BalanceChanges:                    <-balances,
		Transfers:                         <-transfers,
		Votes:                             <-votes,
		Erc20MultiSigSignerAddedBundles:   <-addedEvents,
		Erc20MultiSigSignerRemovedBundles: <-removedEvents,
		Trades:                            <-trades,
		OracleSpecs:                       <-oracleSpecs,
		OracleData:                        <-oracleData,
		Markets:                           <-markets,
		Parties:                           <-parties,
		MarginLevels:                      <-marginLevels,
		Rewards:                           <-rewards,
		Deposits:                          <-deposits,
		Withdrawals:                       <-withdrawals,
		Assets:                            <-assets,
		LiquidityProvisions:               <-lps,
		Proposals:                         <-proposals,
		Delegations:                       <-delegations,
		Nodes:                             <-nodes,
		NodeSignatures:                    <-signatures,
		NetworkParameters:                 <-netParams,
		KeyRotations:                      <-keyRotations,
		EthereumKeyRotations:              <-ethKeyRotations,
		ProtocolUpgradeProposals:          <-pups,
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

func (t *TradingDataServiceV2) GetStopOrder(ctx context.Context, req *v2.GetStopOrderRequest) (*v2.GetStopOrderResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetStopOrder")()

	if len(req.OrderId) == 0 {
		return nil, formatE(ErrMissingOrderID)
	}

	if !crypto.IsValidVegaID(req.OrderId) {
		return nil, formatE(ErrInvalidOrderID)
	}

	order, err := t.stopOrderService.GetStopOrder(ctx, req.OrderId)
	if err != nil {
		return nil, formatE(ErrOrderNotFound, errors.Wrapf(err, "orderID: %s", req.OrderId))
	}

	return &v2.GetStopOrderResponse{
		Order: order.ToProto(),
	}, nil
}

func (t *TradingDataServiceV2) ListStopOrders(ctx context.Context, req *v2.ListStopOrdersRequest) (*v2.ListStopOrdersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetStopOrder")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var filter entities.StopOrderFilter
	if req.Filter != nil {
		dateRange := entities.DateRangeFromProto(req.Filter.DateRange)
		liveOnly := false

		if req.Filter.LiveOnly != nil {
			liveOnly = *req.Filter.LiveOnly
		}

		filter = entities.StopOrderFilter{
			Statuses:       stopOrderStatusesFromProto(req.Filter.Statuses),
			ExpiryStrategy: stopOrderExpiryStrategyFromProto(req.Filter.ExpiryStrategies),
			PartyIDs:       req.Filter.PartyIds,
			MarketIDs:      req.Filter.MarketIds,
			DateRange:      &entities.DateRange{Start: dateRange.Start, End: dateRange.End},
			LiveOnly:       liveOnly,
		}

		if err := VegaIDsSlice(req.Filter.MarketIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more market id is invalid"))
		}

		if err := VegaIDsSlice(req.Filter.PartyIds).Ensure(); err != nil {
			return nil, formatE(err, errors.New("one or more party id is invalid"))
		}
	}

	orders, pageInfo, err := t.stopOrderService.ListStopOrders(ctx, filter, pagination)
	if err != nil {
		if errors.Is(err, sqlstore.ErrLastPaginationNotSupported) {
			return nil, formatE(ErrLastPaginationNotSupported, err)
		}

		return nil, formatE(err)
	}

	edges, err := makeEdges[*v2.StopOrderEdge](orders)
	if err != nil {
		return nil, formatE(err)
	}

	ordersConnection := &v2.StopOrderConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListStopOrdersResponse{
		Orders: ordersConnection,
	}, nil
}

func stopOrderStatusesFromProto(statuses []vega.StopOrder_Status) []entities.StopOrderStatus {
	s := make([]entities.StopOrderStatus, len(statuses))
	for i := range statuses {
		s[i] = entities.StopOrderStatus(statuses[i])
	}
	return s
}

func stopOrderExpiryStrategyFromProto(strategies []vega.StopOrder_ExpiryStrategy) []entities.StopOrderExpiryStrategy {
	es := make([]entities.StopOrderExpiryStrategy, len(strategies))
	for i := range strategies {
		es[i] = entities.StopOrderExpiryStrategy(strategies[i])
	}
	return es
}

func (t *TradingDataServiceV2) ListFundingPeriods(ctx context.Context, req *v2.ListFundingPeriodsRequest) (*v2.ListFundingPeriodsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListFundingPeriods")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	if !crypto.IsValidVegaID(req.MarketId) {
		return nil, formatE(ErrInvalidMarketID)
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	periods, pageInfo, err := t.fundingPeriodService.ListFundingPeriods(ctx, entities.MarketID(req.MarketId), dateRange, pagination)
	if err != nil {
		return nil, formatE(ErrListFundingPeriod, err)
	}

	edges, err := makeEdges[*v2.FundingPeriodEdge](periods)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.FundingPeriodConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListFundingPeriodsResponse{
		FundingPeriods: connection,
	}, nil
}

func (t *TradingDataServiceV2) ListFundingPeriodDataPoints(ctx context.Context, req *v2.ListFundingPeriodDataPointsRequest) (
	*v2.ListFundingPeriodDataPointsResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListFundingPeriodDataPoints")()

	if len(req.MarketId) == 0 {
		return nil, formatE(ErrEmptyMissingMarketID)
	}

	if !crypto.IsValidVegaID(req.MarketId) {
		return nil, formatE(ErrInvalidMarketID)
	}

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	dateRange := entities.DateRangeFromProto(req.DateRange)
	dataPoints, pageInfo, err := t.fundingPeriodService.ListFundingPeriodDataPoints(ctx,
		entities.MarketID(req.MarketId),
		dateRange,
		(*entities.FundingPeriodDataPointSource)(req.Source),
		req.Seq,
		pagination,
	)
	if err != nil {
		return nil, formatE(ErrListFundingPeriodDataPoints, err)
	}

	edges, err := makeEdges[*v2.FundingPeriodDataPointEdge](dataPoints)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.FundingPeriodDataPointConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListFundingPeriodDataPointsResponse{
		FundingPeriodDataPoints: connection,
	}, nil
}

func (t *TradingDataServiceV2) GetCurrentReferralProgram(ctx context.Context, _ *v2.GetCurrentReferralProgramRequest) (
	*v2.GetCurrentReferralProgramResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetCurrentReferralProgram")()
	referralProgram, err := t.referralProgramService.GetCurrentReferralProgram(ctx)
	if err != nil {
		return nil, formatE(ErrGetCurrentReferralProgram, err)
	}

	return &v2.GetCurrentReferralProgramResponse{
		CurrentReferralProgram: referralProgram.ToProto(),
	}, nil
}

func (t *TradingDataServiceV2) ListReferralSets(ctx context.Context, req *v2.ListReferralSetsRequest) (
	*v2.ListReferralSetsResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListReferralSets")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		id       *entities.ReferralSetID
		referrer *entities.PartyID
		referee  *entities.PartyID
	)

	if req.ReferralSetId != nil {
		id = ptr.From(entities.ReferralSetID(*req.ReferralSetId))
	}

	if req.Referrer != nil {
		referrer = ptr.From(entities.PartyID(*req.Referrer))
	}

	if req.Referee != nil {
		referee = ptr.From(entities.PartyID(*req.Referee))
	}

	sets, pageInfo, err := t.referralSetsService.ListReferralSets(ctx, id, referrer, referee, pagination)
	if err != nil {
		return nil, formatE(err)
	}

	edges, err := makeEdges[*v2.ReferralSetEdge](sets)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.ReferralSetConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListReferralSetsResponse{
		ReferralSets: connection,
	}, nil
}

func (t *TradingDataServiceV2) ListReferralSetReferees(ctx context.Context, req *v2.ListReferralSetRefereesRequest) (
	*v2.ListReferralSetRefereesResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListReferralSetReferees")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		id                *entities.ReferralSetID
		referrer          *entities.PartyID
		referee           *entities.PartyID
		epochsToAggregate uint32 = 30 // default is 30 epochs
	)

	if req.ReferralSetId != nil {
		id = ptr.From(entities.ReferralSetID(*req.ReferralSetId))
	}

	if req.Referrer != nil {
		referrer = ptr.From(entities.PartyID(*req.Referrer))
	}

	if req.Referee != nil {
		referee = ptr.From(entities.PartyID(*req.Referee))
	}

	if req.AggregationEpochs != nil {
		epochsToAggregate = *req.AggregationEpochs
	}

	referees, pageInfo, err := t.referralSetsService.ListReferralSetReferees(ctx, id, referrer, referee, pagination, epochsToAggregate)
	if err != nil {
		return nil, formatE(err)
	}

	edges, err := makeEdges[*v2.ReferralSetRefereeEdge](referees)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.ReferralSetRefereeConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListReferralSetRefereesResponse{
		ReferralSetReferees: connection,
	}, nil
}

func (t *TradingDataServiceV2) GetReferralSetStats(ctx context.Context, req *v2.GetReferralSetStatsRequest) (
	*v2.GetReferralSetStatsResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetReferralSetStats")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var setID *entities.ReferralSetID
	if req.ReferralSetId != nil {
		setID = ptr.From(entities.ReferralSetID(*req.ReferralSetId))
	}

	var referee *entities.PartyID
	if req.Referee != nil {
		referee = ptr.From(entities.PartyID(*req.Referee))
	}

	stats, pageInfo, err := t.referralSetsService.GetReferralSetStats(ctx, setID, req.AtEpoch, referee, pagination)
	if err != nil {
		return nil, formatE(ErrGetReferralSetStats, err)
	}

	edges, err := makeEdges[*v2.ReferralSetStatsEdge](stats)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.GetReferralSetStatsResponse{
		Stats: &v2.ReferralSetStatsConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}

func (t *TradingDataServiceV2) ListTeams(ctx context.Context, req *v2.ListTeamsRequest) (*v2.ListTeamsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTeams")()

	if req.TeamId == nil && req.PartyId == nil {
		pagination, err := entities.CursorPaginationFromProto(req.Pagination)
		if err != nil {
			return nil, formatE(ErrInvalidPagination, err)
		}

		return listTeams(ctx, t.teamsService, pagination)
	}

	var (
		teamID  entities.TeamID
		partyID entities.PartyID
	)

	if req.TeamId != nil {
		teamID = entities.TeamID(*req.TeamId)
	}

	if req.PartyId != nil {
		partyID = entities.PartyID(*req.PartyId)
	}

	return listTeam(ctx, t.teamsService, teamID, partyID)
}

func listTeam(ctx context.Context, teamsService *service.Teams, teamID entities.TeamID, partyID entities.PartyID) (*v2.ListTeamsResponse, error) {
	team, err := teamsService.GetTeam(ctx, teamID, partyID)
	if err != nil {
		if pgxscan.NotFound(err) {
			return &v2.ListTeamsResponse{
				Teams: &v2.TeamConnection{
					Edges: []*v2.TeamEdge{},
				},
			}, nil
		}

		return nil, formatE(ErrListTeams, err)
	}

	if team == nil {
		return nil, nil
	}

	edges, err := makeEdges[*v2.TeamEdge]([]entities.Team{*team})
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.ListTeamsResponse{
		Teams: &v2.TeamConnection{
			Edges: edges,
			PageInfo: &v2.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     team.Cursor().Encode(),
				EndCursor:       team.Cursor().Encode(),
			},
		},
	}, nil
}

func listTeams(ctx context.Context, teamsService *service.Teams, pagination entities.CursorPagination) (*v2.ListTeamsResponse, error) {
	teams, pageInfo, err := teamsService.ListTeams(ctx, pagination)
	if err != nil {
		return nil, formatE(ErrListTeams, err)
	}
	edges, err := makeEdges[*v2.TeamEdge](teams)
	if err != nil {
		return nil, formatE(err)
	}
	connection := &v2.TeamConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListTeamsResponse{
		Teams: connection,
	}, nil
}

func (t *TradingDataServiceV2) ListTeamsStatistics(ctx context.Context, req *v2.ListTeamsStatisticsRequest) (*v2.ListTeamsStatisticsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTeamsStatistics")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	filters := sqlstore.ListTeamsStatisticsFilters{
		AggregationEpochs: 10,
	}
	if req.TeamId != nil {
		filters.TeamID = ptr.From(entities.TeamID(*req.TeamId))
	}
	if req.AggregationEpochs != nil {
		filters.AggregationEpochs = *req.AggregationEpochs
	}

	stats, pageInfo, err := t.teamsService.ListTeamsStatistics(ctx, pagination, filters)
	if err != nil {
		return nil, formatE(ErrListTeamStatistics, err)
	}

	edges, err := makeEdges[*v2.TeamStatisticsEdge](stats)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.ListTeamsStatisticsResponse{
		Statistics: &v2.TeamsStatisticsConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}

func (t *TradingDataServiceV2) ListTeamMembersStatistics(ctx context.Context, req *v2.ListTeamMembersStatisticsRequest) (*v2.ListTeamMembersStatisticsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTeamMembersStatistics")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	filters := sqlstore.ListTeamMembersStatisticsFilters{
		TeamID:            entities.TeamID(req.TeamId),
		AggregationEpochs: 10,
	}

	if req.PartyId != nil {
		filters.PartyID = ptr.From(entities.PartyID(*req.PartyId))
	}

	if req.AggregationEpochs != nil && *req.AggregationEpochs > 0 {
		filters.AggregationEpochs = *req.AggregationEpochs
	}

	stats, pageInfo, err := t.teamsService.ListTeamMembersStatistics(ctx, pagination, filters)
	if err != nil {
		return nil, formatE(ErrListTeamReferees, err)
	}

	edges, err := makeEdges[*v2.TeamMemberStatisticsEdge](stats)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.ListTeamMembersStatisticsResponse{
		Statistics: &v2.TeamMembersStatisticsConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}

func (t *TradingDataServiceV2) ListTeamReferees(ctx context.Context, req *v2.ListTeamRefereesRequest) (*v2.ListTeamRefereesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTeamReferees")()
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	teamID := entities.TeamID(req.TeamId)

	referees, pageInfo, err := t.teamsService.ListReferees(ctx, teamID, pagination)
	if err != nil {
		return nil, formatE(ErrListTeamReferees, err)
	}

	edges, err := makeEdges[*v2.TeamRefereeEdge](referees)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.TeamRefereeConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListTeamRefereesResponse{
		TeamReferees: connection,
	}, nil
}

func (t *TradingDataServiceV2) ListTeamRefereeHistory(ctx context.Context, req *v2.ListTeamRefereeHistoryRequest) (*v2.ListTeamRefereeHistoryResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListTeamRefereeHistory")()
	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	refereeID := entities.PartyID(req.Referee)

	history, pageInfo, err := t.teamsService.ListRefereeHistory(ctx, refereeID, pagination)
	if err != nil {
		return nil, formatE(ErrListTeamRefereeHistory, err)
	}

	edges, err := makeEdges[*v2.TeamRefereeHistoryEdge](history)
	if err != nil {
		return nil, formatE(err)
	}

	connection := &v2.TeamRefereeHistoryConnection{
		Edges:    edges,
		PageInfo: pageInfo.ToProto(),
	}

	return &v2.ListTeamRefereeHistoryResponse{
		TeamRefereeHistory: connection,
	}, nil
}

func (t *TradingDataServiceV2) GetFeesStats(ctx context.Context, req *v2.GetFeesStatsRequest) (*v2.GetFeesStatsResponse, error) {
	if req.MarketId == nil && req.AssetId == nil {
		return nil, formatE(ErrFeesStatsRequest)
	}

	var (
		marketID *entities.MarketID
		assetID  *entities.AssetID
	)

	if req.MarketId != nil {
		marketID = ptr.From(entities.MarketID(*req.MarketId))
	}

	if req.AssetId != nil {
		assetID = ptr.From(entities.AssetID(*req.AssetId))
	}

	stats, err := t.feesStatsService.GetFeesStats(ctx, marketID, assetID, req.EpochSeq, req.PartyId)
	if err != nil {
		return nil, formatE(ErrGetFeesStats, err)
	}

	return &v2.GetFeesStatsResponse{
		FeesStats: stats.ToProto(),
	}, nil
}

func (t *TradingDataServiceV2) GetFeesStatsForParty(ctx context.Context, req *v2.GetFeesStatsForPartyRequest) (*v2.GetFeesStatsForPartyResponse, error) {
	var assetID *entities.AssetID
	if req.AssetId != nil {
		assetID = ptr.From(entities.AssetID(*req.AssetId))
	}

	stats, err := t.feesStatsService.StatsForParty(ctx, entities.PartyID(req.PartyId), assetID, req.FromEpoch, req.ToEpoch)
	if err != nil {
		return nil, formatE(ErrGetFeesStatsForParty, err)
	}

	statsProto := make([]*v2.FeesStatsForParty, 0, len(stats))
	for _, s := range stats {
		statsProto = append(statsProto, &v2.FeesStatsForParty{
			AssetId:                 s.AssetID.String(),
			TotalRewardsReceived:    s.TotalRewardsReceived,
			RefereesDiscountApplied: s.RefereesDiscountApplied,
			VolumeDiscountApplied:   s.VolumeDiscountApplied,
			TotalMakerFeesReceived:  s.TotalMakerFeesReceived,
		})
	}
	return &v2.GetFeesStatsForPartyResponse{
		FeesStatsForParty: statsProto,
	}, nil
}

func (t *TradingDataServiceV2) GetCurrentVolumeDiscountProgram(ctx context.Context, _ *v2.GetCurrentVolumeDiscountProgramRequest) (
	*v2.GetCurrentVolumeDiscountProgramResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetCurrentVolumeDiscountProgram")()
	volumeDiscountProgram, err := t.volumeDiscountProgramService.GetCurrentVolumeDiscountProgram(ctx)
	if err != nil {
		return nil, formatE(ErrGetCurrentVolumeDiscountProgram, err)
	}

	return &v2.GetCurrentVolumeDiscountProgramResponse{
		CurrentVolumeDiscountProgram: volumeDiscountProgram.ToProto(),
	}, nil
}

func (t *TradingDataServiceV2) GetVolumeDiscountStats(ctx context.Context, req *v2.GetVolumeDiscountStatsRequest) (
	*v2.GetVolumeDiscountStatsResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVolumeDiscountStats")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	stats, pageInfo, err := t.volumeDiscountStatsService.Stats(ctx, req.AtEpoch, req.PartyId, pagination)
	if err != nil {
		return nil, formatE(ErrGetVolumeDiscountStats, err)
	}

	edges, err := makeEdges[*v2.VolumeDiscountStatsEdge](stats)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.GetVolumeDiscountStatsResponse{
		Stats: &v2.VolumeDiscountStatsConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}

// ObserveTransactionResults opens a subscription to the transaction results.
func (t *TradingDataServiceV2) ObserveTransactionResults(req *v2.ObserveTransactionResultsRequest, srv v2.TradingDataService_ObserveTransactionResultsServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	tradesChan, ref := t.transactionResults.Observe(ctx, t.config.StreamRetries, req.PartyIds, req.Hashes, req.Status)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Transaction results subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "TransactionResults", tradesChan, ref, func(results []events.TransactionResult) error {
		protos := make([]*eventspb.TransactionResult, 0, len(results))
		for _, v := range results {
			p := v.Proto()
			protos = append(protos, &p)
		}

		batches := batch(protos, snapshotPageSize)

		for _, batch := range batches {
			response := &v2.ObserveTransactionResultsResponse{TransactionResults: batch}
			if err := srv.Send(response); err != nil {
				return errors.Wrap(err, "sending transaction results")
			}
		}
		return nil
	})
}

func (t *TradingDataServiceV2) EstimateTransferFee(ctx context.Context, req *v2.EstimateTransferFeeRequest) (
	*v2.EstimateTransferFeeResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateTransferFee")()

	amount, overflow := num.UintFromString(req.Amount, 10)
	if overflow || amount.IsNegative() {
		return nil, formatE(ErrInvalidTransferAmount)
	}

	transferFeeMaxQuantumAmountParam, err := t.networkParameterService.GetByKey(ctx, netparams.TransferFeeMaxQuantumAmount)
	if err != nil {
		return nil, formatE(ErrGetNetworkParameters, errors.Wrapf(err, "key: %s", netparams.TransferFeeMaxQuantumAmount))
	}

	transferFeeMaxQuantumAmount, _ := num.DecimalFromString(transferFeeMaxQuantumAmountParam.Value)

	transferFeeFactorParam, err := t.networkParameterService.GetByKey(ctx, netparams.TransferFeeFactor)
	if err != nil {
		return nil, formatE(ErrGetNetworkParameters, errors.Wrapf(err, "key: %s", netparams.TransferFeeFactor))
	}

	transferFeeFactor, _ := num.DecimalFromString(transferFeeFactorParam.Value)

	asset, err := t.AssetService.GetByID(ctx, req.AssetId)
	if err != nil {
		return nil, formatE(ErrAssetServiceGetByID, err)
	}

	// if we can't get a transfer fee discount, the discount should just be 0, i.e. no discount available
	transferFeesDiscount, err := t.transfersService.GetCurrentTransferFeeDiscount(
		ctx,
		entities.PartyID(req.FromAccount),
		entities.AssetID(req.AssetId),
	)
	if err != nil {
		transferFeesDiscount = &entities.TransferFeesDiscount{
			Amount: decimal.Zero,
		}
	}

	accumulatedDiscount, _ := num.UintFromDecimal(transferFeesDiscount.Amount)

	fee, discount := banking.EstimateFee(
		asset.Quantum,
		transferFeeMaxQuantumAmount,
		transferFeeFactor,
		amount,
		accumulatedDiscount,
		req.FromAccount,
		req.FromAccountType,
		req.ToAccount,
	)

	return &v2.EstimateTransferFeeResponse{
		Fee:      fee.String(),
		Discount: discount.String(),
	}, nil
}

func (t *TradingDataServiceV2) GetTotalTransferFeeDiscount(ctx context.Context, req *v2.GetTotalTransferFeeDiscountRequest) (
	*v2.GetTotalTransferFeeDiscountResponse, error,
) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetTotalTransferFeeDiscount")()

	transferFeesDiscount, err := t.transfersService.GetCurrentTransferFeeDiscount(
		ctx,
		entities.PartyID(req.PartyId),
		entities.AssetID(req.AssetId),
	)
	if err != nil {
		return nil, formatE(ErrTransferServiceGetFeeDiscount, err)
	}

	accumulatedDiscount, _ := num.UintFromDecimal(transferFeesDiscount.Amount)
	return &v2.GetTotalTransferFeeDiscountResponse{
		TotalDiscount: accumulatedDiscount.String(),
	}, nil
}

func (t *TradingDataServiceV2) ListGames(ctx context.Context, req *v2.ListGamesRequest) (*v2.ListGamesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListGames")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	var (
		teamID  *entities.TeamID
		partyID *entities.PartyID
	)

	if req.TeamId != nil {
		teamID = ptr.From(entities.TeamID(*req.TeamId))
	}

	if req.PartyId != nil {
		partyID = ptr.From(entities.PartyID(*req.PartyId))
	}

	games, pageInfo, err := t.gamesService.ListGames(ctx, req.GameId, req.EntityScope, req.EpochFrom, req.EpochTo, teamID, partyID, pagination)
	if err != nil {
		return nil, formatE(ErrListGames, err)
	}

	edges, err := makeEdges[*v2.GameEdge](games)
	if err != nil {
		return nil, formatE(err)
	}
	return &v2.ListGamesResponse{
		Games: &v2.GamesConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}

func (t *TradingDataServiceV2) ListPartyMarginModes(ctx context.Context, req *v2.ListPartyMarginModesRequest) (*v2.ListPartyMarginModesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListPartyMarginModes")()

	pagination, err := entities.CursorPaginationFromProto(req.Pagination)
	if err != nil {
		return nil, formatE(ErrInvalidPagination, err)
	}

	filters := sqlstore.ListPartyMarginModesFilters{}
	if req.MarketId != nil {
		filters.MarketID = ptr.From(entities.MarketID(*req.MarketId))
	}
	if req.PartyId != nil {
		filters.PartyID = ptr.From(entities.PartyID(*req.PartyId))
	}

	modes, pageInfo, err := t.marginModesService.ListPartyMarginModes(ctx, pagination, filters)
	if err != nil {
		return nil, formatE(ErrListPartyMarginModes, err)
	}

	edges, err := makeEdges[*v2.PartyMarginModeEdge](modes)
	if err != nil {
		return nil, formatE(err)
	}

	return &v2.ListPartyMarginModesResponse{
		PartyMarginModes: &v2.PartyMarginModesConnection{
			Edges:    edges,
			PageInfo: pageInfo.ToProto(),
		},
	}, nil
}
