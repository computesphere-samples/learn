// bloated-app is a tiny web app with a deliberately bloated Dockerfile.
// The app is fine; the image is the problem. See README.md.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Read the port from the environment so the platform can choose it.
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "Hello from bloated-app. The code is small; the image is not.")
	})
	// Health checks call this. Keep it cheap: no database, no external calls.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Listen on 0.0.0.0 (all interfaces), not localhost. Inside a container,
	// localhost is only the container itself.
	addr := "0.0.0.0:" + port
	log.Printf("bloated-app listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
