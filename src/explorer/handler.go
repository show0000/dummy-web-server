package explorer

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

// APIInfo is the public representation of a registered API for the explorer UI.
type APIInfo struct {
	Entrypoint  string `json:"entrypoint"`
	Method      string `json:"method"`
	Description string `json:"description"`
	Auth        bool   `json:"auth"`
}

// Handler returns an http.Handler that serves the explorer UI and API list.
// - GET /_explorer → index.html (and static assets)
// - GET /_explorer/apis → JSON list of registered APIs
func Handler(apis []APIInfo) http.HandlerFunc {
	staticFS, _ := fs.Sub(StaticFiles, "static")
	apisJSON, _ := json.Marshal(apis)

	// Pre-read index.html
	indexHTML, _ := fs.ReadFile(StaticFiles, "static/index.html")

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// API list endpoint
		if path == "/_explorer/apis" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(apisJSON)
			return
		}

		// Serve index.html directly for root path
		if path == "/_explorer" || path == "/_explorer/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexHTML)
			return
		}

		// Serve other static files
		r.URL.Path = strings.TrimPrefix(path, "/_explorer")
		http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
	}
}
