// sample-api is the sample API for ComputeSphere Learn labs.
// GET /hello returns the GREETING environment variable, so learners can see
// configuration change without a rebuild. API_KEY is only reported as set or
// not set; its value is never returned.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

var version = "dev"

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, _ *http.Request) {
		greeting := os.Getenv("GREETING")
		if greeting == "" {
			greeting = "Hello, world"
		}
		writeJSON(w, map[string]any{"message": greeting, "api_key_set": os.Getenv("API_KEY") != ""})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"service": "sample-api", "version": version})
	})
	log.Printf("sample-api %s listening on :%s", version, port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
