package http

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/config/encoding"
)

type RateLimitConfig struct {
	CoolDown encoding.Duration `long:"coolDown" description:"rate-limit duration, e.g. 10s, 1m30s, 24h0m0s"`

	AllowList []string `long:"allowList" description:"a list of ip/subnets, e.g. 10.0.0.0/8, 192.168.0.0/16"`

	allowList []net.IPNet
}

type RateLimit struct {
	cfg RateLimitConfig
	// map of any_identifier -> time until request can be allowed
	requests map[string]time.Time

	mu sync.Mutex
}

func NewRateLimit(ctx context.Context, cfg RateLimitConfig) (*RateLimit, error) {
	cfg.allowList = make([]net.IPNet, len(cfg.AllowList))
	for i, allowItem := range cfg.AllowList {
		_, ipnet, err := net.ParseCIDR(allowItem)
		if err != nil {
			return nil, fmt.Errorf("failed to parse AllowList entry: %s", allowItem)
		}
		cfg.allowList[i] = *ipnet
	}
	r := &RateLimit{
		cfg:      cfg,
		requests: map[string]time.Time{},
	}
	go r.startCleanup(ctx)
	return r, nil
}

// NewRequest returns nil if the rate has not been exceeded
func (r *RateLimit) NewRequest(prefix, ip string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isAllowListed(ip) {
		return nil
	}

	identifier := fmt.Sprintf("%s %s", prefix, ip)
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
		return fmt.Errorf("rate-limited (%s for %s) until %v", prefix, ip, r.requests[identifier])
	}

	// greylist for the minimal duration
	r.requests[identifier] = time.Now().Add(r.cfg.CoolDown.Duration)

	return nil
}

func (r *RateLimit) isAllowListed(ip string) bool {
	netIP := net.ParseIP(ip)
	for _, allowItem := range r.cfg.allowList {
		if allowItem.Contains(netIP) {
			return true
		}
	}
	return false
}

func (r *RateLimit) startCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
