package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"dummy-web-server/src/explorer"
	"dummy-web-server/src/internal/api"
	"dummy-web-server/src/internal/auth"
	"dummy-web-server/src/internal/config"
	"dummy-web-server/src/internal/router"
	"dummy-web-server/src/internal/utils"
)

func buildRouterFromConfig(cfg *config.Config) (http.Handler, error) {
	r := router.New()

	// Health check
	r.Handle("GET", "/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Utility endpoints
	r.Handle("POST", "/_utils/schema", utils.SchemaHandler())

	// Load and register dynamic APIs
	registered, err := api.RegisterAPIs(r, cfg.Paths.APIs, cfg.Paths.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to register APIs: %w", err)
	}
	log.Printf("registered %d API endpoint(s)", len(registered))

	// API Explorer
	apiInfos := make([]explorer.APIInfo, len(registered))
	for i, reg := range registered {
		apiInfos[i] = explorer.APIInfo{
			Entrypoint:  reg.Definition.Entrypoint,
			Method:      reg.Definition.Method,
			Description: reg.Definition.Description,
			Auth:        reg.Definition.AuthEnabled(),
		}
	}
	explorerCfg := explorer.ExplorerConfig{JWTEnabled: cfg.JWT.Enabled}
	explorerHandler := explorer.Handler(apiInfos, explorerCfg)
	r.Handle("GET", "/_explorer", explorerHandler)
	r.Handle("GET", "/_explorer/apis", explorerHandler)
	r.Handle("GET", "/_explorer/config", explorerHandler)
	r.Handle("GET", "/_explorer/style.css", explorerHandler)
	r.Handle("GET", "/_explorer/app.js", explorerHandler)

	// JWT authentication
	if cfg.JWT.Enabled {
		accessExpiry, _ := cfg.JWT.AccessTokenDuration()
		refreshExpiry, _ := cfg.JWT.RefreshTokenDuration()
		jwtSvc := auth.NewJWTService(cfg.JWT.Secret, accessExpiry, refreshExpiry)

		auth.RegisterRoutes(r, jwtSvc)
		log.Printf("JWT enabled (access: %s, refresh: %s)", accessExpiry, refreshExpiry)

		// Build skip function from registered APIs with auth: false
		skipAuth := buildSkipAuthFunc(registered)

		middleware := auth.Middleware(jwtSvc, skipAuth)
		return router.LoggerMiddleware(middleware(r)), nil
	}

	return router.LoggerMiddleware(r), nil
}

func buildSkipAuthFunc(registered []api.RegisteredAPI) func(method, path string) bool {
	type routeKey struct {
		method string
		path   string
	}
	skipRoutes := make(map[routeKey]bool)
	for _, reg := range registered {
		if !reg.Definition.AuthEnabled() {
			skipRoutes[routeKey{reg.Definition.Method, reg.Definition.Entrypoint}] = true
		}
	}

	return func(method, path string) bool {
		// Exact match first
		if skipRoutes[routeKey{method, path}] {
			return true
		}
		// Check path variable patterns
		for key := range skipRoutes {
			if key.method != method {
				continue
			}
			if matchPath(key.path, path) {
				return true
			}
		}
		return false
	}
}

func matchPath(pattern, path string) bool {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return false
	}
	for i, p := range patternParts {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			continue
		}
		if p != pathParts[i] {
			return false
		}
	}
	return true
}

type cliFlags struct {
	configPath  string
	port        int
	enableLogin string
}

func run(flags cliFlags) error {
	cfg, err := config.Load(flags.configPath)
	if err != nil {
		return fmt.Errorf("config load failed: %w", err)
	}

	// CLI flag overrides
	if flags.port > 0 {
		cfg.Server.Port = flags.port
	}
	if flags.enableLogin == "y" {
		cfg.JWT.Enabled = true
	} else if flags.enableLogin == "n" {
		cfg.JWT.Enabled = false
	}

	handler, err := buildRouterFromConfig(cfg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("server starting on %s", addr)
	return http.ListenAndServe(addr, handler)
}

func main() {
	flags := cliFlags{}
	flag.StringVar(&flags.configPath, "config", "config.yaml", "path to config.yaml")
	flag.IntVar(&flags.port, "port", 0, "override server port")
	flag.StringVar(&flags.enableLogin, "enable-login", "", "enable JWT login (y|n)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `dummy-web-server - YAML 기반 동적 Mock API 서버

Usage:
  dummy-web-server [options]

Options:
  --config string        config.yaml 경로 (default "config.yaml")
  --port int             서버 포트 (config.yaml 오버라이드)
  --enable-login y|n     JWT 로그인 활성화 (config.yaml 오버라이드)
  --help                 도움말 표시

Examples:
  dummy-web-server
  dummy-web-server --port 3000
  dummy-web-server --port 9090 --enable-login y
  dummy-web-server --config ./my-config.yaml

Endpoints:
  /health              서버 상태 확인
  /_explorer           API Explorer (웹 UI)
  /_utils/schema       JSON → JSON Schema 변환
  /_auth/login         JWT 로그인 (--enable-login y 필요)
  /_auth/logout        JWT 로그아웃
  /_auth/refresh       Refresh Token 갱신
`)
	}

	flag.Parse()

	if err := run(flags); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
}
