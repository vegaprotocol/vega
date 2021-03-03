package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/pkg/errors"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
)

var defaultPagination = protoapi.Pagination{
	Skip:       0,
	Limit:      50,
	Descending: true,
}

// VegaTime ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vega_time_mock.go -package mocks code.vegaprotocol.io/vega/api VegaTime
type VegaTime interface {
	GetTimeNow() (time.Time, error)
}

// OrderService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_service_mock.go -package mocks code.vegaprotocol.io/vega/api OrderService
type OrderService interface {
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) (orders []*types.Order, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool) (orders []*types.Order, err error)
	GetByMarketAndID(ctx context.Context, market string, id string) (order *types.Order, err error)
	GetByOrderID(ctx context.Context, id string, version uint64) (order *types.Order, err error)
	GetByReference(ctx context.Context, ref string) (order *types.Order, err error)
	GetAllVersionsByOrderID(ctx context.Context, id string, skip, limit uint64, descending bool) (orders []*types.Order, err error)
	ObserveOrders(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Order, ref uint64)
	GetOrderSubscribersCount() int32
}

// TradeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_service_mock.go -package mocks code.vegaprotocol.io/vega/api TradeService
type TradeService interface {
	GetByOrderID(ctx context.Context, orderID string) ([]*types.Trade, error)
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) (trades []*types.Trade, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, marketID *string) (trades []*types.Trade, err error)
	GetPositionsByParty(ctx context.Context, party, marketID string) (positions []*types.Position, err error)
	ObserveTrades(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Trade, ref uint64)
	ObservePositions(ctx context.Context, retries int, party, market string) (positions <-chan *types.Position, ref uint64)
	GetTradeSubscribersCount() int32
	GetPositionsSubscribersCount() int32
}

// CandleService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_service_mock.go -package mocks code.vegaprotocol.io/vega/api CandleService
type CandleService interface {
	GetCandles(ctx context.Context, market string, since time.Time, interval types.Interval) (candles []*types.Candle, err error)
	ObserveCandles(ctx context.Context, retries int, market *string, interval *types.Interval) (candleCh <-chan *types.Candle, ref uint64)
	GetCandleSubscribersCount() int32
}

// MarketService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_service_mock.go -package mocks code.vegaprotocol.io/vega/api MarketService
type MarketService interface {
	GetByID(ctx context.Context, name string) (*types.Market, error)
	GetAll(ctx context.Context) ([]*types.Market, error)
	GetDepth(ctx context.Context, market string, limit uint64) (marketDepth *types.MarketDepth, err error)
	ObserveDepth(ctx context.Context, retries int, market string) (depth <-chan *types.MarketDepth, ref uint64)
	ObserveDepthUpdates(ctx context.Context, retries int, market string) (depth <-chan *types.MarketDepthUpdate, ref uint64)
	GetMarketDepthSubscribersCount() int32
	ObserveMarketsData(ctx context.Context, retries int, marketID string) (<-chan []types.MarketData, uint64)
	GetMarketDataSubscribersCount() int32
	GetMarketDataByID(marketID string) (types.MarketData, error)
	GetMarketsData() []types.MarketData
}

// PartyService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/party_service_mock.go -package mocks code.vegaprotocol.io/vega/api PartyService
type PartyService interface {
	GetByID(ctx context.Context, id string) (*types.Party, error)
	GetAll(ctx context.Context) ([]*types.Party, error)
}

// BlockchainClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_client_mock.go -package mocks code.vegaprotocol.io/vega/api BlockchainClient
type BlockchainClient interface {
	SubmitTransaction(ctx context.Context, tx *types.SignedBundle, ty protoapi.SubmitTransactionRequest_Type) error
	GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error)
	GetChainID(ctx context.Context) (chainID string, err error)
	GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error)
	GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error)
	GetUnconfirmedTxCount(ctx context.Context) (count int, err error)
	Health() (*tmctypes.ResultHealth, error)
}

// AccountsService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/accounts_service_mock.go -package mocks code.vegaprotocol.io/vega/api AccountsService
type AccountsService interface {
	GetPartyAccounts(partyID, marketID, asset string, ty types.AccountType) ([]*types.Account, error)
	GetMarketAccounts(marketID, asset string) ([]*types.Account, error)
	GetFeeInfrastructureAccounts(asset string) ([]*types.Account, error)
	ObserveAccounts(ctx context.Context, retries int, marketID, partyID, asset string, ty types.AccountType) (candleCh <-chan []*types.Account, ref uint64)
	GetAccountSubscribersCount() int32
	PrepareWithdraw(context.Context, *types.WithdrawSubmission) error
}

// TransferResponseService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_response_service_mock.go -package mocks code.vegaprotocol.io/vega/api TransferResponseService
type TransferResponseService interface {
	ObserveTransferResponses(ctx context.Context, retries int) (<-chan []*types.TransferResponse, uint64)
}

// GovernanceDataService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_data_service_mock.go -package mocks code.vegaprotocol.io/vega/api  GovernanceDataService
type GovernanceDataService interface {
	GetProposals(inState *types.Proposal_State) []*types.GovernanceData
	GetProposalsByParty(partyID string, inState *types.Proposal_State) []*types.GovernanceData
	GetVotesByParty(partyID string) []*types.Vote

	GetProposalByID(id string) (*types.GovernanceData, error)
	GetProposalByReference(ref string) (*types.GovernanceData, error)

	GetNewMarketProposals(inState *types.Proposal_State) []*types.GovernanceData
	GetUpdateMarketProposals(marketID string, inState *types.Proposal_State) []*types.GovernanceData
	GetNetworkParametersProposals(inState *types.Proposal_State) []*types.GovernanceData
	GetNewAssetProposals(inState *types.Proposal_State) []*types.GovernanceData

	ObserveGovernance(ctx context.Context, retries int) <-chan []types.GovernanceData
	ObservePartyProposals(ctx context.Context, retries int, partyID string) <-chan []types.GovernanceData
	ObservePartyVotes(ctx context.Context, retries int, partyID string) <-chan []types.Vote
	ObserveProposalVotes(ctx context.Context, retries int, proposalID string) <-chan []types.Vote
}

// RiskService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_service_mock.go -package mocks code.vegaprotocol.io/vega/api  RiskService
type RiskService interface {
	ObserveMarginLevels(
		ctx context.Context, retries int, partyID, marketID string,
	) (<-chan []types.MarginLevels, uint64)
	GetMarginLevelsSubscribersCount() int32
	GetMarginLevelsByID(partyID, marketID string) ([]types.MarginLevels, error)
	EstimateMargin(ctx context.Context, order *types.Order) (*types.MarginLevels, error)
}

// Notary ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_service_mock.go -package mocks code.vegaprotocol.io/vega/api  NotaryService
type NotaryService interface {
	GetByID(id string) ([]types.NodeSignature, error)
}

// Withdrawal ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/withdrawal_service_mock.go -package mocks code.vegaprotocol.io/vega/api  WithdrawalService
type WithdrawalService interface {
	GetByID(id string) (types.Withdrawal, error)
	GetByParty(party string, openOnly bool) []types.Withdrawal
}

