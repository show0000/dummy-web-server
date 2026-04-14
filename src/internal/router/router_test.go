package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExactMatch(t *testing.T) {
	r := New()
	r.Handle("GET", "/api/users", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestPathVariable(t *testing.T) {
	r := New()
	r.Handle("GET", "/api/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		params := Params(req.Context())
		w.Write([]byte(params["id"]))
	})

	req := httptest.NewRequest("GET", "/api/users/42", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "42" {
		t.Errorf("expected 42, got %s", body)
	}
}

func TestMultiplePathVariables(t *testing.T) {
	r := New()
	r.Handle("GET", "/api/{group}/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		params := Params(req.Context())
		w.Write([]byte(params["group"] + ":" + params["id"]))
	})

	req := httptest.NewRequest("GET", "/api/admin/users/7", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	if string(body) != "admin:7" {
		t.Errorf("expected admin:7, got %s", body)
	}
}

func TestMethodMismatch(t *testing.T) {
	r := New()
	r.Handle("POST", "/api/users", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestPathMismatch(t *testing.T) {
	r := New()
	r.Handle("GET", "/api/users", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/posts", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestSegmentCountMismatch(t *testing.T) {
	r := New()
	r.Handle("GET", "/api/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestTrailingSlash(t *testing.T) {
	r := New()
	r.Handle("GET", "/api/users", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/api/users/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with trailing slash, got %d", rec.Code)
	}
}

func TestSamePathDifferentMethods(t *testing.T) {
	r := New()
	r.Handle("GET", "/api/users", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("get"))
	})
	r.Handle("POST", "/api/users", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("post"))
	})

	// GET
	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "get" {
		t.Errorf("expected get, got %s", body)
	}

	// POST
	req = httptest.NewRequest("POST", "/api/users", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	body, _ = io.ReadAll(rec.Body)
	if string(body) != "post" {
		t.Errorf("expected post, got %s", body)
	}
}

func TestRootPath(t *testing.T) {
	r := New()
	r.Handle("GET", "/", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("root"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	if string(body) != "root" {
		t.Errorf("expected root, got %s", body)
	}
}

func TestParamsFromContextEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	params := Params(req.Context())
	if params == nil {
		t.Error("expected empty map, got nil")
	}
}
