package restproxy

import (
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
	"code.vegaprotocol.io/vega/internal/logging"
)

func TestNoGzip(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	rec := httptest.NewRecorder()
	NewGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	})(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200 got %d", rec.Code)
	}

	if rec.HeaderMap.Get("Content-Encoding") != "" {
		t.Fatalf(`expected Content-Encoding: "" got %s`, rec.HeaderMap.Get("Content-Encoding"))
	}

	if rec.Body.String() != "test" {
		t.Fatalf(`expected "test" go "%s"`, rec.Body.String())
	}

	if testing.Verbose() {
		b, _ := httputil.DumpResponse(rec.Result(), true)
		t.Log("\n" + string(b))
	}
}

func TestGzip(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	logger := logging.NewLoggerFromEnv("dev")
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

	if rec.HeaderMap.Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected Content-Encoding: gzip got %s", rec.HeaderMap.Get("Content-Encoding"))
	}
	if rec.HeaderMap.Get("Content-Length") != "" {
		t.Fatalf(`expected Content-Length: "" got %s`, rec.HeaderMap.Get("Content-Length"))
	}
	if rec.HeaderMap.Get("Content-Type") != "text/test" {
		t.Fatalf(`expected Content-Type: "text/test" got %s`, rec.HeaderMap.Get("Content-Type"))
	}

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
		b, _ := httputil.DumpResponse(rec.Result(), true)
		t.Log("\n" + string(b))
	}
}

func TestNoBody(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	rec := httptest.NewRecorder()
	NewGzipHandler(*logger, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d got %d", http.StatusNoContent, rec.Code)
	}

	if rec.HeaderMap.Get("Content-Encoding") != "" {
		t.Fatalf(`expected Content-Encoding: "" got %s`, rec.HeaderMap.Get("Content-Encoding"))
	}

	if rec.Body.Len() > 0 {
		t.Logf("%q", rec.Body.String())
		t.Fatalf("no body expected for %d bytes", rec.Body.Len())
	}

	if testing.Verbose() {
		b, _ := httputil.DumpResponse(rec.Result(), true)
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

	logger := logging.NewLoggerFromEnv("dev")
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
