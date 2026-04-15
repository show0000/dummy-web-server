package script

import (
	"fmt"

	"github.com/dop251/goja"
)

// CompiledScript holds a pre-compiled Goja program.
type CompiledScript struct {
	Program *goja.Program
	Source  string
}

// Compile pre-compiles a JavaScript source string. Used at startup for Fail-Fast.
func Compile(name, source string) (*CompiledScript, error) {
	program, err := goja.Compile(name, source, false)
	if err != nil {
		return nil, fmt.Errorf("script compile error [%s]: %w", name, err)
	}
	return &CompiledScript{Program: program, Source: source}, nil
}

// Execute runs a compiled script with the given req context and returns the response.
func Execute(compiled *CompiledScript, req *Request) (*Response, error) {
	vm := goja.New()

	// Sandboxing: block dangerous globals
	for _, name := range []string{"require", "process", "global", "globalThis"} {
		vm.Set(name, goja.Undefined())
	}

	// Inject req (read-only object)
	files := make([]interface{}, len(req.Files))
	for i, f := range req.Files {
		files[i] = map[string]interface{}{
			"fieldName": f.FieldName,
			"fileName":  f.FileName,
			"size":      f.Size,
			"savedPath": f.SavedPath,
		}
	}
	vm.Set("req", map[string]interface{}{
		"body":    req.Body,
		"query":   req.Query,
		"params":  req.Params,
		"headers": req.Headers,
		"files":   files,
	})

	// Inject res helper
	resp := NewResponse()
	helper := NewResHelper(resp)
	vm.Set("res", map[string]interface{}{
		"json":      helper.Json,
		"file":      helper.File,
		"multipart": helper.Multipart,
		"setHeader": helper.SetHeader,
	})

	_, err := vm.RunProgram(compiled.Program)
	if err != nil {
		return nil, fmt.Errorf("script execution error: %w", err)
	}

	if !resp.Responded {
		return nil, fmt.Errorf("script did not produce a response (call res.json or res.file)")
	}

	return resp, nil
}
