package trades

import (
	"context"
	"vega/datastore"
	"vega/proto"
)

type TradeService interface {
	Init(tradeStore datastore.TradeStore)
	GetTrades(ctx context.Context, market string, limit uint64) (trades []msg.Trade, err error)
	GetTradesForOrder(ctx context.Context, market string, orderID string, limit uint64) (trades []msg.Trade, err error)
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
	tr, err := t.tradeStore.GetAll(market, datastore.NewLimitMax())
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
	tr, err := t.tradeStore.GetByOrderId(market, orderId, datastore.NewLimitMax())
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, *trade.ToProtoMessage())
	}
	return tradeMsgs, err
}