// Deposit ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/deposit_service_mock.go -package mocks code.vegaprotocol.io/vega/api  DepositService
type DepositService interface {
	GetByID(id string) (types.Deposit, error)
	GetByParty(party string, openOnly bool) []types.Deposit
}

// AssetService Provides access to assets approved/pending approval in the current network state
//go:generate go run github.com/golang/mock/mockgen -destination mocks/asset_service_mock.go -package mocks code.vegaprotocol.io/vega/api  AssetService
type AssetService interface {
	GetByID(id string) (*types.Asset, error)
	GetAll() ([]types.Asset, error)
}

// FeeService Provides apis to estimate fees
//go:generate go run github.com/golang/mock/mockgen -destination mocks/fee_service_mock.go -package mocks code.vegaprotocol.io/vega/api  FeeService
type FeeService interface {
	EstimateFee(context.Context, *types.Order) (*types.Fee, error)
}

// NetParamsService Provides apis to estimate fees
//go:generate go run github.com/golang/mock/mockgen -destination mocks/net_params_service_mock.go -package mocks code.vegaprotocol.io/vega/api  NetParamsService
type NetParamsService interface {
	GetAll() []types.NetworkParameter
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/event_service_mock.go -package mocks code.vegaprotocol.io/vega/api EventService
type EventService interface {
	ObserveEvents(ctx context.Context, retries int, eTypes []events.Type, batchSize int, filters ...subscribers.EventFilter) (<-chan []*types.BusEvent, chan<- int)
}

type tradingDataService struct {
	log                     *logging.Logger
	Config                  Config
	Client                  BlockchainClient
	Stats                   *stats.Stats
	TimeService             VegaTime
	OrderService            OrderService
	TradeService            TradeService
	CandleService           CandleService
	MarketService           MarketService
	PartyService            PartyService
	AccountsService         AccountsService
	RiskService             RiskService
	NotaryService           NotaryService
	TransferResponseService TransferResponseService
	governanceService       GovernanceDataService
	AssetService            AssetService
	FeeService              FeeService
	eventService            EventService
	statusChecker           *monitoring.Status
	WithdrawalService       WithdrawalService
	DepositService          DepositService
	MarketDepthService      *subscribers.MarketDepthBuilder
	NetParamsService        NetParamsService
	LiquidityService        LiquidityService
	ctx                     context.Context

	chainID                  string
	genesisTime              time.Time
	hasGenesisTimeAndChainID uint32
	mu                       sync.Mutex

	netInfo   *tmctypes.ResultNetInfo
	netInfoMu sync.RWMutex
}

func (t *tradingDataService) LiquidityProvisions(ctx context.Context, req *protoapi.LiquidityProvisionsRequest) (*protoapi.LiquidityProvisionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("LiquidityProvisions")()
	lps, err := t.LiquidityService.Get(req.Party, req.Market)
	if err != nil {
		return nil, err
	}
	out := make([]*types.LiquidityProvision, 0, len(lps))
	for _, v := range lps {
		v := v
		out = append(out, &v)
	}
	return &protoapi.LiquidityProvisionsResponse{
		LiquidityProvisions: out,
	}, nil
}

func (t *tradingDataService) NetworkParameters(ctx context.Context, req *protoapi.NetworkParametersRequest) (*protoapi.NetworkParametersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("NetworkParameters")()
	nps := t.NetParamsService.GetAll()
	out := make([]*types.NetworkParameter, 0, len(nps))
	for _, v := range nps {
		v := v
		out = append(out, &v)
	}
	return &protoapi.NetworkParametersResponse{
		NetworkParameters: out,
	}, nil
}

func (t *tradingDataService) EstimateMargin(ctx context.Context, req *protoapi.EstimateMarginRequest) (*protoapi.EstimateMarginResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateMargin")()
	if req.Order == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("missing order"))
	}

	margin, err := t.RiskService.EstimateMargin(ctx, req.Order)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.EstimateMarginResponse{
		MarginLevels: margin,
	}, nil
}

func (t *tradingDataService) EstimateFee(ctx context.Context, req *protoapi.EstimateFeeRequest) (*protoapi.EstimateFeeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateFee")()
	if req.Order == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("missing order"))
	}

	fee, err := t.FeeService.EstimateFee(ctx, req.Order)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.EstimateFeeResponse{
		Fee: fee,
	}, nil
}

func (t *tradingDataService) ERC20WithdrawalApproval(ctx context.Context, req *protoapi.ERC20WithdrawalApprovalRequest) (*protoapi.ERC20WithdrawalApprovalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ERC20WithdrawalApproval")()
	if len(req.WithdrawalId) <= 0 {
		return nil, ErrMissingWithdrawalID
	}

	// first here we gonna get the withdrawal by its ID,
	withdrawal, err := t.WithdrawalService.GetByID(req.WithdrawalId)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	// then we get the signature and pack them altogether
	signatures, err := t.NotaryService.GetByID(req.WithdrawalId)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}
	// now we pack them
	pack := "0x"
	for _, v := range signatures {
		pack = fmt.Sprintf("%v%v", pack, hex.EncodeToString(v.Sig))
	}
	// now the signature should have the form:
	// 0x + sig1 + sig2 + ... + sigN in hex encoded form

	// then we'll get the asset source to retrieve the asset erc20 ethereum address
	assets, err := t.Assets(ctx, &protoapi.AssetsRequest{})
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	var address string
	for _, v := range assets.Assets {
		if v.Id == withdrawal.Asset {
			switch src := v.Source.Source.(type) {
			case *types.AssetSource_Erc20:
				address = src.Erc20.ContractAddress
			default:
				return nil, fmt.Errorf("invalid asset source")
			}
		}
	}
	if len(address) <= 0 {
		return nil, fmt.Errorf("invalid erc20 token contract address")
	}

	return &protoapi.ERC20WithdrawalApprovalResponse{
		AssetSource: address,
		Amount:      fmt.Sprintf("%v", withdrawal.Amount),
		Expiry:      withdrawal.Expiry,
		Nonce:       withdrawal.Ref,
		Signatures:  pack,
	}, nil
}

func (t *tradingDataService) Withdrawal(ctx context.Context, req *protoapi.WithdrawalRequest) (*protoapi.WithdrawalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Withdrawal")()
	if len(req.Id) <= 0 {
		return nil, ErrMissingWithdrawalID
	}
	withdrawal, err := t.WithdrawalService.GetByID(req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}
	return &protoapi.WithdrawalResponse{
		Withdrawal: &withdrawal,
	}, nil
}

func (t *tradingDataService) Withdrawals(ctx context.Context, req *protoapi.WithdrawalsRequest) (*protoapi.WithdrawalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Withdrawals")()
	if len(req.PartyId) <= 0 {
		return nil, ErrMissingPartyID
	}
	withdrawals := t.WithdrawalService.GetByParty(req.PartyId, false)
	out := make([]*types.Withdrawal, 0, len(withdrawals))
	for _, v := range withdrawals {
		v := v
		out = append(out, &v)
	}
	return &protoapi.WithdrawalsResponse{
		Withdrawals: out,
	}, nil
}

