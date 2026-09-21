package spec_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dbmcp/access"
	"dbmcp/spec"

	_ "dbmcp/connect"
)

func TestLoadYAMLWithEnvExpansion(t *testing.T) {
	t.Setenv("TEST_MONGO_URI", "mongodb://ci-host:27017/")
	t.Setenv("TEST_PG_PASS", "secret")

	dir := t.TempDir()
	path := filepath.Join(dir, "spec.yaml")
	contents := `
server:
  serve_mode: stdio
defaults:
  access: read_only
  max_rows: 50
connections:
  - name: mongo
    type: mongo
    uri: ${TEST_MONGO_URI}
    access: ro
  - name: pg
    type: postgres
    host: localhost
    database: app
    user: postgres
    password: ${TEST_PG_PASS}
    ssl_mode: disable
    access: read_write
  - name: cache
    type: redis
    host: 127.0.0.1
    access: admin
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := spec.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Connections) != 3 {
		t.Fatalf("got %d connections", len(cfg.Connections))
	}
	if cfg.Connections[0].TypeNormalized != "mongodb" {
		t.Fatalf("mongo type alias not normalized: %s", cfg.Connections[0].TypeNormalized)
	}
	if cfg.Connections[0].URI != "mongodb://ci-host:27017/" {
		t.Fatalf("mongo uri not expanded: %s", cfg.Connections[0].URI)
	}
	if cfg.Connections[0].AccessMode != access.ModeReadOnly {
		t.Fatalf("mongo access: %s", cfg.Connections[0].AccessMode)
	}
	if cfg.Connections[1].AccessMode != access.ModeReadWrite {
		t.Fatalf("pg access: %s", cfg.Connections[1].AccessMode)
	}
	if got := cfg.Connections[1].URI; got == "" || cfg.Connections[1].Password != "secret" {
		t.Fatalf("postgres uri/password not built: uri=%s pass=%s", got, cfg.Connections[1].Password)
	}
	if cfg.Connections[2].TypeNormalized != "redis" || cfg.Connections[2].AccessMode != access.ModeAdmin {
		t.Fatalf("redis not normalized: %+v", cfg.Connections[2])
	}
	if cfg.Connections[0].MaxRowsVal != 50 {
		t.Fatalf("max rows default not applied: %d", cfg.Connections[0].MaxRowsVal)
	}
}

func TestLoadRejectsDuplicateNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.yaml")
	contents := `
connections:
  - name: db
    type: redis
    uri: redis://localhost:6379/0
  - name: db
    type: redis
    uri: redis://localhost:6379/1
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := spec.Load(path); err == nil {
		t.Fatal("expected duplicate name error")
	}
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.json")
	contents := `{
  "connections": [
    {"name": "cache", "type": "redis", "uri": "redis://localhost:6379/0"}
  ]
}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := spec.Load(path)
	if err != nil {
		t.Fatalf("Load JSON: %v", err)
	}
	if cfg.Connections[0].AccessMode != access.ModeReadOnly {
		t.Fatalf("default access not applied")
	}
}

func TestLoadMultipleConnectionsPerEngine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.yaml")
	contents := `
connections:
  - name: mongo-dev
    type: mongodb
    uri: mongodb://localhost:27017/dev
    access: read_only
  - name: mongo-prod
    type: mongo
    uri: mongodb://localhost:27018/prod
    access: read_write
  - name: pg-analytics
    type: postgres
    uri: postgres://localhost:5432/analytics
    access: read_only
  - name: pg-app
    type: pg
    uri: postgres://localhost:5432/app
    access: read_write
  - name: redis-cache
    type: redis
    uri: redis://localhost:6379/0
    access: read_only
  - name: redis-jobs
    type: redis
    uri: redis://localhost:6379/1
    access: admin
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := spec.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Connections) != 6 {
		t.Fatalf("got %d connections, want 6", len(loaded.Connections))
	}

	want := []struct {
		name string
		typ  string
		mode access.Mode
	}{
		{"mongo-dev", "mongodb", access.ModeReadOnly},
		{"mongo-prod", "mongodb", access.ModeReadWrite},
		{"pg-analytics", "postgres", access.ModeReadOnly},
		{"pg-app", "postgres", access.ModeReadWrite},
		{"redis-cache", "redis", access.ModeReadOnly},
		{"redis-jobs", "redis", access.ModeAdmin},
	}
	counts := map[string]int{}
	for i, c := range loaded.Connections {
		if c.Name != want[i].name || c.TypeNormalized != want[i].typ || c.AccessMode != want[i].mode {
			t.Fatalf("connection %d: got name=%s type=%s access=%s want %+v", i, c.Name, c.TypeNormalized, c.AccessMode, want[i])
		}
		counts[c.TypeNormalized]++
	}
	if counts["mongodb"] != 2 || counts["postgres"] != 2 || counts["redis"] != 2 {
		t.Fatalf("per-engine counts: %v", counts)
	}
}

func TestUnknownTypeListsRegisteredKinds(t *testing.T) {
	_, err := spec.NormalizeType("not-a-db")
	if err == nil {
		t.Fatal("expected unknown type error")
	}
	msg := err.Error()
	for _, name := range []string{"mongodb", "postgres", "redis"} {
		if !strings.Contains(msg, name) {
			t.Fatalf("error should list %s: %s", name, msg)
		}
	}
}
