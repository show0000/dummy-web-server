package validation

import (
	"testing"
)

func TestValidateValid(t *testing.T) {
	schema := map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"name"},
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "number"},
		},
	}

	data := map[string]interface{}{
		"name": "Alice",
		"age":  float64(30),
	}

	if err := Validate(schema, data); err != nil {
		t.Fatalf("expected valid, got error: %v", err)
	}
}

func TestValidateMissingRequired(t *testing.T) {
	schema := map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"name"},
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
	}

	data := map[string]interface{}{
		"age": float64(30),
	}

	if err := Validate(schema, data); err == nil {
		t.Fatal("expected validation error for missing required field")
	}
}

func TestValidateWrongType(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"age": map[string]interface{}{"type": "integer"},
		},
	}

	data := map[string]interface{}{
		"age": "not-a-number",
	}

	if err := Validate(schema, data); err == nil {
		t.Fatal("expected validation error for wrong type")
	}
}

func TestValidateArray(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"tags": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
		},
	}

	data := map[string]interface{}{
		"tags": []interface{}{"go", "mock"},
	}

	if err := Validate(schema, data); err != nil {
		t.Fatalf("expected valid, got error: %v", err)
	}
}

func TestValidateEmptySchema(t *testing.T) {
	schema := map[string]interface{}{}
	data := map[string]interface{}{"anything": "goes"}

	if err := Validate(schema, data); err != nil {
		t.Fatalf("expected valid with empty schema, got error: %v", err)
	}
}
