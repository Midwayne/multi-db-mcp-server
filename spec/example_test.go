package spec

import (
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..")
}

func TestExampleSpecLoads(t *testing.T) {
	path := filepath.Join(repoRoot(t), "spec.example.yaml")
	spec, err := Load(path)
	if err != nil {
		t.Fatalf("example spec should load: %v", err)
	}
	if len(spec.Connections) != 3 {
		t.Fatalf("expected 3 example connections, got %d", len(spec.Connections))
	}
	seen := map[string]string{}
	for _, c := range spec.Connections {
		seen[c.Name] = c.TypeNormalized
	}
	if seen["local-mongo"] != "mongodb" || seen["analytics"] != "postgres" || seen["cache"] != "redis" {
		t.Fatalf("unexpected example connections: %v", seen)
	}
}

func TestRecipeSpecsLoad(t *testing.T) {
	matches, err := filepath.Glob(filepath.Join(repoRoot(t), "examples", "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("expected example recipe YAML files")
	}
	for _, path := range matches {
		t.Run(filepath.Base(path), func(t *testing.T) {
			spec, err := Load(path)
			if err != nil {
				t.Fatalf("Load %s: %v", path, err)
			}
			if len(spec.Connections) == 0 {
				t.Fatalf("%s has no connections", path)
			}
			if spec.Server.ServeMode != "stdio" {
				t.Fatalf("%s serve_mode=%s", path, spec.Server.ServeMode)
			}
		})
	}
}
