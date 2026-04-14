package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	// Override paths to temp dir
	cfg.Paths.APIs = filepath.Join(dir, "apis.yaml")
	cfg.Paths.Storage = filepath.Join(dir, "storage")

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

func TestFileUpload(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis:
  - entrypoint: /api/upload
    method: POST
    script: |
      if (req.files.length === 0) {
        res.json(400, {error: "no files"});
      } else {
        res.json(200, {
          fileName: req.files[0].fileName,
          size: req.files[0].size,
          savedPath: req.files[0].savedPath
        });
      }
`,
	)
	defer srv.Close()

	// Create multipart body
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("hello file content"))
	writer.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(srv.URL+"/api/upload", writer.FormDataContentType(), &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"test.txt"`) {
		t.Errorf("expected fileName test.txt in body: %s", body)
	}
}

func TestFileDownload(t *testing.T) {
	dir := t.TempDir()

	// Create a file to download
	storageDir := filepath.Join(dir, "storage")
	os.MkdirAll(storageDir, 0755)
	writeFile(t, storageDir, "hello.txt", "hello world content")

	writeFile(t, dir, "config.yaml", `server:
  port: 8080
`)
	writeFile(t, dir, "apis.yaml", fmt.Sprintf(`apis:
  - entrypoint: /api/download/{fileName}
    method: GET
    script: |
      var filePath = "%s/" + req.params.fileName;
      res.file(filePath);
`, strings.ReplaceAll(storageDir, `\`, `\\`)))

	cfgPath := filepath.Join(dir, "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Paths.APIs = filepath.Join(dir, "apis.yaml")
	cfg.Paths.Storage = storageDir

	r, err := buildRouterFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}

	srvr := httptest.NewServer(r)
	defer srvr.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(srvr.URL + "/api/download/hello.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello world content" {
		t.Errorf("expected 'hello world content', got '%s'", body)
	}
}

// --- JWT Integration Tests ---

func setupJWTServer(t *testing.T, apisYAML string) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", `
server:
  port: 8080
jwt:
  enabled: true
  secret: "integration-test-secret"
  accessTokenExpiry: "15m"
  refreshTokenExpiry: "168h"
`)
	writeFile(t, dir, "apis.yaml", apisYAML)

	cfgPath := filepath.Join(dir, "config.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Paths.APIs = filepath.Join(dir, "apis.yaml")
	cfg.Paths.Storage = filepath.Join(dir, "storage")

	handler, err := buildRouterFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}
	return httptest.NewServer(handler)
}

func jwtLogin(t *testing.T, srvURL string) (accessToken, refreshToken string) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(srvURL+"/_auth/login", "application/json",
		strings.NewReader(`{"username":"testuser","password":"testpass"}`))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("login failed with %d: %s", resp.StatusCode, body)
	}
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	return result["accessToken"], result["refreshToken"]
}

func TestJWTLoginAndAccessProtectedAPI(t *testing.T) {
	srv := setupJWTServer(t, `apis:
  - entrypoint: /api/secret
    method: GET
    script: |
      res.json(200, {data: "protected"});
`)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// Without token → 401
	resp, _ := client.Get(srv.URL + "/api/secret")
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}

	// Login
	accessToken, _ := jwtLogin(t, srv.URL)

	// With token → 200
	req, _ := http.NewRequest("GET", srv.URL+"/api/secret", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp2, _ := client.Do(req)
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("expected 200 with token, got %d", resp2.StatusCode)
	}
	body, _ := io.ReadAll(resp2.Body)
	if !strings.Contains(string(body), `"protected"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestJWTAuthFalseSkipsAuth(t *testing.T) {
	srv := setupJWTServer(t, `apis:
  - entrypoint: /api/public
    method: GET
    auth: false
    script: |
      res.json(200, {data: "public"});
  - entrypoint: /api/private
    method: GET
    script: |
      res.json(200, {data: "private"});
`)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// Public API without token → 200
	resp, _ := client.Get(srv.URL + "/api/public")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for auth:false API, got %d", resp.StatusCode)
	}

	// Private API without token → 401
	resp2, _ := client.Get(srv.URL + "/api/private")
	resp2.Body.Close()
	if resp2.StatusCode != 401 {
		t.Errorf("expected 401 for protected API, got %d", resp2.StatusCode)
	}
}

func TestJWTRefreshTokenRotation(t *testing.T) {
	srv := setupJWTServer(t, `apis: []`)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	_, refreshToken := jwtLogin(t, srv.URL)

	// Refresh → new tokens
	resp, _ := client.Post(srv.URL+"/_auth/refresh", "application/json",
		strings.NewReader(fmt.Sprintf(`{"refreshToken":"%s"}`, refreshToken)))
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 on refresh, got %d: %s", resp.StatusCode, body)
	}

	// Old refresh token should be rejected (rotation)
	resp2, _ := client.Post(srv.URL+"/_auth/refresh", "application/json",
		strings.NewReader(fmt.Sprintf(`{"refreshToken":"%s"}`, refreshToken)))
	resp2.Body.Close()
	if resp2.StatusCode != 401 {
		t.Errorf("expected 401 for reused refresh token, got %d", resp2.StatusCode)
	}
}

func TestJWTLogout(t *testing.T) {
	srv := setupJWTServer(t, `apis:
  - entrypoint: /api/data
    method: GET
    script: |
      res.json(200, {ok: true});
`)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	accessToken, refreshToken := jwtLogin(t, srv.URL)

	// Logout
	logoutBody := fmt.Sprintf(`{"refreshToken":"%s"}`, refreshToken)
	req, _ := http.NewRequest("POST", srv.URL+"/_auth/logout", strings.NewReader(logoutBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(req)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 on logout, got %d", resp.StatusCode)
	}

	// Access token should be blacklisted
	req2, _ := http.NewRequest("GET", srv.URL+"/api/data", nil)
	req2.Header.Set("Authorization", "Bearer "+accessToken)
	resp2, _ := client.Do(req2)
	resp2.Body.Close()
	if resp2.StatusCode != 401 {
		t.Errorf("expected 401 after logout, got %d", resp2.StatusCode)
	}
}

func TestJWTHealthSkipsAuth(t *testing.T) {
	srv := setupJWTServer(t, `apis: []`)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, _ := client.Get(srv.URL + "/health")
	defer resp.Body.Close()
	// Health should work without token even with JWT enabled
	// (it's not under /_auth/ but it's also not a dynamic API, so middleware may block it)
	// Let's check the actual behavior
	if resp.StatusCode == 401 {
		t.Error("health endpoint should not require JWT")
	}
}

func TestUtilsSchemaEndpoint(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis: []`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(srv.URL+"/_utils/schema", "application/json",
		strings.NewReader(`{"name":"Alice","age":30,"tags":["go","mock"]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["type"] != "object" {
		t.Errorf("expected type object, got %v", result["type"])
	}

	props := result["properties"].(map[string]interface{})
	nameSchema := props["name"].(map[string]interface{})
	if nameSchema["type"] != "string" {
		t.Errorf("expected name string, got %v", nameSchema["type"])
	}
	tagsSchema := props["tags"].(map[string]interface{})
	if tagsSchema["type"] != "array" {
		t.Errorf("expected tags array, got %v", tagsSchema["type"])
	}
}

func TestExplorerPage(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis:
  - entrypoint: /api/hello
    method: GET
    description: Hello API
    script: |
      res.json(200, {msg: "hello"});
`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// GET /_explorer → HTML
	resp, err := client.Get(srv.URL + "/_explorer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "API Explorer") {
		t.Error("expected HTML page with 'API Explorer'")
	}
}

func TestExplorerAPIs(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis:
  - entrypoint: /api/hello
    method: GET
    description: Hello API
    script: |
      res.json(200, {msg: "hello"});
  - entrypoint: /api/data
    method: POST
    auth: false
    script: |
      res.json(200, {});
`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// GET /_explorer/apis → JSON list
	resp, err := client.Get(srv.URL + "/_explorer/apis")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var apis []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&apis)

	if len(apis) != 2 {
		t.Fatalf("expected 2 APIs, got %d", len(apis))
	}
	if apis[0]["entrypoint"] != "/api/hello" {
		t.Errorf("expected /api/hello, got %v", apis[0]["entrypoint"])
	}
	if apis[0]["method"] != "GET" {
		t.Errorf("expected GET, got %v", apis[0]["method"])
	}
	if apis[0]["description"] != "Hello API" {
		t.Errorf("expected 'Hello API', got %v", apis[0]["description"])
	}
	if apis[1]["auth"] != false {
		t.Errorf("expected auth false for second API")
	}
}

func TestExplorerStaticAssets(t *testing.T) {
	srv := setupServer(t,
		`server:
  port: 8080`,
		`apis: []`,
	)
	defer srv.Close()

	client := &http.Client{Timeout: 2 * time.Second}

	// CSS
	resp, _ := client.Get(srv.URL + "/_explorer/style.css")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for style.css, got %d", resp.StatusCode)
	}

	// JS
	resp2, _ := client.Get(srv.URL + "/_explorer/app.js")
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Errorf("expected 200 for app.js, got %d", resp2.StatusCode)
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
