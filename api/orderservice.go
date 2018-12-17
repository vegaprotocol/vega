package api

import (
	"context"
	"github.com/pkg/errors"
	"time"
	"vega/blockchain"
	"vega/core"
	"vega/datastore"
	"vega/filters"
	"vega/log"
	"vega/msg"
	"fmt"
)

type OrderService interface {
	Init(vega *core.Vega, orderStore datastore.OrderStore)
	Stop()
	ObserveOrders(ctx context.Context, market *string, party *string) (orders <-chan []msg.Order, ref uint64)

	CreateOrder(ctx context.Context, order *msg.Order) (success bool, orderReference string, err error)
	CancelOrder(ctx context.Context, order *msg.Order) (success bool, err error)
	AmendOrder(ctx context.Context, amendment *msg.Amendment) (success bool, err error)

	GetByMarket(ctx context.Context, market string, filters *filters.OrderQueryFilters) (orders []*msg.Order, err error)
	GetByParty(ctx context.Context, party string, filters *filters.OrderQueryFilters) (orders []*msg.Order, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (order *msg.Order, err error)
	GetByPartyAndId(ctx context.Context, market string, id string) (order *msg.Order, err error)

	//GetMarkets(ctx context.Context) ([]string, error)
	GetMarketDepth(ctx context.Context, market string) (marketDepth *msg.MarketDepth, err error)
	ObserveMarketDepth(ctx context.Context, market string) (depth <-chan msg.MarketDepth, ref uint64)

	GetStatistics(ctx context.Context) (*msg.Statistics, error)
	GetCurrentTime(ctx context.Context) (time.Time, error)
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
	//dataDir := "./orderStore"
	//p.orderStore = datastore.NewOrderStore(dataDir)
	p.orderStore = orderStore
	p.blockchain = blockchain.NewClient()
}

func (p *orderService) Stop() {
	p.orderStore.Close()
}

func (p *orderService) CreateOrder(ctx context.Context, order *msg.Order) (success bool, orderReference string, err error) {
	// Set defaults, prevent unwanted external manipulation
	order.Remaining = order.Size
	order.Status = msg.Order_Active
	order.Timestamp = 0
	order.Reference = ""

	// if order is GTT convert datetime to blockchain timestamp
	if order.Type == msg.Order_GTT {
		expirationDateTime, err := time.Parse(time.RFC3339, order.ExpirationDatetime)
		if err != nil {
			return false, "", errors.New("invalid expiration datetime format")
		}

		expirationTimestamp := expirationDateTime.UnixNano()
		if expirationTimestamp <= p.app.State.Timestamp  {
			return false, "", errors.New("invalid expiration datetime error")
		}
		order.ExpirationTimestamp = uint64(expirationTimestamp)
	}

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
	return p.blockchain.CancelOrder(ctx, o)
}

func (p *orderService) AmendOrder(ctx context.Context, amendment *msg.Amendment) (success bool, err error) {

	// Validate order exists using read store
	o, err := p.orderStore.GetByPartyAndId(amendment.Party, amendment.Id)
	if err != nil {
		return false, err
	}

	if o.Status != msg.Order_Active {
		return false, errors.New("order is not active")
	}

	// if order is GTT convert datetime to block chain timestamp
	if amendment.ExpirationDatetime != "" {
		expirationDateTime, err := time.Parse(time.RFC3339, amendment.ExpirationDatetime)
		if err != nil {
			return false, errors.New("invalid format expiration datetime")
		}
		if expirationDateTime.Before(p.app.State.Datetime) || expirationDateTime.Equal(p.app.State.Datetime) {
			return false, errors.New("invalid expiration datetime")
		}
		amendment.ExpirationTimestamp = uint64(expirationDateTime.UnixNano())
	}

	// Send edit request by consensus
	return p.blockchain.AmendOrder(ctx, amendment)
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
		result = append(result, order)
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
		result = append(result, order)
	}
	return result, err
}

