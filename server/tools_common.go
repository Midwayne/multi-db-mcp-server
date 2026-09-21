package server

import (
	"context"

	"dbmcp/db"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListConnections(s *server.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolListConnections,
		mcp.WithDescription("List configured database connections, their engine types, and access modes"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return jsonResult(app.Registry.List())
	})
}

func registerPing(s *server.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolPing,
		mcp.WithDescription("Ping a named database connection to verify it is reachable"),
		connectionOption(),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("connection")
		if err != nil {
			return errResult(err)
		}
		adapter, err := app.Registry.Get(ctx, name)
		if err != nil {
			return errResult(err)
		}
		if err := adapter.Ping(ctx); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"ok": true, "connection": adapter.Name(), "type": adapter.Type()})
	})
}

func registerLandscape(s *server.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolLandscape,
		mcp.WithDescription("Overview of namespaces for every connected database (Mongo collections, Postgres tables, Redis keyspace)"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return jsonResult(collectLandscape(ctx, app.Registry))
	})
}

func collectLandscape(ctx context.Context, reg *db.Registry) map[string]any {
	out := make(map[string]any)
	for _, adapter := range reg.All() {
		entry := map[string]any{
			"type":   adapter.Type(),
			"access": adapter.Access(),
		}
		data, err := adapter.Landscape(ctx)
		if err != nil {
			entry["error"] = err.Error()
		} else {
			entry["namespaces"] = data
		}
		out[adapter.Name()] = entry
	}
	return out
}
