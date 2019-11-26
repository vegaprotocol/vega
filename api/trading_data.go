package api

import (
	"context"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/ptypes/empty"
	google_proto "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

// Errors
var (
	// ErrChainNotConnected signals to the user that he cannot access a given endpoint
	// which require the chain, but the chain is actually offline
	ErrChainNotConnected = errors.New("chain not connected")
	// ErrChannelClosed signals that the channel streaming data is closed
	ErrChannelClosed = errors.New("channel closed")
	// ErrEmptyMissingMarketID signals to the caller that the request expected a
	// market id but the field is missing or empty
	ErrEmptyMissingMarketID = errors.New("empty or missing market ID")
	// ErrEmptyMissingOrderID signals to the caller that the request expected an
	// order id but the field is missing or empty
	ErrEmptyMissingOrderID = errors.New("empty or missing order ID")
	// ErrEmptyMissingOrderReference signals to the caller that the request expected an
	// order reference but the field is missing or empty
	ErrEmptyMissingOrderReference = errors.New("empty or missing order reference")
	// ErrEmptyMissingPartyID signals to the caller that the request expected a
	// party id but the field is missing or empty
	ErrEmptyMissingPartyID = errors.New("empty or missing party ID")
	// ErrEmptyMissingSinceTimestamp signals to the caller that the request expected a
	// timestamp but the field is missing or empty
	ErrEmptyMissingSinceTimestamp = errors.New("empty or missing since-timestamp")
	// ErrServerShutdown signals to the client that the server  is shutting down
	ErrServerShutdown = errors.New("server shutdown")
	// ErrStatisticsNotAvailable signals to the users that the stats endpoint is not available
	ErrStatisticsNotAvailable = errors.New("statistics not available")
	// ErrStreamClosed signals to the users that the grpc stream is closing
	ErrStreamClosed = errors.New("stream closed")
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
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	GetByMarketAndID(ctx context.Context, market string, id string) (order *types.Order, err error)
	GetByReference(ctx context.Context, ref string) (order *types.Order, err error)
	ObserveOrders(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Order, ref uint64)
	GetOrderSubscribersCount() int32
}

// TradeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_service_mock.go -package mocks code.vegaprotocol.io/vega/api TradeService
type TradeService interface {
	GetByOrderID(ctx context.Context, orderID string) ([]*types.Trade, error)
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) (trades []*types.Trade, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, marketID *string) (trades []*types.Trade, err error)
	GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error)
	ObserveTrades(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Trade, ref uint64)
	ObservePositions(ctx context.Context, retries int, party string) (positions <-chan *types.MarketPosition, ref uint64)
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
	GetDepth(ctx context.Context, market string) (marketDepth *types.MarketDepth, err error)
	ObserveDepth(ctx context.Context, retries int, market string) (depth <-chan *types.MarketDepth, ref uint64)
	GetMarketDepthSubscribersCount() int32
	ObserveMarketsData(
		ctx context.Context, retries int, marketID string,
	) (<-chan []types.MarketData, uint64)
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
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error)
	CancelOrder(ctx context.Context, order *types.Order) (success bool, err error)
	CreateOrder(ctx context.Context, order *types.Order) (*types.PendingOrder, error)
	GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error)
	GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error)
	GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error)
	GetUnconfirmedTxCount(ctx context.Context) (count int, err error)
	Health() (*tmctypes.ResultHealth, error)
	NotifyTraderAccount(ctx context.Context, notif *types.NotifyTraderAccount) (success bool, err error)
	Withdraw(context.Context, *types.Withdraw) (success bool, err error)
}

// AccountsService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/accounts_service_mock.go -package mocks code.vegaprotocol.io/vega/api AccountsService
type AccountsService interface {
	GetByParty(partyID string) ([]*types.Account, error)
	GetByPartyAndMarket(partyID string, marketID string) ([]*types.Account, error)
	GetByPartyAndType(partyID string, accType types.AccountType) ([]*types.Account, error)
	GetByPartyAndAsset(partyID string, asset string) ([]*types.Account, error)
	ObserveAccounts(ctx context.Context, retries int, marketID, partyID, asset string, ty types.AccountType) (candleCh <-chan []*types.Account, ref uint64)
	GetAccountSubscribersCount() int32
}

