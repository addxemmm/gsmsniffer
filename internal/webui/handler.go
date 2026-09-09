// Package webui serves the self-contained laboratory console.
package webui

import (
	"embed"
	"net/http"
)

//go:embed static/index.html static/app.css static/app.js
var assets embed.FS

// New serves only the three public console assets; no filesystem is exposed.
func New() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		var name, contentType string
		switch r.URL.Path {
		case "/":
			name, contentType = "index.html", "text/html; charset=utf-8"
		case "/app.css":
			name, contentType = "app.css", "text/css; charset=utf-8"
		case "/app.js":
			name, contentType = "app.js", "text/javascript; charset=utf-8"
		default:
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := assets.ReadFile("static/" + name)
		if err != nil {
			http.Error(w, "asset unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodHead {
			_, _ = w.Write(body)
		}
	})
}
