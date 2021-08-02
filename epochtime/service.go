package epochtime

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/vegatime"
)

type Epoch struct {
	// Unique identifier that increases by one each epoch
	UniqueID uint64
	// What time did this epoch start
	StartTime time.Time
	// What time should this epoch end
	ExpireTime time.Time
	// What time did it actually end
	EndTime time.Time
}

// Svc represents the Service managing epoch inside Vega.
type Svc struct {
	config Config

	epoch Epoch

	netparams *netparams.Store

	listeners []func(context.Context, Epoch)
	mu        sync.Mutex
}

// New instantiates a new epochtime service
func New(conf Config, vt *vegatime.Svc, params *netparams.Store) *Svc {

	s := &Svc{config: conf,
		netparams: params}

	// Subscribe to the vegatime onblocktime event
	vt.NotifyOnTick(s.onTick)

	return s
}

// ReloadConf reload the configuration for the epochtime service
func (s *Svc) ReloadConf(conf Config) {
	// do nothing here, conf is not used for now
}

func (s *Svc) onTick(_ context.Context, t time.Time) {
	if t.IsZero() {
		// We haven't got a blcok time yet, ignore
		return
	}

	if s.epoch.StartTime.IsZero() {
		// First block so let's create our first epoch
		s.epoch.UniqueID = 0
		s.epoch.StartTime = t
		s.epoch.ExpireTime = t // + epoch length
		// Send out new epoch event
	}

	if s.epoch.ExpireTime.Before(t) {
		s.epoch.EndTime = t
		// We have expired, send event

		s.epoch.UniqueID += 1

		// Create a new epoch
		s.epoch.StartTime = t
		s.epoch.ExpireTime = t // + epoch length
		s.epoch.EndTime = time.Time{}
	}
}

// NotifyOnEpoch allows other services to register a callback function
// which will be called once we enter a new epoch
func (s *Svc) NotifyOnEpoch(f func(context.Context, Epoch)) {
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, f)
}

func (s *Svc) notify(ctx context.Context, e Epoch) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.listeners {
		f(ctx, e)
	}
}
