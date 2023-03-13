package ratelimit

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/datanode/contextutil"
	"code.vegaprotocol.io/vega/logging"
	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/libstring"
	"github.com/didip/tollbooth/v7/limiter"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	secret   string
	banMsg   = "temporarily banned for continuing to request while rate limited"
	limitMsg = "api request rate limit exceeded"
)

// init sets our random per-process secret generated at startup.
//
// If the "X-Rate-Limit-Secret": <secret> is present in GRPC metadata, rate limiting will not be applied.
func init() {
	secret = uuid.New().String()
}

// WithSecret is a GRPC dial option that adds the "X-Rate-Limit-Secret": <secret> header to all calls.
func WithSecret() grpc.DialOption {
	interceptor := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "X-Rate-Limit-Secret", secret)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
	return grpc.WithUnaryInterceptor(interceptor)
}

type RateLimit struct {
	lmt         *limiter.Limiter
	cfg         atomic.Pointer[Config]
	log         *logging.Logger
	naughtyStep *naughtyStep
}

func NewFromConfig(cfg *Config, log *logging.Logger) *RateLimit {
	limitOpts := limiter.ExpirableOptions{DefaultExpirationTTL: cfg.TTL.Duration}
	lmt := tollbooth.NewLimiter(cfg.Rate, &limitOpts)
	lmt.SetBurst(cfg.Burst)

	// The naughty step limiter could have a different rate/burst but it seemed likely to add
	// more confusion than it's worth to the configuration & these should be sensible.
	ns := newNaughtyStep(log, cfg.Rate, cfg.Burst, cfg.BanFor.Duration, cfg.TTL.Duration)

	r := &RateLimit{
		lmt:         lmt,
		naughtyStep: ns,
		log:         log,
	}
	r.cfg.Store(cfg)
	return r
}

func (r *RateLimit) ReloadConfig(cfg *Config) {
	r.log.Info("updating rate limit configuration",
		logging.String("old", fmt.Sprintf("%v", r.cfg.Load())),
		logging.String("new", fmt.Sprintf("%v", cfg)))

	r.cfg.Store(cfg)
	r.lmt.SetBurst(cfg.Burst).
		SetMax(cfg.Rate)
	r.naughtyStep.lmt.SetBurst(cfg.Burst).
		SetMax(cfg.Rate)
	r.naughtyStep.banFor = cfg.BanFor.Duration
}

func (r *RateLimit) HTTPMiddleware(next http.Handler) http.Handler {
	middle := func(w http.ResponseWriter, req *http.Request) {
		if !r.cfg.Load().Enabled {
			next.ServeHTTP(w, req)
			return
		}

		ip := r.ipForRequest(req)

		if r.naughtyStep.isBanned(ip) {
			r.expressDisappointment(w, banMsg, ip, http.StatusForbidden, true)
			return
		}

		if httpError := tollbooth.LimitByRequest(r.lmt, w, req); httpError != nil {
			r.naughtyStep.smackBottom(ip)
			r.expressDisappointment(w, limitMsg, ip, http.StatusTooManyRequests, false)
			return
		}

		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(middle)
}

func (r *RateLimit) expressDisappointment(w http.ResponseWriter, msg, ip string, status int, banned bool) {
	w.Header().Add("Content-Type", "application/json")

	if banned {
		expiry := r.naughtyStep.bans[ip]
		remaining := time.Until(expiry).Seconds()

		w.Header().Add("Retry-After", fmt.Sprintf("%0.f", remaining))
	}
	w.WriteHeader(status)
	w.Write([]byte(msg))
}

func (r *RateLimit) ipForRequest(req *http.Request) string {
	ip := libstring.RemoteIP(r.lmt.GetIPLookups(), r.lmt.GetForwardedForIndexFromBehind(), req)
	return libstring.CanonicalizeIP(ip)
}

func (r *RateLimit) GRPCInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	if !r.cfg.Load().Enabled {
		return handler(ctx, req)
	}

	// Check if the client gave the secret in the metadata, if so skip rate limiting
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		mdSecrets := md.Get("X-Rate-Limit-Secret")
		for _, mdSecret := range mdSecrets {
			if mdSecret == secret {
				return handler(ctx, req)
			}
		}
	}

	// Fish out IP address from context
	addr, ok := contextutil.RemoteIPAddrFromContext(ctx)
	if !ok {
		// If we don't have an IP we can't rate limit
		return handler(ctx, req)
	}

	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		ip = addr
	}
	ip = libstring.CanonicalizeIP(ip)

	// Check the naughty step
	if r.naughtyStep.isBanned(ip) {
		expiry := r.naughtyStep.bans[ip]
		remaining := time.Until(expiry).Seconds()

		if err := grpc.SetHeader(ctx, metadata.Pairs("Retry-After", fmt.Sprintf("%0.f", remaining))); err != nil {
			r.log.Error("failed to set header", logging.Error(err))
		}

		return nil, status.Error(codes.Unavailable, banMsg)
	}

	setRateLimitXResponseHeaders(ctx, r.log, r.lmt, ip)
	if r.lmt.LimitReached(ip) {
		r.naughtyStep.smackBottom(ip)
		setRateLimitResponseHeaders(ctx, r.log, r.lmt, 0)
		return nil, status.Error(codes.Unavailable, limitMsg)
	}

	tokensLeft := r.lmt.Tokens(ip)
	setRateLimitResponseHeaders(ctx, r.log, r.lmt, tokensLeft)
	return handler(ctx, req)
}

// setRateLimitXResponseHeaders sets the same set of headers that tollbooth adds to every HTTP response
// when being used as a http server limiter.
func setRateLimitXResponseHeaders(ctx context.Context, log *logging.Logger, lmt *limiter.Limiter, ip string) {
	for _, h := range []metadata.MD{
		metadata.Pairs("X-Rate-Limit-Limit", strconv.FormatFloat(lmt.GetMax(), 'f', -1, 64)),
		metadata.Pairs("X-Rate-Limit-Duration", "1"),
		metadata.Pairs("X-Rate-Limit-Request-Remote-Addr", ip),
	} {
		if errH := grpc.SetHeader(ctx, h); errH != nil {
			log.Error("failed to set header", logging.Error(errH))
		}
	}
}

// setRateLimitResponseHeaders configures RateLimit-Limit, RateLimit-Remaining and RateLimit-Reset
// as seen at https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-ratelimit-headers
func setRateLimitResponseHeaders(ctx context.Context, log *logging.Logger, lmt *limiter.Limiter, tokensLeft int) {
	for _, h := range []metadata.MD{
		metadata.Pairs("RateLimit-Limit", fmt.Sprintf("%d", int(math.Round(lmt.GetMax())))),
		metadata.Pairs("RateLimit-Reset", "1"),
		metadata.Pairs("RateLimit-Remaining", fmt.Sprintf("%d", tokensLeft)),
	} {
		if errH := grpc.SetHeader(ctx, h); errH != nil {
			log.Error("failed to set header", logging.Error(errH))
		}
	}
}
