package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testServer(cfg config, logs io.Writer) *server {
	if logs == nil {
		logs = io.Discard
	}
	s := newServer(cfg, slog.New(slog.NewJSONHandler(logs, nil)))
	return s
}

func do(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, strings.NewReader(body)))
	return rec
}

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestLoadConfigDefaultsAndErrors(t *testing.T) {
	c, err := loadConfig(env(nil))
	if err != nil || c.port != "8080" || !c.requireSigning || c.startDelay != 0 || c.errorRate != 0 {
		t.Fatalf("defaults: %+v %v", c, err)
	}
	c, err = loadConfig(env(map[string]string{"START_DELAY": "150", "CACHE_MB": "900", "ERROR_RATE": "20%", "REQUIRE_SIGNING_KEY": "false"}))
	if err != nil || c.startDelay != 150*time.Second || c.cacheMB != 900 || c.errorRate != 0.2 || c.requireSigning {
		t.Fatalf("parsed: %+v %v", c, err)
	}
	for _, bad := range []map[string]string{
		{"START_DELAY": "soon"}, {"START_DELAY": "-1"}, {"CACHE_MB": "lots"},
		{"ERROR_RATE": "2"}, {"ERROR_RATE": "150%"}, {"REQUIRE_SIGNING_KEY": "maybe"},
	} {
		if _, err := loadConfig(env(bad)); err == nil {
			t.Fatalf("%v: expected an error", bad)
		}
	}
}

func TestHealthzWaitsForStartDelay(t *testing.T) {
	s := testServer(config{startDelay: 30 * time.Second}, nil)
	clock := s.started
	s.now = func() time.Time { return clock }
	h := s.routes()
	if rec := do(h, "GET", "/healthz", ""); rec.Code != 503 || !strings.Contains(rec.Body.String(), `"ready_in_seconds":30`) {
		t.Fatalf("before: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "GET", "/products", ""); rec.Code != 503 {
		t.Fatalf("products while starting: %d", rec.Code)
	}
	clock = clock.Add(30 * time.Second)
	if rec := do(h, "GET", "/healthz", ""); rec.Code != 200 {
		t.Fatalf("after: %d", rec.Code)
	}
}

func TestErrorRate(t *testing.T) {
	s := testServer(config{errorRate: 0.25}, nil)
	vals := []float64{0.1, 0.9, 0.3, 0.5}
	i := 0
	s.rand = func() float64 { v := vals[i%len(vals)]; i++; return v }
	h := s.routes()
	fails := 0
	for n := 0; n < 8; n++ {
		if do(h, "GET", "/products", "").Code == 500 {
			fails++
		}
	}
	if fails != 2 {
		t.Fatalf("fails = %d, want 2", fails)
	}
	if do(h, "GET", "/healthz", "").Code != 200 {
		t.Fatal("healthz must not be affected by ERROR_RATE")
	}
}

func TestOrdersAreSigned(t *testing.T) {
	h := testServer(config{signingKey: "k"}, nil).routes()
	rec := do(h, "POST", "/orders", `{"product_id":"p-101","quantity":2}`)
	var o order
	if rec.Code != 201 || json.Unmarshal(rec.Body.Bytes(), &o) != nil || o.TotalCents != 4400 || len(o.Signature) != 64 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "POST", "/orders", `{"product_id":"nope"}`); rec.Code != 422 {
		t.Fatalf("bad product: %d", rec.Code)
	}
	if rec := do(h, "GET", "/orders", ""); !strings.Contains(rec.Body.String(), o.ID) {
		t.Fatalf("list: %s", rec.Body)
	}
	if rec := do(h, "GET", "/products/p-104", ""); rec.Code != 200 {
		t.Fatalf("product: %d", rec.Code)
	}
	if rec := do(h, "GET", "/products/p-999", ""); rec.Code != 404 {
		t.Fatalf("missing product: %d", rec.Code)
	}
}

func TestLogLinesAreJSONWithFields(t *testing.T) {
	var buf bytes.Buffer
	s := testServer(config{errorRate: 1}, &buf)
	s.rand = func() float64 { return 0 }
	h := s.routes()
	req := httptest.NewRequest("GET", "/orders", nil)
	req.Header.Set("X-Request-Id", "abc123")
	h.ServeHTTP(httptest.NewRecorder(), req)
	do(h, "GET", "/healthz", "") // passing health checks aren't logged
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines: %q", len(lines), buf.String())
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]any{"request_id": "abc123", "route": "/orders", "status": 500.0} {
		if m[k] != want {
			t.Fatalf("%s = %v, want %v", k, m[k], want)
		}
	}
	if _, ok := m["duration_ms"]; !ok || m["error"] == nil {
		t.Fatalf("missing duration_ms or error: %v", m)
	}
}

func TestAllocFreeAndLimits(t *testing.T) {
	h := testServer(config{}, nil).routes()
	if rec := do(h, "GET", "/alloc?mb=2", ""); !strings.Contains(rec.Body.String(), `"alloc_mb":2`) {
		t.Fatalf("alloc: %s", rec.Body)
	}
	if rec := do(h, "GET", "/alloc?mb=3", ""); !strings.Contains(rec.Body.String(), `"alloc_mb":5`) {
		t.Fatalf("alloc again: %s", rec.Body)
	}
	if rec := do(h, "GET", "/free", ""); rec.Code != 200 {
		t.Fatalf("free: %d", rec.Code)
	}
	for _, bad := range []string{"/alloc?mb=99999", "/work?ms=-1", "/burn?seconds=x"} {
		if rec := do(h, "GET", bad, ""); rec.Code != 400 {
			t.Fatalf("%s: %d", bad, rec.Code)
		}
	}
}

func TestBurnStartsAndStops(t *testing.T) {
	s := testServer(config{}, nil)
	h := s.routes()
	if rec := do(h, "GET", "/burn?seconds=5", ""); rec.Code != 202 {
		t.Fatalf("burn: %d", rec.Code)
	}
	if rec := do(h, "GET", "/status", ""); !strings.Contains(rec.Body.String(), `"burn_seconds_left":5`) {
		t.Fatalf("status: %s", rec.Body)
	}
	if rec := do(h, "GET", "/burn?seconds=0", ""); rec.Code != 200 || s.burnUntil.Load() != 0 {
		t.Fatalf("stop: %d", rec.Code)
	}
}
