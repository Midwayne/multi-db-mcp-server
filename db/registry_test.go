package db

import (
	"context"
	"testing"

	"dbmcp/access"
	"dbmcp/spec"
)

type stubAdapter struct {
	Meta
	ready bool
}

func (s *stubAdapter) EnsureConnected(context.Context) error { s.ready = true; return nil }
func (s *stubAdapter) Ping(context.Context) error            { return nil }
func (s *stubAdapter) Close(context.Context) error           { s.ready = false; return nil }
func (s *stubAdapter) Connected() bool                       { return s.ready }
func (s *stubAdapter) Landscape(context.Context) (any, error) {
	return map[string]string{"ok": "true"}, nil
}

func testConn(name, typ string, mode access.Mode, include []string) spec.Connection {
	enabled := true
	return spec.Connection{
		Name:           name,
		TypeNormalized: typ,
		AccessMode:     mode,
		EnabledVal:     true,
		Enabled:        &enabled,
		Tools:          spec.ToolsConfig{Include: include},
		URI:            "stub://",
	}
}

func TestRegistryAnyAllows(t *testing.T) {
	t.Parallel()
	sp := &spec.Spec{
		Connections: []spec.Connection{
			testConn("mongo-ro", "mongodb", access.ModeReadOnly, nil),
			testConn("pg-rw", "postgres", access.ModeReadWrite, []string{"postgres_query", "postgres_execute"}),
		},
	}
	reg, err := NewRegistry(context.Background(), sp, func(c spec.Connection) (Adapter, error) {
		return &stubAdapter{Meta: Meta{Conn: c}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reg.HasType(TypeMongoDB) || !reg.HasType(TypePostgres) || reg.HasType(TypeRedis) {
		t.Fatal("HasType mismatch")
	}
	if !reg.AnyAllows(TypeMongoDB, access.OpRead, "mongo_find") {
		t.Fatal("mongo read should be allowed")
	}
	if reg.AnyAllows(TypeMongoDB, access.OpWrite, "mongo_insert") {
		t.Fatal("mongo write should not be allowed")
	}
	if !reg.AnyAllows(TypePostgres, access.OpWrite, "postgres_execute") {
		t.Fatal("postgres write should be allowed")
	}
	if reg.AnyAllows(TypePostgres, access.OpRead, "postgres_list_tables") {
		t.Fatal("postgres_list_tables is not in the connection include list")
	}
}

func TestRegistryUnknownConnection(t *testing.T) {
	t.Parallel()
	enabled := true
	sp := &spec.Spec{Connections: []spec.Connection{{
		Name: "only", TypeNormalized: "redis", AccessMode: access.ModeReadOnly, EnabledVal: true, Enabled: &enabled,
	}}}
	reg, err := NewRegistry(context.Background(), sp, func(c spec.Connection) (Adapter, error) {
		return &stubAdapter{Meta: Meta{Conn: c}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Lookup("missing"); err == nil {
		t.Fatal("expected unknown connection error")
	}
	info := reg.List()
	if len(info) != 1 || info[0].Name != "only" || info[0].Type != TypeRedis {
		t.Fatalf("unexpected list: %+v", info)
	}
	if !info[0].CanRead || info[0].CanWrite || info[0].CanAdmin {
		t.Fatalf("read_only flags: %+v", info[0])
	}
}
