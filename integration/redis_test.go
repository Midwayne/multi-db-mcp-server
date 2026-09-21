package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"dbmcp/server"
)

func TestRedisLiveBehaviour(t *testing.T) {
	skipIfShort(t)

	cache := startRedis(t)
	jobs := startRedis(t)
	if err := cache.Set("greeting", "from-cache"); err != nil {
		t.Fatal(err)
	}
	if err := jobs.Set("queue", "from-jobs"); err != nil {
		t.Fatal(err)
	}

	app := newLiveApp(t, `
server:
  name: redis-integration
  version: test
defaults:
  connect_timeout: 5s
  fail_on_connect_error: true
  connect_on_start: true
connections:
  - name: redis-cache
    type: redis
    uri: `+redisURI(cache)+`
    access: read_only
  - name: redis-jobs
    type: redis
    uri: `+redisURI(jobs)+`
    access: admin
`)
	cl := newMCPClient(t, app)

	listedText := callToolText(t, cl, server.ToolListConnections, nil)
	var listed []map[string]any
	decodeJSON(t, listedText, &listed)
	if len(listed) != 2 {
		t.Fatalf("list_connections: %s", listedText)
	}

	for _, name := range []string{"redis-cache", "redis-jobs"} {
		ping := callToolText(t, cl, server.ToolPing, map[string]any{"connection": name})
		if !strings.Contains(ping, name) {
			t.Fatalf("ping %s: %s", name, ping)
		}
	}

	cacheGet := callToolText(t, cl, server.ToolRedisGet, map[string]any{
		"connection": "redis-cache",
		"key":        "greeting",
	})
	if !strings.Contains(cacheGet, "from-cache") {
		t.Fatalf("cache get: %s", cacheGet)
	}
	jobsGet := callToolText(t, cl, server.ToolRedisGet, map[string]any{
		"connection": "redis-jobs",
		"key":        "queue",
	})
	if !strings.Contains(jobsGet, "from-jobs") {
		t.Fatalf("jobs get: %s", jobsGet)
	}

	missing := callToolError(t, cl, server.ToolRedisGet, map[string]any{
		"connection": "redis-cache",
		"key":        "queue",
	})
	if !containsFold(missing, "not found") {
		t.Fatalf("cache should not see jobs keys: %s", missing)
	}

	denied := callToolError(t, cl, server.ToolRedisSet, map[string]any{
		"connection": "redis-cache",
		"key":        "greeting",
		"value":      "mutated",
	})
	if !containsFold(denied, "does not allow write") {
		t.Fatalf("read_only set: %s", denied)
	}
	if got, _ := cache.Get("greeting"); got != "from-cache" {
		t.Fatalf("read_only set mutated the key: %q", got)
	}

	cmdDenied := callToolError(t, cl, server.ToolRedisCommand, map[string]any{
		"connection": "redis-cache",
		"command":    "SET",
		"args":       []any{"greeting", "via-command"},
	})
	if !containsFold(cmdDenied, "does not allow write") {
		t.Fatalf("read_only redis_command SET: %s", cmdDenied)
	}

	setOK := callToolText(t, cl, server.ToolRedisSet, map[string]any{
		"connection": "redis-jobs",
		"key":        "worker",
		"value":      "busy",
	})
	if !strings.Contains(setOK, "worker") {
		t.Fatalf("admin set: %s", setOK)
	}
	got := callToolText(t, cl, server.ToolRedisGet, map[string]any{
		"connection": "redis-jobs",
		"key":        "worker",
	})
	if !strings.Contains(got, "busy") {
		t.Fatalf("admin get after set: %s", got)
	}

	scan := callToolText(t, cl, server.ToolRedisScan, map[string]any{
		"connection": "redis-jobs",
		"pattern":    "*",
	})
	var scanned map[string]any
	decodeJSON(t, scan, &scanned)
	keys, _ := scanned["keys"].([]any)
	if len(keys) < 2 {
		t.Fatalf("jobs scan should include seeded and written keys: %s", scan)
	}

	flushDenied := callToolError(t, cl, server.ToolRedisCommand, map[string]any{
		"connection": "redis-cache",
		"command":    "FLUSHALL",
	})
	if !containsFold(flushDenied, "does not allow admin") {
		t.Fatalf("read_only FLUSHALL: %s", flushDenied)
	}

	_ = callToolText(t, cl, server.ToolRedisCommand, map[string]any{
		"connection": "redis-jobs",
		"command":    "FLUSHALL",
	})
	if cache.Exists("greeting") != true {
		t.Fatal("FLUSHALL on jobs must not wipe the other redis instance")
	}
	gone := callToolError(t, cl, server.ToolRedisGet, map[string]any{
		"connection": "redis-jobs",
		"key":        "queue",
	})
	if !containsFold(gone, "not found") {
		t.Fatalf("jobs should be empty after FLUSHALL: %s", gone)
	}

	permText := callToolText(t, cl, server.ToolListPermissions, map[string]any{
		"connection":     "redis-cache",
		"include_server": true,
	})
	var reports []permissionReport
	decodeJSON(t, permText, &reports)
	if len(reports) != 1 || reports[0].CanWrite || !reports[0].Ready {
		t.Fatalf("cache permissions: %s", permText)
	}
	if containsString(reports[0].AllowedTools, server.ToolRedisSet) {
		t.Fatal("read_only redis must not list redis_set as allowed")
	}

	delDenied := callToolError(t, cl, server.ToolRedisDelete, map[string]any{
		"connection": "redis-cache",
		"keys":       []any{"greeting"},
	})
	if !containsFold(delDenied, "does not allow write") {
		t.Fatalf("read_only delete: %s", delDenied)
	}

	landscape := callToolText(t, cl, server.ToolLandscape, nil)
	var namespaces map[string]any
	if err := json.Unmarshal([]byte(landscape), &namespaces); err != nil {
		t.Fatalf("landscape: %v\n%s", err, landscape)
	}
	if _, ok := namespaces["redis-cache"]; !ok {
		t.Fatalf("landscape missing redis-cache: %s", landscape)
	}
	if _, ok := namespaces["redis-jobs"]; !ok {
		t.Fatalf("landscape missing redis-jobs: %s", landscape)
	}
}
