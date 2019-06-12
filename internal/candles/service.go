package candles

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/candles CandleStore
type CandleStore interface {
	Subscribe(iT *storage.InternalTransport) uint64
	Unsubscribe(id uint64) error
	GetCandles(ctx context.Context, market string, since time.Time, interval types.Interval) ([]*types.Candle, error)
}

type Svc struct {
	log          *logging.Logger
	Config       storcfg.CandlesConfig
	tradesBuffer map[string][]*types.Trade
	candleStore  CandleStore
}

func NewService(log *logging.Logger, config storcfg.CandlesConfig, candleStore CandleStore) (*Svc, error) {
	if candleStore == nil {
		return nil, errors.New("candleStore instance is nil when creating candle service instance.")
	}

	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		log:         log,
		Config:      config,
		candleStore: candleStore,
	}, nil
}

func (s *Svc) ReloadConf(cfg storcfg.CandlesConfig) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.Config = cfg
}

func (c *Svc) ObserveCandles(ctx context.Context, retries int, market *string, interval *types.Interval) (<-chan *types.Candle, uint64) {
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

func (c *Svc) GetCandles(ctx context.Context, market string,
	since time.Time, interval types.Interval) (candles []*types.Candle, err error) {

	// sinceTimestamp must be valid and not older than market genesis timestamp
	// interval check if from range of valid intervals

	return c.candleStore.GetCandles(ctx, market, since, interval)
}
