// Copyright (C) 2023 Gobalsky Labs Limited
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

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type ipGetter func(ctx context.Context, method string, log *logging.Logger) (string, error)

func (s *SubscriptionRateLimiter) WithGrpcInterceptor(ipGetterFunc ipGetter) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		addr, err := ipGetterFunc(ss.Context(), info.FullMethod, s.log)
		if err != nil {
			return status.Error(codes.PermissionDenied, err.Error())
		}
		if addr == "" {
			// If we don't have an IP we can't rate limit
			return handler(srv, ss)
		}

		ip, _, err := net.SplitHostPort(addr)
		if err != nil {
			ip = addr
		}

		if err := s.Inc(ip); err != nil {
			s.log.Error("client reached max subscription allowed",
				logging.Error(err))
			// write error
			return status.Error(codes.ResourceExhausted, "client reached max subscription allowed")
		}
		defer func() {
			s.Dec(ip)
		}()
		return handler(srv, ss)
	}
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
