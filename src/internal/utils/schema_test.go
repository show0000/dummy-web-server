package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateSchemaString(t *testing.T) {
	result := GenerateSchema("hello")
	if result["type"] != "string" {
		t.Errorf("expected string, got %v", result["type"])
	}
}

func TestGenerateSchemaInteger(t *testing.T) {
	result := GenerateSchema(float64(42))
	if result["type"] != "integer" {
		t.Errorf("expected integer, got %v", result["type"])
	}
}

func TestGenerateSchemaFloat(t *testing.T) {
	result := GenerateSchema(float64(3.14))
	if result["type"] != "number" {
		t.Errorf("expected number, got %v", result["type"])
	}
}

func TestGenerateSchemaBoolean(t *testing.T) {
	result := GenerateSchema(true)
	if result["type"] != "boolean" {
		t.Errorf("expected boolean, got %v", result["type"])
	}
}

func TestGenerateSchemaNull(t *testing.T) {
	result := GenerateSchema(nil)
	if result["type"] != "null" {
		t.Errorf("expected null, got %v", result["type"])
	}
}

func TestGenerateSchemaObject(t *testing.T) {
	data := map[string]interface{}{
		"name": "Alice",
		"age":  float64(30),
	}
	result := GenerateSchema(data)

	if result["type"] != "object" {
		t.Fatalf("expected object, got %v", result["type"])
	}

	props := result["properties"].(map[string]interface{})
	nameSchema := props["name"].(map[string]interface{})
	if nameSchema["type"] != "string" {
		t.Errorf("expected name type string, got %v", nameSchema["type"])
	}

	ageSchema := props["age"].(map[string]interface{})
	if ageSchema["type"] != "integer" {
		t.Errorf("expected age type integer, got %v", ageSchema["type"])
	}

	required := result["required"].([]interface{})
	if len(required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(required))
	}
}

func TestGenerateSchemaArray(t *testing.T) {
	data := []interface{}{"a", "b", "c"}
	result := GenerateSchema(data)

	if result["type"] != "array" {
		t.Fatalf("expected array, got %v", result["type"])
	}

	items := result["items"].(map[string]interface{})
	if items["type"] != "string" {
		t.Errorf("expected items type string, got %v", items["type"])
	}
}

func TestGenerateSchemaEmptyArray(t *testing.T) {
	data := []interface{}{}
	result := GenerateSchema(data)

	if result["type"] != "array" {
		t.Fatalf("expected array, got %v", result["type"])
	}
	if _, ok := result["items"]; ok {
		t.Error("expected no items for empty array")
	}
}

func TestGenerateSchemaNestedObject(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "Bob",
		},
	}
	result := GenerateSchema(data)
	props := result["properties"].(map[string]interface{})
	userSchema := props["user"].(map[string]interface{})
	if userSchema["type"] != "object" {
		t.Errorf("expected nested object, got %v", userSchema["type"])
	}
}

func TestSchemaHandlerValid(t *testing.T) {
	handler := SchemaHandler()
	body := `{"name":"Alice","age":30,"active":true}`

	req := httptest.NewRequest("POST", "/_utils/schema", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&result)
	if result["type"] != "object" {
		t.Errorf("expected object type, got %v", result["type"])
	}
}

func TestSchemaHandlerInvalidJSON(t *testing.T) {
	handler := SchemaHandler()

	req := httptest.NewRequest("POST", "/_utils/schema", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
