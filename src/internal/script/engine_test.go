package script

import (
	"net/http/httptest"
	"testing"
)

func TestCompileValid(t *testing.T) {
	_, err := Compile("test", `res.json(200, {ok: true});`)
	if err != nil {
		t.Fatalf("unexpected compile error: %v", err)
	}
}

func TestCompileInvalid(t *testing.T) {
	_, err := Compile("test", `this is not valid javascript %%%`)
	if err == nil {
		t.Fatal("expected compile error for invalid script")
	}
}

func TestExecuteResJson(t *testing.T) {
	compiled, _ := Compile("test", `res.json(200, {message: "hello"});`)
	req := &Request{
		Body:    nil,
		Query:   map[string]string{},
		Params:  map[string]string{},
		Headers: map[string]string{},
	}

	resp, err := Execute(compiled, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	body, ok := resp.Body.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map body, got %T", resp.Body)
	}
	if body["message"] != "hello" {
		t.Errorf("expected hello, got %v", body["message"])
	}
}

func TestExecuteResFile(t *testing.T) {
	compiled, _ := Compile("test", `res.file("./storage/test.pdf");`)
	req := &Request{
		Body:    nil,
		Query:   map[string]string{},
		Params:  map[string]string{},
		Headers: map[string]string{},
	}

	resp, err := Execute(compiled, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FilePath != "./storage/test.pdf" {
		t.Errorf("expected ./storage/test.pdf, got %s", resp.FilePath)
	}
}

func TestExecuteSetHeader(t *testing.T) {
	compiled, _ := Compile("test", `
		res.setHeader("X-Custom", "value");
		res.json(200, {});
	`)
	req := &Request{
		Body:    nil,
		Query:   map[string]string{},
		Params:  map[string]string{},
		Headers: map[string]string{},
	}

	resp, err := Execute(compiled, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Headers["X-Custom"] != "value" {
		t.Errorf("expected X-Custom=value, got %s", resp.Headers["X-Custom"])
	}
}

func TestExecuteReqBody(t *testing.T) {
	compiled, _ := Compile("test", `
		var name = req.body.name;
		res.json(200, {greeting: "hello " + name});
	`)
	req := &Request{
		Body:    map[string]interface{}{"name": "world"},
		Query:   map[string]string{},
		Params:  map[string]string{},
		Headers: map[string]string{},
	}

	resp, err := Execute(compiled, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := resp.Body.(map[string]interface{})
	if body["greeting"] != "hello world" {
		t.Errorf("expected hello world, got %v", body["greeting"])
	}
}

func TestExecuteReqQuery(t *testing.T) {
	compiled, _ := Compile("test", `
		res.json(200, {page: req.query.page});
	`)
	req := &Request{
		Body:    nil,
		Query:   map[string]string{"page": "2"},
		Params:  map[string]string{},
		Headers: map[string]string{},
	}

	resp, err := Execute(compiled, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := resp.Body.(map[string]interface{})
	if body["page"] != "2" {
		t.Errorf("expected 2, got %v", body["page"])
	}
}

func TestExecuteReqParams(t *testing.T) {
	compiled, _ := Compile("test", `
		res.json(200, {id: req.params.id});
	`)
	req := &Request{
		Body:    nil,
		Query:   map[string]string{},
		Params:  map[string]string{"id": "42"},
		Headers: map[string]string{},
	}

	resp, err := Execute(compiled, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := resp.Body.(map[string]interface{})
	if body["id"] != "42" {
		t.Errorf("expected 42, got %v", body["id"])
	}
}

func TestExecuteReqHeaders(t *testing.T) {
	compiled, _ := Compile("test", `
		res.json(200, {ct: req.headers["content-type"]});
	`)
	req := &Request{
		Body:    nil,
		Query:   map[string]string{},
		Params:  map[string]string{},
		Headers: map[string]string{"content-type": "application/json"},
	}

	resp, err := Execute(compiled, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := resp.Body.(map[string]interface{})
	if body["ct"] != "application/json" {
		t.Errorf("expected application/json, got %v", body["ct"])
	}
}

func TestExecuteConditionalLogic(t *testing.T) {
	compiled, _ := Compile("test", `
		if (req.params.id === "0") {
			res.json(400, {error: "invalid id"});
		} else {
			res.json(200, {id: req.params.id});
		}
	`)

	// Case: invalid
	req := &Request{Params: map[string]string{"id": "0"}, Query: map[string]string{}, Headers: map[string]string{}}
	resp, err := Execute(compiled, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	// Case: valid
	req2 := &Request{Params: map[string]string{"id": "5"}, Query: map[string]string{}, Headers: map[string]string{}}
	resp2, err := Execute(compiled, req2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp2.StatusCode)
	}
}

func TestExecuteNoResponse(t *testing.T) {
	compiled, _ := Compile("test", `var x = 1 + 1;`)
	req := &Request{Query: map[string]string{}, Params: map[string]string{}, Headers: map[string]string{}}

	_, err := Execute(compiled, req)
	if err == nil {
		t.Fatal("expected error for script without response")
	}
}

func TestExecuteRuntimeError(t *testing.T) {
	compiled, _ := Compile("test", `
		var x = undefined;
		x.property.deep;
	`)
	req := &Request{Query: map[string]string{}, Params: map[string]string{}, Headers: map[string]string{}}

	_, err := Execute(compiled, req)
	if err == nil {
		t.Fatal("expected runtime error")
	}
}

func TestSandboxingRequireBlocked(t *testing.T) {
	compiled, _ := Compile("test", `
		var fs = require("fs");
		res.json(200, {});
	`)
	req := &Request{Query: map[string]string{}, Params: map[string]string{}, Headers: map[string]string{}}

	_, err := Execute(compiled, req)
	if err == nil {
		t.Fatal("expected error: require should be blocked")
	}
}

func TestResponseWriteHTTPJson(t *testing.T) {
	resp := NewResponse()
	resp.StatusCode = 201
	resp.Body = map[string]interface{}{"created": true}
	resp.Headers["X-Request-Id"] = "abc"

	rec := httptest.NewRecorder()
	err := resp.WriteHTTP(rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != 201 {
		t.Errorf("expected 201, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("X-Request-Id") != "abc" {
		t.Errorf("expected X-Request-Id=abc, got %s", rec.Header().Get("X-Request-Id"))
	}
	if rec.Body.String() != `{"created":true}` {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestResponseWriteHTTPFileHeaders(t *testing.T) {
	resp := NewResponse()
	resp.StatusCode = 200
	resp.FilePath = "./storage/report.pdf"

	rec := httptest.NewRecorder()
	err := resp.WriteHTTP(rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("expected application/pdf, got %s", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Content-Disposition") != `attachment; filename="report.pdf"` {
		t.Errorf("unexpected Content-Disposition: %s", rec.Header().Get("Content-Disposition"))
	}
}
