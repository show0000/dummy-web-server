package api

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadValidAPIs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "apis.yaml", `
apis:
  - entrypoint: /api/v1/users
    method: GET
    description: 사용자 목록
    script: |
      return res.json(200, {users: []});
  - entrypoint: /api/v1/users
    method: POST
    scriptFile: ./create.js
`)

	apis, err := LoadAPIs(filepath.Join(dir, "apis.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apis) != 2 {
		t.Fatalf("expected 2 apis, got %d", len(apis))
	}

	if apis[0].Entrypoint != "/api/v1/users" {
		t.Errorf("expected /api/v1/users, got %s", apis[0].Entrypoint)
	}
	if apis[0].Method != "GET" {
		t.Errorf("expected GET, got %s", apis[0].Method)
	}
	if apis[0].Description != "사용자 목록" {
		t.Errorf("expected description, got %s", apis[0].Description)
	}
}

func TestLoadMethodUppercase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "apis.yaml", `
apis:
  - entrypoint: /test
    method: post
    script: "return res.json(200, {});"
`)

	apis, err := LoadAPIs(filepath.Join(dir, "apis.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apis[0].Method != "POST" {
		t.Errorf("expected POST, got %s", apis[0].Method)
	}
}

func TestLoadMissingEntrypoint(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "apis.yaml", `
apis:
  - method: GET
    script: "return res.json(200, {});"
`)

	_, err := LoadAPIs(filepath.Join(dir, "apis.yaml"))
	if err == nil {
		t.Fatal("expected error for missing entrypoint")
	}
}

func TestLoadMissingMethod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "apis.yaml", `
apis:
  - entrypoint: /test
    script: "return res.json(200, {});"
`)

	_, err := LoadAPIs(filepath.Join(dir, "apis.yaml"))
	if err == nil {
		t.Fatal("expected error for missing method")
	}
}

func TestLoadInvalidMethod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "apis.yaml", `
apis:
  - entrypoint: /test
    method: INVALID
    script: "return res.json(200, {});"
`)

	_, err := LoadAPIs(filepath.Join(dir, "apis.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid method")
	}
}

func TestLoadMissingScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "apis.yaml", `
apis:
  - entrypoint: /test
    method: GET
`)

	_, err := LoadAPIs(filepath.Join(dir, "apis.yaml"))
	if err == nil {
		t.Fatal("expected error for missing script and scriptFile")
	}
}

func TestLoadEntrypointNoSlash(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "apis.yaml", `
apis:
  - entrypoint: test
    method: GET
    script: "return res.json(200, {});"
`)

	_, err := LoadAPIs(filepath.Join(dir, "apis.yaml"))
	if err == nil {
		t.Fatal("expected error for entrypoint without leading /")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := LoadAPIs("/nonexistent/apis.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "apis.yaml", "{{invalid")

	_, err := LoadAPIs(filepath.Join(dir, "apis.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestAuthEnabled(t *testing.T) {
	// nil (default) → true
	api := APIDefinition{}
	if !api.AuthEnabled() {
		t.Error("expected AuthEnabled true for nil")
	}

	// explicit true
	tr := true
	api.Auth = &tr
	if !api.AuthEnabled() {
		t.Error("expected AuthEnabled true")
	}

	// explicit false
	fa := false
	api.Auth = &fa
	if api.AuthEnabled() {
		t.Error("expected AuthEnabled false")
	}
}

func TestResolveScriptInline(t *testing.T) {
	api := APIDefinition{Script: "return res.json(200, {});"}
	script, err := api.ResolveScript("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if script != "return res.json(200, {});" {
		t.Errorf("unexpected script: %s", script)
	}
}

func TestResolveScriptFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "handler.js", "return res.json(200, {ok: true});")

	api := APIDefinition{ScriptFile: "handler.js"}
	script, err := api.ResolveScript(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if script != "return res.json(200, {ok: true});" {
		t.Errorf("unexpected script: %s", script)
	}
}

func TestResolveScriptFileMissing(t *testing.T) {
	api := APIDefinition{ScriptFile: "/nonexistent/handler.js"}
	_, err := api.ResolveScript("")
	if err == nil {
		t.Fatal("expected error for missing script file")
	}
}

func TestResolveScriptInlinePriority(t *testing.T) {
	api := APIDefinition{Script: "inline", ScriptFile: "file.js"}
	script, err := api.ResolveScript("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if script != "inline" {
		t.Error("expected inline script to take priority over scriptFile")
	}
}

func TestLoadValidation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "apis.yaml", `
apis:
  - entrypoint: /api/v1/users
    method: POST
    script: "return res.json(201, {});"
    validation:
      schema:
        type: object
        required: [name]
        properties:
          name:
            type: string
`)

	apis, err := LoadAPIs(filepath.Join(dir, "apis.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apis[0].Validation == nil {
		t.Fatal("expected validation to be present")
	}
	if apis[0].Validation.Schema["type"] != "object" {
		t.Errorf("expected schema type object, got %v", apis[0].Validation.Schema["type"])
	}
}
