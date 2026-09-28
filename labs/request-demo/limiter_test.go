package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLimiterPerClientAndRefill(t *testing.T) {
	l := newLimiter(60, 3, 1000)
	clock := time.Unix(0, 0)
	l.now = func() time.Time { return clock }
	for i := 0; i < 3; i++ {
		if !l.allow("a") {
			t.Fatalf("request %d within burst was refused", i+1)
		}
	}
	if l.allow("a") {
		t.Fatal("request past the burst was allowed")
	}
	if !l.allow("b") {
		t.Fatal("another client was throttled by the first")
	}
	clock = clock.Add(2 * time.Second) // 60/min refills 1 token a second
	if !l.allow("a") {
		t.Fatal("bucket didn't refill")
	}
}

func TestLimiterGlobalCeiling(t *testing.T) {
	l := newLimiter(600, 100, 2) // global burst = 4
	clock := time.Unix(0, 0)
	l.now = func() time.Time { return clock }
	allowed := 0
	for i := 0; i < 10; i++ {
		if l.allow(string(rune('a' + i))) {
			allowed++
		}
	}
	if allowed != 4 {
		t.Fatalf("global ceiling allowed %d, want 4", allowed)
	}
}

func TestProtectHeadersAndHealthzExempt(t *testing.T) {
	l := newLimiter(60, 1, 1000)
	h := protect(l, newTestServer().routes())
	do := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("CF-Connecting-IP", "198.51.100.7")
		h.ServeHTTP(rec, req)
		return rec
	}
	first := do("/")
	if first.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
	if do("/").Code != http.StatusTooManyRequests {
		t.Fatal("second request past a burst of 1 wasn't limited")
	}
	for i := 0; i < 5; i++ {
		if do("/healthz").Code != http.StatusOK {
			t.Fatal("health check was rate limited")
		}
	}
}

func TestClientIPPrecedence(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.1:5555"
	if got := clientIP(req); got != "192.0.2.1" {
		t.Fatalf("remote addr: %s", got)
	}
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 192.0.2.1")
	if got := clientIP(req); got != "203.0.113.9" {
		t.Fatalf("forwarded: %s", got)
	}
	req.Header.Set("CF-Connecting-IP", "198.51.100.7")
	if got := clientIP(req); got != "198.51.100.7" {
		t.Fatalf("edge header: %s", got)
	}
}

func TestIndexDoesNotEchoHost(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?code=abc123xy", nil)
	req.Host = "evil.example"
	newTestServer().routes().ServeHTTP(rec, req)
	if got := rec.Body.String(); contains(got, "evil.example") {
		t.Fatalf("host echoed: %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
