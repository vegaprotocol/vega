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
	"compress/gzip"
	"net/http"
	"strings"
	"sync"

	"code.vegaprotocol.io/vega/logging"
)

type gzipResponseWriter struct {
	http.ResponseWriter

	w             *gzip.Writer
	statusCode    int
	headerWritten bool
}

var pool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return &gzipResponseWriter{
			w: w,
		}
	},
}

func (gzr *gzipResponseWriter) WriteHeader(statusCode int) {
	gzr.statusCode = statusCode
	gzr.headerWritten = true

	if gzr.statusCode != http.StatusNotModified && gzr.statusCode != http.StatusNoContent {
		gzr.ResponseWriter.Header().Del("Content-Length")
		gzr.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	}

	gzr.ResponseWriter.WriteHeader(statusCode)
}

func (gzr *gzipResponseWriter) Write(b []byte) (int, error) {
	if _, ok := gzr.Header()["Content-Type"]; !ok {
		// If no content type, apply sniffing algorithm to un-gzipped body.
		gzr.ResponseWriter.Header().Set("Content-Type", http.DetectContentType(b))
	}

	if !gzr.headerWritten {
		// This is exactly what Go would also do if it hasn't been written yet.
		gzr.WriteHeader(http.StatusOK)
	}

	return gzr.w.Write(b)
}

func (gzr *gzipResponseWriter) Flush() {
	if gzr.w != nil {
		gzr.w.Flush()
	}

	if fw, ok := gzr.ResponseWriter.(http.Flusher); ok {
		fw.Flush()
	}
}

func NewGzipHandler(logger logging.Logger, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}

		gzr := pool.Get().(*gzipResponseWriter)
		gzr.statusCode = 0
		gzr.headerWritten = false
		gzr.ResponseWriter = w
		gzr.w.Reset(w)

		defer func() {
			// gzr.w.Close will write a footer even if no data has been written.
			// StatusNotModified and StatusNoContent expect an empty body so don't close it.
			if gzr.statusCode != http.StatusNotModified && gzr.statusCode != http.StatusNoContent {
				if err := gzr.w.Close(); err != nil {
					logger.Error("Failed to Gzip output from REST proxy", logging.Error(err))
				}
			}
			pool.Put(gzr)
		}()

		fn(gzr, r)
	}
}
