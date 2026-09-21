package server

import (
	"context"
	"strings"
	"testing"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/spec"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestTalkToAnyDefinedDB(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, mixedConnections(), spec.ToolsConfig{})

	cases := []struct {
		name     string
		tool     string
		expected db.Type
		op       access.Operation
	}{
		{"mongo", ToolMongoFind, db.TypeMongoDB, access.OpRead},
		{"pg", ToolPostgresQuery, db.TypePostgres, access.OpRead},
		{"cache", ToolRedisGet, db.TypeRedis, access.OpRead},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter, res := app.requireAdapter(context.Background(), connectionRequest(tc.name), tc.tool, tc.expected, tc.op)
			if res != nil {
				t.Fatalf("should talk to %s: %s", tc.name, toolErrText(res))
			}
			if adapter.Name() != tc.name || adapter.Type() != tc.expected {
				t.Fatalf("got %s/%s want %s/%s", adapter.Name(), adapter.Type(), tc.name, tc.expected)
			}
		})
	}
}

func TestTalkToDefinedDBRejectsTypeMismatch(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, mixedConnections(), spec.ToolsConfig{})
	_, res := app.requireAdapter(context.Background(), connectionRequest("pg"), ToolMongoFind, db.TypeMongoDB, access.OpRead)
	if res == nil {
		t.Fatal("postgres connection should not accept mongo tools")
	}
	if !strings.Contains(toolErrText(res), "expected mongodb") {
		t.Fatalf("unexpected error: %s", toolErrText(res))
	}
}

func TestTalkToDefinedDBEnforcesAccessAndAllowlist(t *testing.T) {
	t.Parallel()
	conns := []spec.Connection{
		{Name: "mongo", TypeNormalized: "mongodb", AccessMode: access.ModeReadOnly},
		{
			Name:           "pg",
			TypeNormalized: "postgres",
			AccessMode:     access.ModeReadWrite,
			Tools:          spec.ToolsConfig{Exclude: []string{ToolPostgresStats}},
		},
	}
	app := newTestApp(t, conns, spec.ToolsConfig{})

	_, res := app.requireAdapter(context.Background(), connectionRequest("mongo"), ToolMongoInsert, db.TypeMongoDB, access.OpWrite)
	if res == nil || !strings.Contains(toolErrText(res), "does not allow write") {
		t.Fatalf("read_only mongo should deny writes: %v", res)
	}

	_, res = app.requireAdapter(context.Background(), connectionRequest("pg"), ToolPostgresStats, db.TypePostgres, access.OpRead)
	if res == nil || !strings.Contains(toolErrText(res), "not enabled") {
		t.Fatalf("excluded tool should be denied: %v", res)
	}

	_, res = app.requireAdapter(context.Background(), connectionRequest("missing"), ToolPing, db.TypeRedis, access.OpRead)
	if res == nil || !strings.Contains(toolErrText(res), "unknown connection") {
		t.Fatalf("missing connection: %v", res)
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	_, res = app.requireAdapter(context.Background(), req, ToolPing, db.TypeRedis, access.OpRead)
	if res == nil {
		t.Fatal("connection argument is required")
	}
}

func TestPingAndLandscapeEverySpecDB(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, mixedConnections(), spec.ToolsConfig{})
	for _, name := range []string{"mongo", "pg", "cache"} {
		adapter, err := app.Registry.Get(context.Background(), name)
		if err != nil {
			t.Fatalf("Get(%s): %v", name, err)
		}
		if err := adapter.Ping(context.Background()); err != nil {
			t.Fatalf("Ping(%s): %v", name, err)
		}
	}
	landscape := collectLandscape(context.Background(), app.Registry)
	for _, name := range []string{"mongo", "pg", "cache"} {
		entry, ok := landscape[name].(map[string]any)
		if !ok {
			t.Fatalf("landscape missing %s: %#v", name, landscape)
		}
		if entry["namespaces"] == nil {
			t.Fatalf("landscape namespaces missing for %s: %#v", name, entry)
		}
	}
}

func connectionRequest(name string) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"connection": name}
	return req
}

func toolErrText(res *mcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	if text, ok := res.Content[0].(mcp.TextContent); ok {
		return text.Text
	}
	return ""
}
