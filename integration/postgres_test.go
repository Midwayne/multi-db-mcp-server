package integration

import (
	"strings"
	"testing"

	"dbmcp/server"
)

func TestPostgresLiveBehaviour(t *testing.T) {
	pg := requirePostgres(t)
	seedPostgres(t, pg.Port)

	app := newLiveApp(t, `
server:
  name: postgres-integration
  version: test
defaults:
  connect_timeout: 8s
  fail_on_connect_error: true
  connect_on_start: true
  max_rows: 50
connections:
  - name: pg-analytics
    type: postgres
    uri: `+pgURI(pg.Port, "analytics")+`
    access: read_only
  - name: pg-app
    type: postgres
    uri: `+pgURI(pg.Port, "app")+`
    access: read_write
  - name: pg-admin
    type: postgres
    uri: `+pgURI(pg.Port, "app")+`
    access: admin
`)
	cl := newMCPClient(t, app)

	listed := callToolText(t, cl, server.ToolListConnections, nil)
	if !strings.Contains(listed, "pg-analytics") || !strings.Contains(listed, "pg-app") || !strings.Contains(listed, "pg-admin") {
		t.Fatalf("list_connections: %s", listed)
	}

	for _, name := range []string{"pg-analytics", "pg-app", "pg-admin"} {
		_ = callToolText(t, cl, server.ToolPing, map[string]any{"connection": name})
	}

	metrics := callToolText(t, cl, server.ToolPostgresQuery, map[string]any{
		"connection": "pg-analytics",
		"sql":        "SELECT id, value FROM metrics ORDER BY id",
	})
	var metricRows queryResult
	decodeJSON(t, metrics, &metricRows)
	if metricRows.RowCount != 1 || rowValue(metricRows.Rows[0], "value") != "42" {
		t.Fatalf("analytics seed: %s", metrics)
	}

	items := callToolText(t, cl, server.ToolPostgresQuery, map[string]any{
		"connection": "pg-app",
		"sql":        "SELECT id, name FROM items WHERE id = $1",
		"params":     "[1]",
	})
	var itemRows queryResult
	decodeJSON(t, items, &itemRows)
	if itemRows.RowCount != 1 || rowValue(itemRows.Rows[0], "name") != "widget" {
		t.Fatalf("app seed: %s", items)
	}

	isolated := callTool(t, cl, server.ToolPostgresQuery, map[string]any{
		"connection": "pg-analytics",
		"sql":        "SELECT name FROM items",
	})
	if !isolated.IsError {
		t.Fatalf("analytics should not see app.items, got %s", toolText(isolated))
	}

	writeDenied := callToolError(t, cl, server.ToolPostgresExecute, map[string]any{
		"connection": "pg-analytics",
		"sql":        "INSERT INTO metrics (id, value) VALUES (2, 99)",
	})
	if !containsFold(writeDenied, "does not allow write") {
		t.Fatalf("read_only execute: %s", writeDenied)
	}

	queryWrite := callToolError(t, cl, server.ToolPostgresQuery, map[string]any{
		"connection": "pg-app",
		"sql":        "INSERT INTO items (id, name) VALUES (2, 'sneak')",
	})
	if !containsFold(queryWrite, "postgres_query only allows read") {
		t.Fatalf("writes via postgres_query: %s", queryWrite)
	}

	inserted := callToolText(t, cl, server.ToolPostgresExecute, map[string]any{
		"connection": "pg-app",
		"sql":        "INSERT INTO items (id, name) VALUES ($1, $2)",
		"params":     `[2, "gadget"]`,
	})
	if !strings.Contains(inserted, `"rows_affected": 1`) && !strings.Contains(inserted, `"rows_affected":1`) {
		t.Fatalf("insert: %s", inserted)
	}
	after := callToolText(t, cl, server.ToolPostgresQuery, map[string]any{
		"connection": "pg-app",
		"sql":        "SELECT name FROM items WHERE id = 2",
	})
	if !strings.Contains(after, "gadget") {
		t.Fatalf("app should see inserted row: %s", after)
	}

	ddlDenied := callToolError(t, cl, server.ToolPostgresExecute, map[string]any{
		"connection": "pg-app",
		"sql":        "CREATE TABLE extra (id INT PRIMARY KEY)",
	})
	if !containsFold(ddlDenied, "does not allow admin") {
		t.Fatalf("read_write DDL: %s", ddlDenied)
	}

	_ = callToolText(t, cl, server.ToolPostgresExecute, map[string]any{
		"connection": "pg-admin",
		"sql":        "CREATE TABLE extra (id INT PRIMARY KEY, note TEXT)",
	})
	_ = callToolText(t, cl, server.ToolPostgresExecute, map[string]any{
		"connection": "pg-admin",
		"sql":        "INSERT INTO extra (id, note) VALUES (1, 'ok')",
	})
	seen := callToolText(t, cl, server.ToolPostgresQuery, map[string]any{
		"connection": "pg-app",
		"sql":        "SELECT note FROM extra",
	})
	if !strings.Contains(seen, "ok") {
		t.Fatalf("app connection should see admin-created table: %s", seen)
	}

	tables := callToolText(t, cl, server.ToolPostgresListTables, map[string]any{
		"connection": "pg-analytics",
	})
	if !strings.Contains(tables, "metrics") || strings.Contains(tables, `"table_name": "items"`) {
		t.Fatalf("analytics tables: %s", tables)
	}

	dbs := callToolText(t, cl, server.ToolPostgresListDatabases, map[string]any{
		"connection": "pg-app",
	})
	var databases []struct {
		Name    string   `json:"name"`
		Schemas []string `json:"schemas"`
	}
	decodeJSON(t, dbs, &databases)
	byDB := map[string][]string{}
	for _, info := range databases {
		byDB[info.Name] = info.Schemas
	}
	if !containsString(byDB["app"], "public") {
		t.Fatalf("list databases missing app.public: %s", dbs)
	}
	if !containsString(byDB["analytics"], "public") {
		t.Fatalf("list databases missing analytics.public: %s", dbs)
	}

	desc := callToolText(t, cl, server.ToolPostgresDescribeTable, map[string]any{
		"connection": "pg-app",
		"table":      "items",
	})
	if !strings.Contains(desc, "column_name") || !strings.Contains(desc, "name") {
		t.Fatalf("describe items: %s", desc)
	}

	permAll := callToolText(t, cl, server.ToolListPermissions, map[string]any{"include_server": true})
	var reports []permissionReport
	decodeJSON(t, permAll, &reports)
	if len(reports) != 3 {
		t.Fatalf("permissions for all postgres connections: %s", permAll)
	}
	byName := map[string]permissionReport{}
	for _, r := range reports {
		byName[r.Connection] = r
		if !r.Ready {
			t.Fatalf("%s not ready: %+v", r.Connection, r)
		}
		if r.ServerError != "" {
			t.Fatalf("%s server error: %s", r.Connection, r.ServerError)
		}
		if r.Server == nil {
			t.Fatalf("%s missing live identity", r.Connection)
		}
	}
	if byName["pg-analytics"].CanWrite || !byName["pg-app"].CanWrite || !byName["pg-admin"].CanAdmin {
		t.Fatalf("access flags: %+v", byName)
	}
	if containsString(byName["pg-analytics"].AllowedTools, server.ToolPostgresExecute) {
		t.Fatal("read_only analytics must not allow postgres_execute")
	}
}
