package spec

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestExampleSpecLoads(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "spec.example.yaml")
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