func (t *tradingDataService) Deposit(ctx context.Context, req *protoapi.DepositRequest) (*protoapi.DepositResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Deposit")()
	if len(req.Id) <= 0 {
		return nil, ErrMissingDepositID
	}
	deposit, err := t.DepositService.GetByID(req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}
	return &protoapi.DepositResponse{
		Deposit: &deposit,
	}, nil
}

func (t *tradingDataService) Deposits(ctx context.Context, req *protoapi.DepositsRequest) (*protoapi.DepositsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Deposits")()
	if len(req.PartyId) <= 0 {
		return nil, ErrMissingPartyID
	}
	deposits := t.DepositService.GetByParty(req.PartyId, false)
	out := make([]*types.Deposit, 0, len(deposits))
	for _, v := range deposits {
		v := v
		out = append(out, &v)
	}
	return &protoapi.DepositsResponse{
		Deposits: out,
	}, nil
}

func (t *tradingDataService) AssetByID(ctx context.Context, req *protoapi.AssetByIDRequest) (*protoapi.AssetByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("AssetByID")()
	if len(req.Id) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing ID"))
	}

	asset, err := t.AssetService.GetByID(req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	return &protoapi.AssetByIDResponse{
		Asset: asset,
	}, nil
}

func (t *tradingDataService) Assets(ctx context.Context, req *protoapi.AssetsRequest) (*protoapi.AssetsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Assets")()
	assets, _ := t.AssetService.GetAll()
	out := make([]*types.Asset, 0, len(assets))
	for _, v := range assets {
		v := v
		out = append(out, &v)
	}
	return &protoapi.AssetsResponse{
		Assets: out,
	}, nil
}

func (t *tradingDataService) GetNodeSignaturesAggregate(ctx context.Context,
	req *protoapi.GetNodeSignaturesAggregateRequest) (*protoapi.GetNodeSignaturesAggregateResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodeSignaturesAggregate")()
	if len(req.Id) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing ID"))
	}

	sigs, err := t.NotaryService.GetByID(req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	out := make([]*types.NodeSignature, 0, len(sigs))
	for _, v := range sigs {
		v := v
		out = append(out, &v)
	}

	return &protoapi.GetNodeSignaturesAggregateResponse{
		Signatures: out,
	}, nil
}

// OrdersByMarket provides a list of orders for a given market.
// Pagination: Optional. If not provided, defaults are used.
// Returns a list of orders sorted by timestamp descending (most recent first).
func (t *tradingDataService) OrdersByMarket(ctx context.Context,
	request *protoapi.OrdersByMarketRequest) (*protoapi.OrdersByMarketResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrdersByMarket")()

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	o, err := t.OrderService.GetByMarket(ctx, request.MarketId, p.Skip, p.Limit, p.Descending)
	if err != nil {
		return nil, apiError(codes.Internal, ErrOrderServiceGetByMarket, err)
	}

	var response = &protoapi.OrdersByMarketResponse{}
	if len(o) > 0 {
		response.Orders = o
	}

	return response, nil
}

// OrdersByParty provides a list of orders for a given party.
// Pagination: Optional. If not provided, defaults are used.
// Returns a list of orders sorted by timestamp descending (most recent first).
func (t *tradingDataService) OrdersByParty(ctx context.Context,
	request *protoapi.OrdersByPartyRequest) (*protoapi.OrdersByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrdersByParty")()

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	o, err := t.OrderService.GetByParty(ctx, request.PartyId, p.Skip, p.Limit, p.Descending)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByParty, err)
	}

	var response = &protoapi.OrdersByPartyResponse{}
	if len(o) > 0 {
		response.Orders = o
	}

	return response, nil
}

// Markets provides a list of all current markets that exist on the VEGA platform.
func (t *tradingDataService) Markets(ctx context.Context, _ *protoapi.MarketsRequest) (*protoapi.MarketsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Markets")()
	markets, err := t.MarketService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetMarkets, err)
	}
	return &protoapi.MarketsResponse{
		Markets: markets,
	}, nil
}

// OrdersByMarketAndID provides the given order, searching by Market and (Order)Id.
func (t *tradingDataService) OrderByMarketAndID(ctx context.Context,
	request *protoapi.OrderByMarketAndIDRequest) (*protoapi.OrderByMarketAndIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderByMarketAndID")()

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	order, err := t.OrderService.GetByMarketAndID(ctx, request.MarketId, request.OrderId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrOrderServiceGetByMarketAndID, err)
	}

	return &protoapi.OrderByMarketAndIDResponse{
		Order: order,
	}, nil
}

// OrderByReference provides the (possibly not yet accepted/rejected) order.
func (t *tradingDataService) OrderByReference(ctx context.Context, req *protoapi.OrderByReferenceRequest) (*protoapi.OrderByReferenceResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderByReference")()

	err := req.Validate()
	if err != nil {
		return nil, err
	}

	order, err := t.OrderService.GetByReference(ctx, req.Reference)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByReference, err)
	}
	return &protoapi.OrderByReferenceResponse{
		Order: order,
	}, nil
}

// Candles returns trade OHLC/volume data for the given time period and interval.
// It will fill in any intervals without trades with zero based candles.
// SinceTimestamp must be in RFC3339 string format.
func (t *tradingDataService) Candles(ctx context.Context,
	request *protoapi.CandlesRequest) (*protoapi.CandlesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Candles")()

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	if request.Interval == types.Interval_INTERVAL_UNSPECIFIED {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	c, err := t.CandleService.GetCandles(ctx, request.MarketId, vegatime.UnixNano(request.SinceTimestamp), request.Interval)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCandleServiceGetCandles, err)
	}

	return &protoapi.CandlesResponse{
		Candles: c,
	}, nil
}

// MarketDepth provides the order book for a given market, and also returns the most recent trade
// for the given market.
func (t *tradingDataService) MarketDepth(ctx context.Context, req *protoapi.MarketDepthRequest) (*protoapi.MarketDepthResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketDepth")()

	err := req.Validate()
	if err != nil {
		return nil, err
	}

	// Query market depth statistics
	depth, err := t.MarketService.GetDepth(ctx, req.MarketId, req.MaxDepth)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetDepth, err)
	}
	ts, err := t.TradeService.GetByMarket(ctx, req.MarketId, 0, 1, true)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}

	// Build market depth response, including last trade (if available)
	resp := &protoapi.MarketDepthResponse{
		Buy:            depth.Buy,
		MarketId:       depth.MarketId,
		Sell:           depth.Sell,
		SequenceNumber: depth.SequenceNumber,
	}
	if len(ts) > 0 && ts[0] != nil {
		resp.LastTrade = ts[0]
	}
	return resp, nil
}

// TradesByMarket provides a list of trades for a given market.
// Pagination: Optional. If not provided, defaults are used.
func (t *tradingDataService) TradesByMarket(ctx context.Context, request *protoapi.TradesByMarketRequest) (*protoapi.TradesByMarketResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesByMarket")()

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	ts, err := t.TradeService.GetByMarket(ctx, request.MarketId, p.Skip, p.Limit, p.Descending)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}
	return &protoapi.TradesByMarketResponse{
		Trades: ts,
	}, nil
}

