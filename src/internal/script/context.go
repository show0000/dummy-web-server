package script

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// Request is the read-only req object injected into scripts.
type Request struct {
	Body    interface{}       `json:"body"`
	Query   map[string]string `json:"query"`
	Params  map[string]string `json:"params"`
	Headers map[string]string `json:"headers"`
}

// Response collects the script's response intent.
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       interface{}
	FilePath   string
	Responded  bool
}

func NewResponse() *Response {
	return &Response{
		Headers: make(map[string]string),
	}
}

// ResHelper is the res object exposed to scripts.
type ResHelper struct {
	resp *Response
}

func NewResHelper(resp *Response) *ResHelper {
	return &ResHelper{resp: resp}
}

func (h *ResHelper) Json(status int, body interface{}) interface{} {
	h.resp.StatusCode = status
	h.resp.Body = body
	h.resp.Responded = true
	return nil
}

func (h *ResHelper) File(path string) interface{} {
	h.resp.StatusCode = http.StatusOK
	h.resp.FilePath = path
	h.resp.Responded = true
	return nil
}

func (h *ResHelper) SetHeader(key, value string) {
	h.resp.Headers[key] = value
}

// WriteHTTP writes the Response to an http.ResponseWriter.
func (r *Response) WriteHTTP(w http.ResponseWriter) error {
	for k, v := range r.Headers {
		w.Header().Set(k, v)
	}

	if r.FilePath != "" {
		return r.writeFile(w)
	}
	return r.writeJSON(w)
}

func (r *Response) writeJSON(w http.ResponseWriter) error {
	data, err := json.Marshal(r.Body)
	if err != nil {
		return fmt.Errorf("failed to marshal response body: %w", err)
	}

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(r.StatusCode)
	_, err = w.Write(data)
	return err
}

func (r *Response) writeFile(w http.ResponseWriter) error {
	ext := strings.ToLower(filepath.Ext(r.FilePath))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", mimeType)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(r.FilePath)))

	// Actual file serving is handled by the caller (handler.go) using http.ServeFile.
	// This method only sets headers. The caller checks resp.FilePath != "" to serve the file.
	return nil
}
