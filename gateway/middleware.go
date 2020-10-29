package gateway

import (
	"net/http"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	vhttp "code.vegaprotocol.io/vega/http"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
)

// RemoteAddrMiddleware is a middleware adding to the current request context the
// address of the caller
func RemoteAddrMiddleware(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := vhttp.RemoteAddr(r)
		if err != nil {
			log.Debug("Failed to get remote address in middleware",
				logging.String("remote-addr", r.RemoteAddr),
				logging.String("x-forwarded-for", r.Header.Get("X-Forwarded-For")),
			)
		} else {
			r = r.WithContext(contextutil.WithRemoteIPAddr(r.Context(), ip))
		}
		next.ServeHTTP(w, r)
	})
}

// MetricCollectionMiddleware records the request and the time taken to service it
func MetricCollectionMiddleware(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		next.ServeHTTP(w, r)
		end := time.Now()

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
