package spec

import (
	"os"
	"path/filepath"
	"testing"

	"dbmcp/access"
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

	spec, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(spec.Connections) != 3 {
		t.Fatalf("got %d connections", len(spec.Connections))
	}
	if spec.Connections[0].TypeNormalized != "mongodb" {
		t.Fatalf("mongo type alias not normalized: %s", spec.Connections[0].TypeNormalized)
	}
	if spec.Connections[0].URI != "mongodb://ci-host:27017/" {
		t.Fatalf("mongo uri not expanded: %s", spec.Connections[0].URI)
	}
	if spec.Connections[0].AccessMode != access.ModeReadOnly {
		t.Fatalf("mongo access: %s", spec.Connections[0].AccessMode)
	}
	if spec.Connections[1].AccessMode != access.ModeReadWrite {
		t.Fatalf("pg access: %s", spec.Connections[1].AccessMode)
	}
	if got := spec.Connections[1].URI; got == "" || spec.Connections[1].Password != "secret" {
		t.Fatalf("postgres uri/password not built: uri=%s pass=%s", got, spec.Connections[1].Password)
	}
	if spec.Connections[2].TypeNormalized != "redis" || spec.Connections[2].AccessMode != access.ModeAdmin {
		t.Fatalf("redis not normalized: %+v", spec.Connections[2])
	}
	if spec.Connections[0].MaxRowsVal != 50 {
		t.Fatalf("max rows default not applied: %d", spec.Connections[0].MaxRowsVal)
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
	if _, err := Load(path); err == nil {
		t.Fatal("expected duplicate name error")
	}
}

func TestToolAllowed(t *testing.T) {
	c := Connection{Tools: ToolsConfig{Include: []string{"mongo_find"}, Exclude: []string{"mongo_aggregate"}}}
	if !c.ToolAllowed("mongo_find") {
		t.Fatal("include should allow mongo_find")
	}
	if c.ToolAllowed("mongo_count") {
		t.Fatal("include list should deny unspecified tools")
	}
	c = Connection{Tools: ToolsConfig{Exclude: []string{"mongo_delete"}}}
	if !c.ToolAllowed("mongo_find") || c.ToolAllowed("mongo_delete") {
		t.Fatal("exclude list mismatch")
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
	spec, err := Load(path)
	if err != nil {
		t.Fatalf("Load JSON: %v", err)
	}
	if spec.Connections[0].AccessMode != access.ModeReadOnly {
		t.Fatalf("default access not applied")
	}
}
