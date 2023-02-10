package ratelimit

import (
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	"go.uber.org/zap"
)

// naughtyStep is a struct for keeping track of bad behavior and bans.
//
// You get put on the naughty step if you make requests despite having run out of tokens.
// The naughty step has it's own rate limiter, and its tokens are spent every time a failed
// (due to rate limiting) API call is made. If you run out of naughty tokens, then you get
// banned for a period of time.

type naughtyStep struct {
	log    *logging.Logger
	lmt    *limiter.Limiter
	bans   map[string]time.Time
	mu     sync.RWMutex
	banFor time.Duration
}

func newNaughtyStep(log *logging.Logger, rate float64, burst int, banFor, pruneEvery time.Duration) *naughtyStep {
	limitOpts := limiter.ExpirableOptions{DefaultExpirationTTL: pruneEvery}
	lmt := tollbooth.NewLimiter(rate, &limitOpts)
	lmt.SetBurst(burst)

	ns := naughtyStep{
		log:    log,
		lmt:    lmt,
		bans:   make(map[string]time.Time),
		banFor: banFor,
	}

	go func() {
		for range time.Tick(pruneEvery) {
			ns.prune()
		}
	}()

	return &ns
}

func (n *naughtyStep) enabled() bool {
	return n.banFor > 0
}

func (n *naughtyStep) smackBottom(ip string) {
	if !n.enabled() {
		return
	}

	if n.lmt.LimitReached(ip) {
		n.ban(ip)
		n.log.Info("banned for requesting past rate limit", zap.String("ip", ip))
	}
}

func (n *naughtyStep) ban(ip string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.bans[ip] = time.Now().Add(n.banFor)
}

func (n *naughtyStep) isBanned(ip string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if bannedUntil, ok := n.bans[ip]; ok {
		if time.Now().Before(bannedUntil) {
			return true
		}
	}
	return false
}

func (n *naughtyStep) prune() {
	n.mu.Lock()
	defer n.mu.Unlock()

	for ip, until := range n.bans {
		if time.Now().After(until) {
			delete(n.bans, ip)
		}
	}
}