func (p *orderService) GetByMarketAndId(ctx context.Context, market string, id string) (order *msg.Order, err error) {
	o, err := p.orderStore.GetByMarketAndId(market, id)
	if err != nil {
		return &msg.Order{}, err
	}
	return o, err
}

func (p *orderService) GetByPartyAndId(ctx context.Context, market string, id string) (order *msg.Order, err error) {
	o, err := p.orderStore.GetByPartyAndId(market, id)
	if err != nil {
		return &msg.Order{}, err
	}
	return o, err
}

func (p *orderService) GetMarketDepth(ctx context.Context, marketName string) (orderBookDepth *msg.MarketDepth, err error) {
	return p.orderStore.GetMarketDepth(marketName)
}

func (p *orderService) ObserveOrders(ctx context.Context, market *string, party *string) (<-chan []msg.Order, uint64) {
	orders := make(chan []msg.Order)
	internal := make(chan []msg.Order)
	ref := p.orderStore.Subscribe(internal)

	go func(id uint64, internal chan []msg.Order) {
		<-ctx.Done()
		log.Debugf("OrderService -> Subscriber closed connection: %d", id)
		err := p.orderStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing when context.Done() on OrderService for id: %d", id)
		}
	}(ref, internal)

	go func(id uint64) {

		var validatedOrders []msg.Order

		// read internal channel
		for v := range internal {

			// reset temp slice
			validatedOrders = nil
			for _, item := range v {
				if market != nil && item.Market != *market {
					continue
				}
				if party != nil && item.Party != *party {
					continue
				}
				validatedOrders = append(validatedOrders, item)
			}
			orders <- validatedOrders
		}
		log.Debugf("OrderService -> Channel for subscriber %d has been closed", ref)
	}(ref)

	return orders, ref
}

func (p *orderService) ObserveMarketDepth(ctx context.Context, market string) (<-chan msg.MarketDepth, uint64) {
	depth := make(chan msg.MarketDepth)
	internal := make(chan []msg.Order)
	ref := p.orderStore.Subscribe(internal)

	go func(id uint64, internal chan []msg.Order) {
		<-ctx.Done()
		log.Debugf("OrderService -> Depth closed connection: %d", id)
		err := p.orderStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing depth when context.Done() on OrderService for id: %d", id)
		}
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

func (p *orderService) GetStatistics(ctx context.Context) (*msg.Statistics, error) {
	refused := "dial tcp 127.0.0.1:46657: connect: connection refused"
	rpcErr := "Statistics: block-chain rpc client error [%s] %v"

	p.app.Statistics.CurrentTime = time.Now().UTC().Format(time.RFC3339)
	p.app.Statistics.GenesisTime = p.app.GetGenesisTime().Format(time.RFC3339)

	p.app.Statistics.VegaTime = fmt.Sprintf("%s [%d]", p.app.State.Datetime.Format(time.RFC3339), p.app.State.Timestamp)
	p.app.Statistics.BlockHeight = uint64(p.app.State.Height)

	totalParties := len(p.app.Statistics.Parties)
	p.app.Statistics.TotalParties = uint64(totalParties)

	// Unconfirmed TX count == current transaction backlog length
	backlogLength, err := p.blockchain.GetUnconfirmedTxCount(ctx)
	if err != nil {
		if err.Error() == refused {
			return p.app.Statistics, nil
		}
		log.Errorf(rpcErr, "unconfirmed-tx-count", err)
		return p.app.Statistics, err
	}
	p.app.Statistics.BacklogLength = uint64(backlogLength)

	// Net info provides peer stats etc (block chain network info)
	netInfo, err := p.blockchain.GetNetworkInfo(ctx)
	if err != nil {
		if err.Error() == refused {
			return p.app.Statistics, nil
		}
		log.Errorf(rpcErr, "net-info", err)
		return p.app.Statistics, err
	}
	p.app.Statistics.TotalPeers = uint64(netInfo.NPeers)


	return p.app.Statistics, nil
}

func (p *orderService) GetCurrentTime(ctx context.Context) (time.Time, error) {
	return p.app.State.Datetime, nil
}