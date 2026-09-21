package spec_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"dbmcp/spec"

	_ "dbmcp/connect"
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
	cfg, err := spec.Load(path)
	if err != nil {
		t.Fatalf("example spec should load: %v", err)
	}
	if len(cfg.Connections) != 3 {
		t.Fatalf("expected 3 example connections, got %d", len(cfg.Connections))
	}
	seen := map[string]string{}
	for _, c := range cfg.Connections {
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
			cfg, err := spec.Load(path)
			if err != nil {
				t.Fatalf("Load %s: %v", path, err)
			}
			if len(cfg.Connections) == 0 {
				t.Fatalf("%s has no connections", path)
			}
			if cfg.Server.ServeMode != "stdio" {
				t.Fatalf("%s serve_mode=%s", path, cfg.Server.ServeMode)
			}
		})
	}
}
