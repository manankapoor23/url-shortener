package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeChecker stands in for Safe Browsing so tests never call Google.
type fakeChecker struct {
	unsafe bool
	err    error
}

func (f fakeChecker) isUnsafe(ctx context.Context, target string) (bool, error) {
	return f.unsafe, f.err
}

func newTestServer() http.Handler {
	return newTestServerWith(fakeChecker{})
}

func newTestServerWith(c urlChecker) http.Handler {
	srv := &server{store: newStore(), checker: c, baseURL: "https://short.example", ownHost: "short.example"}
	return srv.routes()
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestShortenThenRedirect(t *testing.T) {
	h := newTestServer()

	rec := do(h, "POST", "/api/shorten", `{"url": "https://example.com/page"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("shorten: status %d, body %s", rec.Code, rec.Body)
	}
	var resp shortenResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.ShortURL != "https://short.example/"+resp.Code {
		t.Fatalf("short_url = %q, code = %q", resp.ShortURL, resp.Code)
	}

	rec = do(h, "GET", "/"+resp.Code, "")
	if rec.Code != http.StatusFound {
		t.Fatalf("redirect: status %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://example.com/page" {
		t.Fatalf("Location = %q", loc)
	}
}

func TestShortenRejectsBadInput(t *testing.T) {
	h := newTestServer()
	tests := map[string]int{
		`{"url": "javascript:alert(1)"}`: http.StatusBadRequest,
		`not json`:                       http.StatusBadRequest,
		`{"url": "` + strings.Repeat("a", maxRequestBytes) + `"}`: http.StatusRequestEntityTooLarge,
	}
	for body, want := range tests {
		if rec := do(h, "POST", "/api/shorten", body); rec.Code != want {
			t.Errorf("body %.40q: status %d, want %d", body, rec.Code, want)
		}
	}
}

func TestShortenRejectsFlaggedURL(t *testing.T) {
	h := newTestServerWith(fakeChecker{unsafe: true})
	rec := do(h, "POST", "/api/shorten", `{"url": "https://evil.example"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "flagged as unsafe") {
		t.Fatalf("status %d, body %s; want 400 flagged", rec.Code, rec.Body)
	}
}

// Fail closed: if the reputation check can't give a verdict, no link is created.
func TestShortenFailsClosedWhenCheckUnavailable(t *testing.T) {
	for _, err := range []error{errNoAPIKey, context.DeadlineExceeded, errors.New("safe browsing: status 500")} {
		srv := &server{store: newStore(), checker: fakeChecker{err: err}, baseURL: "https://short.example", ownHost: "short.example"}
		rec := do(srv.routes(), "POST", "/api/shorten", `{"url": "https://example.com"}`)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%v: status %d, want 503", err, rec.Code)
		}
		if strings.Contains(rec.Body.String(), err.Error()) {
			t.Errorf("%v: internal error leaked to user: %s", err, rec.Body)
		}
		if len(srv.store.codes) != 0 {
			t.Errorf("%v: link was stored despite failed check", err)
		}
	}
}

// Unknown and malformed codes must be indistinguishable.
func TestUnknownAndMalformedCodesGetSame404(t *testing.T) {
	h := newTestServer()
	unknown := do(h, "GET", "/aB3x9Kq", "")
	malformed := do(h, "GET", "/abc", "")

	for _, rec := range []*httptest.ResponseRecorder{unknown, malformed} {
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d, want 404", rec.Code)
		}
	}
	a, _ := io.ReadAll(unknown.Body)
	b, _ := io.ReadAll(malformed.Body)
	if string(a) != string(b) {
		t.Fatalf("404 bodies differ: %q vs %q", a, b)
	}
}

func TestIndexServesPage(t *testing.T) {
	rec := do(newTestServer(), "GET", "/", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<form") {
		t.Fatalf("index: status %d", rec.Code)
	}
}
