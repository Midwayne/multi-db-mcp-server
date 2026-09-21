package server

import (
	"context"
	"testing"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/spec"
)

type stubAdapter struct {
	db.Meta
	ready bool
}

func (s *stubAdapter) EnsureConnected(context.Context) error { s.ready = true; return nil }
func (s *stubAdapter) Ping(context.Context) error            { return nil }
func (s *stubAdapter) Close(context.Context) error           { s.ready = false; return nil }
func (s *stubAdapter) Connected() bool                       { return s.ready }
func (s *stubAdapter) Landscape(context.Context) (any, error) {
	return map[string]string{"ok": "true"}, nil
}

func TestShouldRegister(t *testing.T) {
	t.Parallel()
	enabled := true
	sp := &spec.Spec{
		Connections: []spec.Connection{
			{
				Name:           "mongo",
				TypeNormalized: "mongodb",
				AccessMode:     access.ModeReadOnly,
				EnabledVal:     true,
				Enabled:        &enabled,
			},
			{
				Name:           "pg",
				TypeNormalized: "postgres",
				AccessMode:     access.ModeReadWrite,
				EnabledVal:     true,
				Enabled:        &enabled,
				Tools:          spec.ToolsConfig{Exclude: []string{ToolPostgresExecute}},
			},
		},
		Tools: spec.ToolsConfig{Exclude: []string{ToolMongoSchema}},
	}
	reg, err := db.NewRegistry(context.Background(), sp, func(c spec.Connection) (db.Adapter, error) {
		return &stubAdapter{Meta: db.Meta{Conn: c}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{Spec: sp, Registry: reg}

	if !ShouldRegister(toolSpec{Name: ToolListConnections, Op: access.OpRead}, app) {
		t.Fatal("common tools should register")
	}
	if !ShouldRegister(toolSpec{Name: ToolMongoFind, DBType: db.TypeMongoDB, Op: access.OpRead}, app) {
		t.Fatal("mongo read should register")
	}
	if ShouldRegister(toolSpec{Name: ToolMongoInsert, DBType: db.TypeMongoDB, Op: access.OpWrite}, app) {
		t.Fatal("mongo write should not register when all mongo connections are read_only")
	}
	if ShouldRegister(toolSpec{Name: ToolMongoSchema, DBType: db.TypeMongoDB, Op: access.OpRead}, app) {
		t.Fatal("globally excluded tool should not register")
	}
	if ShouldRegister(toolSpec{Name: ToolRedisGet, DBType: db.TypeRedis, Op: access.OpRead}, app) {
		t.Fatal("redis tools should not register without a redis connection")
	}
	if ShouldRegister(toolSpec{Name: ToolPostgresExecute, DBType: db.TypePostgres, Op: access.OpWrite}, app) {
		t.Fatal("per-connection exclude should hide postgres_execute")
	}
	if !ShouldRegister(toolSpec{Name: ToolPostgresQuery, DBType: db.TypePostgres, Op: access.OpRead}, app) {
		t.Fatal("postgres_query should register")
	}
	if !ShouldRegister(toolSpec{Name: ToolListPermissions, Op: access.OpRead}, app) {
		t.Fatal("list_permissions should always register")
	}
}

func TestPermissionReports(t *testing.T) {
	t.Parallel()
	enabled := true
	sp := &spec.Spec{
		Connections: []spec.Connection{
			{
				Name:           "mongo",
				TypeNormalized: "mongodb",
				AccessMode:     access.ModeReadOnly,
				EnabledVal:     true,
				Enabled:        &enabled,
				Database:       "app",
			},
			{
				Name:           "pg",
				TypeNormalized: "postgres",
				AccessMode:     access.ModeReadWrite,
				EnabledVal:     true,
				Enabled:        &enabled,
				Tools:          spec.ToolsConfig{Exclude: []string{ToolPostgresExecute}},
			},
		},
		Tools: spec.ToolsConfig{Exclude: []string{ToolMongoSchema}},
	}
	reg, err := db.NewRegistry(context.Background(), sp, func(c spec.Connection) (db.Adapter, error) {
		return &stubAdapter{Meta: db.Meta{Conn: c}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{Spec: sp, Registry: reg}

	all, err := app.permissionReports(context.Background(), "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(all))
	}

	one, err := app.permissionReports(context.Background(), "mongo", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(one) != 1 || one[0].Connection != "mongo" {
		t.Fatalf("single connection report: %+v", one)
	}
	if !one[0].CanRead || one[0].CanWrite || one[0].CanAdmin {
		t.Fatalf("mongo should be read_only: %+v", one[0])
	}
	if !containsString(one[0].AllowedTools, ToolMongoFind) {
		t.Fatalf("mongo_find should be allowed: %v", one[0].AllowedTools)
	}
	if !containsString(one[0].DeniedTools, ToolMongoInsert) {
		t.Fatalf("mongo_insert should be denied: %v", one[0].DeniedTools)
	}
	if !containsString(one[0].DeniedTools, ToolMongoSchema) {
		t.Fatalf("globally excluded mongo_schema should be denied: %v", one[0].DeniedTools)
	}
	if containsString(one[0].AllowedTools, ToolPostgresQuery) || containsString(one[0].DeniedTools, ToolPostgresQuery) {
		t.Fatal("postgres tools should not appear on a mongo permission report")
	}

	pg, err := app.permissionReports(context.Background(), "pg", false)
	if err != nil {
		t.Fatal(err)
	}
	if !pg[0].CanWrite || containsString(pg[0].AllowedTools, ToolPostgresExecute) {
		t.Fatalf("postgres_execute is excluded: %+v", pg[0])
	}
	if !containsString(pg[0].DeniedTools, ToolPostgresExecute) {
		t.Fatalf("postgres_execute should be in denied_tools: %v", pg[0].DeniedTools)
	}

	if _, err := app.permissionReports(context.Background(), "missing", false); err == nil {
		t.Fatal("expected unknown connection error")
	}
}
