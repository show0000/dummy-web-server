package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.JWT.Enabled != false {
		t.Error("expected JWT disabled by default")
	}
	if cfg.Paths.APIs != "./apis.yaml" {
		t.Errorf("expected default apis path ./apis.yaml, got %s", cfg.Paths.APIs)
	}
}

func TestLoadValidConfig(t *testing.T) {
	yaml := `
server:
  port: 9090
jwt:
  enabled: true
  secret: "my-secret"
  accessTokenExpiry: "30m"
  refreshTokenExpiry: "72h"
paths:
  apis: "./my-apis.yaml"
  storage: "./data"
  scripts: "./js"
`
	path := writeTestConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if !cfg.JWT.Enabled {
		t.Error("expected JWT enabled")
	}
	if cfg.JWT.Secret != "my-secret" {
		t.Errorf("expected secret my-secret, got %s", cfg.JWT.Secret)
	}
	if cfg.Paths.APIs != "./my-apis.yaml" {
		t.Errorf("expected apis path ./my-apis.yaml, got %s", cfg.Paths.APIs)
	}
	if cfg.Paths.Storage != "./data" {
		t.Errorf("expected storage path ./data, got %s", cfg.Paths.Storage)
	}
	if cfg.Paths.Scripts != "./js" {
		t.Errorf("expected scripts path ./js, got %s", cfg.Paths.Scripts)
	}
}

func TestLoadPartialConfigUsesDefaults(t *testing.T) {
	yaml := `
server:
  port: 3000
`
	path := writeTestConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Server.Port)
	}
	// JWT should keep defaults
	if cfg.JWT.Enabled != false {
		t.Error("expected JWT disabled by default")
	}
	if cfg.Paths.APIs != "./apis.yaml" {
		t.Errorf("expected default apis path, got %s", cfg.Paths.APIs)
	}
}

func TestLoadInvalidPort(t *testing.T) {
	yaml := `
server:
  port: 99999
`
	path := writeTestConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestLoadInvalidPortZero(t *testing.T) {
	yaml := `
server:
  port: 0
`
	path := writeTestConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for port 0")
	}
}

func TestLoadJWTEnabledWithoutSecret(t *testing.T) {
	yaml := `
jwt:
  enabled: true
  secret: ""
`
	path := writeTestConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for JWT enabled without secret")
	}
}

func TestLoadInvalidTokenExpiry(t *testing.T) {
	yaml := `
jwt:
  enabled: true
  secret: "test"
  accessTokenExpiry: "not-a-duration"
`
	path := writeTestConfig(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid accessTokenExpiry")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := writeTestConfig(t, "{{invalid yaml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestJWTTokenDurations(t *testing.T) {
	cfg := DefaultConfig()

	accessDur, err := cfg.JWT.AccessTokenDuration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if accessDur != 15*time.Minute {
		t.Errorf("expected 15m, got %v", accessDur)
	}

	refreshDur, err := cfg.JWT.RefreshTokenDuration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if refreshDur != 168*time.Hour {
		t.Errorf("expected 168h, got %v", refreshDur)
	}
}
