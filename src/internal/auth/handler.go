package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"dummy-web-server/src/internal/router"
)

// RegisterRoutes registers /_auth/* endpoints on the router.
func RegisterRoutes(r *router.Router, svc *JWTService) {
	r.Handle("POST", "/_auth/login", loginHandler(svc))
	r.Handle("POST", "/_auth/logout", logoutHandler(svc))
	r.Handle("POST", "/_auth/refresh", refreshHandler(svc))
}

func loginHandler(svc *JWTService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if req.Username == "" || req.Password == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
			return
		}

		// Mock server: accept any non-empty username/password
		pair, err := svc.GenerateTokenPair(req.Username)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, pair)
	}
}

func logoutHandler(svc *JWTService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		accessToken := extractBearerToken(r)
		svc.Logout(accessToken, req.RefreshToken)

		writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
	}
}

func refreshHandler(svc *JWTService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "refreshToken is required"})
			return
		}

		pair, err := svc.Refresh(req.RefreshToken)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, pair)
	}
}

// Middleware returns an HTTP middleware that validates JWT Bearer tokens.
// It skips paths starting with /_auth/ and /_explorer, and APIs with auth:false.
func Middleware(svc *JWTService, skipAuth func(method, path string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Skip auth endpoints, explorer, and health check
			if strings.HasPrefix(path, "/_auth/") || strings.HasPrefix(path, "/_explorer") || path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Skip if this API has auth: false
			if skipAuth != nil && skipAuth(r.Method, path) {
				next.ServeHTTP(w, r)
				return
			}

			token := extractBearerToken(r)
			if token == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization token"})
				return
			}

			_, err := svc.ValidateAccessToken(token)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
