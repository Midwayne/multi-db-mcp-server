package server

import (
	"context"
	"encoding/json"
	"testing"

	"dbmcp/spec"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestMCPListAndViewPermissions(t *testing.T) {
	t.Parallel()
	app := newTestApp(t, mixedConnections(), spec.ToolsConfig{})
	cl := newMCPClient(t, app)
	defer cl.Close()

	tools, err := cl.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{ToolListConnections, ToolListPermissions, ToolPing, ToolMongoFind, ToolPostgresQuery, ToolRedisGet, ToolRedisSet, ToolPostgresExecute} {
		if !names[want] {
			t.Fatalf("expected tool %s to be registered, got %v", want, names)
		}
	}
	if names[ToolMongoInsert] {
		t.Fatal("mongo write tools should not register when the only mongo connection is read_only")
	}

	allText := callToolText(t, cl, ToolListConnections, nil)
	var listed []map[string]any
	if err := json.Unmarshal([]byte(allText), &listed); err != nil {
		t.Fatalf("list_connections JSON: %v\n%s", err, allText)
	}
	if len(listed) != 3 {
		t.Fatalf("list_connections should return every spec DB, got %d: %s", len(listed), allText)
	}

	permAll := callToolText(t, cl, ToolListPermissions, nil)
	var allReports []permissionReport
	if err := json.Unmarshal([]byte(permAll), &allReports); err != nil {
		t.Fatalf("list_permissions all JSON: %v\n%s", err, permAll)
	}
	if len(allReports) != 3 {
		t.Fatalf("list_permissions without connection should cover all DBs, got %d", len(allReports))
	}

	permOne := callToolText(t, cl, ToolListPermissions, map[string]any{"connection": "cache"})
	var one []permissionReport
	if err := json.Unmarshal([]byte(permOne), &one); err != nil {
		t.Fatalf("list_permissions one JSON: %v\n%s", err, permOne)
	}
	if len(one) != 1 || one[0].Connection != "cache" || !one[0].CanAdmin {
		t.Fatalf("view permissions for cache: %+v", one)
	}

	missing := callTool(t, cl, ToolListPermissions, map[string]any{"connection": "does-not-exist"})
	if !missing.IsError {
		t.Fatal("list_permissions for an unknown DB should be a tool error")
	}

	for _, name := range []string{"mongo", "pg", "cache"} {
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

func newMCPClient(t *testing.T, app *App) *mcpclient.Client {
	t.Helper()
	cl, err := mcpclient.NewInProcessClient(NewMCPServer(app))
	if err != nil {
		t.Fatal(err)
	}
	if err := cl.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "test-client", Version: "1.0.0"}
	if _, err := cl.Initialize(context.Background(), initReq); err != nil {
		t.Fatal(err)
	}
	return cl
}

func callTool(t *testing.T, cl *mcpclient.Client, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	if args == nil {
		args = map[string]any{}
	}
	result, err := cl.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return result
}

func callToolText(t *testing.T, cl *mcpclient.Client, name string, args map[string]any) string {
	t.Helper()
	result := callTool(t, cl, name, args)
	if result.IsError {
		t.Fatalf("%s returned error: %s", name, toolErrText(result))
	}
	if len(result.Content) == 0 {
		t.Fatalf("%s returned no content", name)
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("%s content is %T", name, result.Content[0])
	}
	return text.Text
}
