package candles

import (
	"context"

	types "vega/proto"

	"vega/internal/storage"
	"vega/internal/logging"
)

type Service interface {
	ObserveCandles(ctx context.Context, market *string, interval *types.Interval) (candleCh <-chan types.Candle, ref uint64)
	GetCandles(ctx context.Context, market string, sinceTimestamp uint64, interval types.Interval) (candles []*types.Candle, err error)
}

type candleService struct {
	*Config
	tradesBuffer map[string][]*types.Trade
	candleStore  storage.CandleStore
}

func NewCandleService(config *Config, candleStore storage.CandleStore) Service {
	return &candleService{
		Config: config,
		candleStore: candleStore,
	}
}

func (c *candleService) ObserveCandles(ctx context.Context, market *string, interval *types.Interval) (<-chan types.Candle, uint64) {
	candleCh := make(chan types.Candle)
	iT := storage.InternalTransport{Market: *market, Interval: *interval, Transport: make(chan types.Candle)}
	ref := c.candleStore.Subscribe(&iT)

	go func(id uint64, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		<-ctx.Done()
		c.log.Debugf("CandleService -> Subscriber closed connection: %d [%s]", id, ip)
		err := c.candleStore.Unsubscribe(id)
		if err != nil {
			c.log.Errorf("Error un-subscribing when context.Done() on CandleService for id: %d [%s]", id, ip)
		}
	}(ref, ctx)

	go func(iT *storage.InternalTransport, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		for v := range iT.Transport {
			select {
				case candleCh <- v:
					c.log.Debugf("CandleService -> Candles for subscriber %d [%s] sent successfully", ref, ip)
				default:
					c.log.Debugf("CandleService -> Candles for subscriber %d [%s] not sent", ref, ip)
			}
		}
		c.log.Debugf("CandleService -> Channel for subscriber %d has been closed [%s]", ref, ip)
	}(&iT, ctx)

	return candleCh, ref
}

func (c *candleService) GetCandles(ctx context.Context, market string,
	sinceTimestamp uint64, interval types.Interval) (candles []*types.Candle, err error) {

	// sinceTimestamp must be valid and not older than market genesis timestamp,

	// interval check if from range of valid intervals

	return c.candleStore.GetCandles(market, sinceTimestamp, interval)
}
