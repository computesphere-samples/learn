// request-demo is the service learners inspect in the Cloud foundations lab
// "Read a real request". It answers every request normally, and when the URL
// carries a learner's code (?code=…) it adds an X-Learn-Nonce header derived
// from that code. The lab page asks for the nonce back and checks it with
// /verify, which proves the learner read a real response header without any
// account.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var version = "dev"

// A code is what the lab page shows the learner: lowercase letters and digits.
var codePattern = regexp.MustCompile(`^[a-z0-9]{6,32}$`)

const nonceLength = 10

type server struct {
	secret []byte
}

// nonce derives the value for a code. HMAC with a server-side secret, so a
// nonce can't be worked out without asking the server.
func (s *server) nonce(code string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(code))
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(mac.Sum(nil))
	return strings.ToLower(enc[:nonceLength])
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/verify", s.verify)
	mux.HandleFunc("/", s.index)
	return mux
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	code := r.URL.Query().Get("code")
	if code != "" && codePattern.MatchString(code) {
		w.Header().Set("X-Learn-Nonce", s.nonce(code))
		fmt.Fprintf(w, "Hello from request-demo %s.\nYour nonce is in the X-Learn-Nonce response header. Read it with curl -v and your code.\n", version)
		return
	}
	fmt.Fprintf(w, "Hello from request-demo %s.\nAdd ?code=<your code from the lesson> to get your nonce header.\n", version)
}

func (s *server) verify(w http.ResponseWriter, r *http.Request) {
	// Anyone may call this from the lesson page; the answer reveals nothing.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	code := r.URL.Query().Get("code")
	got := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("nonce")))
	ok := codePattern.MatchString(code) && hmac.Equal([]byte(got), []byte(s.nonce(code)))
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": ok})
}

func main() {
	secret := os.Getenv("NONCE_SECRET")
	if len(secret) < 32 {
		log.Fatal("NONCE_SECRET must be set to at least 32 characters")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr: ":" + port,
		// 60 requests a minute per client (bursts of 20), 50 a second overall.
		Handler:           protect(newLimiter(60, 20, 50), (&server{secret: []byte(secret)}).routes()),
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	log.Printf("request-demo %s listening on :%s", version, port)
	log.Fatal(srv.ListenAndServe())
}
