package server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/spec"
)

func TestListPermissionsForAllAndOne(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, mixedConnections(), spec.ToolsConfig{})

	all, err := app.permissionReports(context.Background(), "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("expected permissions for every spec DB, got %d", len(all))
	}

	byName := map[string]permissionReport{}
	for _, report := range all {
		byName[report.Connection] = report
	}
	for _, name := range []string{"mongo", "pg", "cache"} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("missing permissions for %s", name)
		}
	}

	mongo := byName["mongo"]
	if mongo.Type != db.TypeMongoDB || !mongo.CanRead || mongo.CanWrite || mongo.CanAdmin {
		t.Fatalf("mongo permissions: %+v", mongo)
	}
	if mongo.Database != "app" {
		t.Fatalf("mongo default database: %s", mongo.Database)
	}
	if !containsString(mongo.AllowedTools, ToolMongoFind) || containsString(mongo.AllowedTools, ToolMongoInsert) {
		t.Fatalf("mongo tools: allowed=%v denied=%v", mongo.AllowedTools, mongo.DeniedTools)
	}

	pg := byName["pg"]
	if !pg.CanRead || !pg.CanWrite || pg.CanAdmin {
		t.Fatalf("postgres should be read_write: %+v", pg)
	}
	if !containsString(pg.AllowedTools, ToolPostgresQuery) || !containsString(pg.AllowedTools, ToolPostgresExecute) {
		t.Fatalf("postgres DML should be allowed: %v", pg.AllowedTools)
	}
	if !strings.Contains(strings.Join(pg.Notes, " "), "DDL") {
		t.Fatalf("postgres read_write should note that DDL needs admin: %v", pg.Notes)
	}

	cache := byName["cache"]
	if !cache.CanRead || !cache.CanWrite || !cache.CanAdmin {
		t.Fatalf("redis admin flags: %+v", cache)
	}
	if !containsString(cache.AllowedTools, ToolRedisGet) || !containsString(cache.AllowedTools, ToolRedisSet) {
		t.Fatalf("redis admin should allow get and set: %v", cache.AllowedTools)
	}
	if len(cache.Notes) == 0 {
		t.Fatal("redis should note redis_command write/admin gating")
	}

	one, err := app.permissionReports(context.Background(), "pg", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(one) != 1 || one[0].Connection != "pg" {
		t.Fatalf("single DB view should return only pg: %+v", one)
	}
	if _, err := app.permissionReports(context.Background(), "no-such-db", false); err == nil {
		t.Fatal("unknown connection should error")
	}
}

func TestListPermissionsJSONRoundTrip(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, mixedConnections(), spec.ToolsConfig{})
	reports, err := app.permissionReports(context.Background(), "", false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.ToJSON(reports)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []permissionReport
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("permission JSON should decode: %v\n%s", err, raw)
	}
	if len(decoded) != 3 {
		t.Fatalf("decoded %d reports", len(decoded))
	}
}

func TestListPermissionsIncludeServer(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, mixedConnections(), spec.ToolsConfig{})
	reports, err := app.permissionReports(context.Background(), "mongo", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("got %d reports", len(reports))
	}
	if reports[0].ServerError != "" {
		t.Fatalf("stub identity should not error: %s", reports[0].ServerError)
	}
	if reports[0].Server != nil {
		t.Fatalf("stub adapters have no live identity, got %#v", reports[0].Server)
	}
	if !reports[0].Ready {
		t.Fatal("include_server should connect the named DB")
	}
}

func TestListPermissionsIncludeServerConnectError(t *testing.T) {
	t.Parallel()
	conns := []spec.Connection{{
		Name:           "down",
		TypeNormalized: "redis",
		AccessMode:     access.ModeReadOnly,
		EnabledVal:     true,
		Enabled:        boolPtr(true),
		URI:            "stub://",
		Options:        map[string]any{"connect_error": errors.New("offline")},
	}}
	app := newTestApp(t, conns, spec.ToolsConfig{})
	reports, err := app.permissionReports(context.Background(), "down", true)
	if err != nil {
		t.Fatal(err)
	}
	if reports[0].ServerError == "" {
		t.Fatal("expected server_error when the DB cannot be reached")
	}
	if reports[0].Access != access.ModeReadOnly {
		t.Fatal("spec permissions should still be returned when the server is down")
	}
}

func TestListConnectionsPermissionFlags(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, mixedConnections(), spec.ToolsConfig{})
	info := app.Registry.List()
	if len(info) != 3 {
		t.Fatalf("expected 3 connections, got %d", len(info))
	}
	flags := map[string]db.Info{}
	for _, item := range info {
		flags[item.Name] = item
	}
	if !flags["mongo"].CanRead || flags["mongo"].CanWrite {
		t.Fatalf("mongo flags: %+v", flags["mongo"])
	}
	if !flags["pg"].CanWrite || flags["pg"].CanAdmin {
		t.Fatalf("pg flags: %+v", flags["pg"])
	}
	if !flags["cache"].CanAdmin {
		t.Fatalf("cache flags: %+v", flags["cache"])
	}
}
