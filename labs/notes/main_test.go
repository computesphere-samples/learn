package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := &server{path: filepath.Join(dir, "note.txt"), log: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	get := func() string {
		t.Helper()
		res, err := http.Get(ts.URL + "/note")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET /note: %d", res.StatusCode)
		}
		b, _ := io.ReadAll(res.Body)
		return string(b)
	}

	if got := get(); got != "" {
		t.Fatalf("before any save: got %q, want empty", got)
	}
	for _, note := range []string{"before redeploy", "sk 2026-09-27"} {
		res, err := http.Post(ts.URL+"/note", "application/x-www-form-urlencoded", strings.NewReader(note))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusNoContent {
			t.Fatalf("POST /note: %d", res.StatusCode)
		}
		if got := get(); got != note {
			t.Fatalf("got %q, want %q", got, note)
		}
	}

	// A new server on the same directory, as after a redeploy with a volume.
	s2 := &server{path: s.path, log: s.log}
	ts2 := httptest.NewServer(s2.routes())
	defer ts2.Close()
	res, err := http.Get(ts2.URL + "/note")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(b) != "sk 2026-09-27" {
		t.Fatalf("after restart: got %q", b)
	}

	big := strings.Repeat("x", maxNote+1)
	res, err = http.Post(ts.URL+"/note", "text/plain", strings.NewReader(big))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized note: %d", res.StatusCode)
	}
}

func TestHealthz(t *testing.T) {
	s := &server{path: filepath.Join(t.TempDir(), "note.txt"), log: slog.New(slog.NewJSONHandler(io.Discard, nil))}
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("healthz: %d %s", rec.Code, rec.Body)
	}
}
