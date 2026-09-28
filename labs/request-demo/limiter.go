package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Per-client rate limiting, plus a global ceiling as a backstop. The service
// sits behind the platform's edge, so the client's address arrives in a
// forwarding header; the connection's own address would be the edge's and
// would throttle everyone together.

type bucket struct {
	tokens float64
	last   time.Time
}

type limiter struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64
	clients map[string]*bucket
	global  *bucket
	gRate   float64
	gBurst  float64
	now     func() time.Time
}

func newLimiter(perMinute, burst, globalPerSecond int) *limiter {
	return &limiter{
		rate:    float64(perMinute) / 60,
		burst:   float64(burst),
		clients: map[string]*bucket{},
		global:  &bucket{tokens: float64(globalPerSecond) * 2},
		gRate:   float64(globalPerSecond),
		gBurst:  float64(globalPerSecond) * 2,
		now:     time.Now,
	}
}

func take(b *bucket, now time.Time, rate, burst float64) bool {
	if !b.last.IsZero() {
		b.tokens += now.Sub(b.last).Seconds() * rate
		if b.tokens > burst {
			b.tokens = burst
		}
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *limiter) allow(client string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if !take(l.global, now, l.gRate, l.gBurst) {
		return false
	}
	b, ok := l.clients[client]
	if !ok {
		// Bound memory: forget idle clients once the table grows.
		if len(l.clients) > 50_000 {
			for k, v := range l.clients {
				if now.Sub(v.last) > 10*time.Minute {
					delete(l.clients, k)
				}
			}
		}
		b = &bucket{tokens: l.burst}
		l.clients[client] = b
	}
	return take(b, now, l.rate, l.burst)
}

// clientIP prefers the edge's client header, then the first forwarded
// address, then the connection itself.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// protect adds safety headers to every response and rate-limits everything
// except the health check, which the platform itself calls.
func protect(l *limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		if r.URL.Path != "/healthz" && !l.allow(clientIP(r)) {
			h.Set("Retry-After", "60")
			http.Error(w, "Too many requests. Try again in a minute.", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