// PositionsByParty provides a list of positions for a given party.
func (t *tradingDataService) PositionsByParty(ctx context.Context, request *protoapi.PositionsByPartyRequest) (*protoapi.PositionsByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("PositionsByParty")()

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	// Check here for a valid marketID so we don't fail later
	if request.MarketId != "" {
		_, err := t.MarketService.GetByID(ctx, request.MarketId)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, ErrInvalidMarketID, err)
		}
	}

	positions, err := t.TradeService.GetPositionsByParty(ctx, request.PartyId, request.MarketId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetPositionsByParty, err)
	}
	var response = &protoapi.PositionsByPartyResponse{}
	response.Positions = positions
	return response, nil
}

// MarginLevels returns the current margin levels for a given party and market.
func (t *tradingDataService) MarginLevels(_ context.Context, req *protoapi.MarginLevelsRequest) (*protoapi.MarginLevelsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarginLevels")()

	err := req.Validate()
	if err != nil {
		return nil, err
	}

	mls, err := t.RiskService.GetMarginLevelsByID(req.PartyId, req.MarketId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrRiskServiceGetMarginLevelsByID, err)
	}
	levels := make([]*types.MarginLevels, 0, len(mls))
	for _, v := range mls {
		v := v
		levels = append(levels, &v)
	}
	return &protoapi.MarginLevelsResponse{
		MarginLevels: levels,
	}, nil
}

// MarketDataByID provides market data for the given ID.
func (t *tradingDataService) MarketDataByID(_ context.Context, req *protoapi.MarketDataByIDRequest) (*protoapi.MarketDataByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketDataByID")()

	err := req.Validate()
	if err != nil {
		return nil, err
	}

	md, err := t.MarketService.GetMarketDataByID(req.MarketId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetMarketData, err)
	}
	return &protoapi.MarketDataByIDResponse{
		MarketData: &md,
	}, nil
}

// MarketsData provides all market data for all markets on this network.
func (t *tradingDataService) MarketsData(_ context.Context, _ *protoapi.MarketsDataRequest) (*protoapi.MarketsDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketsData")()
	mds := t.MarketService.GetMarketsData()
	mdptrs := make([]*types.MarketData, 0, len(mds))
	for _, v := range mds {
		v := v
		mdptrs = append(mdptrs, &v)
	}
	return &protoapi.MarketsDataResponse{
		MarketsData: mdptrs,
	}, nil
}

// Statistics provides various blockchain and Vega statistics, including:
// Blockchain height, backlog length, current time, orders and trades per block, tendermint version
// Vega counts for parties, markets, order actions (amend, cancel, submit), Vega version
func (t *tradingDataService) Statistics(ctx context.Context, _ *protoapi.StatisticsRequest) (*protoapi.StatisticsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Statistics")()
	// Call tendermint and related services to get information for statistics
	// We load read-only internal statistics through each package level statistics structs
	epochTime, err := t.TimeService.GetTimeNow()
	if err != nil {
		return nil, apiError(codes.Internal, ErrTimeServiceGetTimeNow, err)
	}

	// Call tendermint via rpc client
	var (
		backlogLength, numPeers int
		gt                      *time.Time
		chainID                 string
	)

	backlogLength, numPeers, gt, chainID, err = t.getTendermintStats(ctx)
	if err != nil {
		// do not return an error, let just eventually log it
		t.log.Debug("could not load tendermint stats", logging.Error(err))
	}

	// If the chain is replaying then genesis time can be nil
	genesisTime := ""
	if gt != nil {
		genesisTime = vegatime.Format(*gt)
	}

	// Load current markets details
	m, err := t.MarketService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Unavailable, ErrMarketServiceGetMarkets, err)
	}

	stats := &types.Statistics{
		BlockHeight:              t.Stats.Blockchain.Height(),
		BacklogLength:            uint64(backlogLength),
		TotalPeers:               uint64(numPeers),
		GenesisTime:              genesisTime,
		CurrentTime:              vegatime.Format(vegatime.Now()),
		VegaTime:                 vegatime.Format(epochTime),
		Uptime:                   vegatime.Format(t.Stats.GetUptime()),
		TxPerBlock:               uint64(t.Stats.Blockchain.TotalTxLastBatch()),
		AverageTxBytes:           uint64(t.Stats.Blockchain.AverageTxSizeBytes()),
		AverageOrdersPerBlock:    uint64(t.Stats.Blockchain.AverageOrdersPerBatch()),
		TradesPerSecond:          t.Stats.Blockchain.TradesPerSecond(),
		OrdersPerSecond:          t.Stats.Blockchain.OrdersPerSecond(),
		Status:                   t.statusChecker.ChainStatus(),
		TotalMarkets:             uint64(len(m)),
		AppVersionHash:           t.Stats.GetVersionHash(),
		AppVersion:               t.Stats.GetVersion(),
		ChainVersion:             t.Stats.GetChainVersion(),
		TotalAmendOrder:          t.Stats.Blockchain.TotalAmendOrder(),
		TotalCancelOrder:         t.Stats.Blockchain.TotalCancelOrder(),
		TotalCreateOrder:         t.Stats.Blockchain.TotalCreateOrder(),
		TotalOrders:              t.Stats.Blockchain.TotalOrders(),
		TotalTrades:              t.Stats.Blockchain.TotalTrades(),
		BlockDuration:            t.Stats.Blockchain.BlockDuration(),
		OrderSubscriptions:       uint32(t.OrderService.GetOrderSubscribersCount()),
		TradeSubscriptions:       uint32(t.TradeService.GetTradeSubscribersCount()),
		PositionsSubscriptions:   uint32(t.TradeService.GetPositionsSubscribersCount()),
		MarketDepthSubscriptions: uint32(t.MarketService.GetMarketDepthSubscribersCount()),
		CandleSubscriptions:      uint32(t.CandleService.GetCandleSubscribersCount()),
		AccountSubscriptions:     uint32(t.AccountsService.GetAccountSubscribersCount()),
		MarketDataSubscriptions:  uint32(t.MarketService.GetMarketDataSubscribersCount()),
		ChainId:                  chainID,
	}
	return &protoapi.StatisticsResponse{
		Statistics: stats,
	}, nil
}

// GetVegaTime returns the latest blockchain header timestamp, in UnixNano format.
// Example: "1568025900111222333" corresponds to 2019-09-09T10:45:00.111222333Z.
func (t *tradingDataService) GetVegaTime(ctx context.Context, _ *protoapi.GetVegaTimeRequest) (*protoapi.GetVegaTimeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVegaTime")()
	ts, err := t.TimeService.GetTimeNow()
	if err != nil {
		return nil, apiError(codes.Internal, ErrTimeServiceGetTimeNow, err)
	}

	return &protoapi.GetVegaTimeResponse{
		Timestamp: ts.UnixNano(),
	}, nil

}

