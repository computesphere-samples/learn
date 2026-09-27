// hello-web is the sample web service for ComputeSphere Learn labs.
// It serves a page on / and a health check on /healthz, on $PORT (8080).
// Built with -ldflags "-X main.broken=true" it fails its health check:
// that build is the "bad release" in the rollback lab.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var (
	version = "dev"
	broken  = "false"
)

const page = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>hello-web</title>
<style>body{margin:0;min-height:100vh;display:grid;place-items:center;background:#0a0a0b;color:#ecedee;font-family:system-ui,sans-serif}
main{text-align:center;padding:24px}h1{font-weight:400;font-size:clamp(32px,6vw,56px);margin:0 0 12px;letter-spacing:-.02em}
p{color:#9ca3ac;margin:0}code{color:#8fd3e3}</style></head>
<body><main><h1>Hello from ComputeSphere</h1><p>hello-web <code>%s</code> is live.</p></main></body></html>`

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, page, version)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		if broken == "true" {
			http.Error(w, "unhealthy: this is the bad release", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	log.Printf("hello-web %s listening on :%s", version, port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
