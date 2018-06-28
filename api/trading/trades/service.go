package trades

import (
	"vega/datastore"
	"vega/proto"
)

type TradeService interface {
	Init(tradeStore datastore.TradeStore)
	GetTrades(market string) (trades []msg.Trade, err error)
	GetTradesForOrder(market string, orderID string) (trades []msg.Trade, err error)
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

func(t *tradeService) GetTrades(market string) (trades []msg.Trade, err error) {
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

func(t *tradeService) GetTradesForOrder(market string, orderID string) (trades []msg.Trade, err error) {
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
