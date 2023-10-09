// Copyright (C) 2023  Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package http

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/libs/config/encoding"
)

type RateLimitConfig struct {
	CoolDown encoding.Duration `description:"rate-limit duration, e.g. 10s, 1m30s, 24h0m0s" long:"coolDown"`

	AllowList []string `description:"a list of ip/subnets, e.g. 10.0.0.0/8, 192.168.0.0/16" long:"allowList"`

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

// NewRequest returns nil if the rate has not been exceeded.
func (r *RateLimit) NewRequest(identifier, ip string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isAllowListed(ip) {
		return nil
	}

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
		return fmt.Errorf("rate-limited (%s) until %v", identifier, r.requests[identifier])
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
