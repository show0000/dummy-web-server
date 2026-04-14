package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"dummy-web-server/src/internal/router"
	"dummy-web-server/src/internal/script"
	"dummy-web-server/src/internal/validation"
)

// RegisteredAPI holds a parsed API definition with its compiled script.
type RegisteredAPI struct {
	Definition APIDefinition
	Compiled   *script.CompiledScript
}

// RegisterAPIs loads apis.yaml, compiles all scripts (Fail-Fast), and registers routes.
func RegisterAPIs(r *router.Router, apisPath string) ([]RegisteredAPI, error) {
	apis, err := LoadAPIs(apisPath)
	if err != nil {
		return nil, err
	}

	basePath := filepath.Dir(apisPath)
	var registered []RegisteredAPI

	for _, apiDef := range apis {
		source, err := apiDef.ResolveScript(basePath)
		if err != nil {
			return nil, fmt.Errorf("api %s %s: %w", apiDef.Method, apiDef.Entrypoint, err)
		}

		compiled, err := script.Compile(apiDef.Entrypoint, source)
		if err != nil {
			return nil, fmt.Errorf("api %s %s: %w", apiDef.Method, apiDef.Entrypoint, err)
		}

		reg := RegisteredAPI{Definition: apiDef, Compiled: compiled}
		registered = append(registered, reg)

		handler := makeHandler(reg)
		r.Handle(apiDef.Method, apiDef.Entrypoint, handler)
	}

	return registered, nil
}

func makeHandler(reg RegisteredAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var body interface{}
		if r.Body != nil && r.ContentLength != 0 {
			data, err := io.ReadAll(r.Body)
			if err != nil {
				writeError(w, http.StatusBadRequest, "failed to read request body")
				return
			}
			if len(data) > 0 {
				if err := json.Unmarshal(data, &body); err != nil {
					writeError(w, http.StatusBadRequest, "invalid JSON body")
					return
				}
			}
		}

		// Validation
		if reg.Definition.Validation != nil && reg.Definition.Validation.Schema != nil {
			if body == nil {
				writeError(w, http.StatusBadRequest, "request body is required for validation")
				return
			}
			if err := validation.Validate(reg.Definition.Validation.Schema, body); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		// Build script request context
		params := router.Params(r.Context())
		query := make(map[string]string)
		for k, v := range r.URL.Query() {
			if len(v) > 0 {
				query[k] = v[0]
			}
		}
		headers := make(map[string]string)
		for k, v := range r.Header {
			if len(v) > 0 {
				headers[strings.ToLower(k)] = v[0]
			}
		}

		req := &script.Request{
			Body:    body,
			Query:   query,
			Params:  params,
			Headers: headers,
		}

		// Execute script
		resp, err := script.Execute(reg.Compiled, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Write response
		if resp.FilePath != "" {
			// Set custom headers before serving file
			for k, v := range resp.Headers {
				w.Header().Set(k, v)
			}
			http.ServeFile(w, r, resp.FilePath)
			return
		}

		if err := resp.WriteHTTP(w); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to write response")
		}
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
