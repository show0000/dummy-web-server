package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dummy-web-server/src/internal/config"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// setupServer creates a test server with config.yaml + apis.yaml and returns the server URL.
func setupServer(t *testing.T, configYAML, apisYAML string) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", configYAML)
	writeFile(t, dir, "apis.yaml", apisYAML)

	cfgPath := filepath.Join(dir, "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	// Override apis path to temp dir
	cfg.Paths.APIs = filepath.Join(dir, "apis.yaml")

	r, err := buildRouterFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}

	return httptest.NewServer(r)
}

func TestRunFailsWithMissingConfig(t *testing.T) {
	err := run("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestRunFailsWithInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", `server:
  port: 0
`)
	err := run(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis: []`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestDynamicAPIJsonResponse(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis:
  - entrypoint: /api/greet/{name}
    method: GET
    script: |
      res.json(200, {message: "hello " + req.params.name});
`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(srv.URL + "/api/greet/world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"hello world"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestDynamicAPIWithQueryParams(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis:
  - entrypoint: /api/search
    method: GET
    script: |
      res.json(200, {q: req.query.q, page: req.query.page});
`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(srv.URL + "/api/search?q=test&page=2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"test"`) || !strings.Contains(string(body), `"2"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestDynamicAPIWithRequestBody(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis:
  - entrypoint: /api/echo
    method: POST
    script: |
      res.json(200, {received: req.body.message});
`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(srv.URL+"/api/echo", "application/json", strings.NewReader(`{"message":"ping"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"ping"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestDynamicAPIWithValidation(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis:
  - entrypoint: /api/users
    method: POST
    script: |
      res.json(201, {name: req.body.name});
    validation:
      schema:
        type: object
        required: [name]
        properties:
          name:
            type: string
`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// Valid request
	resp, err := client.Post(srv.URL+"/api/users", "application/json", strings.NewReader(`{"name":"Alice"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	// Invalid request (missing required field)
	resp2, err := client.Post(srv.URL+"/api/users", "application/json", strings.NewReader(`{"age":30}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 400 {
		t.Errorf("expected 400 for validation failure, got %d", resp2.StatusCode)
	}
}

func TestDynamicAPIWithSetHeader(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis:
  - entrypoint: /api/custom-header
    method: GET
    script: |
      res.setHeader("X-Custom", "hello");
      res.json(200, {});
`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(srv.URL + "/api/custom-header")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Custom") != "hello" {
		t.Errorf("expected X-Custom=hello, got %s", resp.Header.Get("X-Custom"))
	}
}

func TestDynamicAPIConditionalResponse(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis:
  - entrypoint: /api/items/{id}
    method: GET
    script: |
      if (req.params.id === "0") {
        res.json(404, {error: "not found"});
      } else {
        res.json(200, {id: req.params.id});
      }
`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// Found
	resp, _ := client.Get(srv.URL + "/api/items/5")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Not found
	resp2, _ := client.Get(srv.URL + "/api/items/0")
	defer resp2.Body.Close()
	if resp2.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp2.StatusCode)
	}
}

func TestDynamicAPIExternalScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", `server:
  port: 8080
paths:
  apis: "./apis.yaml"
`)
	writeFile(t, dir, "handler.js", `res.json(200, {source: "external"});`)
	writeFile(t, dir, "apis.yaml", `apis:
  - entrypoint: /api/ext
    method: GET
    scriptFile: ./handler.js
`)

	cfgPath := filepath.Join(dir, "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Paths.APIs = filepath.Join(dir, "apis.yaml")

	r, err := buildRouterFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}

	srv := httptest.NewServer(r)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(srv.URL + "/api/ext")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"external"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestNotFoundRoute(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis: []`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, _ := client.Get(srv.URL + "/nonexistent")
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
