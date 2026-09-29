// shop-api is the sample service for ComputeSphere Learn, Path 6 (Operate).
// It's a small shop API (/products, /orders) with switches, all environment
// variables, that make it misbehave on purpose: slow to start, missing a
// secret, short of memory, busy on CPU, or failing a share of requests.
// Every log line is one JSON object on stdout.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	mrand "math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var version = "dev"

// levelFatal is logged just before the process exits on bad configuration.
const levelFatal = slog.Level(12)

const (
	maxAllocPerCallMB = 1024
	maxWorkMS         = 10_000
	maxBurnSeconds    = 3600
	maxOrders         = 100
)

type config struct {
	shopName       string
	port           string
	signingKey     string
	requireSigning bool
	startDelay     time.Duration
	cacheMB        int
	errorRate      float64
}

// loadConfig reads the environment. A bad value is an error, not a silent
// default, so a typo shows up as a clear log line.
func loadConfig(getenv func(string) string) (config, error) {
	c := config{port: getenv("PORT"), signingKey: getenv("SIGNING_KEY"), requireSigning: true}
	if c.port == "" {
		c.port = "8080"
	}
	c.shopName = getenv("SHOP_NAME")
	if c.shopName == "" {
		c.shopName = "Learn shop"
	}
	if v := getenv("REQUIRE_SIGNING_KEY"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return c, fmt.Errorf("REQUIRE_SIGNING_KEY must be true or false, got %q", v)
		}
		c.requireSigning = b
	}
	if v := getenv("START_DELAY"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 3600 {
			return c, fmt.Errorf("START_DELAY must be a whole number of seconds from 0 to 3600, got %q", v)
		}
		c.startDelay = time.Duration(n) * time.Second
	}
	if v := getenv("CACHE_MB"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 16384 {
			return c, fmt.Errorf("CACHE_MB must be a whole number of megabytes from 0 to 16384, got %q", v)
		}
		c.cacheMB = n
	}
	if v := getenv("ERROR_RATE"); v != "" {
		r, err := parseRate(v)
		if err != nil {
			return c, err
		}
		c.errorRate = r
	}
	return c, nil
}

// parseRate accepts a share as 0.2 or 20%.
func parseRate(v string) (float64, error) {
	s := strings.TrimSpace(v)
	pct := strings.HasSuffix(s, "%")
	f, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
	if pct {
		f /= 100
	}
	if err != nil || math.IsNaN(f) || f < 0 || f > 1 {
		return 0, fmt.Errorf("ERROR_RATE must be a share from 0 to 1 (0.2) or a percentage (20%%), got %q", v)
	}
	return f, nil
}

type product struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
}

var catalog = []product{
	{"p-100", "Enamel mug", 1400},
	{"p-101", "Canvas tote", 2200},
	{"p-102", "Sticker pack", 600},
	{"p-103", "Wool beanie", 2800},
	{"p-104", "Notebook", 1100},
}

type order struct {
	ID         string    `json:"id"`
	ProductID  string    `json:"product_id"`
	Quantity   int       `json:"quantity"`
	TotalCents int       `json:"total_cents"`
	CreatedAt  time.Time `json:"created_at"`
	Signature  string    `json:"signature,omitempty"`
}

type server struct {
	cfg     config
	log     *slog.Logger
	started time.Time
	now     func() time.Time
	rand    func() float64

	cache []byte // CACHE_MB, held for the life of the process

	mu     sync.Mutex
	orders []order
	held   [][]byte // /alloc

	burnUntil atomic.Int64 // unix nanos; 0 when idle
	burnGen   atomic.Int64
}

func newServer(cfg config, log *slog.Logger) *server {
	return &server{cfg: cfg, log: log, started: time.Now(), now: time.Now, rand: mrand.Float64}
}

func (s *server) ready() bool { return s.now().Sub(s.started) >= s.cfg.startDelay }

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /status", s.status)
	mux.HandleFunc("GET /products", s.shop(s.listProducts))
	mux.HandleFunc("GET /products/{id}", s.shop(s.getProduct))
	mux.HandleFunc("GET /orders", s.shop(s.listOrders))
	mux.HandleFunc("POST /orders", s.shop(s.createOrder))
	mux.HandleFunc("GET /work", s.work)
	mux.HandleFunc("GET /burn", s.burn)
	mux.HandleFunc("GET /alloc", s.alloc)
	mux.HandleFunc("GET /free", s.free)
	return s.logRequests(mux)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *server) index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, `%s (learn-shop-api %s)

GET  /products          the catalog
GET  /products/{id}     one product
GET  /orders            recent orders
POST /orders            {"product_id":"p-100","quantity":2}
GET  /healthz           200 once started, 503 before
GET  /status            version, uptime and what the app is doing
GET  /work?ms=200       burn CPU inside this request
GET  /burn?seconds=60   burn CPU in the background
GET  /alloc?mb=64       hold more memory
GET  /free              release memory from /alloc
`, s.cfg.shopName, version)
}

