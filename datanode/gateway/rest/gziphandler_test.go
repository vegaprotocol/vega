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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"code.vegaprotocol.io/vega/logging"
)

func headerNotPresent(t *testing.T, x *httptest.ResponseRecorder, key string) {
	t.Helper()
	res := x.Result() //nolint:bodyclose
	h, found := res.Header[key]
	if found || len(h) > 0 {
		t.Fatalf("Unexpected header: %s", key)
	}
}

func headerPresent(t *testing.T, x *httptest.ResponseRecorder, key string, expected []string) {
	t.Helper()
	res := x.Result() //nolint:bodyclose
	h, found := res.Header[key]
	if !found || len(h) == 0 {
		t.Fatalf("Missing header: %s", key)
	}
	if len(h) != len(expected) {
		t.Fatalf("Unexpected number of headers for \"%s\": expected %d, got %d", key, len(expected), len(h))
	}
	for i := range h {
		if h[i] != expected[i] {
			t.Fatalf("Unexpected header for \"%s\": #%d, expected \"%s\", got \"%s\"", key, i, expected[i], h[i])
		}
	}
}

func TestNoGzip(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	logger := logging.NewTestLogger()
	defer logger.Sync()

	rec := httptest.NewRecorder()
	NewGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	})(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200 got %d", rec.Code)
	}

	headerNotPresent(t, rec, "Content-Encoding")

	if rec.Body.String() != "test" {
		t.Fatalf(`expected "test" go "%s"`, rec.Body.String())
	}

	if testing.Verbose() {
		b, _ := httputil.DumpResponse(rec.Result(), true) //nolint:bodyclose
		t.Log("\n" + string(b))
	}
}

func TestGzip(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	logger := logging.NewTestLogger()
	defer logger.Sync()

	rec := httptest.NewRecorder()
	NewGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4")
		w.Header().Set("Content-Type", "text/test")
		w.Write([]byte("test"))
	})(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200 got %d", rec.Code)
	}

	headerPresent(t, rec, "Content-Encoding", []string{"gzip"})
	headerNotPresent(t, rec, "Content-Length")
	headerPresent(t, rec, "Content-Type", []string{"text/test"})

	r, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "test" {
		t.Fatalf(`expected "test" go "%s"`, string(body))
	}

	if testing.Verbose() {
		b, _ := httputil.DumpResponse(rec.Result(), true) //nolint:bodyclose
		t.Log("\n" + string(b))
	}
}

func TestNoBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	logger := logging.NewTestLogger()
	defer logger.Sync()

	rec := httptest.NewRecorder()
	NewGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d got %d", http.StatusNoContent, rec.Code)
	}

	headerNotPresent(t, rec, "Content-Encoding")

	if rec.Body.Len() > 0 {
		t.Logf("%q", rec.Body.String())
		t.Fatalf("no body expected for %d bytes", rec.Body.Len())
	}

	if testing.Verbose() {
		b, _ := httputil.DumpResponse(rec.Result(), true) // nolint: bodyclose
		t.Log("\n" + string(b))
	}
}

func BenchmarkGzip(b *testing.B) {
	body := []byte("testtesttesttesttesttesttesttesttesttesttesttesttest")

	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		b.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	logger := logging.NewTestLogger()
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			NewGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
				w.Write(body)
			})(rec, req)

			if rec.Code != http.StatusOK {
				b.Fatalf("expected %d got %d", http.StatusOK, rec.Code)
			}
			if rec.Body.Len() != 49 {
				b.Fatalf("expected 49 bytes, got %d bytes", rec.Body.Len())
			}
		}
	})
}
