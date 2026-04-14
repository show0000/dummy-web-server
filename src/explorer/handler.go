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

// ExplorerConfig holds server configuration exposed to the explorer UI.
type ExplorerConfig struct {
	JWTEnabled bool `json:"jwtEnabled"`
}

// Handler returns an http.Handler that serves the explorer UI and API list.
func Handler(apis []APIInfo, cfg ExplorerConfig) http.HandlerFunc {
	staticFS, _ := fs.Sub(StaticFiles, "static")
	apisJSON, _ := json.Marshal(apis)
	cfgJSON, _ := json.Marshal(cfg)
	indexHTML, _ := fs.ReadFile(StaticFiles, "static/index.html")

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/_explorer/apis" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(apisJSON)
			return
		}

		if path == "/_explorer/config" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(cfgJSON)
			return
		}

		if path == "/_explorer" || path == "/_explorer/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexHTML)
			return
		}

		r.URL.Path = strings.TrimPrefix(path, "/_explorer")
		http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
	}
}
