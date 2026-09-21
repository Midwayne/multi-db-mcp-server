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

func TestRegistryGetEveryNamedConnection(t *testing.T) {
	t.Parallel()
	enabled := true
	sp := &spec.Spec{Connections: []spec.Connection{
		testConn("mongo-ro", "mongodb", access.ModeReadOnly, nil),
		testConn("pg-rw", "postgres", access.ModeReadWrite, nil),
		testConn("cache", "redis", access.ModeAdmin, nil),
	}}
	for i := range sp.Connections {
		sp.Connections[i].Enabled = &enabled
		sp.Connections[i].EnabledVal = true
	}
	reg, err := NewRegistry(context.Background(), sp, func(c spec.Connection) (Adapter, error) {
		return &stubAdapter{Meta: Meta{Conn: c}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"mongo-ro", "pg-rw", "cache"} {
		adapter, err := reg.Get(context.Background(), name)
		if err != nil {
			t.Fatalf("Get(%s): %v", name, err)
		}
		if adapter.Name() != name {
			t.Fatalf("Get(%s) returned %s", name, adapter.Name())
		}
	}
	info := reg.List()
	if len(info) != 3 {
		t.Fatalf("List()=%d want 3", len(info))
	}
	byName := map[string]Info{}
	for _, item := range info {
		byName[item.Name] = item
	}
	if !byName["mongo-ro"].CanRead || byName["mongo-ro"].CanWrite {
		t.Fatalf("mongo-ro flags: %+v", byName["mongo-ro"])
	}
	if !byName["pg-rw"].CanWrite || byName["pg-rw"].CanAdmin {
		t.Fatalf("pg-rw flags: %+v", byName["pg-rw"])
	}
	if !byName["cache"].CanAdmin {
		t.Fatalf("cache flags: %+v", byName["cache"])
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

func TestRegistryMultipleConnectionsPerEngine(t *testing.T) {
	t.Parallel()
	sp := &spec.Spec{Connections: []spec.Connection{
		testConn("mongo-dev", "mongodb", access.ModeReadOnly, nil),
		testConn("mongo-prod", "mongodb", access.ModeReadWrite, nil),
		testConn("pg-analytics", "postgres", access.ModeReadOnly, nil),
		testConn("pg-app", "postgres", access.ModeReadWrite, nil),
		testConn("redis-cache", "redis", access.ModeReadOnly, nil),
		testConn("redis-jobs", "redis", access.ModeAdmin, nil),
	}}
	reg, err := NewRegistry(context.Background(), sp, func(c spec.Connection) (Adapter, error) {
		return &stubAdapter{Meta: Meta{Conn: c}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	info := reg.List()
	if len(info) != 6 {
		t.Fatalf("List()=%d want 6", len(info))
	}
	byName := map[string]Info{}
	counts := map[Type]int{}
	for _, item := range info {
		byName[item.Name] = item
		counts[item.Type]++
	}
	if counts[TypeMongoDB] != 2 || counts[TypePostgres] != 2 || counts[TypeRedis] != 2 {
		t.Fatalf("per-engine counts: %v", counts)
	}

	for _, name := range []string{"mongo-dev", "mongo-prod", "pg-analytics", "pg-app", "redis-cache", "redis-jobs"} {
		adapter, err := reg.Get(context.Background(), name)
		if err != nil {
			t.Fatalf("Get(%s): %v", name, err)
		}
		if adapter.Name() != name {
			t.Fatalf("Get(%s) returned %s", name, adapter.Name())
		}
	}

	dev, err := reg.Get(context.Background(), "mongo-dev")
	if err != nil {
		t.Fatal(err)
	}
	prod, err := reg.Get(context.Background(), "mongo-prod")
	if err != nil {
		t.Fatal(err)
	}
	if dev.Name() == prod.Name() || dev.Access() == prod.Access() {
		t.Fatal("two mongo connections must stay distinct")
	}
	if !reg.AnyAllows(TypeMongoDB, access.OpWrite, "mongo_insert") {
		t.Fatal("mongo_insert should be allowed because mongo-prod is read_write")
	}
	if !byName["mongo-dev"].CanRead || byName["mongo-dev"].CanWrite {
		t.Fatalf("mongo-dev flags: %+v", byName["mongo-dev"])
	}
	if !byName["mongo-prod"].CanWrite {
		t.Fatalf("mongo-prod flags: %+v", byName["mongo-prod"])
	}
}
