package candles

import (
	"context"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

type Service interface {
	ObserveCandles(ctx context.Context, retries int, market *string, interval *types.Interval) (candleCh <-chan *types.Candle, ref uint64)
	GetCandles(ctx context.Context, market string, sinceTimestamp uint64, interval types.Interval) (candles []*types.Candle, err error)
}

type candleService struct {
	*Config
	tradesBuffer map[string][]*types.Trade
	candleStore  storage.CandleStore
}

func NewCandleService(config *Config, candleStore storage.CandleStore) (Service, error) {
	if config == nil {
		return nil, errors.New("candle config is nil when creating candle service instance.")
	}
	if candleStore == nil {
		return nil, errors.New("candleStore instance is nil when creating candle service instance.")
	}
	return &candleService{
		Config:      config,
		candleStore: candleStore,
	}, nil
}

func (c *candleService) ObserveCandles(ctx context.Context, retries int, market *string, interval *types.Interval) (<-chan *types.Candle, uint64) {
	candleCh := make(chan *types.Candle)
	iT := storage.InternalTransport{
		Market:    *market,
		Interval:  *interval,
		Transport: make(chan *types.Candle),
	}
	ref := c.candleStore.Subscribe(&iT)
	retryCount := retries

	go func() {
		ctx, cfunc := context.WithCancel(ctx)
		defer cfunc()
		ip := logging.IPAddressFromContext(ctx)
		for {
			select {
			case <-ctx.Done():
				c.log.Debug(
					"Candles subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				if err := c.candleStore.Unsubscribe(ref); err != nil {
					c.log.Error(
						"Failure un-subscribing candles subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(iT.Transport)
				close(candleCh)
				return
			case v := <-iT.Transport:
				select {
				case candleCh <- v:
					c.log.Debug(
						"Candles for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
					)
					// reset retry counter
					retryCount = retries
				default:
					retryCount--
					// no retries left?
					if retryCount == 0 {
						c.log.Warn(
							"Candles subscriber ran out of retries",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
							logging.Int("retries", retries),
						)
						cfunc()
					}
					c.log.Debug(
						"Candles for subscriber not sent",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
					)
				}
			}
		}
	}()

	return candleCh, ref
}

func (c *candleService) GetCandles(ctx context.Context, market string,
	sinceTimestamp uint64, interval types.Interval) (candles []*types.Candle, err error) {

	// sinceTimestamp must be valid and not older than market genesis timestamp
	// interval check if from range of valid intervals

	return c.candleStore.GetCandles(ctx, market, sinceTimestamp, interval)
}
