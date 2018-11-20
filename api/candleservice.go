package api

import (
	"context"
	"vega/core"
	"vega/datastore"
	"vega/log"
	"vega/msg"
)

type CandleService interface {
	Init(app *core.Vega, candleStore datastore.CandleStore)
	Stop()
	//AddTrade(trade *msg.Trade)
	//Generate(ctx context.Context, market string) error
	ObserveCandles(ctx context.Context, market *string, interval *msg.Interval) (candleCh <-chan msg.Candle, ref uint64)
	GetCandles(ctx context.Context, market string, sinceTimestamp uint64, interval msg.Interval) (candles []*msg.Candle, err error)
}

type candleService struct {
	app          *core.Vega
	tradesBuffer map[string][]*msg.Trade
	candleStore  datastore.CandleStore
}

func NewCandleService() CandleService {
	return &candleService{}
}

func (c *candleService) Init(app *core.Vega, candleStore datastore.CandleStore) {
	c.app = app
	c.candleStore = candleStore
//	c.tradesBuffer = make(map[string][]*msg.Trade, 0)
}

func (c *candleService) Stop() {
	c.candleStore.Close()
}

//func (c *candleService) AddTrade(trade *msg.Trade) {
//	c.tradesBuffer[trade.Market] = append(c.tradesBuffer[trade.Market], trade)
//}
//
//// this should act as a separate slow go routine triggered after block is committed
//func (c *candleService) Generate(ctx context.Context, market string) error {
//	if _, ok := c.tradesBuffer[market]; !ok {
//		return errors.New("Market not found")
//	}
//
//	// in case there is no trading activity on this market, generate empty candles based on historical values
//	if len(c.tradesBuffer) == 0 {
//		if err := c.candleStore.GenerateEmptyCandles(market, c.app.GetCurrentTimestamp()); err != nil {
//			return err
//		}
//		return nil
//	}
//
//	// generate/update  candles for each trade in the buffer
//	for idx := range c.tradesBuffer[market] {
//		if err := c.candleStore.GenerateCandles(c.tradesBuffer[market][idx]); err != nil {
//			return err
//		}
//	}
//
//	// Notify all subscribers
//	c.candleStore.Notify()
//
//	// Flush the buffer
//	c.tradesBuffer[market] = nil
//
//	return nil
//}

func (c *candleService) ObserveCandles(ctx context.Context, market *string, interval *msg.Interval) (<-chan msg.Candle, uint64) {
	candleCh := make(chan msg.Candle)
	internalTransport := make(map[msg.Interval]chan msg.Candle, 0)
	ref := c.candleStore.Subscribe(internalTransport)

	go func(id uint64) {
		<-ctx.Done()
		log.Debugf("CandleService -> Subscriber closed connection: %d", id)
		err := c.candleStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing when context.Done() on CandleService for id: %d", id)
		}
	}(ref)

	go func(internalTransport map[msg.Interval]chan msg.Candle) {
		var tempCandle msg.Candle
		for v := range internalTransport[*interval] {
			tempCandle = v
			candleCh <- tempCandle
		}
		log.Debugf("CandleService -> Channel for subscriber %d has been closed", ref)
	}(internalTransport)

	return candleCh, ref
}

func (c *candleService) GetCandles(ctx context.Context, market string,
	sinceTimestamp uint64, interval msg.Interval) (candles []*msg.Candle, err error) {
	// sinceTimestamp must be valid and not older than market genesis timestamp,

	// interval check if from range of valid intervals

	return c.candleStore.GetCandles(market, sinceTimestamp, interval), nil
}
