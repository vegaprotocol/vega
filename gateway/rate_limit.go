package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
)

type RateLimitConfig struct {
	CoolDown encoding.Duration `long:"coolDown"`
}

type RateLimit struct {
	cfg RateLimitConfig
	// map of any_identifier -> time until request can be allowed
	requests map[string]time.Time

	mu sync.Mutex
}

func NewRateLimit(ctx context.Context, cfg RateLimitConfig) *RateLimit {
	r := &RateLimit{
		cfg:      cfg,
		requests: map[string]time.Time{},
	}
	go r.startCleanup(ctx)
	return r
}

// NewRequest returns nil if the rate has not been exceeded
func (r *RateLimit) NewRequest(identifier string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	until, found := r.requests[identifier]
	if !found {
		until = time.Time{}
		r.requests[identifier] = until
	}
	// just check in case the time is expired already, and
	// we had not the cleanup run
	if time.Now().Before(until) {
		// we are already greylisted,
		// another request came in while still greylisted
		// add a penalty time
		r.requests[identifier] = until.Add(r.cfg.CoolDown.Duration)
		return fmt.Errorf("rate-limited until until %v", r.requests[identifier])
	}

	// greylist for the minimal duration
	r.requests[identifier] = time.Now().Add(r.cfg.CoolDown.Duration)

	return nil
}

func (r *RateLimit) startCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case _ = <-ticker.C:
			now := time.Now()
			r.mu.Lock()
			for identifier, until := range r.requests {
				// if time is elapsed, remove from the map
				if until.Before(now) {
					delete(r.requests, identifier)
				}
			}
			r.mu.Unlock()
		}
	}
}
