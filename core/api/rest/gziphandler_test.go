// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package rest

import (
	"compress/gzip"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"code.vegaprotocol.io/vega/logging"
)

func headerNotPresent(t *testing.T, x *httptest.ResponseRecorder, key string) {
	t.Helper()
	res := x.Result()
	defer res.Body.Close()
	h, found := res.Header[key]
	if found || len(h) > 0 {
		t.Fatalf("Unexpected header: %s", key)
	}
}

func headerPresent(t *testing.T, x *httptest.ResponseRecorder, key string, expected []string) {
	t.Helper()
	res := x.Result()
	defer res.Body.Close()
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
	req, err := http.NewRequestWithContext(
		context.Background(), "GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	logger := logging.NewTestLogger()
	defer logger.Sync()

	rec := httptest.NewRecorder()
	newGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
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
		res := rec.Result()
		defer res.Body.Close()
		b, _ := httputil.DumpResponse(res, true)
		t.Log("\n" + string(b))
	}
}

func TestGzip(t *testing.T) {
	req, err := http.NewRequestWithContext(
		context.Background(), "GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	logger := logging.NewTestLogger()
	defer logger.Sync()

	rec := httptest.NewRecorder()
	newGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
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
		res := rec.Result()
		defer res.Body.Close()
		b, _ := httputil.DumpResponse(res, true)
		t.Log("\n" + string(b))
	}
}

func TestNoBody(t *testing.T) {
	req, err := http.NewRequestWithContext(
		context.Background(), "GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	logger := logging.NewTestLogger()
	defer logger.Sync()

	rec := httptest.NewRecorder()
	newGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
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
		res := rec.Result()
		defer res.Body.Close()
		b, _ := httputil.DumpResponse(res, true)
		t.Log("\n" + string(b))
	}
}

func BenchmarkGzip(b *testing.B) {
	body := []byte("testtesttesttesttesttesttesttesttesttesttesttesttest")

	req, err := http.NewRequest("GET", "http://example.com/", nil)
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
			newGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
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
