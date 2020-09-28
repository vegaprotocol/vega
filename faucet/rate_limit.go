package faucet

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type RateLimit struct {
	cfg Config
	// map of pubkeys -> assets id -> time until request can be allowed
	requests map[string]map[string]time.Time

	mu sync.Mutex
}

func NewRateLimit(ctx context.Context, cfg Config) *RateLimit {
	r := &RateLimit{
		cfg:      cfg,
		requests: map[string]map[string]time.Time{},
	}
	go r.startCleanup(ctx)
	return r
}

// NewRequest returns nil if the party can request new funds
func (r *RateLimit) NewRequest(pubkey, asset string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	assets, ok := r.requests[pubkey]
	if !ok {
		r.requests[pubkey] = map[string]time.Time{}
		assets = r.requests[pubkey]
	}
	until, ok := assets[asset]
	if ok {
		// just check in case the time is expired already, and
		// we had not the cleanup run
		if time.Now().Before(until) {
			// we already have this asset greylisted,
			// the trader is trying to get more fuunds while still greylisted
			// give him a penalty
			assets[asset] = until.Add(r.cfg.CoolDown.Duration)
			return fmt.Errorf("you are greylisted - your pubkey is now greylisted for an extended period until %v", assets[asset])
		}
	}

	// greylist for the minimal duration
	assets[asset] = time.Now().Add(r.cfg.CoolDown.Duration)

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
			for pubkey, assets := range r.requests {
				for asset, tim := range assets {
					// if time is elapsed, remove from the map
					if tim.Before(now) {
						delete(assets, asset)
					}
				}
				// if no assets greylisted anymore for this pubkey
				// we remove the pubkey
				if len(assets) <= 0 {
					delete(r.requests, pubkey)
				}
			}
			r.mu.Unlock()
		}
	}
}