func (s *server) healthz(w http.ResponseWriter, _ *http.Request) {
	if !s.ready() {
		left := s.cfg.startDelay - s.now().Sub(s.started)
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "starting", "ready_in_seconds": int(math.Ceil(left.Seconds())),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) status(w http.ResponseWriter, _ *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	s.mu.Lock()
	heldMB := 0
	for _, b := range s.held {
		heldMB += len(b) >> 20
	}
	orders := len(s.orders)
	s.mu.Unlock()
	burning := 0
	if until := s.burnUntil.Load(); until > 0 && s.now().UnixNano() < until {
		burning = int(math.Ceil(time.Duration(until - s.now().UnixNano()).Seconds()))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version":           version,
		"shop":              s.cfg.shopName,
		"ready":             s.ready(),
		"uptime_seconds":    int(s.now().Sub(s.started).Seconds()),
		"start_delay":       int(s.cfg.startDelay.Seconds()),
		"signing_key_set":   s.cfg.signingKey != "",
		"error_rate":        s.cfg.errorRate,
		"cache_mb":          s.cfg.cacheMB,
		"alloc_mb":          heldMB,
		"burn_seconds_left": burning,
		"orders":            orders,
		"heap_mb":           int(ms.HeapAlloc >> 20),
	})
}

// shop wraps the normal endpoints: 503 while starting, and ERROR_RATE's share
// of simulated 500s.
func (s *server) shop(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.ready() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "starting up, try again shortly"})
			return
		}
		if s.cfg.errorRate > 0 && s.rand() < s.cfg.errorRate {
			setLogError(r, fmt.Sprintf("simulated failure (ERROR_RATE=%g)", s.cfg.errorRate))
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		h(w, r)
	}
}

func (s *server) listProducts(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"shop": s.cfg.shopName, "products": catalog})
}

func findProduct(id string) (product, bool) {
	for _, p := range catalog {
		if p.ID == id {
			return p, true
		}
	}
	return product{}, false
}

func (s *server) getProduct(w http.ResponseWriter, r *http.Request) {
	p, ok := findProduct(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such product"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) listOrders(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	out := make([]order, len(s.orders))
	copy(out, s.orders)
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	writeJSON(w, http.StatusOK, map[string]any{"orders": out})
}

func (s *server) createOrder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `send JSON like {"product_id":"p-100","quantity":1}`})
		return
	}
	if in.Quantity == 0 {
		in.Quantity = 1
	}
	p, ok := findProduct(in.ProductID)
	if !ok || in.Quantity < 1 || in.Quantity > 100 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "unknown product_id or quantity not 1-100"})
		return
	}
	o := order{
		ID: "o-" + randomHex(6), ProductID: p.ID, Quantity: in.Quantity,
		TotalCents: p.PriceCents * in.Quantity, CreatedAt: s.now().UTC(),
	}
	if s.cfg.signingKey != "" {
		mac := hmac.New(sha256.New, []byte(s.cfg.signingKey))
		fmt.Fprintf(mac, "%s|%s|%d|%d", o.ID, o.ProductID, o.Quantity, o.TotalCents)
		o.Signature = hex.EncodeToString(mac.Sum(nil))
	}
	s.mu.Lock()
	s.orders = append(s.orders, o)
	if len(s.orders) > maxOrders {
		s.orders = s.orders[len(s.orders)-maxOrders:]
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, o)
}

func intParam(r *http.Request, name string, def, max int) (int, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > max {
		return 0, fmt.Errorf("%s must be a whole number from 0 to %d", name, max)
	}
	return n, nil
}

// itersPerMS is roughly a millisecond of work on one modern core.
const itersPerMS = 400_000

var sink atomic.Uint64

// cpuWork does a fixed amount of computation, not a fixed amount of time, so
// it takes longer when the spherelet is short of CPU: that's the latency you
// see rise when CPU is saturated.
func cpuWork(ms int) {
	x := uint64(88172645463325252)
	for i := 0; i < ms*itersPerMS; i++ {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
	}
	sink.Store(x)
}

func spin(until time.Time) {
	x := 0
	for time.Now().Before(until) {
		for i := 0; i < 10_000; i++ {
			x += i * i
		}
	}
	_ = x
}

// work burns about ms of CPU inside the request. When CPU is short it takes
// longer than ms: compare took_ms with worked_ms.
func (s *server) work(w http.ResponseWriter, r *http.Request) {
	ms, err := intParam(r, "ms", 100, maxWorkMS)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	start := time.Now()
	cpuWork(ms)
	writeJSON(w, http.StatusOK, map[string]any{"worked_ms": ms, "took_ms": time.Since(start).Milliseconds()})
}

