// webhook-inbox is the webhook receiver for ComputeSphere Learn, Path 6
// (Operate). POST anything to /hooks/<token>; open /<token> to see the last
// 20 messages for that token, as a page or as JSON. Messages live in memory
// only: a restart empties every inbox.
package main

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

var version = "dev"

const (
	perInbox  = 20       // messages kept per token
	maxInbox  = 1000     // tokens kept; the least recently written is dropped
	maxBody   = 64 << 10 // bytes accepted per message
	shownBody = 8 << 10  // bytes of a body shown on the page
)

// A token is the learner's own name for their inbox.
var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{6,64}$`)

var reserved = map[string]bool{"healthz": true, "hooks": true, "favicon.ico": true}

type message struct {
	ReceivedAt  time.Time         `json:"received_at"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers"`
	Summary     string            `json:"summary,omitempty"`
	Body        string            `json:"body"`
	JSON        json.RawMessage   `json:"json,omitempty"`
}

type inbox struct {
	token string
	msgs  []message // newest last
}

type store struct {
	mu    sync.Mutex
	order *list.List // of *inbox, most recently written at the front
	index map[string]*list.Element
}

func newStore() *store { return &store{order: list.New(), index: map[string]*list.Element{}} }

func (s *store) add(token string, m message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	el, ok := s.index[token]
	if ok {
		s.order.MoveToFront(el)
	} else {
		el = s.order.PushFront(&inbox{token: token})
		s.index[token] = el
		if s.order.Len() > maxInbox {
			last := s.order.Back()
			s.order.Remove(last)
			delete(s.index, last.Value.(*inbox).token)
		}
	}
	ib := el.Value.(*inbox)
	ib.msgs = append(ib.msgs, m)
	if len(ib.msgs) > perInbox {
		ib.msgs = ib.msgs[len(ib.msgs)-perInbox:]
	}
}

// list returns the token's messages, newest first.
func (s *store) list(token string) []message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []message{}
	if el, ok := s.index[token]; ok {
		msgs := el.Value.(*inbox).msgs
		for i := len(msgs) - 1; i >= 0; i-- {
			out = append(out, msgs[i])
		}
	}
	return out
}

// Headers worth showing. Everything else (cookies, auth) is left out.
var shownHeaders = []string{"User-Agent", "X-Request-Id", "X-Webhook-Event", "X-Event-Type"}

// summarize picks the human-readable line chat webhooks use.
func summarize(raw []byte) (json.RawMessage, string) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, ""
	}
	if obj, ok := v.(map[string]any); ok {
		for _, k := range []string{"text", "content", "message", "title", "event"} {
			if s, ok := obj[k].(string); ok && s != "" {
				return json.RawMessage(raw), s
			}
		}
	}
	return json.RawMessage(raw), ""
}

type server struct {
	store *store
	log   *slog.Logger
	now   func() time.Time
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("POST /hooks/{token}", s.receive)
	mux.HandleFunc("GET /hooks/{token}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Send webhooks here with POST. Read them at /"+r.PathValue("token"), http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("GET /{token}", s.show)
	return headers(mux)
}

func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
		next.ServeHTTP(w, r)
	})
}

func validToken(t string) bool { return tokenPattern.MatchString(t) && !reserved[t] }

