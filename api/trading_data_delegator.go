package api

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/sqlstore"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	"code.vegaprotocol.io/protos/vega"
	"google.golang.org/grpc/codes"
)

type tradingDataDelegator struct {
	*tradingDataService
	orderStore      *sqlstore.Orders
	tradeStore      *sqlstore.Trades
	assetStore      *sqlstore.Assets
	accountStore    *sqlstore.Accounts
	marketDataStore *sqlstore.MarketData
	rewardStore     *sqlstore.Rewards
	marketsStore    *sqlstore.Markets
	delegationStore *sqlstore.Delegations
	epochStore      *sqlstore.Epochs
}

var defaultEntityPagination = entities.Pagination{
	Skip:       0,
	Limit:      50,
	Descending: true,
}

/****************************** Epochs **************************************/

func (t *tradingDataDelegator) GetEpoch(ctx context.Context, req *protoapi.GetEpochRequest) (*protoapi.GetEpochResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetEpoch SQL")()

	var epoch entities.Epoch
	var err error

	if req.GetId() == 0 {
		epoch, err = t.epochStore.GetCurrent(ctx)
	} else {
		epoch, err = t.epochStore.Get(ctx, int64(req.GetId()))
	}

	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoEpoch := epoch.ToProto()

	delegations, err := t.delegationStore.Get(ctx, nil, nil, &epoch.ID, nil)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoDelegations := make([]*vega.Delegation, len(delegations))
	for i, delegation := range delegations {
		protoDelegations[i] = delegation.ToProto()
	}
	protoEpoch.Delegations = protoDelegations

	// TODO: Add in nodes once we've got them in the sql store too

	return &protoapi.GetEpochResponse{
		Epoch: protoEpoch,
	}, nil
}

/****************************** Delegations **************************************/

func (t *tradingDataDelegator) Delegations(ctx context.Context,
	req *protoapi.DelegationsRequest) (*protoapi.DelegationsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Delegations SQL")()

	var delegations []entities.Delegation
	var err error

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	var epochID *int64
	var partyID *string
	var nodeID *string

	if req.EpochSeq != "" {
		epochNum, err := strconv.ParseInt(req.EpochSeq, 10, 64)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, err)
		}
		epochID = &epochNum
	}

	if req.Party != "" {
		partyID = &req.Party
	}

	if req.NodeId != "" {
		nodeID = &req.NodeId
	}

	delegations, err = t.delegationStore.Get(ctx, partyID, nodeID, epochID, &p)

	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoDelegations := make([]*vega.Delegation, len(delegations))
	for i, delegation := range delegations {
		protoDelegations[i] = delegation.ToProto()
	}

	return &protoapi.DelegationsResponse{
		Delegations: protoDelegations,
	}, nil
}

/****************************** Rewards **************************************/

func (t *tradingDataDelegator) GetRewards(ctx context.Context,
	req *protoapi.GetRewardsRequest) (*protoapi.GetRewardsResponse, error) {

	defer metrics.StartAPIRequestAndTimeGRPC("GetRewards-SQL")()
	if len(req.PartyId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrGetRewards)
	}

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	var rewards []entities.Reward
	var err error

	if len(req.AssetId) <= 0 {
		rewards, err = t.rewardStore.Get(ctx, &req.PartyId, nil, &p)
	} else {
		rewards, err = t.rewardStore.Get(ctx, &req.PartyId, &req.AssetId, &p)
	}

	if err != nil {
		return nil, apiError(codes.Internal, ErrGetRewards, err)
	}

	protoRewards := make([]*vega.Reward, len(rewards))
	for i, reward := range rewards {
		protoRewards[i] = reward.ToProto()
	}

	return &protoapi.GetRewardsResponse{Rewards: protoRewards}, nil
}

func (t *tradingDataDelegator) GetRewardSummaries(ctx context.Context,
	req *protoapi.GetRewardSummariesRequest) (*protoapi.GetRewardSummariesResponse, error) {

	defer metrics.StartAPIRequestAndTimeGRPC("GetRewardSummaries-SQL")()

	if len(req.PartyId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrTradeServiceGetByParty)
	}

	var summaries []entities.RewardSummary
	var err error

	if len(req.AssetId) <= 0 {
		summaries, err = t.rewardStore.GetSummaries(ctx, &req.PartyId, nil)
	} else {
		summaries, err = t.rewardStore.GetSummaries(ctx, &req.PartyId, &req.AssetId)
	}

	if err != nil {
		return nil, apiError(codes.Internal, ErrGetRewards, err)
	}

	protoSummaries := make([]*vega.RewardSummary, len(summaries))
	for i, summary := range summaries {
		protoSummaries[i] = summary.ToProto()
	}

	return &protoapi.GetRewardSummariesResponse{Summaries: protoSummaries}, nil
}

