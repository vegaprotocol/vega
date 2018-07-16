package api

import (
	"context"
	"errors"
	"time"

	"vega/core"
	"vega/datastore"
	"vega/proto"
	"vega/tendermint/rpc"

	"github.com/golang/protobuf/proto"
	"sync"
)

var (
	clients []*rpc.Client
	mux sync.Mutex
)

type TradeService interface {
	Init(app *core.Vega, tradeStore datastore.TradeStore)
	GetById(ctx context.Context, market string, id string) (trade msg.Trade, err error)
	GetTrades(ctx context.Context, market string, limit uint64) (trades []msg.Trade, err error)
	GetTradesForOrder(ctx context.Context, market string, orderId string, limit uint64) (trades []msg.Trade, err error)
	GetCandlesChart(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error)
}

type tradeService struct {
	app        *core.Vega
	tradeStore datastore.TradeStore
}

func NewTradeService() TradeService {
	return &tradeService{}
}

func (t *tradeService) Init(app *core.Vega, tradeStore datastore.TradeStore) {
	t.app = app
	t.tradeStore = tradeStore
}

func (t *tradeService) GetTrades(ctx context.Context, market string, limit uint64) (trades []msg.Trade, err error) {
	tr, err := t.tradeStore.GetAll(market, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, *trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetTradesForOrder(ctx context.Context, market string, orderId string, limit uint64) (trades []msg.Trade, err error) {
	tr, err := t.tradeStore.GetByOrderId(market, orderId, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, *trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetById(ctx context.Context, market string, id string) (trade msg.Trade, err error) {
	tr, err := t.tradeStore.Get(market, id)
	if err != nil {
		return msg.Trade{}, err
	}
	return *tr.ToProtoMessage(), err
}

func (t *tradeService) GetCandlesChart(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error) {
	// compare time and translate it into timestamps
	appCurrentTime := t.app.GetTime()

	delta := appCurrentTime.Sub(since)
	deltaInSeconds := int64(delta.Seconds())
	if deltaInSeconds < 0 {
		return msg.Candles{}, errors.New("INVALID_REQUEST")
	}

	sinceBlock := t.app.GetAbciHeight() - deltaInSeconds
	if sinceBlock < 0 {
		sinceBlock = 0
	}
	
	c, err := t.tradeStore.GetCandles(market, uint64(sinceBlock), uint64(t.app.GetAbciHeight()), interval)
	if err != nil {
		return msg.Candles{}, err
	}

	aggregationStartTime := appCurrentTime.Add(-delta)
	for i, candle := range c.Candles {
		candleDuration := time.Duration(i*int(interval)) * time.Second
		candle.Date = aggregationStartTime.Add(candleDuration).Format(time.RFC3339)
	}

	return c, nil
}

type OrderService interface {
	Init(vega *core.Vega, orderStore datastore.OrderStore)
	GetById(ctx context.Context, market string, id string) (order msg.Order, err error)
	CreateOrder(ctx context.Context, order *msg.Order) (success bool, err error)
	GetOrders(ctx context.Context, market string, party string, limit uint64) (orders []msg.Order, err error)
	GetOrderBookDepthChart(ctx context.Context, market string) (orderBookDepth *msg.OrderBookDepth, err error)
}

type orderService struct {
	app        *core.Vega
	orderStore datastore.OrderStore
}

func NewOrderService() OrderService {
	return &orderService{}
}

func (p *orderService) Init(app *core.Vega, orderStore datastore.OrderStore) {
	p.app = app
	p.orderStore = orderStore
}

func (p *orderService) CreateOrder(ctx context.Context, order *msg.Order) (success bool, err error) {
	order.Remaining = order.Size

	// Protobuf marshall the incoming order to byte slice.
	bytes, err := proto.Marshal(order)
	if err != nil {
		return false, err
	}
	if len(bytes) == 0 {
		return false, errors.New("order message cannot be empty")
	}

	// Tendermint requires unique transactions so we pre-pend a guid + pipe to the byte array.
	// It's split on arrival out of concensus.
	bytes, err = bytesWithPipedGuid(bytes)
	if err != nil {
		return false, err
	}

	// Get a lightweight RPC client (our custom Tendermint client) from a pool (create one if n/a).
	client, err := getClient()
	if err != nil {
		return false, err
	}

	// Fire off the transaction for consensus
	err = client.AsyncTransaction(ctx, bytes)
	if err != nil {
		if !client.HasError() {
			releaseClient(client)
		}
		return false, err
	}

	// If all went well we return the client to the pool for another caller.
	if client != nil {
		releaseClient(client)
	}
	return true, err
}

func (p *orderService) GetOrders(ctx context.Context, market string, party string, limit uint64) (orders []msg.Order, err error) {
	o, err := p.orderStore.GetAll(market, party, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	result := make([]msg.Order, 0)
	for _, order := range o {
		result = append(result, msg.Order{
			Id:        order.Id,
			Market:    order.Market,
			Party:     order.Party,
			Side:      order.Side,
			Price:     order.Price,
			Size:      order.Timestamp,
			Remaining: order.Remaining,
			Timestamp: order.Timestamp,
			Type:      order.Type,
		})
	}
	return result, err
}

func (p *orderService) GetById(ctx context.Context, market string, id string) (order msg.Order, err error) {
	or, err := p.orderStore.Get(market, id)
	if err != nil {
		return msg.Order{}, err
	}
	return *or.ToProtoMessage(), err
}

func (p *orderService) GetOrderBookDepthChart(ctx context.Context, marketName string) (orderBookDepth *msg.OrderBookDepth, err error) {
	return p.orderStore.GetOrderBookDepth(marketName)
}

func getClient() (*rpc.Client, error) {
	mux.Lock()
	if len(clients) == 0 {
		mux.Unlock()
		client := rpc.Client{
		}
		if err := client.Connect(); err != nil {
			return nil, err
		}
		return &client, nil
	}
	client := clients[0]
	clients = clients[1:]
	mux.Unlock()
	return client, nil
}

func releaseClient(c *rpc.Client) {
	mux.Lock()
	clients = append(clients, c)
	mux.Unlock()
}
