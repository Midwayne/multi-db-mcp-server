package server

import (
	"context"
	"encoding/json"
	"testing"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/spec"
)

func TestMultipleConnectionsPerEngine(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, multipleSameTypeConnections(), spec.ToolsConfig{})

	info := app.Registry.List()
	if len(info) != 6 {
		t.Fatalf("expected 6 connections, got %d", len(info))
	}
	counts := map[db.Type]int{}
	byName := map[string]db.Info{}
	for _, item := range info {
		counts[item.Type]++
		byName[item.Name] = item
	}
	if counts[db.TypeMongoDB] != 2 || counts[db.TypePostgres] != 2 || counts[db.TypeRedis] != 2 {
		t.Fatalf("want two of each engine, got %v", counts)
	}

	all, err := app.permissionReports(context.Background(), "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 6 {
		t.Fatalf("list_permissions for all should include every instance, got %d", len(all))
	}

	dev, err := app.permissionReports(context.Background(), "mongo-dev", false)
	if err != nil {
		t.Fatal(err)
	}
	prod, err := app.permissionReports(context.Background(), "mongo-prod", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(dev) != 1 || len(prod) != 1 {
		t.Fatal("viewing one mongo instance should return a single report")
	}
	if dev[0].CanWrite || !prod[0].CanWrite {
		t.Fatalf("mongo-dev write=%v mongo-prod write=%v", dev[0].CanWrite, prod[0].CanWrite)
	}
	if containsString(dev[0].AllowedTools, ToolMongoInsert) {
		t.Fatal("read_only mongo-dev must not allow mongo_insert")
	}
	if !containsString(prod[0].AllowedTools, ToolMongoInsert) {
		t.Fatal("read_write mongo-prod must allow mongo_insert")
	}

	if !ShouldRegister(toolSpec{Name: ToolMongoInsert, DBType: db.TypeMongoDB, Op: access.OpWrite}, app) {
		t.Fatal("mongo write tools should register when any mongo connection is read_write")
	}

	_, res := app.requireAdapter(context.Background(), connectionRequest("mongo-dev"), ToolMongoInsert, db.TypeMongoDB, access.OpWrite)
	if res == nil {
		t.Fatal("writes against mongo-dev should be denied")
	}
	adapter, res := app.requireAdapter(context.Background(), connectionRequest("mongo-prod"), ToolMongoInsert, db.TypeMongoDB, access.OpWrite)
	if res != nil {
		t.Fatalf("writes against mongo-prod should be allowed: %s", toolErrText(res))
	}
	if adapter.Name() != "mongo-prod" || adapter.DefaultDatabase() != "prod" {
		t.Fatalf("routed to the wrong mongo instance: %+v", adapter)
	}

	cases := []struct {
		name     string
		tool     string
		expected db.Type
	}{
		{"mongo-dev", ToolMongoFind, db.TypeMongoDB},
		{"mongo-prod", ToolMongoFind, db.TypeMongoDB},
		{"pg-analytics", ToolPostgresQuery, db.TypePostgres},
		{"pg-app", ToolPostgresQuery, db.TypePostgres},
		{"redis-cache", ToolRedisGet, db.TypeRedis},
		{"redis-jobs", ToolRedisGet, db.TypeRedis},
	}
	for _, tc := range cases {
		got, res := app.requireAdapter(context.Background(), connectionRequest(tc.name), tc.tool, tc.expected, access.OpRead)
		if res != nil {
			t.Fatalf("should talk to %s: %s", tc.name, toolErrText(res))
		}
		if got.Name() != tc.name || got.Type() != tc.expected {
			t.Fatalf("Get(%s) => %s/%s", tc.name, got.Name(), got.Type())
		}
	}

	_, res = app.requireAdapter(context.Background(), connectionRequest("pg-analytics"), ToolPostgresExecute, db.TypePostgres, access.OpWrite)
	if res == nil {
		t.Fatal("read_only pg-analytics should deny execute")
	}
	if _, res = app.requireAdapter(context.Background(), connectionRequest("pg-app"), ToolPostgresExecute, db.TypePostgres, access.OpWrite); res != nil {
		t.Fatalf("read_write pg-app should allow execute: %s", toolErrText(res))
	}
}

func TestMCPMultipleConnectionsPerEngine(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, multipleSameTypeConnections(), spec.ToolsConfig{})
	cl := newMCPClient(t, app)
	defer cl.Close()

	listedText := callToolText(t, cl, ToolListConnections, nil)
	var listed []map[string]any
	if err := json.Unmarshal([]byte(listedText), &listed); err != nil {
		t.Fatalf("list_connections: %v\n%s", err, listedText)
	}
	if len(listed) != 6 {
		t.Fatalf("list_connections should return all six instances, got %d", len(listed))
	}
	names := map[string]string{}
	for _, item := range listed {
		name, _ := item["name"].(string)
		typ, _ := item["type"].(string)
		names[name] = typ
	}
	want := map[string]string{
		"mongo-dev": "mongodb", "mongo-prod": "mongodb",
		"pg-analytics": "postgres", "pg-app": "postgres",
		"redis-cache": "redis", "redis-jobs": "redis",
	}
	for name, typ := range want {
		if names[name] != typ {
			t.Fatalf("missing or wrong type for %s: %v", name, names)
		}
	}

	permAll := callToolText(t, cl, ToolListPermissions, nil)
	var allReports []permissionReport
	if err := json.Unmarshal([]byte(permAll), &allReports); err != nil {
		t.Fatalf("list_permissions all: %v", err)
	}
	if len(allReports) != 6 {
		t.Fatalf("list_permissions all got %d", len(allReports))
	}

	devText := callToolText(t, cl, ToolListPermissions, map[string]any{"connection": "mongo-dev"})
	var dev []permissionReport
	if err := json.Unmarshal([]byte(devText), &dev); err != nil {
		t.Fatal(err)
	}
	if len(dev) != 1 || dev[0].Connection != "mongo-dev" || dev[0].CanWrite {
		t.Fatalf("mongo-dev permissions: %+v", dev)
	}

	prodText := callToolText(t, cl, ToolListPermissions, map[string]any{"connection": "mongo-prod"})
	var prod []permissionReport
	if err := json.Unmarshal([]byte(prodText), &prod); err != nil {
		t.Fatal(err)
	}
	if len(prod) != 1 || prod[0].Connection != "mongo-prod" || !prod[0].CanWrite {
		t.Fatalf("mongo-prod permissions: %+v", prod)
	}

	for name := range want {
		text := callToolText(t, cl, ToolPing, map[string]any{"connection": name})
		var payload map[string]any
		if err := json.Unmarshal([]byte(text), &payload); err != nil {
			t.Fatalf("ping %s: %v", name, err)
		}
		if payload["connection"] != name {
			t.Fatalf("ping %s payload: %s", name, text)
		}
	}
}
