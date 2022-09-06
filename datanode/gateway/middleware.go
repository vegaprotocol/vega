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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"google.golang.org/grpc/metadata"

	"code.vegaprotocol.io/vega/datanode/contextutil"
	"code.vegaprotocol.io/vega/datanode/metrics"
	vhttp "code.vegaprotocol.io/vega/libs/http"
	"code.vegaprotocol.io/vega/logging"
)

// RemoteAddrMiddleware is a middleware adding to the current request context the
// address of the caller.
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

// MetricCollectionMiddleware records the request and the time taken to service it.
func MetricCollectionMiddleware(next http.Handler) http.Handler {
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

func AddMDHeadersToContext(ctx context.Context, headers metadata.MD) error {
	resctx := graphql.GetResolverContext(ctx)
	if resctx == nil {
		return fmt.Errorf("no resolver context")
	}
	resctx.Args = map[string]interface{}{"headers": headers}
	return nil
}

func HeadersFromContext(ctx context.Context) (http.Header, bool) {
	resctx := graphql.GetResolverContext(ctx)

	args := resctx.Args
	if args == nil {
		return nil, false
	}

	headersRaw, ok := args["headers"]
	if !ok {
		return nil, false
	}

	mdHeader, ok := headersRaw.(metadata.MD)
	if !ok {
		return nil, false
	}

	return http.Header(mdHeader), true
}

// Chain builds the middleware Chain recursively, functions are first class
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
		iw := &InjectableResponseWriter{ResponseWriter: w}
		ctx = context.WithValue(ctx, injectableWriterKey, iw)
		next.ServeHTTP(iw, r.WithContext(ctx))
	})
}

type InjectableResponseWriter struct {
	http.ResponseWriter
	headers http.Header
}

type key string

const injectableWriterKey key = "injectable-writer-key"

func InjectableWriterFromContext(ctx context.Context) (*InjectableResponseWriter, bool) {
	if ctx == nil {
		return nil, false
	}
	val := ctx.Value(injectableWriterKey)
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
