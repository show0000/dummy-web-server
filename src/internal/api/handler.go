package api

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
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
func RegisterAPIs(r *router.Router, apisPath, storagePath string) ([]RegisteredAPI, error) {
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

		handler := makeHandler(reg, storagePath)
		r.Handle(apiDef.Method, apiDef.Entrypoint, handler)
	}

	return registered, nil
}

func makeHandler(reg RegisteredAPI, storagePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body interface{}
		var files []script.FileInfo

		ct := r.Header.Get("Content-Type")
		isMultipart := strings.HasPrefix(ct, "multipart/form-data")

		if isMultipart {
			// Parse multipart form (max 32MB)
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				writeError(w, http.StatusBadRequest, "failed to parse multipart form")
				return
			}

			// Save uploaded files
			for fieldName, fileHeaders := range r.MultipartForm.File {
				for _, fh := range fileHeaders {
					savedPath, err := saveUploadedFile(fh, storagePath)
					if err != nil {
						writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to save file: %v", err))
						return
					}
					files = append(files, script.FileInfo{
						FieldName: fieldName,
						FileName:  fh.Filename,
						Size:      fh.Size,
						SavedPath: savedPath,
					})
				}
			}

			// Parse form fields as body
			formData := make(map[string]interface{})
			for k, v := range r.MultipartForm.Value {
				if len(v) == 1 {
					formData[k] = v[0]
				} else {
					formData[k] = v
				}
			}
			if len(formData) > 0 {
				body = formData
			}
		} else if r.Body != nil && r.ContentLength != 0 {
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
			Files:   files,
		}

		// Execute script
		resp, err := script.Execute(reg.Compiled, req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Write response
		if resp.IsMultipart {
			for k, v := range resp.Headers {
				w.Header().Set(k, v)
			}
			if err := writeMultipartResponse(w, resp.StatusCode, resp.MultipartParts); err != nil {
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to write multipart response: %v", err))
			}
			return
		}

		if resp.FilePath != "" {
			for k, v := range resp.Headers {
				w.Header().Set(k, v)
			}
			// Set Content-Disposition with filename for download
			if w.Header().Get("Content-Disposition") == "" {
				filename := filepath.Base(resp.FilePath)
				w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
			}
			// Expose Content-Disposition to browsers (needed for fetch() access)
			w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
			http.ServeFile(w, r, resp.FilePath)
			return
		}

		if err := resp.WriteHTTP(w); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to write response")
		}
	}
}

func saveUploadedFile(fh *multipart.FileHeader, storagePath string) (string, error) {
	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return "", err
	}

	destPath := filepath.Join(storagePath, fh.Filename)
	dst, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return destPath, nil
}

// writeMultipartResponse serializes parts as multipart/mixed.
// Each part must be a map with one of: "json", "file", "text".
// Optional: "name" (form field name), "filename", "contentType", "headers" (map).
func writeMultipartResponse(w http.ResponseWriter, status int, parts []interface{}) error {
	if status == 0 {
		status = http.StatusOK
	}

	mw := multipart.NewWriter(w)
	w.Header().Set("Content-Type", "multipart/mixed; boundary="+mw.Boundary())
	w.WriteHeader(status)
	realWriter := mw

	for i, p := range parts {
		part, ok := p.(map[string]interface{})
		if !ok {
			return fmt.Errorf("part %d: expected object, got %T", i, p)
		}

		hdr := textproto.MIMEHeader{}

		// Custom headers from the part
		if hMap, ok := part["headers"].(map[string]interface{}); ok {
			for k, v := range hMap {
				if s, ok := v.(string); ok {
					hdr.Set(k, s)
				}
			}
		}

		// Content-Disposition
		name, _ := part["name"].(string)
		filename, _ := part["filename"].(string)
		if name != "" || filename != "" {
			disp := "form-data"
			if name != "" {
				disp += fmt.Sprintf(`; name="%s"`, name)
			}
			if filename != "" {
				disp += fmt.Sprintf(`; filename="%s"`, filename)
			}
			hdr.Set("Content-Disposition", disp)
		}

		// Optional explicit Content-Type override
		explicitCT, _ := part["contentType"].(string)

		switch {
		case part["json"] != nil:
			if explicitCT == "" {
				hdr.Set("Content-Type", "application/json")
			} else {
				hdr.Set("Content-Type", explicitCT)
			}
			pw, err := realWriter.CreatePart(hdr)
			if err != nil {
				return fmt.Errorf("part %d: %w", i, err)
			}
			data, err := json.Marshal(part["json"])
			if err != nil {
				return fmt.Errorf("part %d json marshal: %w", i, err)
			}
			if _, err := pw.Write(data); err != nil {
				return fmt.Errorf("part %d write: %w", i, err)
			}

		case part["text"] != nil:
			text, ok := part["text"].(string)
			if !ok {
				return fmt.Errorf("part %d: text must be a string", i)
			}
			if explicitCT == "" {
				hdr.Set("Content-Type", "text/plain; charset=utf-8")
			} else {
				hdr.Set("Content-Type", explicitCT)
			}
			pw, err := realWriter.CreatePart(hdr)
			if err != nil {
				return fmt.Errorf("part %d: %w", i, err)
			}
			if _, err := pw.Write([]byte(text)); err != nil {
				return fmt.Errorf("part %d write: %w", i, err)
			}

		case part["file"] != nil:
			filePath, ok := part["file"].(string)
			if !ok {
				return fmt.Errorf("part %d: file must be a string path", i)
			}
			if filename == "" {
				filename = filepath.Base(filePath)
				hdr.Set("Content-Disposition", func() string {
					disp := "form-data"
					if name != "" {
						disp += fmt.Sprintf(`; name="%s"`, name)
					}
					disp += fmt.Sprintf(`; filename="%s"`, filename)
					return disp
				}())
			}
			if explicitCT == "" {
				ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(filePath)))
				if ct == "" {
					ct = "application/octet-stream"
				}
				hdr.Set("Content-Type", ct)
			} else {
				hdr.Set("Content-Type", explicitCT)
			}
			pw, err := realWriter.CreatePart(hdr)
			if err != nil {
				return fmt.Errorf("part %d: %w", i, err)
			}
			f, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("part %d open file: %w", i, err)
			}
			_, copyErr := io.Copy(pw, f)
			f.Close()
			if copyErr != nil {
				return fmt.Errorf("part %d copy file: %w", i, copyErr)
			}

		default:
			return fmt.Errorf("part %d: must have one of json, file, text", i)
		}
	}

	return realWriter.Close()
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
