package epochtime

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/vegatime"
)

type Broker interface {
	Send(e events.Event)
}

// Svc represents the Service managing epoch inside Vega.
type Svc struct {
	config Config

	epoch types.Epoch

	netparams *netparams.Store

	listeners []func(context.Context, types.Epoch)
	mu        sync.Mutex

	log *logging.Logger

	broker Broker
}

// New instantiates a new epochtime service
func NewService(l *logging.Logger, conf Config, vt *vegatime.Svc, params *netparams.Store, broker Broker) *Svc {
	s := &Svc{config: conf,
		netparams: params,
		log:       l,
		broker:    broker}

	// Subscribe to the vegatime onblocktime event
	vt.NotifyOnTick(s.onTick)

	return s
}

// ReloadConf reload the configuration for the epochtime service
func (s *Svc) ReloadConf(conf Config) {
	// do nothing here, conf is not used for now
}

func (s *Svc) getEpochLength() (*time.Duration, error) {
	// Get the epoch length from the network params
	length, err := s.netparams.Get(netparams.ValidatorsEpochLength)

	if err != nil {
		return nil, err
	}

	// Convert string to time
	d, err := time.ParseDuration(length)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Svc) onTick(ctx context.Context, t time.Time) {
	if t.IsZero() {
		// We haven't got a block time yet, ignore
		return
	}

	if s.epoch.StartTime.IsZero() {
		// First block so let's create our first epoch
		s.epoch.Seq = 0
		s.epoch.StartTime = t

		d, err := s.getEpochLength()
		if err != nil {
			// Something bad has happened, we should stop
			s.log.Panic("Unable to get the epoch length", logging.Error(err))
		}
		s.epoch.ExpireTime = t.Add(*d) // current time + epoch length

		// Send out new epoch event
		s.notify(ctx, s.epoch)
	}

	if s.epoch.ExpireTime.Before(t) {
		s.epoch.EndTime = t
		// We have expired, send event
		s.notify(ctx, s.epoch)

		s.epoch.Seq += 1

		// Create a new epoch
		s.epoch.StartTime = t

		d, err := s.getEpochLength()
		if err != nil {
			// Something bad has happened, we should stop
			s.log.Panic("Unable to get the epoch length", logging.Error(err))
		}
		s.epoch.ExpireTime = t.Add(*d) // + epoch length
		s.epoch.EndTime = time.Time{}
		s.notify(ctx, s.epoch)
	}
}

// NotifyOnEpoch allows other services to register a callback function
// which will be called once we enter a new epoch
func (s *Svc) NotifyOnEpoch(f func(context.Context, types.Epoch)) {
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, f)
}

func (s *Svc) notify(ctx context.Context, e types.Epoch) {
	// Push this updated epoch message onto the event bus
	s.broker.Send(events.NewEpochEvent(ctx, &e))
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.listeners {
		f(ctx, e)
	}
}
