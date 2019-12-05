package risk

import (
	"context"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

// RiskStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_store_mock.go -package mocks code.vegaprotocol.io/vega/risk RiskStore
type RiskStore interface {
	GetMarginLevelsByID(partyID string, marketID string) ([]types.MarginLevels, error)
	Subscribe(c chan []types.MarginLevels) uint64
	Unsubscribe(id uint64) error
}

// Svc represent the market service
type Svc struct {
	Config
	log           *logging.Logger
	store         RiskStore
	subscriberCnt int32
}

func NewService(log *logging.Logger, c Config, store RiskStore) *Svc {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	return &Svc{
		Config: c,
		log:    log,
		store:  store,
	}
}

// ReloadConf update the market service internal configuration
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

// GetMarginLevelsByID returns the margin levels for a given party
func (s *Svc) GetMarginLevelsByID(partyID, marketID string) ([]types.MarginLevels, error) {
	return s.store.GetMarginLevelsByID(partyID, marketID)
}

func (s *Svc) ObserveMarginLevels(
	ctx context.Context, retries int, partyID, marketID string,
) (accountCh <-chan []types.MarginLevels, ref uint64) {

	margins := make(chan []types.MarginLevels)
	internal := make(chan []types.MarginLevels)
	ref = s.store.Subscribe(internal)

	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	go func() {
		atomic.AddInt32(&s.subscriberCnt, 1)
		defer atomic.AddInt32(&s.subscriberCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug(
					"Risks subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				// this error only happens when the subscriber reference doesn't exist
				// so we can still safely close the channels
				if err := s.store.Unsubscribe(ref); err != nil {
					s.log.Error(
						"Failure un-subscribing accounts subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(margins)
				return
			case accs := <-internal:
				filtered := make([]types.MarginLevels, 0, len(accs))
				for _, acc := range accs {
					acc := acc
					if (len(marketID) <= 0 || marketID == acc.MarketID) &&
						partyID == acc.PartyID {
						filtered = append(filtered, acc)
					}
				}
				retryCount := retries
				success := false
				for !success && retryCount >= 0 {
					select {
					case margins <- filtered:
						retryCount = retries
						s.log.Debug(
							"Risks for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
						success = true
					default:
						retryCount--
						if retryCount > 0 {
							s.log.Debug(
								"Risks for subscriber not sent",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip))
						}
						time.Sleep(time.Duration(10) * time.Millisecond)
					}
				}
				if !success && retryCount <= 0 {
					s.log.Warn(
						"Risk subscriber has hit the retry limit",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Int("retries", retries))
					cancel()
				}

			}
		}
	}()

	return margins, ref

}

// GetMarginLevelsSubscribersCount returns the total number of active subscribers for ObserveMarginLevels.
func (s *Svc) GetMarginLevelsSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscriberCnt)
}
