package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer() *server {
	return &server{secret: []byte("test-secret-test-secret-test-secret-000")}
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestIndexSetsNonceForValidCode(t *testing.T) {
	s := newTestServer()
	rec := get(t, s.routes(), "/?code=abc123xy")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	n := rec.Header().Get("X-Learn-Nonce")
	if len(n) != nonceLength || n != s.nonce("abc123xy") {
		t.Fatalf("nonce = %q", n)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("index must not be cached")
	}
}

func TestIndexOmitsNonceForBadOrMissingCode(t *testing.T) {
	h := newTestServer().routes()
	for _, target := range []string{"/", "/?code=AB", "/?code=has%20space", "/?code=UPPERCASE1"} {
		if n := get(t, h, target).Header().Get("X-Learn-Nonce"); n != "" {
			t.Fatalf("%s: unexpected nonce %q", target, n)
		}
	}
}

func TestNonceDependsOnCodeAndSecret(t *testing.T) {
	a := newTestServer()
	b := &server{secret: []byte("another-secret-another-secret-000000")}
	if a.nonce("abc123xy") == a.nonce("abc123xz") {
		t.Fatal("different codes gave the same nonce")
	}
	if a.nonce("abc123xy") == b.nonce("abc123xy") {
		t.Fatal("different secrets gave the same nonce")
	}
}

func TestVerify(t *testing.T) {
	s := newTestServer()
	h := s.routes()
	good := s.nonce("abc123xy")
	cases := []struct {
		target string
		want   bool
	}{
		{"/verify?code=abc123xy&nonce=" + good, true},
		{"/verify?code=abc123xy&nonce=%20" + good + "%20", true},
		{"/verify?code=abc123xy&nonce=wrongnonce", false},
		{"/verify?code=abc123xz&nonce=" + good, false},
		{"/verify?code=AB&nonce=" + good, false},
		{"/verify", false},
	}
	for _, c := range cases {
		rec := get(t, h, c.target)
		var body map[string]bool
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: %v", c.target, err)
		}
		if body["ok"] != c.want {
			t.Fatalf("%s: ok = %v, want %v", c.target, body["ok"], c.want)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Fatalf("%s: missing CORS header", c.target)
		}
	}
}

func TestVerifyRejectsPost(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer().routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/verify", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHealthzAndUnknownPath(t *testing.T) {
	h := newTestServer().routes()
	if rec := get(t, h, "/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("healthz = %d", rec.Code)
	}
	if rec := get(t, h, "/nope"); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown path = %d", rec.Code)
	}
}