// burn keeps `threads` goroutines busy in the background for `seconds`. A new
// call replaces the running burn; seconds=0 stops it.
func (s *server) burn(w http.ResponseWriter, r *http.Request) {
	secs, err := intParam(r, "seconds", 60, maxBurnSeconds)
	threads := 1
	if err == nil {
		threads, err = intParam(r, "threads", 1, 64)
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if threads < 1 {
		threads = 1
	}
	gen := s.burnGen.Add(1)
	if secs == 0 {
		s.burnUntil.Store(0)
		s.log.Info("cpu burn stopped", "route", "/burn")
		writeJSON(w, http.StatusOK, map[string]any{"burning": false})
		return
	}
	until := time.Now().Add(time.Duration(secs) * time.Second)
	s.burnUntil.Store(until.UnixNano())
	for i := 0; i < threads; i++ {
		go func() {
			for s.burnGen.Load() == gen && time.Now().Before(until) {
				spin(time.Now().Add(50 * time.Millisecond))
			}
		}()
	}
	s.log.Info("cpu burn started", "route", "/burn", "seconds", secs, "threads", threads)
	writeJSON(w, http.StatusAccepted, map[string]any{"burning": true, "seconds": secs, "threads": threads, "until": until.UTC()})
}

// touch writes to every page so the memory is really resident, not just
// reserved.
func touch(b []byte) {
	for i := 0; i < len(b); i += 4096 {
		b[i] = 1
	}
}

func (s *server) alloc(w http.ResponseWriter, r *http.Request) {
	mb, err := intParam(r, "mb", 64, maxAllocPerCallMB)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	b := make([]byte, mb<<20)
	touch(b)
	s.mu.Lock()
	s.held = append(s.held, b)
	total := 0
	for _, h := range s.held {
		total += len(h) >> 20
	}
	s.mu.Unlock()
	s.log.Info("memory held", "route", "/alloc", "added_mb", mb, "alloc_mb", total)
	writeJSON(w, http.StatusOK, map[string]any{"added_mb": mb, "alloc_mb": total})
}

func (s *server) free(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.held = nil
	s.mu.Unlock()
	debug.FreeOSMemory()
	s.log.Info("memory released", "route", "/free")
	writeJSON(w, http.StatusOK, map[string]any{"alloc_mb": 0})
}

// --- request logging -------------------------------------------------------

type ctxKey struct{}

type logInfo struct{ err string }

func setLogError(r *http.Request, msg string) {
	if li, ok := r.Context().Value(ctxKey{}).(*logInfo); ok {
		li.err = msg
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// logRequests writes one line per request. Passing health checks aren't
// logged (the platform calls /healthz every few seconds); failing ones are.
func (s *server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := r.Header.Get("X-Request-Id")
		if id == "" || len(id) > 64 {
			id = randomHex(8)
		}
		w.Header().Set("X-Request-Id", id)
		li := &logInfo{}
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r.WithContext(context.WithValue(r.Context(), ctxKey{}, li)))
		if r.URL.Path == "/healthz" && sw.status == http.StatusOK {
			return
		}
		level, msg := slog.LevelInfo, "request"
		switch {
		case sw.status >= 500 && r.URL.Path == "/healthz":
			level, msg = slog.LevelWarn, "health check failed: still starting"
		case sw.status >= 500:
			level, msg = slog.LevelError, "request failed"
		case sw.status >= 400:
			level = slog.LevelWarn
		}
		attrs := []any{
			"request_id", id, "method", r.Method, "route", r.URL.Path,
			"status", sw.status, "duration_ms", time.Since(start).Milliseconds(),
		}
		if li.err != "" {
			attrs = append(attrs, "error", li.err)
		}
		s.log.Log(r.Context(), level, msg, attrs...)
	})
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				if l, ok := a.Value.Any().(slog.Level); ok {
					if l >= levelFatal {
						return slog.String("level", "fatal")
					}
					return slog.String("level", strings.ToLower(l.String()))
				}
			}
			return a
		},
	}))
}

func fatal(log *slog.Logger, msg string, attrs ...any) {
	log.Log(context.Background(), levelFatal, msg, attrs...)
	os.Exit(1)
}

func main() {
	log := newLogger()
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		fatal(log, "invalid configuration: "+err.Error())
	}
	if cfg.signingKey == "" && cfg.requireSigning {
		fatal(log, "SIGNING_KEY is not set: the shop can't sign orders without it. Add SIGNING_KEY as a secret variable and redeploy",
			"missing", "SIGNING_KEY")
	}
	log.Info("starting learn-shop-api", "version", version, "shop", cfg.shopName, "port", cfg.port,
		"start_delay", int(cfg.startDelay.Seconds()), "cache_mb", cfg.cacheMB, "error_rate", cfg.errorRate,
		"signing_key_set", cfg.signingKey != "")
	s := newServer(cfg, log)

	if cfg.cacheMB > 0 {
		log.Info("warming the product cache", "cache_mb", cfg.cacheMB)
		s.cache = make([]byte, cfg.cacheMB<<20)
		touch(s.cache)
		log.Info("product cache warm", "cache_mb", cfg.cacheMB)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	if cfg.startDelay > 0 {
		log.Info("loading the catalog; /healthz answers 503 until it's done", "start_delay", int(cfg.startDelay.Seconds()))
		time.AfterFunc(cfg.startDelay, func() { log.Info("ready: catalog loaded, /healthz answers 200") })
	} else {
		log.Info("ready: catalog loaded, /healthz answers 200")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-stop
		log.Info("shutting down: finishing requests in flight")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	log.Info("listening", "port", cfg.port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fatal(log, "can't listen: "+err.Error(), "port", cfg.port)
	}
	log.Info("stopped")
}
