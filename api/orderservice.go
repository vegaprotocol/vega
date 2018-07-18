package api

import (
	"context"
	"errors"
	"sync"

	"vega/proto"
	"vega/core"
	"vega/datastore"
	"vega/tendermint/rpc"

	"github.com/golang/protobuf/proto"
)

var (
	clients []*rpc.Client
	mux sync.Mutex
)

type OrderService interface {
	Init(vega *core.Vega, orderStore datastore.OrderStore)
	CreateOrder(ctx context.Context, order *msg.Order) (success bool, err error)
	GetByMarket(ctx context.Context, market string, limit uint64) (orders []*msg.Order, err error)
	GetByParty(ctx context.Context, party string, limit uint64) (orders []*msg.Order, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (order *msg.Order, err error)
	GetByPartyAndId(ctx context.Context, market string, id string) (order *msg.Order, err error)
	GetMarkets(ctx context.Context) ([]string, error)
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

func (p *orderService) GetByMarket(ctx context.Context, market string, limit uint64) (orders []*msg.Order, err error) {
	o, err := p.orderStore.GetByMarket(market, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	result := make([]*msg.Order, 0)
	for _, order := range o {
		//if order.Remaining == 0 {
		//	continue
		//}
		o := &msg.Order{
			Id:        order.Id,
			Market:    order.Market,
			Party:     order.Party,
			Side:      order.Side,
			Price:     order.Price,
			Size:      order.Timestamp,
			Remaining: order.Remaining,
			Timestamp: order.Timestamp,
			Type:      order.Type,
		}
		result = append(result, o)
	}
	return result, err
}

func (p *orderService) GetByParty(ctx context.Context, party string, limit uint64) (orders []*msg.Order, err error) {
	o, err := p.orderStore.GetByParty(party, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	result := make([]*msg.Order, 0)
	for _, order := range o {
		//if order.Remaining == 0 {
		//	continue
		//}
		o := &msg.Order{
			Id:        order.Id,
			Market:    order.Market,
			Party:     order.Party,
			Side:      order.Side,
			Price:     order.Price,
			Size:      order.Timestamp,
			Remaining: order.Remaining,
			Timestamp: order.Timestamp,
			Type:      order.Type,
		}
		result = append(result, o)
	}
	return result, err
}

func (p *orderService) GetByMarketAndId(ctx context.Context, market string, id string) (order *msg.Order, err error) {
	o, err := p.orderStore.GetByMarketAndId(market, id)
	if err != nil {
		return &msg.Order{}, err
	}
	return o.ToProtoMessage(), err
}

func (p *orderService) GetByPartyAndId(ctx context.Context, market string, id string) (order *msg.Order, err error) {
	o, err := p.orderStore.GetByPartyAndId(market, id)
	if err != nil {
		return &msg.Order{}, err
	}
	return o.ToProtoMessage(), err
}

func (p *orderService) GetMarkets(ctx context.Context) ([]string, error) {
	markets, err := p.orderStore.GetMarkets()
	if err != nil {
		return []string{}, err
	}
	return markets, err
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