// TransferResponseService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_response_service_mock.go -package mocks code.vegaprotocol.io/vega/api TransferResponseService
type TransferResponseService interface {
	ObserveTransferResponses(ctx context.Context, retries int) (<-chan []*types.TransferResponse, uint64)
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
	TransferResponseService TransferResponseService
	statusChecker           *monitoring.Status
	ctx                     context.Context
}

// defaultLimit specifies the result size limit to use if none is provided by the incoming API
// call. This prevents returning all results every time a careless query is made.
const defaultLimit = uint64(1000)

// OrdersByMarket provides a list of orders for a given market.
// Pagination: Optional. If not provided, defaults are used.
// Returns a list of orders sorted by timestamp descending (most recent first).
func (h *tradingDataService) OrdersByMarket(ctx context.Context,
	request *protoapi.OrdersByMarketRequest) (*protoapi.OrdersByMarketResponse, error) {

	if request.MarketID == "" {
		return nil, ErrEmptyMissingMarketID
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	o, err := h.OrderService.GetByMarket(ctx, request.MarketID, p.Skip, p.Limit, p.Descending, &request.Open)
	if err != nil {
		return nil, err
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
func (h *tradingDataService) OrdersByParty(ctx context.Context,
	request *protoapi.OrdersByPartyRequest) (*protoapi.OrdersByPartyResponse, error) {

	if request.PartyID == "" {
		return nil, ErrEmptyMissingPartyID
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	o, err := h.OrderService.GetByParty(ctx, request.PartyID, p.Skip, p.Limit, p.Descending, &request.Open)
	if err != nil {
		return nil, err
	}

	var response = &protoapi.OrdersByPartyResponse{}
	if len(o) > 0 {
		response.Orders = o
	}

	return response, nil
}

// Markets provides a list of all current markets that exist on the VEGA platform.
func (h *tradingDataService) Markets(ctx context.Context, request *google_proto.Empty) (*protoapi.MarketsResponse, error) {
	mkts, err := h.MarketService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return &protoapi.MarketsResponse{
		Markets: mkts,
	}, nil
}

// OrdersByMarketAndId provides the given order, searching by Market and (Order)Id.
func (h *tradingDataService) OrderByMarketAndId(ctx context.Context,
	request *protoapi.OrderByMarketAndIdRequest) (*protoapi.OrderByMarketAndIdResponse, error) {

	if request.MarketID == "" {
		return nil, ErrEmptyMissingMarketID
	}
	if request.OrderID == "" {
		return nil, ErrEmptyMissingOrderID
	}
	order, err := h.OrderService.GetByMarketAndID(ctx, request.MarketID, request.OrderID)
	if err != nil {
		return nil, err
	}

	return &protoapi.OrderByMarketAndIdResponse{
		Order: order,
	}, nil
}

// OrderByReference provides the (possibly not yet accepted/rejected) order.
func (h *tradingDataService) OrderByReference(ctx context.Context, req *protoapi.OrderByReferenceRequest) (*protoapi.OrderByReferenceResponse, error) {
	if req.Reference == "" {
		return nil, ErrEmptyMissingOrderReference
	}
	order, err := h.OrderService.GetByReference(ctx, req.Reference)
	if err != nil {
		return nil, err
	}
	return &protoapi.OrderByReferenceResponse{
		Order: order,
	}, nil
}

// Candles returns trade OHLC/volume data for the given time period and interval.
// It will fill in any intervals without trades with zero based candles.
// SinceTimestamp must be in RFC3339 string format.
func (h *tradingDataService) Candles(ctx context.Context,
	request *protoapi.CandlesRequest) (*protoapi.CandlesResponse, error) {

	marketID := request.MarketID
	if marketID == "" {
		return nil, ErrEmptyMissingMarketID
	}

	if request.SinceTimestamp == 0 {
		return nil, ErrEmptyMissingSinceTimestamp
	}

	c, err := h.CandleService.GetCandles(ctx, marketID, vegatime.UnixNano(request.SinceTimestamp), request.Interval)
	if err != nil {
		return nil, err
	}

	return &protoapi.CandlesResponse{
		Candles: c,
	}, nil
}

// MarketDepth provides the order book for a given market, and also returns the most recent trade
// for the given market.
func (h *tradingDataService) MarketDepth(ctx context.Context, req *protoapi.MarketDepthRequest) (*protoapi.MarketDepthResponse, error) {
	if req.MarketID == "" {
		return nil, ErrEmptyMissingMarketID
	}

	// Query market depth statistics
	depth, err := h.MarketService.GetDepth(ctx, req.MarketID)
	if err != nil {
		return nil, err
	}
	t, err := h.TradeService.GetByMarket(ctx, req.MarketID, 0, 1, true)
	if err != nil {
		return nil, err
	}

	// Build market depth response, including last trade (if available)
	resp := &protoapi.MarketDepthResponse{
		Buy:      depth.Buy,
		MarketID: depth.MarketID,
		Sell:     depth.Sell,
	}
	if len(t) > 0 && t[0] != nil {
		resp.LastTrade = t[0]
	}
	return resp, nil
}

// TradesByMarket provides a list of trades for a given market.
// Pagination: Optional. If not provided, defaults are used.
func (h *tradingDataService) TradesByMarket(ctx context.Context, request *protoapi.TradesByMarketRequest) (*protoapi.TradesByMarketResponse, error) {
	if request.MarketID == "" {
		return nil, ErrEmptyMissingMarketID
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	t, err := h.TradeService.GetByMarket(ctx, request.MarketID, p.Skip, p.Limit, p.Descending)
	if err != nil {
		return nil, err
	}
	return &protoapi.TradesByMarketResponse{
		Trades: t,
	}, nil
}

// PositionsByParty provides a list of positions for a given party.
func (h *tradingDataService) PositionsByParty(ctx context.Context, request *protoapi.PositionsByPartyRequest) (*protoapi.PositionsByPartyResponse, error) {
	if request.PartyID == "" {
		return nil, ErrEmptyMissingPartyID
	}
	positions, err := h.TradeService.GetPositionsByParty(ctx, request.PartyID)
	if err != nil {
		return nil, err
	}
	var response = &protoapi.PositionsByPartyResponse{}
	response.Positions = positions
	return response, nil
}

func (h *tradingDataService) MarketDataByID(_ context.Context, req *protoapi.MarketDataByIDRequest) (*protoapi.MarketDataByIDResponse, error) {
	if len(req.MarketID) <= 0 {
		return nil, ErrEmptyMissingMarketID
	}
	md, err := h.MarketService.GetMarketDataByID(req.MarketID)
	if err != nil {
		return nil, err
	}
	return &protoapi.MarketDataByIDResponse{
		MarketData: &md,
	}, nil
}

func (h *tradingDataService) MarketsData(_ context.Context, _ *empty.Empty) (*protoapi.MarketsDataResponse, error) {
	mds := h.MarketService.GetMarketsData()
	mdptrs := make([]*types.MarketData, 0, len(mds))
	for _, v := range mds {
		mdptrs = append(mdptrs, &v)
	}
	return &protoapi.MarketsDataResponse{
		MarketsData: mdptrs,
	}, nil
}

// Statistics provides various blockchain and Vega statistics, including:
// Blockchain height, backlog length, current time, orders and trades per block, tendermint version
// Vega counts for parties, markets, order actions (amend, cancel, submit), Vega version
func (h *tradingDataService) Statistics(ctx context.Context, request *google_proto.Empty) (*types.Statistics, error) {
	// Call tendermint and related services to get information for statistics
	// We load read-only internal statistics through each package level statistics structs
	epochTime, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	if h.Stats == nil || h.Stats.Blockchain == nil {
		return nil, ErrStatisticsNotAvailable
	}

	// Call out to tendermint via rpc client
	backlogLength, numPeers, gt, err := h.getTendermintStats(ctx)
	if err != nil {
		return nil, err
	}

	// If the chain is replaying then genesis time can be nil
	genesisTime := ""
	if gt != nil {
		genesisTime = vegatime.Format(*gt)
	}

	// Load current markets details
	m, err := h.MarketService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	// Load current parties details
	p, err := h.PartyService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	// Extract names for ease of reading in stats
	partyNames := make([]string, 0, len(p))
	for _, v := range p {
		if v != nil {
			pp := *v
			partyNames = append(partyNames, pp.Id)
		}
	}

	return &types.Statistics{
		BlockHeight:              h.Stats.Blockchain.Height(),
		BacklogLength:            uint64(backlogLength),
		TotalPeers:               uint64(numPeers),
		GenesisTime:              genesisTime,
		CurrentTime:              vegatime.Format(vegatime.Now()),
		VegaTime:                 vegatime.Format(epochTime),
		TxPerBlock:               uint64(h.Stats.Blockchain.TotalTxLastBatch()),
		AverageTxBytes:           uint64(h.Stats.Blockchain.AverageTxSizeBytes()),
		AverageOrdersPerBlock:    uint64(h.Stats.Blockchain.AverageOrdersPerBatch()),
		TradesPerSecond:          uint64(h.Stats.Blockchain.TradesPerSecond()),
		OrdersPerSecond:          uint64(h.Stats.Blockchain.OrdersPerSecond()),
		Status:                   h.statusChecker.ChainStatus(),
		TotalMarkets:             uint64(len(m)),
		TotalParties:             uint64(len(p)),
		Parties:                  partyNames,
		AppVersionHash:           h.Stats.GetVersionHash(),
		AppVersion:               h.Stats.GetVersion(),
		ChainVersion:             h.Stats.GetChainVersion(),
		TotalAmendOrder:          h.Stats.Blockchain.TotalAmendOrder(),
		TotalCancelOrder:         h.Stats.Blockchain.TotalCancelOrder(),
		TotalCreateOrder:         h.Stats.Blockchain.TotalCreateOrder(),
		TotalOrders:              h.Stats.Blockchain.TotalOrders(),
		TotalTrades:              h.Stats.Blockchain.TotalTrades(),
		BlockDuration:            h.Stats.Blockchain.BlockDuration(),
		OrderSubscriptions:       h.OrderService.GetOrderSubscribersCount(),
		TradeSubscriptions:       h.TradeService.GetTradeSubscribersCount(),
		PositionsSubscriptions:   h.TradeService.GetPositionsSubscribersCount(),
		MarketDepthSubscriptions: h.MarketService.GetMarketDepthSubscribersCount(),
		CandleSubscriptions:      h.CandleService.GetCandleSubscribersCount(),
		AccountSubscriptions:     h.AccountsService.GetAccountSubscribersCount(),
		MarketDataSubscriptions:  h.MarketService.GetMarketDataSubscribersCount(),
	}, nil
}

// GetVegaTime returns the latest blockchain header timestamp, in UnixNano format.
// Example: "1568025900111222333" corresponds to 2019-09-09T10:45:00.111222333Z.
func (h *tradingDataService) GetVegaTime(ctx context.Context, request *google_proto.Empty) (*protoapi.VegaTimeResponse, error) {
	ts, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	return &protoapi.VegaTimeResponse{
		Timestamp: ts.UnixNano(),
	}, nil

}

func (h *tradingDataService) TransferResponsesSubscribe(
	req *empty.Empty, srv protoapi.TradingData_TransferResponsesSubscribeServer) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	transferResponseschan, ref := h.TransferResponseService.ObserveTransferResponses(
		ctx, h.Config.StreamRetries)
	h.log.Debug("TransferResponses subscriber - new rpc stream", logging.Uint64("ref", ref))
	var err error

	for {
		select {
		case transferResponses := <-transferResponseschan:
			if transferResponses == nil {
				err = ErrChannelClosed
				h.log.Error("transferResponses subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}
			for _, tr := range transferResponses {
				tr := tr
				err = srv.Send(tr)
				if err != nil {
					h.log.Error("TransferResponses subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return err
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			h.log.Debug("TransferResponses subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return ErrServerShutdown
		}

		if transferResponseschan == nil {
			h.log.Debug("TransferResponses subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return ErrStreamClosed
		}
	}
}

func (h *tradingDataService) MarketsDataSubscribe(req *protoapi.MarketsDataSubscribeRequest, srv protoapi.TradingData_MarketsDataSubscribeServer) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	marketdatachan, ref := h.MarketService.ObserveMarketsData(
		ctx, h.Config.StreamRetries, req.MarketID)
	h.log.Debug("Accounts subscriber - new rpc stream", logging.Uint64("ref", ref))

	var err error

	for {
		select {
		case mds := <-marketdatachan:
			if mds == nil {
				err = ErrChannelClosed
				h.log.Error("markets data subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}
			for _, md := range mds {
				err = srv.Send(&md)
				if err != nil {
					h.log.Error("markets data subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return err
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			h.log.Debug("Markets data subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return ErrServerShutdown
		}

		if marketdatachan == nil {
			h.log.Debug("Markets data subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return ErrStreamClosed
		}
	}
}

// AccountsSubscribe opens a subscription to the Accounts service.
func (h *tradingDataService) AccountsSubscribe(req *protoapi.AccountsSubscribeRequest, srv protoapi.TradingData_AccountsSubscribeServer) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	accountschan, ref := h.AccountsService.ObserveAccounts(
		ctx, h.Config.StreamRetries, req.MarketID, req.PartyID, req.Asset, req.Type)
	h.log.Debug("Accounts subscriber - new rpc stream", logging.Uint64("ref", ref))

	var err error

	for {
		select {
		case accounts := <-accountschan:
			if accounts == nil {
				err = ErrChannelClosed
				h.log.Error("accounts subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}
			for _, account := range accounts {
				account := account
				err = srv.Send(account)
				if err != nil {
					h.log.Error("Accounts subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return err
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			h.log.Debug("Accounts subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return ErrServerShutdown
		}

		if accountschan == nil {
			h.log.Debug("Accounts subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return ErrStreamClosed
		}
	}
}

// OrdersSubscribe opens a subscription to the Orders service.
// MarketID: Optional.
// PartyID: Optional.
func (h *tradingDataService) OrdersSubscribe(
	req *protoapi.OrdersSubscribeRequest, srv protoapi.TradingData_OrdersSubscribeServer) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	var (
		err               error
		marketID, partyID *string
	)
	if len(req.MarketID) > 0 {
		marketID = &req.MarketID
	}
	if len(req.PartyID) > 0 {
		partyID = &req.PartyID
	}

	orderschan, ref := h.OrderService.ObserveOrders(
		ctx, h.Config.StreamRetries, marketID, partyID)
	h.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))

	for {
		select {
		case orders := <-orderschan:
			if orders == nil {
				err = ErrChannelClosed
				h.log.Error("Orders subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}
			out := make([]*types.Order, 0, len(orders))
			for _, v := range orders {
				v := v
				out = append(out, &v)
			}
			err = srv.Send(&protoapi.OrdersStream{Orders: out})
			if err != nil {
				h.log.Error("Orders subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}
		case <-ctx.Done():
			err = ctx.Err()
			h.log.Debug("Orders subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return ErrServerShutdown
		}

		if orderschan == nil {
			h.log.Debug("Orders subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return ErrStreamClosed
		}
	}
}

// TradesSubscribe opens a subscription to the Trades service.
func (h *tradingDataService) TradesSubscribe(req *protoapi.TradesSubscribeRequest, srv protoapi.TradingData_TradesSubscribeServer) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	var (
		err               error
		marketID, partyID *string
	)
	if len(req.MarketID) > 0 {
		marketID = &req.MarketID
	}
	if len(req.PartyID) > 0 {
		partyID = &req.PartyID
	}

	tradeschan, ref := h.TradeService.ObserveTrades(
		ctx, h.Config.StreamRetries, marketID, partyID)
	h.log.Debug("Trades subscriber - new rpc stream", logging.Uint64("ref", ref))

	for {
		select {
		case trades := <-tradeschan:
			if len(trades) <= 0 {
				err = ErrChannelClosed
				h.log.Error("Trades subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}

			out := make([]*types.Trade, 0, len(trades))
			for _, v := range trades {
				v := v
				out = append(out, &v)
			}
			err = srv.Send(&protoapi.TradesStream{Trades: out})
			if err != nil {
				h.log.Error("Trades subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}
		case <-ctx.Done():
			err = ctx.Err()
			h.log.Debug("Trades subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return ErrServerShutdown
		}

		if tradeschan == nil {
			h.log.Debug("Trades subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return ErrStreamClosed
		}
	}
}

// CandlesSubscribe opens a subscription to the Candles service.
func (h *tradingDataService) CandlesSubscribe(req *protoapi.CandlesSubscribeRequest, srv protoapi.TradingData_CandlesSubscribeServer) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	var (
		err      error
		marketID *string
	)
	if len(req.MarketID) > 0 {
		marketID = &req.MarketID
	} else {
		return ErrEmptyMissingMarketID
	}

	candleschan, ref := h.CandleService.ObserveCandles(
		ctx, h.Config.StreamRetries, marketID, &req.Interval)
	h.log.Debug("Candles subscriber - new rpc stream", logging.Uint64("ref", ref))

	for {
		select {
		case candle := <-candleschan:
			if candle == nil {
				err = ErrChannelClosed
				h.log.Error("Candles subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}

			err = srv.Send(candle)
			if err != nil {
				h.log.Error("Candles subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}
		case <-ctx.Done():
			err = ctx.Err()
			h.log.Debug("Candles subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return ErrServerShutdown
		}

		if candleschan == nil {
			h.log.Debug("Candles subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return ErrStreamClosed
		}
	}
}

// MarketDepthSubscribe opens a subscription to the MarketDepth service.
func (h *tradingDataService) MarketDepthSubscribe(
	req *protoapi.MarketDepthSubscribeRequest,
	srv protoapi.TradingData_MarketDepthSubscribeServer,
) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	_, err := validateMarket(ctx, req.MarketID, h.MarketService)
	if err != nil {
		return err
	}

	depthchan, ref := h.MarketService.ObserveDepth(
		ctx, h.Config.StreamRetries, req.MarketID)
	h.log.Debug("Depth subscriber - new rpc stream", logging.Uint64("ref", ref))

	for {
		select {
		case depth := <-depthchan:
			if depth == nil {
				err = ErrChannelClosed
				h.log.Error("Depth subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}

			err = srv.Send(depth)
			if err != nil {
				h.log.Error("Depth subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}

		case <-ctx.Done():
			err = ctx.Err()
			h.log.Debug("Depth subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return ErrServerShutdown
		}

		if depthchan == nil {
			h.log.Debug("Depth subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return ErrStreamClosed
		}
	}
}

// PositionsSubscribe opens a subscription to the Positions service.
func (h *tradingDataService) PositionsSubscribe(
	req *protoapi.PositionsSubscribeRequest,
	srv protoapi.TradingData_PositionsSubscribeServer,
) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	positionschan, ref := h.TradeService.ObservePositions(
		ctx, h.Config.StreamRetries, req.PartyID)
	h.log.Debug("Positions subscriber - new rpc stream", logging.Uint64("ref", ref))

	for {
		select {
		case position := <-positionschan:
			if position == nil {
				err := ErrChannelClosed
				h.log.Error("Positions subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}
			err := srv.Send(position)
			if err != nil {
				h.log.Error("Positions subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return err
			}
		case <-ctx.Done():
			err := ctx.Err()
			h.log.Debug("Positions subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return ErrServerShutdown
		}

		if positionschan == nil {
			h.log.Debug("Positions subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return ErrStreamClosed
		}
	}
}

// MarketByID provides the given market.
func (h *tradingDataService) MarketByID(ctx context.Context, req *protoapi.MarketByIDRequest) (*protoapi.MarketByIDResponse, error) {
	mkt, err := validateMarket(ctx, req.MarketID, h.MarketService)
	if err != nil {
		return nil, err
	}

	return &protoapi.MarketByIDResponse{
		Market: mkt,
	}, nil
}

// Parties provides a list of all parties.
func (h *tradingDataService) Parties(ctx context.Context, req *google_proto.Empty) (*protoapi.PartiesResponse, error) {
	pties, err := h.PartyService.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return &protoapi.PartiesResponse{
		Parties: pties,
	}, nil
}

// PartyByID provides the given party.
func (h *tradingDataService) PartyByID(ctx context.Context, req *protoapi.PartyByIDRequest) (*protoapi.PartyByIDResponse, error) {
	pty, err := validateParty(ctx, req.PartyID, h.PartyService)
	if err != nil {
		return nil, err
	}
	return &protoapi.PartyByIDResponse{
		Party: pty,
	}, nil
}

// TradesByParty provides a list of trades for the given party.
// Pagination: Optional. If not provided, defaults are used.
func (h *tradingDataService) TradesByParty(
	ctx context.Context, req *protoapi.TradesByPartyRequest,
) (*protoapi.TradesByPartyResponse, error) {

	p := defaultPagination
	if req.Pagination != nil {
		p = *req.Pagination
	}

	trades, err := h.TradeService.GetByParty(ctx, req.PartyID, p.Skip, p.Limit, p.Descending, &req.MarketID)
	if err != nil {
		return nil, err
	}

	return &protoapi.TradesByPartyResponse{
		Trades: trades,
	}, nil
}

// TradesByOrder provides a list of the trades that correspond to a given order.
func (h *tradingDataService) TradesByOrder(
	ctx context.Context, req *protoapi.TradesByOrderRequest,
) (*protoapi.TradesByOrderResponse, error) {
	trades, err := h.TradeService.GetByOrderID(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	return &protoapi.TradesByOrderResponse{
		Trades: trades,
	}, nil
}

// LastTrade provides the last trade for the given market.
func (h *tradingDataService) LastTrade(
	ctx context.Context, req *protoapi.LastTradeRequest,
) (*protoapi.LastTradeResponse, error) {
	if len(req.MarketID) <= 0 {
		return nil, ErrEmptyMissingMarketID
	}
	t, err := h.TradeService.GetByMarket(ctx, req.MarketID, 0, 1, true)
	if err != nil {
		return nil, err
	}
	if t != nil && len(t) > 0 && t[0] != nil {
		return &protoapi.LastTradeResponse{Trade: t[0]}, nil
	}
	// No trades found on the market yet (and no errors)
	// this can happen at the beginning of a new market
	return &protoapi.LastTradeResponse{}, nil
}

// AccountsByParty provides a list of accounts for the given party.
func (h *tradingDataService) AccountsByParty(ctx context.Context, req *protoapi.AccountsByPartyRequest) (*protoapi.AccountsByPartyResponse, error) {
	accs, err := h.AccountsService.GetByParty(req.PartyID)
	if err != nil {
		return nil, err
	}
	return &protoapi.AccountsByPartyResponse{
		Accounts: accs,
	}, nil
}

// AccountsByPartyAndMarket provides a list of accounts for the given party and market.
func (h *tradingDataService) AccountsByPartyAndMarket(ctx context.Context, req *protoapi.AccountsByPartyAndMarketRequest) (*protoapi.AccountsByPartyAndMarketResponse, error) {
	accs, err := h.AccountsService.GetByPartyAndMarket(req.PartyID, req.MarketID)
	if err != nil {
		return nil, err
	}
	return &protoapi.AccountsByPartyAndMarketResponse{
		Accounts: accs,
	}, nil
}

// AccountsByPartyAndType provides a list of accounts of the given type for the given party.
func (h *tradingDataService) AccountsByPartyAndType(ctx context.Context, req *protoapi.AccountsByPartyAndTypeRequest) (*protoapi.AccountsByPartyAndTypeResponse, error) {
	accs, err := h.AccountsService.GetByPartyAndType(req.PartyID, req.Type)
	if err != nil {
		return nil, err
	}
	return &protoapi.AccountsByPartyAndTypeResponse{
		Accounts: accs,
	}, nil
}

// AccountsByPartyAndAsset provides a list of accounts for the given party.
func (h *tradingDataService) AccountsByPartyAndAsset(ctx context.Context, req *protoapi.AccountsByPartyAndAssetRequest) (*protoapi.AccountsByPartyAndAssetResponse, error) {
	accs, err := h.AccountsService.GetByPartyAndAsset(req.PartyID, req.Asset)
	if err != nil {
		return nil, err
	}
	return &protoapi.AccountsByPartyAndAssetResponse{
		Accounts: accs,
	}, nil
}

func validateMarket(ctx context.Context, marketID string, marketService MarketService) (*types.Market, error) {
	var mkt *types.Market
	var err error
	if len(marketID) == 0 {
		return nil, ErrEmptyMissingMarketID
	}
	mkt, err = marketService.GetByID(ctx, marketID)
	if err != nil {
		return nil, err
	}

	return mkt, nil
}

func validateParty(ctx context.Context, partyID string, partyService PartyService) (*types.Party, error) {
	var pty *types.Party
	var err error
	if len(partyID) == 0 {
		return nil, ErrEmptyMissingPartyID
	}
	pty, err = partyService.GetByID(ctx, partyID)
	if err != nil {
		return nil, err
	}
	return pty, err
}

func (h *tradingDataService) getTendermintStats(ctx context.Context) (backlogLength int,
	numPeers int, genesis *time.Time, err error) {

	refused := "connection refused"

	// Unconfirmed TX count == current transaction backlog length
	backlogLength, err = h.Client.GetUnconfirmedTxCount(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return 0, 0, nil, nil
		}
		return 0, 0, nil, err
	}

	// Net info provides peer stats etc (block chain network info) == number of peers
	netInfo, err := h.Client.GetNetworkInfo(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return backlogLength, 0, nil, nil
		}
		return backlogLength, 0, nil, err
	}

	// Genesis retrieves the current genesis date/time for the blockchain
	genesisTime, err := h.Client.GetGenesisTime(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return backlogLength, 0, nil, nil
		}
		return backlogLength, 0, nil, err
	}

	return backlogLength, netInfo.NPeers, &genesisTime, nil
}