/****************************** Trades **************************************/
// TradesByParty provides a list of trades for the given party.
// Pagination: Optional. If not provided, defaults are used.
func (t *tradingDataDelegator) TradesByParty(ctx context.Context,
	req *protoapi.TradesByPartyRequest) (*protoapi.TradesByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesByParty-SQL")()

	p := defaultEntityPagination
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	trades, err := t.tradeStore.GetByParty(ctx, req.PartyId, &req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByParty, err)
	}

	protoTrades := tradesToProto(trades)

	return &protoapi.TradesByPartyResponse{Trades: protoTrades}, nil
}

func tradesToProto(trades []entities.Trade) []*vega.Trade {
	protoTrades := []*vega.Trade{}
	for _, trade := range trades {
		protoTrades = append(protoTrades, trade.ToProto())
	}
	return protoTrades
}

// TradesByOrder provides a list of the trades that correspond to a given order.
func (t *tradingDataDelegator) TradesByOrder(ctx context.Context,
	req *protoapi.TradesByOrderRequest) (*protoapi.TradesByOrderResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesByOrder-SQL")()

	trades, err := t.tradeStore.GetByOrderID(ctx, req.OrderId, nil, defaultEntityPagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByOrderID, err)
	}

	protoTrades := tradesToProto(trades)

	return &protoapi.TradesByOrderResponse{Trades: protoTrades}, nil
}

// TradesByMarket provides a list of trades for a given market.
// Pagination: Optional. If not provided, defaults are used.
func (t *tradingDataDelegator) TradesByMarket(ctx context.Context, req *protoapi.TradesByMarketRequest) (*protoapi.TradesByMarketResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesByMarket-SQL")()

	p := defaultEntityPagination
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	trades, err := t.tradeStore.GetByMarket(ctx, req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}

	protoTrades := tradesToProto(trades)
	return &protoapi.TradesByMarketResponse{
		Trades: protoTrades,
	}, nil
}

// LastTrade provides the last trade for the given market.
func (t *tradingDataDelegator) LastTrade(ctx context.Context,
	req *protoapi.LastTradeRequest) (*protoapi.LastTradeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("LastTrade-SQL")()

	if len(req.MarketId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}

	p := entities.Pagination{
		Skip:       0,
		Limit:      1,
		Descending: true,
	}

	trades, err := t.tradeStore.GetByMarket(ctx, req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}

	protoTrades := tradesToProto(trades)

	if len(protoTrades) > 0 && protoTrades[0] != nil {
		return &protoapi.LastTradeResponse{Trade: protoTrades[0]}, nil
	}
	// No trades found on the market yet (and no errors)
	// this can happen at the beginning of a new market
	return &protoapi.LastTradeResponse{}, nil
}

func (t *tradingDataDelegator) OrderByID(ctx context.Context, req *protoapi.OrderByIDRequest) (*protoapi.OrderByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderByID-SQL")()

	if len(req.OrderId) == 0 {
		return nil, ErrMissingOrderIDParameter
	}

	version := int32(req.Version)
	order, err := t.orderStore.GetByOrderID(ctx, req.OrderId, &version)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	resp := &protoapi.OrderByIDResponse{Order: order.ToProto()}
	return resp, nil
}

func (t *tradingDataDelegator) OrderByMarketAndID(ctx context.Context,
	req *protoapi.OrderByMarketAndIDRequest) (*protoapi.OrderByMarketAndIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderByMarketAndID-SQL")()

	// This function is no longer needed; IDs are globally unique now, but keep it for compatibility for now
	if len(req.OrderId) == 0 {
		return nil, ErrMissingOrderIDParameter
	}

	order, err := t.orderStore.GetByOrderID(ctx, req.OrderId, nil)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	resp := &protoapi.OrderByMarketAndIDResponse{Order: order.ToProto()}
	return resp, nil
}

// OrderByReference provides the (possibly not yet accepted/rejected) order.
func (t *tradingDataDelegator) OrderByReference(ctx context.Context, req *protoapi.OrderByReferenceRequest) (*protoapi.OrderByReferenceResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderByReference-SQL")()

	orders, err := t.orderStore.GetByReference(ctx, req.Reference, entities.Pagination{})
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByReference, err)
	}

	if len(orders) == 0 {
		return nil, ErrOrderNotFound
	}
	return &protoapi.OrderByReferenceResponse{
		Order: orders[0].ToProto(),
	}, nil
}

func (t *tradingDataDelegator) OrdersByParty(ctx context.Context,
	req *protoapi.OrdersByPartyRequest) (*protoapi.OrdersByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrdersByParty-SQL")()

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	orders, err := t.orderStore.GetByParty(ctx, req.PartyId, p)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByParty, err)
	}

	pbOrders := make([]*vega.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = order.ToProto()
	}

	return &protoapi.OrdersByPartyResponse{
		Orders: pbOrders,
	}, nil
}

