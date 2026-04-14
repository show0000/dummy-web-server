package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"dummy-web-server/src/internal/api"
	"dummy-web-server/src/internal/config"
	"dummy-web-server/src/internal/router"
)

func buildRouterFromConfig(cfg *config.Config) (*router.Router, error) {
	r := router.New()

	// Health check
	r.Handle("GET", "/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Load and register dynamic APIs
	registered, err := api.RegisterAPIs(r, cfg.Paths.APIs, cfg.Paths.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to register APIs: %w", err)
	}
	log.Printf("registered %d API endpoint(s)", len(registered))

	return r, nil
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("config load failed: %w", err)
	}

	r, err := buildRouterFromConfig(cfg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("server starting on %s", addr)
	return http.ListenAndServe(addr, r)
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config.yaml")
	flag.Parse()

	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
}
