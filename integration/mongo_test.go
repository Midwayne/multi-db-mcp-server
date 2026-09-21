package integration

import (
	"strings"
	"testing"

	"dbmcp/server"
)

func TestMongoLiveBehaviour(t *testing.T) {
	mongo := requireMongo(t)
	seedMongo(t, mongo.URI())

	app := newLiveApp(t, `
server:
  name: mongo-integration
  version: test
defaults:
  connect_timeout: 8s
  fail_on_connect_error: true
  connect_on_start: true
  max_rows: 50
connections:
  - name: mongo-ro
    type: mongodb
    uri: `+mongo.URI()+`
    database: inventory
    access: read_only
  - name: mongo-rw
    type: mongodb
    uri: `+mongo.URI()+`
    database: orders
    access: read_write
`)
	cl := newMCPClient(t, app)

	listed := callToolText(t, cl, server.ToolListConnections, nil)
	if !strings.Contains(listed, "mongo-ro") || !strings.Contains(listed, "mongo-rw") {
		t.Fatalf("list_connections: %s", listed)
	}
	_ = callToolText(t, cl, server.ToolPing, map[string]any{"connection": "mongo-ro"})
	_ = callToolText(t, cl, server.ToolPing, map[string]any{"connection": "mongo-rw"})

	inv := callToolText(t, cl, server.ToolMongoFind, map[string]any{
		"connection":      "mongo-ro",
		"collection_name": "products",
	})
	var invResult queryResult
	decodeJSON(t, inv, &invResult)
	if !docsHave(invResult.Documents, "sku", "A-1") {
		t.Fatalf("inventory find: %s", inv)
	}

	orders := callToolText(t, cl, server.ToolMongoFind, map[string]any{
		"connection":      "mongo-rw",
		"collection_name": "tickets",
	})
	var orderResult queryResult
	decodeJSON(t, orders, &orderResult)
	if !docsHave(orderResult.Documents, "sku", "T-9") {
		t.Fatalf("orders find: %s", orders)
	}

	cross := callToolText(t, cl, server.ToolMongoFind, map[string]any{
		"connection":      "mongo-ro",
		"collection_name": "tickets",
	})
	var crossResult queryResult
	decodeJSON(t, cross, &crossResult)
	if docsHave(crossResult.Documents, "sku", "T-9") {
		t.Fatalf("inventory connection should not default to orders data: %s", cross)
	}

	countText := callToolText(t, cl, server.ToolMongoCount, map[string]any{
		"connection":      "mongo-ro",
		"collection_name": "products",
		"filter":          `{"sku":"A-1"}`,
	})
	if !strings.Contains(countText, `"count": 1`) && !strings.Contains(countText, `"count":1`) {
		t.Fatalf("count: %s", countText)
	}

	insertDenied := callToolError(t, cl, server.ToolMongoInsert, map[string]any{
		"connection":      "mongo-ro",
		"collection_name": "products",
		"documents":       `{"sku":"blocked"}`,
	})
	if !containsFold(insertDenied, "does not allow write") {
		t.Fatalf("read_only insert: %s", insertDenied)
	}

	inserted := callToolText(t, cl, server.ToolMongoInsert, map[string]any{
		"connection":      "mongo-rw",
		"collection_name": "tickets",
		"documents":       `{"sku":"T-10","status":"new"}`,
	})
	if !strings.Contains(inserted, "inserted_ids") {
		t.Fatalf("insert: %s", inserted)
	}

	foundNew := callToolText(t, cl, server.ToolMongoFind, map[string]any{
		"connection":      "mongo-rw",
		"collection_name": "tickets",
		"filter":          `{"sku":"T-10"}`,
	})
	var found queryResult
	decodeJSON(t, foundNew, &found)
	if !docsHave(found.Documents, "status", "new") {
		t.Fatalf("find inserted ticket: %s", foundNew)
	}

	updated := callToolText(t, cl, server.ToolMongoUpdate, map[string]any{
		"connection":      "mongo-rw",
		"collection_name": "tickets",
		"filter":          `{"sku":"T-10"}`,
		"update":          `{"$set":{"status":"closed"}}`,
	})
	if !strings.Contains(updated, `"modified": 1`) && !strings.Contains(updated, `"modified":1`) {
		t.Fatalf("update: %s", updated)
	}
	closed := callToolText(t, cl, server.ToolMongoFind, map[string]any{
		"connection":      "mongo-rw",
		"collection_name": "tickets",
		"filter":          `{"sku":"T-10"}`,
	})
	var closedResult queryResult
	decodeJSON(t, closed, &closedResult)
	if !docsHave(closedResult.Documents, "status", "closed") {
		t.Fatalf("updated ticket: %s", closed)
	}

	deleted := callToolText(t, cl, server.ToolMongoDelete, map[string]any{
		"connection":      "mongo-rw",
		"collection_name": "tickets",
		"filter":          `{"sku":"T-10"}`,
	})
	if !strings.Contains(deleted, `"deleted": 1`) && !strings.Contains(deleted, `"deleted":1`) {
		t.Fatalf("delete: %s", deleted)
	}

	colls := callToolText(t, cl, server.ToolMongoListCollections, map[string]any{
		"connection": "mongo-ro",
	})
	if !strings.Contains(colls, "products") {
		t.Fatalf("list collections: %s", colls)
	}

	dbs := callToolText(t, cl, server.ToolMongoListDatabases, map[string]any{
		"connection": "mongo-rw",
	})
	var listed []struct {
		Name        string   `json:"name"`
		Collections []string `json:"collections"`
	}
	decodeJSON(t, dbs, &listed)
	byName := map[string][]string{}
	for _, info := range listed {
		byName[info.Name] = info.Collections
	}
	if !containsString(byName["inventory"], "products") {
		t.Fatalf("list databases missing inventory.products: %s", dbs)
	}
	if !containsString(byName["orders"], "tickets") {
		t.Fatalf("list databases missing orders.tickets: %s", dbs)
	}

	perm := callToolText(t, cl, server.ToolListPermissions, map[string]any{
		"connection":     "mongo-ro",
		"include_server": true,
	})
	var reports []permissionReport
	decodeJSON(t, perm, &reports)
	if len(reports) != 1 || reports[0].CanWrite {
		t.Fatalf("mongo-ro permissions: %s", perm)
	}
	if containsString(reports[0].AllowedTools, server.ToolMongoInsert) {
		t.Fatal("read_only mongo must not allow mongo_insert")
	}
	if reports[0].ServerError != "" {
		t.Fatalf("include_server: %s", reports[0].ServerError)
	}
	if !reports[0].Ready {
		t.Fatal("mongo-ro should be ready after include_server")
	}

	agg := callToolText(t, cl, server.ToolMongoAggregate, map[string]any{
		"connection":      "mongo-ro",
		"collection_name": "products",
		"pipeline":        `[{"$match":{"sku":"A-1"}},{"$project":{"_id":0,"sku":1}}]`,
	})
	var aggResult queryResult
	decodeJSON(t, agg, &aggResult)
	if !docsHave(aggResult.Documents, "sku", "A-1") {
		t.Fatalf("aggregate: %s", agg)
	}
}
