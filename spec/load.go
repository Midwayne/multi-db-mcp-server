package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const specEnvVar = "DBMCP_SPEC"

// Load reads a spec file, expands ${ENV} placeholders, unmarshals, and normalizes.
func Load(path string) (*Spec, error) {
	resolved, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read spec %s: %w", resolved, err)
	}

	expanded := expandEnv(string(raw))
	spec, err := parse(resolved, []byte(expanded))
	if err != nil {
		return nil, err
	}
	if err := spec.Normalize(); err != nil {
		return nil, err
	}
	return spec, nil
}

// ResolvePath picks the spec file from an explicit path, DBMCP_SPEC, or common defaults.
func ResolvePath(path string) (string, error) {
	if strings.TrimSpace(path) != "" {
		return path, nil
	}
	if env := strings.TrimSpace(os.Getenv(specEnvVar)); env != "" {
		return env, nil
	}
	candidates := []string{"spec.yaml", "spec.yml", "spec.json"}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("no spec file found (pass -spec, set %s, or add spec.yaml)", specEnvVar)
}

func parse(path string, raw []byte) (*Spec, error) {
	var spec Spec
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		if err := json.Unmarshal(raw, &spec); err != nil {
			return nil, fmt.Errorf("parse spec JSON: %w", err)
		}
	case ".yaml", ".yml", "":
		if err := yaml.Unmarshal(raw, &spec); err != nil {
			return nil, fmt.Errorf("parse spec YAML: %w", err)
		}
	default:
		if err := yaml.Unmarshal(raw, &spec); err != nil {
			if jsonErr := json.Unmarshal(raw, &spec); jsonErr != nil {
				return nil, fmt.Errorf("parse spec: yaml: %v; json: %v", err, jsonErr)
			}
		}
	}
	return &spec, nil
}

func expandEnv(s string) string {
	return os.Expand(s, func(key string) string {
		name, def, hasDefault := strings.Cut(key, ":-")
		if hasDefault {
			if v, ok := os.LookupEnv(name); ok {
				return v
			}
			return def
		}
		return os.Getenv(key)
	})
}
