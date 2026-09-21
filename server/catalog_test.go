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
}
