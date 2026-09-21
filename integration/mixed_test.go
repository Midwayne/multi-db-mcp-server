package integration

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"dbmcp/server"
)

func TestMixedEnginesLiveBehaviour(t *testing.T) {
	skipIfShort(t)

	cache := startRedis(t)
	if err := cache.Set("health", "ok"); err != nil {
		t.Fatal(err)
	}

	var b strings.Builder
	b.WriteString(`
server:
  name: mixed-integration
  version: test
defaults:
  connect_timeout: 8s
  fail_on_connect_error: true
  connect_on_start: true
connections:
  - name: cache
    type: redis
    uri: ` + redisURI(cache) + `
    access: read_only
`)

	engineCount := 1
	pg := optionalPostgres(t)
	if pg != nil {
		seedPostgres(t, pg.Port)
		fmt.Fprintf(&b, `
  - name: app
    type: postgres
    uri: %s
    access: read_write
`, pgURI(pg.Port, "app"))
		engineCount++
	}

	mongo := optionalMongo(t)
	if mongo != nil {
		seedMongo(t, mongo.URI())
		fmt.Fprintf(&b, `
  - name: catalog
    type: mongodb
    uri: %s
    database: inventory
    access: read_only
`, mongo.URI())
		engineCount++
	}

	if engineCount < 2 {
		t.Skip("need postgres or mongo in addition to redis for mixed-engine coverage")
	}

	app := newLiveApp(t, b.String())
	cl := newMCPClient(t, app)

	listedText := callToolText(t, cl, server.ToolListConnections, nil)
	var listed []map[string]any
	decodeJSON(t, listedText, &listed)
	if len(listed) != engineCount {
		t.Fatalf("list_connections want %d, got %s", engineCount, listedText)
	}

	permText := callToolText(t, cl, server.ToolListPermissions, nil)
	var reports []permissionReport
	decodeJSON(t, permText, &reports)
	if len(reports) != engineCount {
		t.Fatalf("list_permissions want %d, got %s", engineCount, permText)
	}

	_ = callToolText(t, cl, server.ToolPing, map[string]any{"connection": "cache"})
	got := callToolText(t, cl, server.ToolRedisGet, map[string]any{"connection": "cache", "key": "health"})
	if !strings.Contains(got, "ok") {
		t.Fatalf("redis get: %s", got)
	}
	_ = callToolError(t, cl, server.ToolRedisSet, map[string]any{
		"connection": "cache",
		"key":        "health",
		"value":      "mutated",
	})

	if pg != nil {
		_ = callToolText(t, cl, server.ToolPing, map[string]any{"connection": "app"})
		rows := callToolText(t, cl, server.ToolPostgresQuery, map[string]any{
			"connection": "app",
			"sql":        "SELECT name FROM items WHERE id = 1",
		})
		if !strings.Contains(rows, "widget") {
			t.Fatalf("postgres query: %s", rows)
		}
		cross := callToolError(t, cl, server.ToolRedisGet, map[string]any{
			"connection": "app",
			"key":        "health",
		})
		if !containsFold(cross, "expected redis") {
			t.Fatalf("postgres must not answer redis tools: %s", cross)
		}
	}

	if mongo != nil {
		_ = callToolText(t, cl, server.ToolPing, map[string]any{"connection": "catalog"})
		docs := callToolText(t, cl, server.ToolMongoFind, map[string]any{
			"connection":      "catalog",
			"collection_name": "products",
		})
		var found queryResult
		decodeJSON(t, docs, &found)
		if !docsHave(found.Documents, "sku", "A-1") {
			t.Fatalf("mongo find: %s", docs)
		}
		writeDenied := callToolError(t, cl, server.ToolMongoInsert, map[string]any{
			"connection":      "catalog",
			"collection_name": "products",
			"documents":       `{"sku":"nope"}`,
		})
		if !containsFold(writeDenied, "does not allow write") {
			t.Fatalf("read_only mongo write: %s", writeDenied)
		}
		cross := callToolError(t, cl, server.ToolRedisGet, map[string]any{
			"connection": "catalog",
			"key":        "health",
		})
		if !containsFold(cross, "expected redis") {
			t.Fatalf("mongo must not answer redis tools: %s", cross)
		}
	}

	landscape := callToolText(t, cl, server.ToolLandscape, nil)
	var namespaces map[string]any
	if err := json.Unmarshal([]byte(landscape), &namespaces); err != nil {
		t.Fatalf("landscape: %v\n%s", err, landscape)
	}
	if _, ok := namespaces["cache"]; !ok {
		t.Fatalf("landscape missing cache: %s", landscape)
	}
	if pg != nil {
		if _, ok := namespaces["app"]; !ok {
			t.Fatalf("landscape missing app: %s", landscape)
		}
	}
	if mongo != nil {
		if _, ok := namespaces["catalog"]; !ok {
			t.Fatalf("landscape missing catalog: %s", landscape)
		}
	}
}
