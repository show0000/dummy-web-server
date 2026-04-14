package api

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Validation struct {
	Schema     map[string]interface{} `yaml:"schema"`
	SchemaFile string                 `yaml:"schemaFile"`
}

type APIDefinition struct {
	Entrypoint  string     `yaml:"entrypoint"`
	Method      string     `yaml:"method"`
	Description string     `yaml:"description"`
	Auth        *bool      `yaml:"auth"`
	Validation  *Validation `yaml:"validation"`
	Script      string     `yaml:"script"`
	ScriptFile  string     `yaml:"scriptFile"`
}

type APIsFile struct {
	APIs []APIDefinition `yaml:"apis"`
}

var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true,
}

func LoadAPIs(path string) ([]APIDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read apis file: %w", err)
	}

	var file APIsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse apis file: %w", err)
	}

	for i, api := range file.APIs {
		if err := validateAPI(i, api); err != nil {
			return nil, err
		}
		file.APIs[i].Method = strings.ToUpper(api.Method)
	}

	return file.APIs, nil
}

func validateAPI(index int, api APIDefinition) error {
	prefix := fmt.Sprintf("apis[%d]", index)

	if api.Entrypoint == "" {
		return fmt.Errorf("%s: entrypoint is required", prefix)
	}
	if !strings.HasPrefix(api.Entrypoint, "/") {
		return fmt.Errorf("%s: entrypoint must start with /", prefix)
	}

	method := strings.ToUpper(api.Method)
	if method == "" {
		return fmt.Errorf("%s: method is required", prefix)
	}
	if !validMethods[method] {
		return fmt.Errorf("%s: invalid method %q", prefix, api.Method)
	}

	if api.Script == "" && api.ScriptFile == "" {
		return fmt.Errorf("%s: script or scriptFile is required", prefix)
	}

	return nil
}

func (a *APIDefinition) AuthEnabled() bool {
	if a.Auth == nil {
		return true
	}
	return *a.Auth
}

func (a *APIDefinition) ResolveScript(basePath string) (string, error) {
	if a.Script != "" {
		return a.Script, nil
	}

	path := a.ScriptFile
	if basePath != "" && !strings.HasPrefix(path, "/") {
		path = basePath + "/" + path
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read script file %q: %w", a.ScriptFile, err)
	}
	return string(data), nil
}
