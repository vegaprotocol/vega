package api

import (
	"context"
	"errors"
	"time"

	"vega/core"
	"vega/datastore"
	"vega/msg"
	"vega/log"
)

type TradeService interface {
	Init(app *core.Vega, tradeStore datastore.TradeStore)
	ObserveTrades(ctx context.Context) (orders <-chan msg.Trade, ref uint64)

	GetByMarket(ctx context.Context, market string, limit uint64) (trades []*msg.Trade, err error)
	GetByParty(ctx context.Context, party string, limit uint64) (trades []*msg.Trade, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (trade *msg.Trade, err error)
	GetByPartyAndId(ctx context.Context, party string, id string) (trade *msg.Trade, err error)

	GetCandles(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error)
	ObserveCandles(ctx context.Context, market string, interval uint64) (candle <-chan msg.Candles, ref uint64)

	GetPositionsByParty(ctx context.Context, party string) (positions []*msg.MarketPosition, err error)
	ObservePositions(ctx context.Context, party string) (positions <-chan msg.MarketPosition, ref uint64)
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

func (t *tradeService) GetByMarket(ctx context.Context, market string, limit uint64) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByMarket(market, datastore.GetTradeParams{Limit: limit})
	if err != nil {
		return nil, err
	}
	tradeMsgs := make([]*msg.Trade, 0)
	for _, trade := range tr {
		tradeMsgs = append(tradeMsgs, trade.ToProtoMessage())
	}
	return tradeMsgs, err
}

func (t *tradeService) GetByParty(ctx context.Context, party string, limit uint64) (trades []*msg.Trade, err error) {
	tr, err := t.tradeStore.GetByParty(party, datastore.GetTradeParams{Limit: limit})
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

func (t *tradeService) GetCandles(ctx context.Context, market string, since time.Time, interval uint64) (candles msg.Candles, err error) {
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

func (t *tradeService) GetPositionsByParty(ctx context.Context, party string) (positions []*msg.MarketPosition, err error) {
	mapOfMarketPositions := t.tradeStore.GetPositionsByParty(party)
	for _, marketPositions := range mapOfMarketPositions {
		positions = append(positions, marketPositions)
	}
	return positions, nil
}

func (t *tradeService) ObserveTrades(ctx context.Context) (<-chan msg.Trade, uint64) {
	trades := make(chan msg.Trade)
	internal := make(chan []datastore.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []datastore.Trade) {
		<-ctx.Done()
		log.Debugf("TradeService -> Subscriber closed connection: %d", id)
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing when context.Done() on TradeService for id: %d", id)
		}
		close(internal)
	}(ref, internal)

	go func(id uint64) {
		for v := range internal {
			for _, item := range v {
				trades <- *item.ToProtoMessage()
			}
		}
		log.Debugf("TradeService -> Channel for subscriber %d has been closed", ref)
	}(ref)

	return trades, ref
}

func (t *tradeService) ObservePositions(ctx context.Context, party string) (<-chan msg.MarketPosition, uint64) {
	positions := make(chan msg.MarketPosition)
	internal := make(chan []datastore.Trade)
	ref := t.tradeStore.Subscribe(internal)

	go func(id uint64, internal chan []datastore.Trade) {
		<-ctx.Done()
		log.Debugf("TradeService -> Positions subscriber closed connection: %d", id)
		err := t.tradeStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing positions when context.Done() on TradeService for id: %d", id)
		}
		close(internal)
	}(ref, internal)

	go func(id uint64) {
		for range internal {
			mapOfMarketPositions := t.tradeStore.GetPositionsByParty(party)
			for _, marketPositions := range mapOfMarketPositions {
				positions <- *marketPositions
			}
		}
		log.Debugf("TradeService -> Channel for positions subscriber %d has been closed", ref)
	}(ref)

	return positions, ref
}

func (t *tradeService) ObserveCandles(ctx context.Context, market string, interval uint64) (candle <-chan msg.Candles, ref uint64) {

	log.Fatalf("ObserveCandles not implemented yet, not sure of best approach :(")

	return nil, 0
}