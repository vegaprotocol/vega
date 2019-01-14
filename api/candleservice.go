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
}

func (c *candleService) Stop() {
	c.candleStore.Close()
}

func (c *candleService) ObserveCandles(ctx context.Context, market *string, interval *msg.Interval) (<-chan msg.Candle, uint64) {
	candleCh := make(chan msg.Candle)
	iT := datastore.InternalTransport{Market: *market, Interval: *interval, Transport: make(chan msg.Candle)}
	ref := c.candleStore.Subscribe(&iT)

	go func(id uint64, ctx context.Context) {
		ip := ipAddressFromContext(ctx)
		<-ctx.Done()
		log.Debugf("CandleService -> Subscriber closed connection: %d [%s]", id, ip)
		err := c.candleStore.Unsubscribe(id)
		if err != nil {
			log.Errorf("Error un-subscribing when context.Done() on CandleService for id: %d [%s]", id, ip)
		}
	}(ref, ctx)

	go func(iT *datastore.InternalTransport, ctx context.Context) {
		ip := ipAddressFromContext(ctx)
		for v := range iT.Transport {
			select {
				case candleCh <- v:
					log.Debugf("CandleService -> Candles for subscriber %d [%s] sent successfully", ref, ip)
				default:
					log.Debugf("CandleService -> Candles for subscriber %d [%s] not sent", ref, ip)
			}
		}
		log.Debugf("CandleService -> Channel for subscriber %d has been closed [%s]", ref, ip)
	}(&iT, ctx)

	return candleCh, ref
}

func (c *candleService) GetCandles(ctx context.Context, market string,
	sinceTimestamp uint64, interval msg.Interval) (candles []*msg.Candle, err error) {

	// sinceTimestamp must be valid and not older than market genesis timestamp,

	// interval check if from range of valid intervals

	return c.candleStore.GetCandles(market, sinceTimestamp, interval)
}
