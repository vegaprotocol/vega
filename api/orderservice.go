package api

import (
	"context"
	"vega/blockchain"
	"vega/core"
	"vega/datastore"
	"vega/msg"
	"github.com/pkg/errors"
	"vega/log"
	"vega/filters"
)

type OrderService interface {
	Init(vega *core.Vega, orderStore datastore.OrderStore)
	ObserveOrders(ctx context.Context, market *string, party *string) (orders <-chan msg.Order, ref uint64)

	CreateOrder(ctx context.Context, order *msg.Order) (success bool, err error)
	CancelOrder(ctx context.Context, order *msg.Order) (success bool, err error)
	GetByMarket(ctx context.Context, market string, filters *filters.OrderQueryFilters) (orders []*msg.Order, err error)
	GetByParty(ctx context.Context, party string, filters *filters.OrderQueryFilters) (orders []*msg.Order, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (order *msg.Order, err error)
	GetByPartyAndId(ctx context.Context, market string, id string) (order *msg.Order, err error)

	GetMarkets(ctx context.Context) ([]string, error)
	GetMarketDepth(ctx context.Context, market string) (marketDepth *msg.MarketDepth, err error)
	ObserveMarketDepth(ctx context.Context, market string) (depth <-chan msg.MarketDepth, ref uint64)
}

type orderService struct {
	app        *core.Vega
	orderStore datastore.OrderStore
	blockchain blockchain.Client
}

func NewOrderService() OrderService {
	return &orderService{}
}

func (p *orderService) Init(app *core.Vega, orderStore datastore.OrderStore) {
	p.app = app
	p.orderStore = orderStore
	p.blockchain = blockchain.NewClient()
}

func (p *orderService) CreateOrder(ctx context.Context, order *msg.Order) (success bool, err error) {
	// Set defaults, prevent unwanted external manipulation
	order.Remaining = order.Size
	order.Status = msg.Order_Active
	order.Type = msg.Order_GTC // VEGA only supports GTC at present
	order.Timestamp = 0

	// TODO validate

	// Call out to the blockchain package/layer and use internal client to gain consensus
	return p.blockchain.CreateOrder(ctx, order)
}

// CancelOrder requires valid ID, Market, Party on an attempt to cancel the given active order via consensus
func (p *orderService) CancelOrder(ctx context.Context, order *msg.Order) (success bool, err error) {
	// Validate order exists using read store
	o, err := p.orderStore.GetByMarketAndId(order.Market, order.Id)
	if err != nil {
		return false, err
	}
	if o.Status == msg.Order_Cancelled {
		return false, errors.New("order has already been cancelled")
	}
	if o.Remaining == 0 {
		return false, errors.New("order has been fully filled")
	}
	if o.Party != order.Party {
		return false, errors.New("party mis-match cannot cancel order")
	}
	// Send cancellation request by consensus 
	return p.blockchain.CancelOrder(ctx, o.ToProtoMessage())
}

func (p *orderService) GetByMarket(ctx context.Context, market string, filters *filters.OrderQueryFilters) (orders []*msg.Order, err error) {
	o, err := p.orderStore.GetByMarket(market, filters)
	if err != nil {
		return nil, err
	}
	filterOpen := filters != nil && filters.Open == true
	result := make([]*msg.Order, 0)
	for _, order := range o {
		if filterOpen && (order.Remaining == 0 || order.Status != msg.Order_Active) {
			continue
		}
		o := &msg.Order{
			Id:        order.Id,
			Market:    order.Market,
			Party:     order.Party,
			Side:      order.Side,
			Price:     order.Price,
			Size:      order.Size,
			Remaining: order.Remaining,
			Timestamp: order.Timestamp,
			Type:      order.Type,
			Status:    order.Status,
		}
		result = append(result, o)
	}
	return result, err
}

func (p *orderService) GetByParty(ctx context.Context, party string, filters *filters.OrderQueryFilters) (orders []*msg.Order, err error) {
	o, err := p.orderStore.GetByParty(party, filters)
	if err != nil {
		return nil, err
	}
	filterOpen := filters != nil && filters.Open == true
	result := make([]*msg.Order, 0)
	for _, order := range o {
		if filterOpen && (order.Remaining == 0 || order.Status != msg.Order_Active) {
			continue
		}
		o := &msg.Order{
			Id:        order.Id,
			Market:    order.Market,
			Party:     order.Party,
			Side:      order.Side,
			Price:     order.Price,
			Size:      order.Size,
			Remaining: order.Remaining,
			Timestamp: order.Timestamp,
			Type:      order.Type,
			Status:    order.Status,
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

func (p *orderService) GetMarketDepth(ctx context.Context, marketName string) (orderBookDepth *msg.MarketDepth, err error) {
	return p.orderStore.GetMarketDepth(marketName)
}

func (p *orderService) ObserveOrders(ctx context.Context, market *string, party *string) (<-chan msg.Order, uint64) {
	orders := make(chan msg.Order)
	internal := make(chan []datastore.Order)
	ref := p.orderStore.Subscribe(internal)

	go func(id uint64, internal chan []datastore.Order) {
		<-ctx.Done()
		log.Debugf("OrderService -> Subscriber closed connection: %d", id)
		err := p.orderStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing when context.Done() on OrderService for id: %d", id)
		}
		close(internal)
	}(ref, internal)

	go func(id uint64) {
		for v := range internal {
			for _, item := range v {
				if market != nil && item.Market != *market {
					continue
				}
				if party != nil && item.Party != *party {
					continue
				}
				orders <- *item.ToProtoMessage()
			}
		}
		log.Debugf("OrderService -> Channel for subscriber %d has been closed", ref)
	}(ref)

	return orders, ref
}

func (p *orderService) ObserveMarketDepth(ctx context.Context, market string) (<-chan msg.MarketDepth, uint64) {
	depth := make(chan msg.MarketDepth)
	internal := make(chan []datastore.Order)
	ref := p.orderStore.Subscribe(internal)

	go func(id uint64, internal chan []datastore.Order) {
		<-ctx.Done()
		log.Debugf("OrderService -> Depth closed connection: %d", id)
		err := p.orderStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing depth when context.Done() on OrderService for id: %d", id)
		}
		close(internal)
	}(ref, internal)

	go func(id uint64) {
		for range internal {

			d, err := p.orderStore.GetMarketDepth(market)
			if err != nil {
				log.Errorf("error calculating market depth", err)
			} else {
				depth <- *d
			}

		}
		log.Debugf("OrderService -> Channel for depth subscriber %d has been closed", ref)
	}(ref)

	return depth, ref
}
