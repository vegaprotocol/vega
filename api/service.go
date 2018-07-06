package api


import (
	"context"
	"time"
	"net/http"
	"vega/datastore"
	"vega/proto"
)


type TradeService interface {
	Init(tradeStore datastore.TradeStore)
	GetById(ctx context.Context, market string, id string) (trade msg.Trade, err error)
	GetTrades(ctx context.Context, market string, limit uint64) (trades []msg.Trade, err error)
	GetTradesForOrder(ctx context.Context, market string, orderId string, limit uint64) (trades []msg.Trade, err error)
}

type tradeService struct {
	tradeStore datastore.TradeStore
}

func NewTradeService() TradeService {
	return &tradeService{}
}

func (t *tradeService) Init(tradeStore datastore.TradeStore) {
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

type OrderService interface {
	Init(orderStore datastore.OrderStore)
	GetById(ctx context.Context, market string, id string) (order msg.Order, err error)
	CreateOrder(ctx context.Context, order msg.Order) (success bool, err error)
	GetOrders(ctx context.Context, market string, party string, limit uint64) (orders []msg.Order, err error)
}

type orderService struct {
	orderStore datastore.OrderStore
}

func NewOrderService() OrderService {
	return &orderService{}
}

func (p *orderService) Init(orderStore datastore.OrderStore) {
	p.orderStore = orderStore
}

func (p *orderService) CreateOrder(ctx context.Context, order msg.Order) (success bool, err error) {

	utcNow := time.Now().UTC()
	order.Timestamp = unixTimestamp(utcNow)
	order.Remaining = order.Size

	payload, err := jsonWithEncoding(order)
	if err != nil {
		return false, err
	}

	reqUrl := "http://localhost:46657/broadcast_tx_async?tx=%22" + newGuid() + "|" + payload + "%22"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(reqUrl)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// For debugging only
	// body, err := ioutil.ReadAll(resp.Body)
	//if err == nil {
	//	fmt.Println(string(body))
	//}

	return true, err
}

func (p *orderService) GetOrders(ctx context.Context, market string, party string, limit uint64) (orders []msg.Order, err error) {
	o, err := p.orderStore.GetAll(market, party, datastore.GetParams{ Limit: limit })
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

