package gateway

import (
	"context"
	"net"
	"net/http"
	"strings"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/vegatime"
)

const bearerPrefix = "Bearer "

type tokenKeyTy int

var tokenKey tokenKeyTy

// TokenFromContext extract a token from the context
func TokenFromContext(ctx context.Context) string {
	u, _ := ctx.Value(tokenKey).(string)
	return u
}

// AddTokenToContext adds a new token to the given context
func AddTokenToContext(ctx context.Context, tkn string) context.Context {
	return context.WithValue(ctx, tokenKey, tkn)
}

// TokenMiddleware is used to add middleware checking for token in the
// processing of the http request
func TokenMiddleware(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if authhdr := r.Header.Get("Authorization"); len(authhdr) > 0 {
			if strings.HasPrefix(authhdr, bearerPrefix) {
				tkn := strings.TrimPrefix(authhdr, bearerPrefix)
				r = r.WithContext(context.WithValue(r.Context(), tokenKey, tkn))
				log.Debug("request with auth token",
					logging.String("token", tkn),
					logging.String("remote-addr", r.RemoteAddr),
				)
			} else {
				log.Debug("token specified but invalid fmt",
					logging.String("remote-addr", r.RemoteAddr),
				)
			}
		} else {
			log.Debug("no auth token",
				logging.String("remote-addr", r.RemoteAddr),
			)
		}
		next.ServeHTTP(w, r)
	})
}

// RemoteAddrMiddleware is a middleware adding to the current request context the
// address of the caller
func RemoteAddrMiddleware(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		found := false
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Warn("Remote address is not splittable in middleware",
				logging.String("remote-addr", r.RemoteAddr))
		} else {
			userIP := net.ParseIP(ip)
			if userIP == nil {
				log.Warn("Remote address is not IP:port format in middleware",
					logging.String("remote-addr", r.RemoteAddr))
			} else {
				found = true

				// Only defined when site is accessed via non-anonymous proxy
				// and takes precedence over RemoteAddr
				forward := r.Header.Get("X-Forwarded-For")
				if forward != "" {
					ip = forward
				}
			}
		}

		if found {
			r = r.WithContext(contextutil.WithRemoteIPAddr(r.Context(), ip))
		}
		next.ServeHTTP(w, r)
	})
}

// MetricCollectionMiddleware records the request and the time taken to service it
func MetricCollectionMiddleware(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := vegatime.Now()
		next.ServeHTTP(w, r)
		end := vegatime.Now()

		uri := r.RequestURI

		// Remove the first slash if it has one
		if strings.Index(uri, "/") == 0 {
			uri = uri[1:]
		}
		// Trim the URI down to something useful
		if strings.Count(uri, "/") >= 1 {
			uri = uri[:strings.Index(uri, "/")]
		}

		// Update the call count and timings in metrics
		timetaken := end.Sub(start)

		metrics.APIRequestAndTimeREST(uri, timetaken.Seconds())
	})
}