func toEntityPagination(pagination *protoapi.Pagination) entities.Pagination {
	return entities.Pagination{
		Skip:       pagination.Skip,
		Limit:      pagination.Limit,
		Descending: pagination.Descending,
	}
}

func (t *tradingDataDelegator) AssetByID(ctx context.Context, req *protoapi.AssetByIDRequest) (*protoapi.AssetByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("AssetByID-SQL")()
	if len(req.Id) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing ID"))
	}

	asset, err := t.assetStore.GetByID(ctx, req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	return &protoapi.AssetByIDResponse{
		Asset: asset.ToProto(),
	}, nil
}

func (t *tradingDataDelegator) Assets(ctx context.Context, _ *protoapi.AssetsRequest) (*protoapi.AssetsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Assets-SQL")()

	assets, _ := t.assetStore.GetAll(ctx)

	out := make([]*vega.Asset, 0, len(assets))
	for _, v := range assets {
		out = append(out, v.ToProto())
	}
	return &protoapi.AssetsResponse{
		Assets: out,
	}, nil
}

func isValidAccountType(accountType vega.AccountType, validAccountTypes ...vega.AccountType) bool {
	for _, vt := range validAccountTypes {
		if accountType == vt {
			return true
		}
	}

	return false
}

func (t *tradingDataDelegator) PartyAccounts(ctx context.Context, req *protoapi.PartyAccountsRequest) (*protoapi.PartyAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("PartyAccounts_SQL")()

	// This is just nicer to read and update if the list of valid account types change than multiple AND statements
	if !isValidAccountType(req.Type, vega.AccountType_ACCOUNT_TYPE_GENERAL, vega.AccountType_ACCOUNT_TYPE_MARGIN,
		vega.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW, vega.AccountType_ACCOUNT_TYPE_BOND, vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED) {
		return nil, errors.New("invalid type for query, only GENERAL, MARGIN, LOCK_WITHDRAW AND BOND accounts for a party supported")
	}

	pagination := entities.Pagination{}

	filter := entities.AccountFilter{
		Asset:        toAccountsFilterAsset(req.Asset),
		Parties:      toAccountsFilterParties(req.PartyId),
		AccountTypes: toAccountsFilterAccountTypes(req.Type),
		Markets:      toAccountsFilterMarkets(req.MarketId),
	}

	accountBalances, err := t.accountStore.QueryBalances(ctx, filter, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetPartyAccounts, err)
	}

	return &protoapi.PartyAccountsResponse{
		Accounts: accountBalancesToProtoAccountList(accountBalances),
	}, nil
}

func toAccountsFilterAccountTypes(accountTypes ...vega.AccountType) []vega.AccountType {
	accountTypesProto := make([]vega.AccountType, 0)

	for _, accountType := range accountTypes {
		if accountType == vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
			return nil
		}

		accountTypesProto = append(accountTypesProto, accountType)
	}

	return accountTypesProto
}

func accountBalancesToProtoAccountList(accounts []entities.AccountBalance) []*vega.Account {
	accountsProto := make([]*vega.Account, 0, len(accounts))

	for _, acc := range accounts {
		accountsProto = append(accountsProto, acc.ToProto())
	}

	return accountsProto
}

func toAccountsFilterAsset(assetID string) entities.Asset {
	asset := entities.Asset{}

	if len(assetID) > 0 {
		assetIDBytes, _ := hex.DecodeString(assetID)
		asset.ID = assetIDBytes
	}

	return asset
}

func toAccountsFilterParties(partyIDs ...string) []entities.Party {
	parties := make([]entities.Party, 0, len(partyIDs))
	for _, id := range partyIDs {
		if id == "" {
			continue
		}

		idBytes, err := hex.DecodeString(id)

		if err != nil {
			continue
		}

		party := entities.Party{
			ID: idBytes,
		}
		parties = append(parties, party)
	}

	return parties
}

func toAccountsFilterMarkets(marketIDs ...string) []entities.Market {
	markets := make([]entities.Market, 0, len(marketIDs))
	for _, id := range marketIDs {
		if id == "" {
			continue
		}
		idBytes, err := hex.DecodeString(id)
		if err != nil {
			continue
		}

		market := entities.Market{
			ID: idBytes,
		}
		markets = append(markets, market)
	}

	return markets
}

