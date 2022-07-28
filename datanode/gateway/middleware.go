// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gateway

import (
	"net/http"
	"strings"
	"time"

	"code.vegaprotocol.io/data-node/contextutil"
	vhttp "code.vegaprotocol.io/data-node/http"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
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
