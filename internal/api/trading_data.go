package api

import (
	"context"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	google_proto "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

var ErrChainNotConnected = errors.New("Chain not connected")

//go:generate go run github.com/golang/mock/mockgen -destination mocks/vega_time_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc VegaTime
type VegaTime interface {
	GetTimeNow() (time.Time, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc OrderService
type OrderService interface {
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (order *types.Order, err error)
	ObserveOrders(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Order, ref uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc TradeService
type TradeService interface {
	GetByOrderId(ctx context.Context, orderID string) ([]*types.Trade, error)
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) (trades []*types.Trade, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, marketID *string) (trades []*types.Trade, err error)
	GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error)
	ObserveTrades(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Trade, ref uint64)
	ObservePositions(ctx context.Context, retries int, party string) (positions <-chan *types.MarketPosition, ref uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc CandleService
type CandleService interface {
	GetCandles(ctx context.Context, market string, since time.Time, interval types.Interval) (candles []*types.Candle, err error)
	ObserveCandles(ctx context.Context, retries int, market *string, interval *types.Interval) (candleCh <-chan *types.Candle, ref uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc MarketService
type MarketService interface {
	GetByID(ctx context.Context, name string) (*types.Market, error)
	GetAll(ctx context.Context) ([]*types.Market, error)
	GetDepth(ctx context.Context, market string) (marketDepth *types.MarketDepth, err error)
	ObserveDepth(ctx context.Context, retries int, market string) (depth <-chan *types.MarketDepth, ref uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/party_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc PartyService
type PartyService interface {
	GetByID(ctx context.Context, id string) (*types.Party, error)
	GetAll(ctx context.Context) ([]*types.Party, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_client_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc BlockchainClient
type BlockchainClient interface {
	GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error)
	GetUnconfirmedTxCount(ctx context.Context) (count int, err error)
	GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error)
}

type tradingDataService struct {
	log           *logging.Logger
	Config        Config
	Client        BlockchainClient
	Stats         *internal.Stats
	TimeService   VegaTime
	OrderService  OrderService
	TradeService  TradeService
	CandleService CandleService
	MarketService MarketService
	PartyService  PartyService
	statusChecker *monitoring.Status
	ctx           context.Context
}

// If no limit is provided at the gRPC API level, the system will use this limit instead.
// (Prevent returning all results every time a careless query is made)
const defaultLimit = uint64(1000)

// OrdersByMarket provides a list of orders for a given market. Optional limits can be provided. Most recent first.
func (h *tradingDataService) OrdersByMarket(ctx context.Context,
	request *protoapi.OrdersByMarketRequest) (*protoapi.OrdersByMarketResponse, error) {

	if request.MarketID == "" {
		return nil, errors.New("Market empty or missing")
	}

	var (
		skip, limit uint64
		descending  bool
		open        *bool
	)
	if request.Params != nil && request.Params.Limit > 0 {
		descending = true
		limit = request.Params.Limit
	}

	o, err := h.OrderService.GetByMarket(ctx, request.MarketID, skip, limit, descending, open)
	if err != nil {
		return nil, err
	}

	var response = &protoapi.OrdersByMarketResponse{}
	if len(o) > 0 {
		response.Orders = o
	}

	return response, nil
}

// OrdersByParty provides a list of orders for a given party. Optional limits can be provided. Most recent first.
func (h *tradingDataService) OrdersByParty(ctx context.Context,
	request *protoapi.OrdersByPartyRequest) (*protoapi.OrdersByPartyResponse, error) {

	if request.PartyID == "" {
		return nil, errors.New("Party empty or missing")
	}

	var limit uint64
	if request.Params != nil && request.Params.Limit > 0 {
		limit = request.Params.Limit
	}

	o, err := h.OrderService.GetByParty(ctx, request.PartyID, 0, limit, true, nil)
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

// OrdersByMarketAndId searches for the given order by Id and Market. If found it will return
// an Order types otherwise it will return an error.
func (h *tradingDataService) OrderByMarketAndId(ctx context.Context,
	request *protoapi.OrderByMarketAndIdRequest) (*protoapi.OrderByMarketAndIdResponse, error) {

	if request.MarketID == "" {
		return nil, errors.New("Market empty or missing")
	}
	if request.Id == "" {
		return nil, errors.New("Id empty or missing")
	}
	order, err := h.OrderService.GetByMarketAndId(ctx, request.MarketID, request.Id)
	if err != nil {
		return nil, err
	}

	return &protoapi.OrderByMarketAndIdResponse{
		Order: order,
	}, nil
}

// TradeCandles returns trade open/close/volume data for the given time period and interval.
// It will fill in any trade-less intervals with zero based candles. Since time period must be in RFC3339 string format.
func (h *tradingDataService) Candles(ctx context.Context,
	request *protoapi.CandlesRequest) (*protoapi.CandlesResponse, error) {

	market := request.Market
	if market == "" {
		return nil, errors.New("Market empty or missing")
	}

	if request.SinceTimestamp == 0 {
		return nil, errors.New("Since date is missing")
	}

	c, err := h.CandleService.GetCandles(ctx, market, vegatime.UnixNano(request.SinceTimestamp), request.Interval)
	if err != nil {
		return nil, err
	}

	return &protoapi.CandlesResponse{
		Candles: c,
	}, nil

}

func (h *tradingDataService) MarketDepth(ctx context.Context, req *protoapi.MarketDepthRequest) (*protoapi.MarketDepthResponse, error) {
	if req.Market == "" {
		return nil, errors.New("Market empty or missing")
	}

	// Query market depth statistics
	depth, err := h.MarketService.GetDepth(ctx, req.Market)
	if err != nil {
		return nil, err
	}
	t, err := h.TradeService.GetByMarket(ctx, req.Market, 0, 1, true)
	if err != nil {
		return nil, err
	}

	// Build market depth response, including last trade (if available)
	resp := &protoapi.MarketDepthResponse{
		Buy:      depth.Buy,
		MarketID: depth.Name,
		Sell:     depth.Sell,
	}
	if t != nil && t[0] != nil {
		resp.LastTrade = t[0]
	}
	return resp, nil
}

func (h *tradingDataService) TradesByMarket(ctx context.Context, request *protoapi.TradesByMarketRequest) (*protoapi.TradesByMarketResponse, error) {
	if request.MarketID == "" {
		return nil, errors.New("Market empty or missing")
	}
	limit := defaultLimit
	if request.Params != nil && request.Params.Limit > 0 {
		limit = request.Params.Limit
	}

	t, err := h.TradeService.GetByMarket(ctx, request.MarketID, 0, limit, true)
	if err != nil {
		return nil, err
	}
	return &protoapi.TradesByMarketResponse{
		Trades: t,
	}, nil
}

func (h *tradingDataService) PositionsByParty(ctx context.Context, request *protoapi.PositionsByPartyRequest) (*protoapi.PositionsByPartyResponse, error) {
	if request.PartyID == "" {
		return nil, errors.New("Party empty or missing")
	}
	positions, err := h.TradeService.GetPositionsByParty(ctx, request.PartyID)
	if err != nil {
		return nil, err
	}
	var response = &protoapi.PositionsByPartyResponse{}
	response.Positions = positions
	return response, nil
}

func (h *tradingDataService) Statistics(ctx context.Context, request *google_proto.Empty) (*types.Statistics, error) {
	// Call out to tendermint and related services to get related information for statistics
	// We load read-only internal statistics through each package level statistics structs
	epochTime, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	if h.Stats == nil || h.Stats.Blockchain == nil {
		return nil, errors.New("Internal error: statistics not available")
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
			partyNames = append(partyNames, pp.Name)
		}
	}

	return &types.Statistics{
		BlockHeight:           h.Stats.Blockchain.Height(),
		BacklogLength:         uint64(backlogLength),
		TotalPeers:            uint64(numPeers),
		GenesisTime:           genesisTime,
		CurrentTime:           vegatime.Format(vegatime.Now()),
		VegaTime:              vegatime.Format(epochTime),
		TxPerBlock:            uint64(h.Stats.Blockchain.TotalTxLastBatch()),
		AverageTxBytes:        uint64(h.Stats.Blockchain.AverageTxSizeBytes()),
		AverageOrdersPerBlock: uint64(h.Stats.Blockchain.AverageOrdersPerBatch()),
		TradesPerSecond:       uint64(h.Stats.Blockchain.TradesPerSecond()),
		OrdersPerSecond:       uint64(h.Stats.Blockchain.OrdersPerSecond()),
		Status:                h.statusChecker.ChainStatus(),
		TotalMarkets:          uint64(len(m)),
		TotalParties:          uint64(len(p)),
		Parties:               partyNames,
		AppVersionHash:        h.Stats.GetVersionHash(),
		AppVersion:            h.Stats.GetVersion(),
		TotalAmendOrder:       h.Stats.Blockchain.TotalAmendOrder(),
		TotalCancelOrder:      h.Stats.Blockchain.TotalCancelOrder(),
		TotalCreateOrder:      h.Stats.Blockchain.TotalCreateOrder(),
		TotalOrders:           h.Stats.Blockchain.TotalOrders(),
		TotalTrades:           h.Stats.Blockchain.TotalTrades(),
	}, nil
}

func (h *tradingDataService) GetVegaTime(ctx context.Context, request *google_proto.Empty) (*protoapi.VegaTimeResponse, error) {
	ts, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	return &protoapi.VegaTimeResponse{
		Timestamp: ts.UnixNano(),
	}, nil

}

func (h *tradingDataService) OrdersSubscribe(
	req *protoapi.OrdersSubscribeRequest, srv protoapi.TradingData_OrdersSubscribeServer) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	_, err := validateMarket(ctx, req.MarketID, h.MarketService)
	if err != nil {
		return err
	}

	orderschan, ref := h.OrderService.ObserveOrders(
		ctx, h.Config.StreamRetries, &req.MarketID, &req.PartyID)
	h.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))

	for {
		select {
		case orders := <-orderschan:
			for _, o := range orders {
				err := srv.Send(&o)
				if err != nil {
					h.log.Error("Orders subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return err
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			h.log.Debug("Orders subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return errors.New("server shutdown")
		}

		if orderschan == nil {
			h.log.Debug("Orders subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return errors.New("stream closed")
		}
	}
}

func (h *tradingDataService) TradesSubscribe(req *protoapi.TradesSubscribeRequest, srv protoapi.TradingData_TradesSubscribeServer) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	_, err := validateMarket(ctx, req.MarketID, h.MarketService)
	if err != nil {
		return err
	}

	tradeschan, ref := h.TradeService.ObserveTrades(
		ctx, h.Config.StreamRetries, &req.MarketID, &req.PartyID)
	h.log.Debug("Trades subscriber - new rpc stream", logging.Uint64("ref", ref))

	for {
		select {
		case trades := <-tradeschan:
			for _, o := range trades {
				err := srv.Send(&o)
				if err != nil {
					h.log.Error("Trades subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return err
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			h.log.Debug("Trades subscriber - rpc stream ctx error",
				logging.Error(err),
				logging.Uint64("ref", ref),
			)
			return err
		case <-h.ctx.Done():
			return errors.New("server shutdown")
		}

		if tradeschan == nil {
			h.log.Debug("Trades subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return errors.New("stream closed")
		}
	}
}

func (h *tradingDataService) CandlesSubscribe(req *protoapi.CandlesSubscribeRequest, srv protoapi.TradingData_CandlesSubscribeServer) error {
	// wrap context from the request into cancellable. we can closed internal chan in error
	ctx, cfunc := context.WithCancel(srv.Context())
	defer cfunc()

	_, err := validateMarket(ctx, req.MarketID, h.MarketService)
	if err != nil {
		return err
	}

	candleschan, ref := h.CandleService.ObserveCandles(
		ctx, h.Config.StreamRetries, &req.MarketID, &req.Interval)
	h.log.Debug("Candles subscriber - new rpc stream", logging.Uint64("ref", ref))

	for {
		select {
		case candle := <-candleschan:
			err := srv.Send(candle)
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
			return errors.New("server shutdown")
		}

		if candleschan == nil {
			h.log.Debug("Candles subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return errors.New("stream closed")
		}
	}
}

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
			err := srv.Send(depth)
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
			return errors.New("server shutdown")
		}

		if depthchan == nil {
			h.log.Debug("Depth subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return errors.New("stream closed")
		}
	}
}

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
			return errors.New("server shutdown")
		}

		if positionschan == nil {
			h.log.Debug("Positions subscriber - rpc stream closed",
				logging.Uint64("ref", ref),
			)
			return errors.New("stream closed")
		}
	}
}

func (h *tradingDataService) MarketByID(ctx context.Context, req *protoapi.MarketByIDRequest) (*protoapi.MarketByIDResponse, error) {
	mkt, err := validateMarket(ctx, req.Id, h.MarketService)
	if err != nil {
		return nil, err
	}

	return &protoapi.MarketByIDResponse{
		Market: mkt,
	}, nil
}
func (h *tradingDataService) Parties(ctx context.Context, req *google_proto.Empty) (*protoapi.PartiesResponse, error) {
	pties, err := h.PartyService.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return &protoapi.PartiesResponse{
		Parties: pties,
	}, nil
}
func (h *tradingDataService) PartyByID(ctx context.Context, req *protoapi.PartyByIDRequest) (*protoapi.PartyByIDResponse, error) {
	pty, err := validateParty(ctx, req.Id, h.PartyService)
	if err != nil {
		return nil, err
	}
	return &protoapi.PartyByIDResponse{
		Party: pty,
	}, nil
}
func (h *tradingDataService) TradesByParty(
	ctx context.Context, req *protoapi.TradesByPartyRequest,
) (*protoapi.TradesByPartyResponse, error) {
	var (
		skip, limit uint64
		descending  bool
	)
	if req.Params != nil && req.Params.Limit > 0 {
		descending = true
		limit = req.Params.Limit
	}

	trades, err := h.TradeService.GetByParty(ctx, req.PartyID, skip, limit, descending, &req.MarketID)
	if err != nil {
		return nil, err
	}

	return &protoapi.TradesByPartyResponse{
		Trades: trades,
	}, nil
}
func (h *tradingDataService) TradesByOrder(
	ctx context.Context, req *protoapi.TradesByOrderRequest,
) (*protoapi.TradesByOrderResponse, error) {
	trades, err := h.TradeService.GetByOrderId(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	return &protoapi.TradesByOrderResponse{
		Trades: trades,
	}, nil
}

func (h *tradingDataService) LastTrade(
	ctx context.Context, req *protoapi.LastTradeRequest,
) (*protoapi.LastTradeResponse, error) {
	if len(req.MarketID) <= 0 {
		return nil, errors.New("missing market ID")
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

func validateMarket(ctx context.Context, marketID string, marketService MarketService) (*types.Market, error) {
	var mkt *types.Market
	var err error
	if len(marketID) == 0 {
		return nil, errors.New("market must not be empty")
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
		return nil, errors.New("party must not be empty")
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
