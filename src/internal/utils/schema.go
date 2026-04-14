package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

// GenerateSchema converts a JSON value into a JSON Schema definition.
func GenerateSchema(data interface{}) map[string]interface{} {
	if data == nil {
		return map[string]interface{}{"type": "null"}
	}

	switch v := data.(type) {
	case map[string]interface{}:
		return generateObjectSchema(v)
	case []interface{}:
		return generateArraySchema(v)
	case string:
		return map[string]interface{}{"type": "string"}
	case float64:
		if v == float64(int64(v)) {
			return map[string]interface{}{"type": "integer"}
		}
		return map[string]interface{}{"type": "number"}
	case bool:
		return map[string]interface{}{"type": "boolean"}
	default:
		return map[string]interface{}{"type": "string"}
	}
}

func generateObjectSchema(obj map[string]interface{}) map[string]interface{} {
	properties := make(map[string]interface{})
	required := make([]interface{}, 0, len(obj))

	// Sort keys for deterministic output
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		properties[k] = GenerateSchema(obj[k])
		required = append(required, k)
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func generateArraySchema(arr []interface{}) map[string]interface{} {
	schema := map[string]interface{}{
		"type": "array",
	}
	if len(arr) > 0 {
		schema["items"] = GenerateSchema(arr[0])
	}
	return schema
}

// Handler returns an http.HandlerFunc for POST /_utils/schema.
func SchemaHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		var data interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid JSON: %v", err)})
			return
		}

		schema := GenerateSchema(data)
		writeJSON(w, http.StatusOK, schema)
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
