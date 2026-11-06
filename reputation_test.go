package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newFakeGoogle starts a local server that plays the Safe Browsing API.
func newFakeGoogle(t *testing.T, handler http.HandlerFunc) *safeBrowsing {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	sb := newSafeBrowsing("test-key")
	sb.endpoint = ts.URL
	return sb
}

func TestSafeBrowsingNoMatch(t *testing.T) {
	sb := newFakeGoogle(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("key = %q", r.URL.Query().Get("key"))
		}
		var req safeBrowsingRequest
		json.NewDecoder(r.Body).Decode(&req)
		if got := req.ThreatInfo.ThreatEntries; len(got) != 1 || got[0].URL != "https://example.com" {
			t.Errorf("threatEntries = %+v", got)
		}
		w.Write([]byte(`{}`))
	})
	unsafe, err := sb.isUnsafe(context.Background(), "https://example.com")
	if err != nil || unsafe {
		t.Fatalf("got %v, %v; want safe", unsafe, err)
	}
}

func TestSafeBrowsingMatch(t *testing.T) {
	sb := newFakeGoogle(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"matches":[{"threatType":"SOCIAL_ENGINEERING","platformType":"ANY_PLATFORM","threat":{"url":"https://evil.example"}}]}`))
	})
	unsafe, err := sb.isUnsafe(context.Background(), "https://evil.example")
	if err != nil || !unsafe {
		t.Fatalf("got %v, %v; want unsafe", unsafe, err)
	}
}

func TestSafeBrowsingErrors(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		if _, err := newSafeBrowsing("").isUnsafe(context.Background(), "https://example.com"); !errors.Is(err, errNoAPIKey) {
			t.Fatalf("got %v, want errNoAPIKey", err)
		}
	})

	t.Run("bad status", func(t *testing.T) {
		sb := newFakeGoogle(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		if _, err := sb.isUnsafe(context.Background(), "https://example.com"); err == nil {
			t.Fatal("want error on 403")
		}
	})

	t.Run("timeout without leaking key", func(t *testing.T) {
		sb := newFakeGoogle(t, func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
		})
		sb.client.Timeout = 50 * time.Millisecond
		_, err := sb.isUnsafe(context.Background(), "https://example.com")
		if err == nil {
			t.Fatal("want timeout error")
		}
		if strings.Contains(err.Error(), "test-key") {
			t.Fatalf("error leaks API key: %v", err)
		}
	})
}
