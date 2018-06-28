package trades

import (
	"vega/datastore"
	"vega/proto"
	"context"
)

type TradeService interface {
	Init(tradeStore datastore.TradeStore)
	GetTrades(c context.Context, market string) (trades []msg.Trade, err error)
	GetTradesForOrder(c context.Context, market string, orderID string) (trades []msg.Trade, err error)
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

func(t *tradeService) GetTrades(ctx context.Context, market string) (trades []msg.Trade, err error) {
	tr, err := t.tradeStore.All(market)
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, *trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func(t *tradeService) GetTradesForOrder(ctx context.Context, market string, orderID string) (trades []msg.Trade, err error) {
	tr, err := t.tradeStore.FindByOrderID(market, orderID)
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, *trade.ToProtoMessage())
	}
	return tradeMsgs, err
}