// TransferResponsesSubscribe opens a subscription to transfer response data provided by the transfer response service.
func (t *tradingDataService) TransferResponsesSubscribe(
	_ *protoapi.TransferResponsesSubscribeRequest, srv protoapi.TradingDataService_TransferResponsesSubscribeServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("TransferResponseSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan in error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	transferResponsesChan, ref := t.TransferResponseService.ObserveTransferResponses(ctx, t.Config.StreamRetries)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("TransferResponses subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	var err error
	for {
		select {
		case transferResponses := <-transferResponsesChan:
			if transferResponses == nil {
				err = ErrChannelClosed
				t.log.Error("TransferResponses subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			for _, tr := range transferResponses {
				tr := tr
				if err := srv.Send(&protoapi.TransferResponsesSubscribeResponse{
					Response: tr,
				}); err != nil {
					t.log.Error("TransferResponses subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("TransferResponses subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if transferResponsesChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("TransferResponses subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// MarketsDataSubscribe opens a subscription to market data provided by the markets service.
func (t *tradingDataService) MarketsDataSubscribe(req *protoapi.MarketsDataSubscribeRequest,
	srv protoapi.TradingDataService_MarketsDataSubscribeServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketsDataSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	marketsDataChan, ref := t.MarketService.ObserveMarketsData(ctx, t.Config.StreamRetries, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Markets data subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	var err error
	for {
		select {
		case mds := <-marketsDataChan:
			if mds == nil {
				err = ErrChannelClosed
				t.log.Error("Markets data subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			for _, md := range mds {
				resp := &protoapi.MarketsDataSubscribeResponse{
					MarketData: &md,
				}
				if err := srv.Send(resp); err != nil {
					t.log.Error("Markets data subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Markets data subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if marketsDataChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Markets data subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// AccountsSubscribe opens a subscription to the Margin Levels provided by the risk service.
func (t *tradingDataService) MarginLevelsSubscribe(req *protoapi.MarginLevelsSubscribeRequest, srv protoapi.TradingDataService_MarginLevelsSubscribeServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("MarginLevelsSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	err := req.Validate()
	if err != nil {
		return err
	}

	marginLevelsChan, ref := t.RiskService.ObserveMarginLevels(ctx, t.Config.StreamRetries, req.PartyId, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Margin levels subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case mls := <-marginLevelsChan:
			if mls == nil {
				err = ErrChannelClosed
				t.log.Error("Margin levels subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			for _, ml := range mls {
				ml := ml
				resp := &protoapi.MarginLevelsSubscribeResponse{
					MarginLevels: &ml,
				}
				if err := srv.Send(resp); err != nil {
					t.log.Error("Margin levels data subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Margin levels data subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if marginLevelsChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Margin levels data subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// AccountsSubscribe opens a subscription to the Accounts service.
func (t *tradingDataService) AccountsSubscribe(req *protoapi.AccountsSubscribeRequest,
	srv protoapi.TradingDataService_AccountsSubscribeServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("AccountsSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	accountsChan, ref := t.AccountsService.ObserveAccounts(
		ctx, t.Config.StreamRetries, req.MarketId, req.PartyId, req.Asset, req.Type)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Accounts subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	var err error
	for {
		select {
		case accounts := <-accountsChan:
			if accounts == nil {
				err = ErrChannelClosed
				t.log.Error("Accounts subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			for _, account := range accounts {
				account := account
				resp := &protoapi.AccountsSubscribeResponse{
					Account: account,
				}
				err = srv.Send(resp)
				if err != nil {
					t.log.Error("Accounts subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Accounts subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if accountsChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Accounts subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// OrdersSubscribe opens a subscription to the Orders service.
// MarketID: Optional.
// PartyID: Optional.
func (t *tradingDataService) OrdersSubscribe(
	req *protoapi.OrdersSubscribeRequest, srv protoapi.TradingDataService_OrdersSubscribeServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("OrdersSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var (
		err               error
		marketID, partyID *string
	)

	if len(req.MarketId) > 0 {
		marketID = &req.MarketId
	}
	if len(req.PartyId) > 0 {
		partyID = &req.PartyId
	}

	ordersChan, ref := t.OrderService.ObserveOrders(ctx, t.Config.StreamRetries, marketID, partyID)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case orders := <-ordersChan:
			if orders == nil {
				err = ErrChannelClosed
				t.log.Error("Orders subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			out := make([]*types.Order, 0, len(orders))
			for _, v := range orders {
				v := v
				out = append(out, &v)
			}
			err = srv.Send(&protoapi.OrdersSubscribeResponse{Orders: out})
			if err != nil {
				t.log.Error("Orders subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err = ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Orders subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if ordersChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Orders subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// TradesSubscribe opens a subscription to the Trades service.
func (t *tradingDataService) TradesSubscribe(req *protoapi.TradesSubscribeRequest,
	srv protoapi.TradingDataService_TradesSubscribeServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var (
		err               error
		marketID, partyID *string
	)
	if len(req.MarketId) > 0 {
		marketID = &req.MarketId
	}
	if len(req.PartyId) > 0 {
		partyID = &req.PartyId
	}

	tradesChan, ref := t.TradeService.ObserveTrades(ctx, t.Config.StreamRetries, marketID, partyID)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Trades subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case trades := <-tradesChan:
			if len(trades) <= 0 {
				err = ErrChannelClosed
				t.log.Error("Trades subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}

			out := make([]*types.Trade, 0, len(trades))
			for _, v := range trades {
				v := v
				out = append(out, &v)
			}
			if err := srv.Send(&protoapi.TradesSubscribeResponse{Trades: out}); err != nil {
				t.log.Error("Trades subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err = ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Trades subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}
		if tradesChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Trades subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// CandlesSubscribe opens a subscription to the Candles service.
func (t *tradingDataService) CandlesSubscribe(req *protoapi.CandlesSubscribeRequest,
	srv protoapi.TradingDataService_CandlesSubscribeServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("CandlesSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var (
		err      error
		marketID *string
	)
	if len(req.MarketId) > 0 {
		marketID = &req.MarketId
	} else {
		return apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}

	candlesChan, ref := t.CandleService.ObserveCandles(ctx, t.Config.StreamRetries, marketID, &req.Interval)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Candles subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case candle := <-candlesChan:
			if candle == nil {
				err = ErrChannelClosed
				t.log.Error("Candles subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			resp := &protoapi.CandlesSubscribeResponse{
				Candle: candle,
			}
			if err := srv.Send(resp); err != nil {
				t.log.Error("Candles subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err = ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Candles subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if candlesChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Candles subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// MarketDepthSubscribe opens a subscription to the MarketDepth service.
func (t *tradingDataService) MarketDepthSubscribe(
	req *protoapi.MarketDepthSubscribeRequest,
	srv protoapi.TradingDataService_MarketDepthSubscribeServer,
) error {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketDepthSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	_, err := validateMarket(ctx, req.MarketId, t.MarketService)
	if err != nil {
		return err // validateMarket already returns an API error, no additional wrapping needed
	}

	depthChan, ref := t.MarketService.ObserveDepth(
		ctx, t.Config.StreamRetries, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Depth subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case depth := <-depthChan:
			if depth == nil {
				err = ErrChannelClosed
				t.log.Error("Depth subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			resp := &protoapi.MarketDepthSubscribeResponse{
				MarketDepth: depth,
			}
			if err := srv.Send(resp); err != nil {
				if t.log.GetLevel() == logging.DebugLevel {
					t.log.Debug("Depth subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
				}
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err = ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Depth subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if depthChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Depth subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// MarketDepthUpdatesSubscribe opens a subscription to the MarketDepth Updates service.
func (t *tradingDataService) MarketDepthUpdatesSubscribe(
	req *protoapi.MarketDepthUpdatesSubscribeRequest,
	srv protoapi.TradingDataService_MarketDepthUpdatesSubscribeServer,
) error {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketDepthUpdatesSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	_, err := validateMarket(ctx, req.MarketId, t.MarketService)
	if err != nil {
		return err // validateMarket already returns an API error, no additional wrapping needed
	}

	depthChan, ref := t.MarketService.ObserveDepthUpdates(
		ctx, t.Config.StreamRetries, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Depth updates subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case depth := <-depthChan:
			if depth == nil {
				err = ErrChannelClosed
				if t.log.GetLevel() == logging.DebugLevel {
					t.log.Debug("Depth updates subscriber closed",
						logging.Error(err),
						logging.Uint64("ref", ref))
				}
				return apiError(codes.Internal, err)
			}
			resp := &protoapi.MarketDepthUpdatesSubscribeResponse{
				Update: depth,
			}

			if err := srv.Send(resp); err != nil {
				if t.log.GetLevel() == logging.DebugLevel {
					t.log.Debug("Depth updates subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
				}
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err = ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Depth updates subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if depthChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Depth updates subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// PositionsSubscribe opens a subscription to the Positions service.
func (t *tradingDataService) PositionsSubscribe(
	req *protoapi.PositionsSubscribeRequest,
	srv protoapi.TradingDataService_PositionsSubscribeServer,
) error {
	defer metrics.StartAPIRequestAndTimeGRPC("PositionsSubscribe")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	positionsChan, ref := t.TradeService.ObservePositions(ctx, t.Config.StreamRetries, req.PartyId, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Positions subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case position := <-positionsChan:
			if position == nil {
				err := ErrChannelClosed
				t.log.Error("Positions subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			resp := &protoapi.PositionsSubscribeResponse{
				Position: position,
			}
			if err := srv.Send(resp); err != nil {
				t.log.Error("Positions subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err := ctx.Err()
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Positions subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-t.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if positionsChan == nil {
			if t.log.GetLevel() == logging.DebugLevel {
				t.log.Debug("Positions subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// MarketByID provides the given market.
func (t *tradingDataService) MarketByID(ctx context.Context, req *protoapi.MarketByIDRequest) (*protoapi.MarketByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketByID")()
	mkt, err := validateMarket(ctx, req.MarketId, t.MarketService)
	if err != nil {
		return nil, err // validateMarket already returns an API error, no need to additionally wrap
	}

	return &protoapi.MarketByIDResponse{
		Market: mkt,
	}, nil
}

// Parties provides a list of all parties.
func (t *tradingDataService) Parties(ctx context.Context, _ *protoapi.PartiesRequest) (*protoapi.PartiesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Parties")()
	parties, err := t.PartyService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPartyServiceGetAll, err)
	}
	return &protoapi.PartiesResponse{
		Parties: parties,
	}, nil
}

// PartyByID provides the given party.
func (t *tradingDataService) PartyByID(ctx context.Context, req *protoapi.PartyByIDRequest) (*protoapi.PartyByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("PartyByID")()
	pty, err := validateParty(ctx, t.log, req.PartyId, t.PartyService)
	if err != nil {
		return nil, err // validateParty already returns an API error, no need to additionally wrap
	}
	return &protoapi.PartyByIDResponse{
		Party: pty,
	}, nil
}

// TradesByParty provides a list of trades for the given party.
// Pagination: Optional. If not provided, defaults are used.
func (t *tradingDataService) TradesByParty(ctx context.Context,
	req *protoapi.TradesByPartyRequest) (*protoapi.TradesByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesByParty")()

	p := defaultPagination
	if req.Pagination != nil {
		p = *req.Pagination
	}
	trades, err := t.TradeService.GetByParty(ctx, req.PartyId, p.Skip, p.Limit, p.Descending, &req.MarketId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByParty, err)
	}

	return &protoapi.TradesByPartyResponse{Trades: trades}, nil
}

// TradesByOrder provides a list of the trades that correspond to a given order.
func (t *tradingDataService) TradesByOrder(ctx context.Context,
	req *protoapi.TradesByOrderRequest) (*protoapi.TradesByOrderResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesByOrder")()
	trades, err := t.TradeService.GetByOrderID(ctx, req.OrderId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByOrderID, err)
	}
	return &protoapi.TradesByOrderResponse{
		Trades: trades,
	}, nil
}

// LastTrade provides the last trade for the given market.
func (t *tradingDataService) LastTrade(ctx context.Context,
	req *protoapi.LastTradeRequest) (*protoapi.LastTradeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("LastTrade")()
	if len(req.MarketId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}
	ts, err := t.TradeService.GetByMarket(ctx, req.MarketId, 0, 1, true)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}
	if len(ts) > 0 && ts[0] != nil {
		return &protoapi.LastTradeResponse{Trade: ts[0]}, nil
	}
	// No trades found on the market yet (and no errors)
	// this can happen at the beginning of a new market
	return &protoapi.LastTradeResponse{}, nil
}

func (t *tradingDataService) MarketAccounts(_ context.Context,
	req *protoapi.MarketAccountsRequest) (*protoapi.MarketAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketAccounts")()
	accs, err := t.AccountsService.GetMarketAccounts(req.MarketId, req.Asset)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetMarketAccounts, err)
	}
	return &protoapi.MarketAccountsResponse{
		Accounts: accs,
	}, nil
}

func (t *tradingDataService) FeeInfrastructureAccounts(_ context.Context,
	req *protoapi.FeeInfrastructureAccountsRequest) (*protoapi.FeeInfrastructureAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("FeeInfrastructureAccounts")()
	accs, err := t.AccountsService.GetFeeInfrastructureAccounts(req.Asset)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetFeeInfrastructureAccounts, err)
	}
	return &protoapi.FeeInfrastructureAccountsResponse{
		Accounts: accs,
	}, nil
}

func (t *tradingDataService) PartyAccounts(_ context.Context,
	req *protoapi.PartyAccountsRequest) (*protoapi.PartyAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("PartyAccounts")()
	accs, err := t.AccountsService.GetPartyAccounts(req.PartyId, req.MarketId, req.Asset, req.Type)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetPartyAccounts, err)
	}
	return &protoapi.PartyAccountsResponse{
		Accounts: accs,
	}, nil
}

func validateMarket(ctx context.Context, marketID string, marketService MarketService) (*types.Market, error) {
	var mkt *types.Market
	var err error
	if len(marketID) == 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}
	mkt, err = marketService.GetByID(ctx, marketID)
	if err != nil {
		// We return nil for error as we do not want
		// to return an error when a market is not found
		// but just a nil value.
		return nil, nil
	}
	return mkt, nil
}

func validateParty(ctx context.Context, log *logging.Logger, partyID string, partyService PartyService) (*types.Party, error) {
	pty, err := partyService.GetByID(ctx, partyID)
	if err != nil {
		// we just log the error here, then return an nil error.
		// right now the only error possible is about not finding a party
		// we just not an actual error
		log.Debug("error getting party by ID",
			logging.Error(err),
			logging.String("party-id", partyID))
		err = nil
	}
	return pty, err
}

func (t *tradingDataService) getTendermintStats(
	ctx context.Context,
) (
	backlogLength, numPeers int,
	genesis *time.Time,
	chainID string,
	err error,
) {

	if t.Stats == nil || t.Stats.Blockchain == nil {
		return 0, 0, nil, "", apiError(codes.Internal, ErrChainNotConnected)
	}

	const refused = "connection refused"

	// Unconfirmed TX count == current transaction backlog length
	backlogLength, err = t.Client.GetUnconfirmedTxCount(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return 0, 0, nil, "", nil
		}
		return 0, 0, nil, "", apiError(codes.Internal, ErrBlockchainBacklogLength, err)
	}

	if atomic.LoadUint32(&t.hasGenesisTimeAndChainID) == 0 {
		if err = t.getGenesisTimeAndChainID(ctx); err != nil {
			return 0, 0, nil, "", err
		}
	}

	// Net info provides peer stats etc (block chain network info) == number of peers
	netInfo, err := t.getTMNetInfo(ctx)
	if err != nil {
		return backlogLength, 0, &t.genesisTime, t.chainID, nil
	}

	return backlogLength, netInfo.NPeers, &t.genesisTime, t.chainID, nil
}

func (t *tradingDataService) getTMNetInfo(ctx context.Context) (tmctypes.ResultNetInfo, error) {
	t.netInfoMu.RLock()
	defer t.netInfoMu.RUnlock()

	if t.netInfo == nil {
		return tmctypes.ResultNetInfo{}, apiError(codes.Internal, ErrBlockchainNetworkInfo)
	}

	return *t.netInfo, nil
}

func (t *tradingDataService) updateNetInfo(ctx context.Context) {
	// update the net info every 1 minutes
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			netInfo, err := t.Client.GetNetworkInfo(ctx)
			if err != nil {
				continue
			}
			t.netInfoMu.Lock()
			t.netInfo = netInfo
			t.netInfoMu.Unlock()
		}
	}
}

func (t *tradingDataService) getGenesisTimeAndChainID(ctx context.Context) error {
	const refused = "connection refused"
	// just lock in here, ideally we'ill come here only once, so not a big issue to lock
	t.mu.Lock()
	defer t.mu.Unlock()

	var err error
	// Genesis retrieves the current genesis date/time for the blockchain
	t.genesisTime, err = t.Client.GetGenesisTime(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return nil
		}
		return apiError(codes.Internal, ErrBlockchainGenesisTime, err)
	}

	t.chainID, err = t.Client.GetChainID(ctx)
	if err != nil {
		return apiError(codes.Internal, ErrBlockchainChainID, err)
	}

	atomic.StoreUint32(&t.hasGenesisTimeAndChainID, 1)
	return nil
}

func (t *tradingDataService) OrderByID(ctx context.Context, in *protoapi.OrderByIDRequest) (*protoapi.OrderByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderByID")()
	if len(in.OrderId) == 0 {
		// Invalid parameter
		return nil, ErrMissingOrderIDParameter
	}

	order, err := t.OrderService.GetByOrderID(ctx, in.OrderId, in.Version)
	if err != nil {
		// If we get here then no match was found
		return nil, ErrOrderNotFound
	}

	resp := &protoapi.OrderByIDResponse{
		Order: order,
	}
	return resp, nil

}

// OrderVersionsByID returns all versions of the order by its orderID
func (t *tradingDataService) OrderVersionsByID(
	ctx context.Context,
	in *protoapi.OrderVersionsByIDRequest,
) (*protoapi.OrderVersionsByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderVersionsByID")()

	err := in.Validate()
	if err != nil {
		return nil, err
	}
	p := defaultPagination
	if in.Pagination != nil {
		p = *in.Pagination
	}
	orders, err := t.OrderService.GetAllVersionsByOrderID(ctx,
		in.OrderId,
		p.Skip,
		p.Limit,
		p.Descending)
	if err == nil {
		return &protoapi.OrderVersionsByIDResponse{
			Orders: orders,
		}, nil
	}
	return nil, err
}

func (t *tradingDataService) GetProposals(_ context.Context,
	in *protoapi.GetProposalsRequest,
) (*protoapi.GetProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetProposals")()

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetProposalsResponse{
		Data: t.governanceService.GetProposals(inState),
	}, nil
}

func (t *tradingDataService) GetProposalsByParty(_ context.Context,
	in *protoapi.GetProposalsByPartyRequest,
) (*protoapi.GetProposalsByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetProposalsByParty")()

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetProposalsByPartyResponse{
		Data: t.governanceService.GetProposalsByParty(in.PartyId, inState),
	}, nil
}

func (t *tradingDataService) GetVotesByParty(_ context.Context,
	in *protoapi.GetVotesByPartyRequest,
) (*protoapi.GetVotesByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVotesByParty")()

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	return &protoapi.GetVotesByPartyResponse{
		Votes: t.governanceService.GetVotesByParty(in.PartyId),
	}, nil
}

func (t *tradingDataService) GetNewMarketProposals(_ context.Context,
	in *protoapi.GetNewMarketProposalsRequest,
) (*protoapi.GetNewMarketProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNewMarketProposals")()

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetNewMarketProposalsResponse{
		Data: t.governanceService.GetNewMarketProposals(inState),
	}, nil
}

func (t *tradingDataService) GetUpdateMarketProposals(_ context.Context,
	in *protoapi.GetUpdateMarketProposalsRequest,
) (*protoapi.GetUpdateMarketProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetUpdateMarketProposals")()

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetUpdateMarketProposalsResponse{
		Data: t.governanceService.GetUpdateMarketProposals(in.MarketId, inState),
	}, nil
}

func (t *tradingDataService) GetNetworkParametersProposals(_ context.Context,
	in *protoapi.GetNetworkParametersProposalsRequest,
) (*protoapi.GetNetworkParametersProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkParametersProposals")()

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetNetworkParametersProposalsResponse{
		Data: t.governanceService.GetNetworkParametersProposals(inState),
	}, nil
}

func (t *tradingDataService) GetNewAssetProposals(_ context.Context,
	in *protoapi.GetNewAssetProposalsRequest,
) (*protoapi.GetNewAssetProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNewAssetProposals")()

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetNewAssetProposalsResponse{
		Data: t.governanceService.GetNewAssetProposals(inState),
	}, nil
}

func (t *tradingDataService) GetProposalByID(_ context.Context,
	in *protoapi.GetProposalByIDRequest,
) (*protoapi.GetProposalByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetProposalByID")()

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	proposal, err := t.governanceService.GetProposalByID(in.ProposalId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMissingProposalID, err)
	}
	return &protoapi.GetProposalByIDResponse{Data: proposal}, nil
}

func (t *tradingDataService) GetProposalByReference(_ context.Context,
	in *protoapi.GetProposalByReferenceRequest,
) (*protoapi.GetProposalByReferenceResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetProposalByReference")()

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	proposal, err := t.governanceService.GetProposalByReference(in.Reference)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMissingProposalReference, err)
	}
	return &protoapi.GetProposalByReferenceResponse{Data: proposal}, nil
}

func (t *tradingDataService) ObserveGovernance(
	_ *protoapi.ObserveGovernanceRequest,
	stream protoapi.TradingDataService_ObserveGovernanceServer,
) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ObserveGovernance")()
	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming governance updates")
	}
	ch := t.governanceService.ObserveGovernance(ctx, t.Config.StreamRetries)
	for {
		select {
		case props, ok := <-ch:
			if !ok {
				cfunc()
				return nil
			}
			for _, p := range props {
				resp := &protoapi.ObserveGovernanceResponse{
					Data: &p,
				}
				if err := stream.Send(resp); err != nil {
					t.log.Error("failed to send governance data into stream",
						logging.Error(err))
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-t.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}
	}
}
func (t *tradingDataService) ObservePartyProposals(
	in *protoapi.ObservePartyProposalsRequest,
	stream protoapi.TradingDataService_ObservePartyProposalsServer,
) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ObservePartyProposals")()

	if err := in.Validate(); err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming party proposals")
	}
	ch := t.governanceService.ObservePartyProposals(ctx, t.Config.StreamRetries, in.PartyId)
	for {
		select {
		case props, ok := <-ch:
			if !ok {
				cfunc()
				return nil
			}
			for _, p := range props {
				resp := &protoapi.ObservePartyProposalsResponse{
					Data: &p,
				}
				if err := stream.Send(resp); err != nil {
					t.log.Error("failed to send party proposal into stream",
						logging.Error(err))
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-t.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}
	}
}

func (t *tradingDataService) ObservePartyVotes(
	in *protoapi.ObservePartyVotesRequest,
	stream protoapi.TradingDataService_ObservePartyVotesServer,
) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ObservePartyVotes")()

	if err := in.Validate(); err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming party votes")
	}
	ch := t.governanceService.ObservePartyVotes(ctx, t.Config.StreamRetries, in.PartyId)
	for {
		select {
		case votes, ok := <-ch:
			if !ok {
				cfunc()
				return nil
			}
			for _, p := range votes {
				resp := &protoapi.ObservePartyVotesResponse{
					Vote: &p,
				}
				if err := stream.Send(resp); err != nil {
					t.log.Error("failed to send party vote into stream",
						logging.Error(err))
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-t.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}
	}
}

func (t *tradingDataService) ObserveProposalVotes(
	in *protoapi.ObserveProposalVotesRequest,
	stream protoapi.TradingDataService_ObserveProposalVotesServer,
) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ObserveProposalVotes")()

	if err := in.Validate(); err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming proposal votes")
	}
	ch := t.governanceService.ObserveProposalVotes(ctx, t.Config.StreamRetries, in.ProposalId)
	for {
		select {
		case votes, ok := <-ch:
			if !ok {
				cfunc()
				return nil
			}
			for _, p := range votes {
				resp := &protoapi.ObserveProposalVotesResponse{
					Vote: &p,
				}
				if err := stream.Send(resp); err != nil {
					t.log.Error("failed to send proposal vote into stream",
						logging.Error(err))
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-t.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}
	}
}

func (t *tradingDataService) ObserveEventBus(
	stream protoapi.TradingDataService_ObserveEventBusServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ObserveEventBus")()

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()

	// now we start listening for a few seconds in order to get at least the very first message
	// this will be blocking until the connection by the client is closed
	// and we will not start processing any events until we receive the original request
	// indicating filters and batch size.
	req, err := t.recvEventRequest(stream)
	if err != nil {
		// client exited, nothing to do
		return nil
	}

	if err := req.Validate(); err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	// now we will aggregaate filter out of the initial reuqest
	types, err := events.ProtoToInternal(req.Type...)
	if err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	if len(req.PartyId) == 0 {
		// no PartyID filter
		for _, t := range types {
			// subscription to TxErr events
			if t == events.TxErrEvent {
				return apiError(codes.InvalidArgument, ErrMalformedRequest, errors.New("missing party filter for TxError stream"))
			}
		}
	}
	filters := []subscribers.EventFilter{}
	if len(req.MarketId) > 0 && len(req.PartyId) > 0 {
		filters = append(filters, events.GetPartyAndMarketFilter(req.MarketId, req.PartyId))
	} else {
		if len(req.MarketId) > 0 {
			filters = append(filters, events.GetMarketIDFilter(req.MarketId))
		}
		if len(req.PartyId) > 0 {
			filters = append(filters, events.GetPartyIDFilter(req.PartyId))
		}
	}

	// number of retries to -1 to have pretty much unlimited retries
	ch, bCh := t.eventService.ObserveEvents(ctx, t.Config.StreamRetries, types, int(req.BatchSize), filters...)
	defer close(bCh)

	if req.BatchSize > 0 {
		err := t.observeEventsWithAck(ctx, stream, req.BatchSize, ch, bCh)
		return err

	}
	err = t.observeEvents(ctx, stream, ch)
	return err
}

func (t *tradingDataService) observeEvents(
	ctx context.Context,
	stream protoapi.TradingDataService_ObserveEventBusServer,
	ch <-chan []*types.BusEvent,
) error {
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return nil
			}
			resp := &protoapi.ObserveEventBusResponse{
				Events: data,
			}
			if err := stream.Send(resp); err != nil {
				t.log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-t.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}
	}
}

func (t *tradingDataService) recvEventRequest(
	stream protoapi.TradingDataService_ObserveEventBusServer,
) (*protoapi.ObserveEventBusRequest, error) {
	readCtx, cfunc := context.WithTimeout(stream.Context(), 5*time.Second)
	oebCh := make(chan protoapi.ObserveEventBusRequest)
	var err error
	go func() {
		defer close(oebCh)
		nb := protoapi.ObserveEventBusRequest{}
		if err = stream.RecvMsg(&nb); err != nil {
			cfunc()
			return
		}
		oebCh <- nb
	}()
	select {
	case <-readCtx.Done():
		if err != nil {
			// this means the client disconnectd
			return nil, err
		}
		// this mean we timedout
		return nil, readCtx.Err()
	case nb := <-oebCh:
		return &nb, nil
	}
}

func (t *tradingDataService) observeEventsWithAck(
	ctx context.Context,
	stream protoapi.TradingDataService_ObserveEventBusServer,
	batchSize int64,
	ch <-chan []*types.BusEvent,
	bCh chan<- int,
) error {
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return nil
			}
			resp := &protoapi.ObserveEventBusResponse{
				Events: data,
			}
			if err := stream.Send(resp); err != nil {
				t.log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-t.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}

		// now we try to read again the new size / ack
		req, err := t.recvEventRequest(stream)
		if err != nil {
			return err
		}

		if req.BatchSize != batchSize {
			batchSize = req.BatchSize
			bCh <- int(batchSize)
		}
	}
}