func (s *server) receive(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if !validToken(token) {
		http.Error(w, "The token must be 6 to 64 letters, digits, - or _.", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		http.Error(w, fmt.Sprintf("Body too large: the limit is %d KB.", maxBody>>10), http.StatusRequestEntityTooLarge)
		return
	}
	m := message{ReceivedAt: s.now().UTC(), ContentType: r.Header.Get("Content-Type"), Headers: map[string]string{}, Body: string(body)}
	for _, k := range shownHeaders {
		if v := r.Header.Get(k); v != "" {
			m.Headers[k] = v
		}
	}
	m.JSON, m.Summary = summarize(body)
	s.store.add(token, m)
	s.log.Info("webhook received", "token", token, "bytes", len(body), "content_type", m.ContentType, "json", m.JSON != nil)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprint(w, `{"ok":true}`)
}

func wantsJSON(r *http.Request) bool {
	return r.URL.Query().Get("format") == "json" || strings.Contains(r.Header.Get("Accept"), "application/json")
}

func (s *server) show(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSuffix(r.PathValue("token"), ".json")
	asJSON := wantsJSON(r) || strings.HasSuffix(r.PathValue("token"), ".json")
	if !validToken(token) {
		http.NotFound(w, r)
		return
	}
	msgs := s.store.list(token)
	w.Header().Set("Cache-Control", "no-store")
	if asJSON {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"token": token, "count": len(msgs), "messages": msgs})
		return
	}
	type row struct {
		message
		When, Pretty string
	}
	rows := make([]row, 0, len(msgs))
	for _, m := range msgs {
		pretty := m.Body
		if m.JSON != nil {
			var buf strings.Builder
			var v any
			if json.Unmarshal(m.JSON, &v) == nil {
				enc := json.NewEncoder(&buf)
				enc.SetIndent("", "  ")
				enc.SetEscapeHTML(false)
				_ = enc.Encode(v)
				pretty = buf.String()
			}
		}
		if len(pretty) > shownBody {
			pretty = pretty[:shownBody] + "\n… (cut; the JSON view has it all)"
		}
		rows = append(rows, row{message: m, When: m.ReceivedAt.Format("2006-01-02 15:04:05 UTC"), Pretty: pretty})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = page.Execute(w, map[string]any{"Token": token, "Rows": rows, "Max": perInbox, "Version": version})
}

func (s *server) index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, `learn-webhook-inbox %s

Pick a token (6-64 letters, digits, - or _), then:

POST /hooks/<token>          send a webhook here
GET  /<token>                the last %d messages, newest first (refreshes itself)
GET  /<token>?format=json    the same as JSON

Messages are kept in memory only. Anyone who knows a token can read its inbox,
so use one nobody would guess, and don't send anything secret.
`, version, perInbox)
}

var page = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="refresh" content="5">
<title>Inbox: {{.Token}}</title>
<style>
body{margin:0;background:#0a0a0b;color:#ecedee;font:15px/1.5 system-ui,sans-serif}
main{max-width:860px;margin:0 auto;padding:32px 16px}
h1{font-weight:500;font-size:22px;margin:0 0 4px}
.sub{color:#9ca3ac;margin:0 0 24px}
code{color:#8fd3e3}
.msg{border:1px solid #26282c;border-radius:10px;padding:14px 16px;margin:0 0 14px;background:#111214}
.meta{color:#9ca3ac;font-size:13px}
.summary{font-size:16px;margin:6px 0 8px}
pre{margin:8px 0 0;padding:10px;background:#0a0a0b;border-radius:6px;overflow-x:auto;font-size:13px;white-space:pre-wrap;word-break:break-word}
.empty{color:#9ca3ac;border:1px dashed #26282c;border-radius:10px;padding:24px;text-align:center}
</style></head>
<body><main>
<h1>Inbox <code>{{.Token}}</code></h1>
<p class="sub">Send webhooks to <code>/hooks/{{.Token}}</code>. The last {{.Max}} are shown, newest first. This page refreshes every 5 seconds. <a href="?format=json" style="color:#8fd3e3">JSON</a></p>
{{if not .Rows}}<p class="empty">Nothing yet. POST something to <code>/hooks/{{.Token}}</code>.</p>{{end}}
{{range .Rows}}<div class="msg">
<div class="meta">{{.When}}{{if .ContentType}} · {{.ContentType}}{{end}}{{range $k, $v := .Headers}} · {{$k}}: {{$v}}{{end}}</div>
{{if .Summary}}<div class="summary">{{.Summary}}</div>{{end}}
<pre>{{.Pretty}}</pre>
</div>{{end}}
<p class="meta">learn-webhook-inbox {{.Version}}</p>
</main></body></html>`))

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	s := &server{store: newStore(), log: log, now: time.Now}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	log.Info("learn-webhook-inbox listening", "version", version, "port", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("can't listen", "error", err.Error())
		os.Exit(1)
	}
}
