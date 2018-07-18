package api

import (
	"context"
	"errors"
	"time"

	"vega/core"
	"vega/datastore"
	"vega/proto"
)

type TradeService interface {
	Init(app *core.Vega, tradeStore datastore.TradeStore)
	GetTradesByMarket(ctx context.Context, market string, limit uint64) (trades []*msg.Trade, err error)
	GetTradesByParty(ctx context.Context, party string, limit uint64) (trades []*msg.Trade, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (trade *msg.Trade, err error)
	GetByPartyAndId(ctx context.Context, party string, id string) (trade *msg.Trade, err error)
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

func (t *tradeService) GetTradesByMarket(ctx context.Context, market string, limit uint64) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByMarket(market, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]*msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetTradesByParty(ctx context.Context, party string, limit uint64) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByParty(party, datastore.GetParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]*msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetByMarketAndId(ctx context.Context, market string, id string) (trade *msg.Trade, err error) {
	tr, err := t.tradeStore.GetByMarketAndId(market, id)
	if err != nil {
		return &msg.Trade{}, err
	}
	return tr.ToProtoMessage(), err
}

func (t *tradeService) GetByPartyAndId(ctx context.Context, party string, id string) (trade *msg.Trade, err error) {
	tr, err := t.tradeStore.GetByPartyAndId(party, id)
	if err != nil {
		return &msg.Trade{}, err
	}
	return tr.ToProtoMessage(), err
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


