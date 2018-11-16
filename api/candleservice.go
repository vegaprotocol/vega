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

type CandleService interface {
	Init(app *core.Vega, candleStore datastore.CandleStore)
	Stop()
	AddTrade(trade *msg.Trade)
	Generate(ctx context.Context, market string) error
	ObserveCandles(ctx context.Context, market *string, party *string, interval *string) (candleCh <-chan msg.Candle, ref uint64)
	GetCandles(ctx context.Context, market string, since time.Time, interval string) (candles []*msg.Candle, err error)
}

type candleService struct {
	app        *core.Vega
	tradesBuffer map[string][]*msg.Trade
	candleStore datastore.CandleStore
}

func NewCandleService() CandleService {
	return &candleService{}
}

func (c *candleService) Init(app *core.Vega, candleStore datastore.CandleStore) {
	c.app = app
	//dataDir := "./tradeStore"
	//t.candleStore = datastore.NewCandleStore(dataDir)
	c.candleStore = candleStore
	c.tradesBuffer = make(map[string][]*msg.Trade, 0)
}

func (c *candleService) Stop() {
	c.candleStore.Close()
}

func (c *candleService) AddTrade(trade *msg.Trade) {
	c.tradesBuffer[trade.Market] = append(c.tradesBuffer[trade.Market], trade)
}

// this should act as a separate slow go routine triggered after block is committed
func (c *candleService) Generate(ctx context.Context, market string) error {
	if _, ok := c.tradesBuffer[market]; !ok {
		return errors.New("Market not found")
	}

	// TODO: change to c.app.timestamp
	currentTime := uint64(time.Now().UnixNano())
	// in case there is no trading activity on this market, generate empty candles based on historical values
	if len(c.tradesBuffer) == 0 {
		if err := c.candleStore.GenerateEmptyCandles(market, uint64(currentTime)); err != nil {
			return err
		}
		return nil
	}

	// generate/update  candles for each trade in the buffer
	for idx := range c.tradesBuffer[market] {
		if err := c.candleStore.GenerateCandles(c.tradesBuffer[market][idx]); err != nil {
			return err
		}
	}

	// Flush the buffer
	c.tradesBuffer[market] = nil

	return nil
}

func (c *candleService) ObserveCandles(ctx context.Context, market *string, party *string, interval *string) (candleCh <-chan msg.Candle, ref uint64) {
	candleCh = make(chan msg.Candle)
	internalTransport := make(map[string]chan msg.Candle, 0)
	ref = c.candleStore.Subscribe(internalTransport)

	go func(id uint64) {
		<-ctx.Done()
		log.Debugf("CandleService -> Subscriber closed connection: %d", id)
		err := c.candleStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing when context.Done() on CandleService for id: %d", id)
		}
	}(ref)

	go func(internalTransport map[string]chan msg.Candle) {
		var tempCandle msg.Candle
		for v := range internalTransport[*interval] {
			tempCandle = v
			candleCh <- tempCandle
		}
		log.Debugf("CandleService -> Channel for subscriber %d has been closed", ref)
	}(internalTransport)


	return candleCh, ref
}

func (c *candleService) GetCandles(ctx context.Context, market string, sinceTimestamp time.Time, interval string) (candles []*msg.Candle, err error) {
	// sinceTimestamp must be valid and not older than market genesis timestamp,

	// interval check if from range of valid intervals

	return c.candleStore.GetCandles(market, uint64(sinceTimestamp.UnixNano()), interval), nil
}