func (t *tradingDataDelegator) MarketAccounts(ctx context.Context,
	req *protoapi.MarketAccountsRequest) (*protoapi.MarketAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketAccounts")()

	filter := entities.AccountFilter{
		Asset:   toAccountsFilterAsset(req.Asset),
		Markets: toAccountsFilterMarkets(req.MarketId),
		AccountTypes: toAccountsFilterAccountTypes(
			vega.AccountType_ACCOUNT_TYPE_INSURANCE,
			vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY,
		),
	}

	pagination := entities.Pagination{}

	accountBalances, err := t.accountStore.QueryBalances(ctx, filter, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetMarketAccounts, err)
	}

	return &protoapi.MarketAccountsResponse{
		Accounts: accountBalancesToProtoAccountList(accountBalances),
	}, nil
}

func (t *tradingDataDelegator) FeeInfrastructureAccounts(ctx context.Context,
	req *protoapi.FeeInfrastructureAccountsRequest) (*protoapi.FeeInfrastructureAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("FeeInfrastructureAccounts")()

	filter := entities.AccountFilter{
		Asset: toAccountsFilterAsset(req.Asset),
		AccountTypes: toAccountsFilterAccountTypes(
			vega.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE,
		),
	}
	pagination := entities.Pagination{}

	accountBalances, err := t.accountStore.QueryBalances(ctx, filter, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetFeeInfrastructureAccounts, err)
	}
	return &protoapi.FeeInfrastructureAccountsResponse{
		Accounts: accountBalancesToProtoAccountList(accountBalances),
	}, nil
}

func (t *tradingDataDelegator) GlobalRewardPoolAccounts(ctx context.Context,
	req *protoapi.GlobalRewardPoolAccountsRequest) (*protoapi.GlobalRewardPoolAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GloabRewardPoolAccounts")()
	filter := entities.AccountFilter{
		Asset: toAccountsFilterAsset(req.Asset),
		AccountTypes: toAccountsFilterAccountTypes(
			vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
		),
	}
	pagination := entities.Pagination{}

	accountBalances, err := t.accountStore.QueryBalances(ctx, filter, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetGlobalRewardPoolAccounts, err)
	}
	return &protoapi.GlobalRewardPoolAccountsResponse{
		Accounts: accountBalancesToProtoAccountList(accountBalances),
	}, nil
}

// MarketDataByID provides market data for the given ID.
func (t *tradingDataDelegator) MarketDataByID(ctx context.Context, req *protoapi.MarketDataByIDRequest) (*protoapi.MarketDataByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketDataByID_SQL")()

	// validate the market exist
	if req.MarketId != "" {
		_, err := t.marketsStore.GetByID(ctx, req.MarketId)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, ErrInvalidMarketID, err)
		}
	}

	md, err := t.marketDataStore.GetMarketDataByID(ctx, req.MarketId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetMarketData, err)
	}
	return &protoapi.MarketDataByIDResponse{
		MarketData: md.ToProto(),
	}, nil
}

// MarketsData provides all market data for all markets on this network.
func (t *tradingDataDelegator) MarketsData(ctx context.Context, _ *protoapi.MarketsDataRequest) (*protoapi.MarketsDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketsData_SQL")()
	mds, _ := t.marketDataStore.GetMarketsData(ctx)

	mdptrs := make([]*vega.MarketData, 0, len(mds))
	for _, v := range mds {
		mdptrs = append(mdptrs, v.ToProto())
	}

	return &protoapi.MarketsDataResponse{
		MarketsData: mdptrs,
	}, nil
}

// MarketByID provides the given market.
func (t *tradingDataDelegator) MarketByID(ctx context.Context, req *protoapi.MarketByIDRequest) (*protoapi.MarketByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketByID_SQL")()

	mkt, err := validateMarketSQL(ctx, req.MarketId, t.marketsStore)
	if err != nil {
		return nil, err // validateMarket already returns an API error, no need to additionally wrap
	}

	return &protoapi.MarketByIDResponse{
		Market: mkt,
	}, nil
}

func validateMarketSQL(ctx context.Context, marketID string, marketsStore *sqlstore.Markets) (*vega.Market, error) {
	if len(marketID) == 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}

	market, err := marketsStore.GetByID(ctx, marketID)

	if err != nil {
		// We return nil for error as we do not want
		// to return an error when a market is not found
		// but just a nil value.
		return nil, nil
	}

	mkt, err := market.ToProto()

	if err != nil {
		return nil, nil
	}

	return mkt, nil
}

// Markets provides a list of all current markets that exist on the VEGA platform.
func (t *tradingDataDelegator) Markets(ctx context.Context, _ *protoapi.MarketsRequest) (*protoapi.MarketsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Markets_SQL")()
	markets, err := t.marketsStore.GetAll(ctx, entities.Pagination{})
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetMarkets, err)
	}

	results := make([]*vega.Market, 0, len(markets))
	for _, m := range markets {
		mkt, err := m.ToProto()
		if err != nil {
			continue
		}

		results = append(results, mkt)
	}

	return &protoapi.MarketsResponse{
		Markets: results,
	}, nil
}
