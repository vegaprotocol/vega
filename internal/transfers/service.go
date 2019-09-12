package transfers

import (
	"context"
	"sync/atomic"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_response_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/transfers TransferResponseStore
type TransferResponseStore interface {
	Subscribe(c chan []*types.TransferResponse) uint64
	Unsubscribe(id uint64) error
}

type Svc struct {
	Config
	log           *logging.Logger
	store         TransferResponseStore
	subscriberCnt int32
}

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

func (s *Svc) ObserveTransferResponses(
	ctx context.Context, retries int,
) (<-chan []*types.TransferResponse, uint64) {

	transfers := make(chan []*types.TransferResponse)
	internal := make(chan []*types.TransferResponse)
	ref := s.store.Subscribe(internal)

	retryCount := retries
	go func() {
		atomic.AddInt32(&s.subscriberCnt, 1)
		defer atomic.AddInt32(&s.subscriberCnt, -1)
		ip := logging.IPAddressFromContext(ctx)
		ctx, cancel := context.WithCancel(ctx)
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
						"Failure un-subscribing transfer reponses subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(transfers)
				return
			case tmptrs := <-internal:
				select {
				case transfers <- tmptrs:
					retryCount = retries
					s.log.Debug(
						"TransferResponses for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
					)
				default:
					retryCount--
					if retryCount == 0 {
						s.log.Warn(
							"TransferResponses subscriber has hit the retry limit",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
							logging.Int("retries", retries))

						cancel()
					}
					s.log.Debug(
						"TransferResponses for subscriber not sent",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				}
			}
		}
	}()

	return transfers, ref
}

func (s *Svc) GetTransferResponsesSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscriberCnt)
}
