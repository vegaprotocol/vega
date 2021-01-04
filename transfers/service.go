package transfers

import (
	"context"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

// TransferResponseStore represent an abstraction over a transfer response storage
//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_response_store_mock.go -package mocks code.vegaprotocol.io/vega/transfers TransferResponseStore
type TransferResponseStore interface {
	Subscribe(c chan []*types.TransferResponse) uint64
	Unsubscribe(id uint64) error
}

// Svc is the service handling all the transfer responses (leger movement)
type Svc struct {
	Config
	log           *logging.Logger
	store         TransferResponseStore
	subscriberCnt int32
}

// NewService retunrs a new instance of the transfer response service
func NewService(log *logging.Logger, cfg Config, store TransferResponseStore) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	return &Svc{
		Config: cfg,
		log:    log,
		store:  store,
	}
}

// ReloadConf reloads configuration for this package from toml config files.
func (s *Svc) ReloadConf(cfg Config) {
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

// ObserveTransferResponses start a new goroutine and return a channels
// allowing the caller to listen to all new TransferResponse created by the
// core
func (s *Svc) ObserveTransferResponses(
	ctx context.Context, retries int,
) (<-chan []*types.TransferResponse, uint64) {

	transfers := make(chan []*types.TransferResponse)
	internal := make(chan []*types.TransferResponse)
	ref := s.store.Subscribe(internal)

	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	retryCount := retries
	go func() {
		atomic.AddInt32(&s.subscriberCnt, 1)
		defer atomic.AddInt32(&s.subscriberCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug(
					"TransferResponses subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				// this error only happens when the subscriber reference doesn't exist
				// so we can still safely close the channels
				if err := s.store.Unsubscribe(ref); err != nil {
					s.log.Error(
						"Failure un-subscribing transfer responses subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(transfers)
				return
			case tmptrs := <-internal:
				retryCount = retries
				success := false
				for !success && retryCount > 0 {
					select {
					case transfers <- tmptrs:
						s.log.Debug(
							"TransferResponses for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
						success = true
					default:
						retryCount--
						if retryCount > 0 {
							s.log.Debug(
								"TransferResponses for subscriber not sent",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip))
							time.Sleep(time.Duration(10) * time.Millisecond)
						}
					}
				}
				if retryCount <= 0 {
					s.log.Warn(
						"TransferResponses subscriber has hit the retry limit",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Int("retries", retries))
					cancel()
				}
			}
		}
	}()

	return transfers, ref
}

// GetTransferResponsesSubscribersCount return the number of subscribers to the
// transfer responses updates.
func (s *Svc) GetTransferResponsesSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscriberCnt)
}
