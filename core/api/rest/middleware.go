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

package rest

import (
	"net/http"
	"time"

	"code.vegaprotocol.io/vega/core/metrics"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	vfmt "code.vegaprotocol.io/vega/libs/fmt"
	vghttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
)

// RemoteAddrMiddleware is a middleware adding to the current request context the
// address of the caller.
func RemoteAddrMiddleware(log *logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := vghttp.RemoteAddr(r)
		if err != nil {
			log.Debug("Failed to get remote address in middleware",
				logging.String("remote-addr", r.RemoteAddr),
				logging.String("x-forwarded-for", vfmt.Escape(r.Header.Get("X-Forwarded-For"))),
			)
		} else {
			r = r.WithContext(vgcontext.WithRemoteIPAddr(r.Context(), ip))
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
