// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gateway

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/datanode/contextutil"
	"code.vegaprotocol.io/vega/datanode/metrics"
	vfmt "code.vegaprotocol.io/vega/libs/fmt"
	vhttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
)

var ErrMaxSubscriptionReached = func(ip string, max uint32) error {
	return fmt.Errorf("max subscriptions count (%v) reached for ip (%s)", max, ip)
}

// RemoteAddrMiddleware is a middleware adding to the current request context the
// address of the caller.
func RemoteAddrMiddleware(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := vhttp.RemoteAddr(r)
		if err != nil {
			log.Debug("Failed to get remote address in middleware",
				logging.String("remote-addr", r.RemoteAddr),
				logging.String("x-forwarded-for", vfmt.Escape(r.Header.Get("X-Forwarded-For"))),
			)
		} else {
			r = r.WithContext(contextutil.WithRemoteIPAddr(r.Context(), ip))
		}
		next.ServeHTTP(w, r)
	})
}

// MetricCollectionMiddleware records the request and the time taken to service it.
func MetricCollectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		end := time.Now()

		// Update the call count and timings in metrics
		timetaken := end.Sub(start)

		metrics.APIRequestAndTimeREST(r.Method, r.RequestURI, timetaken.Seconds())
	})
}

// Chain builds the middleware Chain recursively, functions are first class.
func Chain(f http.Handler, m ...func(http.Handler) http.Handler) http.Handler {
	// if our Chain is done, use the original handler func
	if len(m) == 0 {
		return f
	}
	// otherwise nest the handler funcs
	return m[0](Chain(f, m[1:cap(m)]...))
}

func WithAddHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		iw := &InjectableResponseWriter{
			ResponseWriter: w,
			Hijacker:       hijacker,
		}
		ctx = context.WithValue(ctx, injectableWriterKey{}, iw)
		next.ServeHTTP(iw, r.WithContext(ctx))
	})
}

type InjectableResponseWriter struct {
	http.ResponseWriter
	http.Hijacker
	headers http.Header
}

type injectableWriterKey struct{}

func InjectableWriterFromContext(ctx context.Context) (*InjectableResponseWriter, bool) {
	if ctx == nil {
		return nil, false
	}
	val := ctx.Value(injectableWriterKey{})
	if val == nil {
		return nil, false
	}
	return val.(*InjectableResponseWriter), true
}

func (i *InjectableResponseWriter) Write(data []byte) (int, error) {
	for k, v := range i.headers {
		if len(v) > 0 {
			i.ResponseWriter.Header().Add(k, v[0])
		}
	}
	return i.ResponseWriter.Write(data)
}

func (i *InjectableResponseWriter) SetHeaders(headers http.Header) {
	i.headers = headers
}

type SubscriptionRateLimiter struct {
	log *logging.Logger
	m   map[string]uint32
	mu  sync.Mutex

	MaxSubscriptions uint32
}

func NewSubscriptionRateLimiter(
	log *logging.Logger,
	maxSubscriptions uint32,
) *SubscriptionRateLimiter {
	return &SubscriptionRateLimiter{
		log:              log,
		MaxSubscriptions: maxSubscriptions,
		m:                map[string]uint32{},
	}
}

func (s *SubscriptionRateLimiter) Inc(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cnt := s.m[ip]
	if cnt == s.MaxSubscriptions {
		return ErrMaxSubscriptionReached(ip, s.MaxSubscriptions)
	}
	s.m[ip] = cnt + 1
	return nil
}

func (s *SubscriptionRateLimiter) Dec(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cnt := s.m[ip]
	s.m[ip] = cnt - 1
}

func (s *SubscriptionRateLimiter) WithSubscriptionRateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// is that a subscription?
		if _, ok := w.(http.Hijacker); !ok {
			next.ServeHTTP(w, r)
			return
		}

		if ip, err := getIP(r); err != nil {
			s.log.Debug("couldn't get client ip", logging.Error(err))
		} else {
			if err := s.Inc(ip); err != nil {
				s.log.Error("client reached max subscription allowed",
					logging.Error(err))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(err.Error()))
				// write error
				return
			}
			defer func() {
				s.Dec(ip)
			}()
		}

		next.ServeHTTP(w, r)
	})
}

func getIP(r *http.Request) (string, error) {
	ip := r.Header.Get("X-Real-IP")
	if net.ParseIP(ip) != nil {
		return ip, nil
	}

	ip = r.Header.Get("X-Forward-For")
	for _, i := range strings.Split(ip, ",") {
		if net.ParseIP(i) != nil {
			return i, nil
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	if net.ParseIP(ip) != nil {
		return ip, nil
	}

	return "", errors.New("no valid ip found")
}
