// notes is the volume lab app for ComputeSphere Learn (lesson 5.5.4). It keeps
// one note in a file, /data/note.txt by default. POST /note saves the request
// body there and GET /note returns it. Mount a volume at /data and the note
// survives a redeploy; without one it's lost with the spherelet.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var version = "dev"

const maxNote = 16 << 10 // bytes accepted per note

type server struct {
	path string // the note file
	mu   sync.Mutex
	log  *slog.Logger
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// read returns the saved note, or "" when none has been saved yet.
func (s *server) read() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(b), err
}

// save replaces the note: it writes a temporary file next to it, then renames
// it over the old one, so a reader never sees half a note.
func (s *server) save(note []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, note, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /note", func(w http.ResponseWriter, _ *http.Request) {
		note, err := s.read()
		if err != nil {
			s.log.Error("can't read note", "path", s.path, "error", err.Error())
			http.Error(w, "can't read the note", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, note)
	})
	mux.HandleFunc("POST /note", func(w http.ResponseWriter, r *http.Request) {
		note, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxNote))
		if err != nil {
			http.Error(w, "a note can be at most 16 KB", http.StatusRequestEntityTooLarge)
			return
		}
		if err := s.save(note); err != nil {
			s.log.Error("can't save note", "path", s.path, "error", err.Error())
			http.Error(w, "can't save the note", http.StatusInternalServerError)
			return
		}
		s.log.Info("note saved", "path", s.path, "bytes", len(note))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"service": "learn-notes",
			"version": version,
			"note":    s.path,
			"usage":   "POST /note saves the body; GET /note returns it",
		})
	})
	return s.logRequests(mux)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if r.URL.Path == "/healthz" {
			return // health checks would drown out everything else
		}
		s.log.Info("request", "method", r.Method, "path", r.URL.Path,
			"status", rec.status, "duration_ms", time.Since(start).Milliseconds())
	})
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dir := os.Getenv("DATA_DIR")
	if dir == "" {
		dir = "/data"
	}
	s := &server{path: filepath.Join(dir, "note.txt"), log: log}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	log.Info("learn-notes listening", "version", version, "port", port, "note", s.path)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("can't listen", "error", err.Error())
		os.Exit(1)
	}
}
