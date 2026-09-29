package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer() *server {
	return &server{store: newStore(), log: slog.New(slog.NewJSONHandler(io.Discard, nil)), now: time.Now}
}

func do(s *server, method, target, body string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.routes().ServeHTTP(rec, req)
	return rec
}

func TestReceiveAndShowJSON(t *testing.T) {
	s := newTestServer()
	rec := do(s, "POST", "/hooks/team-alerts", `{"text":"Deployment learn-shop-api is Running"}`, map[string]string{"Content-Type": "application/json"})
	if rec.Code != 202 {
		t.Fatalf("post: %d", rec.Code)
	}
	var got struct {
		Count    int       `json:"count"`
		Messages []message `json:"messages"`
	}
	rec = do(s, "GET", "/team-alerts?format=json", "", nil)
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Count != 1 {
		t.Fatalf("json: %s %v", rec.Body, err)
	}
	if got.Messages[0].Summary != "Deployment learn-shop-api is Running" {
		t.Fatalf("summary: %+v", got.Messages[0])
	}
	if rec := do(s, "GET", "/team-alerts", "", map[string]string{"Accept": "application/json"}); !strings.Contains(rec.Body.String(), `"count":1`) {
		t.Fatalf("accept header: %s", rec.Body)
	}
}

func TestPageEscapesAndOrders(t *testing.T) {
	s := newTestServer()
	do(s, "POST", "/hooks/abcdef", "msg-one", nil)
	do(s, "POST", "/hooks/abcdef", `<script>alert(1)</script>`, nil)
	body := do(s, "GET", "/abcdef", "", nil).Body.String()
	if strings.Contains(body, "<script>") {
		t.Fatal("body not escaped")
	}
	if strings.Index(body, "&lt;script") > strings.Index(body, "msg-one") {
		t.Fatal("newest message should come first")
	}
	if !strings.Contains(do(s, "GET", "/zzzzzz", "", nil).Body.String(), "Nothing yet") {
		t.Fatal("empty inbox page")
	}
}

func TestKeepsLastTwentyAndSeparatesTokens(t *testing.T) {
	s := newTestServer()
	for i := 0; i < 25; i++ {
		do(s, "POST", "/hooks/aaaaaa", fmt.Sprintf(`{"message":"m%d"}`, i), nil)
	}
	msgs := s.store.list("aaaaaa")
	if len(msgs) != perInbox || msgs[0].Summary != "m24" || msgs[len(msgs)-1].Summary != "m5" {
		t.Fatalf("kept %d, first %q", len(msgs), msgs[0].Summary)
	}
	if len(s.store.list("bbbbbb")) != 0 {
		t.Fatal("tokens leaked into each other")
	}
}

func TestRejectsBadTokensAndBigBodies(t *testing.T) {
	s := newTestServer()
	for _, target := range []string{"/hooks/abc", "/hooks/healthz", "/hooks/has%20space"} {
		if rec := do(s, "POST", target, "x", nil); rec.Code != 400 {
			t.Fatalf("%s: %d", target, rec.Code)
		}
	}
	if rec := do(s, "POST", "/hooks/abcdef", strings.Repeat("x", maxBody+1), nil); rec.Code != 413 {
		t.Fatalf("big body: %d", rec.Code)
	}
	if rec := do(s, "GET", "/hooks/abcdef", "", nil); rec.Code != 405 {
		t.Fatalf("GET hooks: %d", rec.Code)
	}
	if rec := do(s, "GET", "/healthz", "", nil); rec.Code != 200 {
		t.Fatalf("healthz: %d", rec.Code)
	}
}

func TestEvictsOldestInbox(t *testing.T) {
	st := newStore()
	for i := 0; i <= maxInbox; i++ {
		st.add(fmt.Sprintf("token%04d", i), message{})
	}
	if len(st.list("token0000")) != 0 || len(st.list(fmt.Sprintf("token%04d", maxInbox))) != 1 {
		t.Fatal("eviction")
	}
}
