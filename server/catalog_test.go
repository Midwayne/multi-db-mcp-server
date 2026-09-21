package server

import (
	"testing"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/spec"
)

func TestShouldRegister(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, []spec.Connection{
		{Name: "mongo", TypeNormalized: "mongodb", AccessMode: access.ModeReadOnly},
		{
			Name:           "pg",
			TypeNormalized: "postgres",
			AccessMode:     access.ModeReadWrite,
			Tools:          spec.ToolsConfig{Exclude: []string{ToolPostgresExecute}},
		},
	}, spec.ToolsConfig{Exclude: []string{ToolMongoSchema}})

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
