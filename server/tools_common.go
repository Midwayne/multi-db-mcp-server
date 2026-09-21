package server

import (
	"context"

	"dbmcp/db"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerListConnections(s *server.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolListConnections,
		mcp.WithDescription("List every database defined in the spec file: name, engine type, access mode (read_only/read_write/admin), and can_read/can_write/can_admin flags. Use the name as the connection argument on later tools."),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return jsonResult(app.Registry.List())
	})
}

func registerListPermissions(s *server.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolListPermissions,
		mcp.WithDescription("View permissions for one database or every database in the spec: access mode, allowed/denied tools, and optional live server identity. Omit connection to list all."),
		mcp.WithString("connection", mcp.Description("Spec connection name. Omit to view permissions for every database.")),
		mcp.WithBoolean("include_server", mcp.Description("If true, connect and include live engine identity/privileges (Mongo connectionStatus, Postgres role flags, Redis ACL user). Defaults to false.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		reports, err := app.permissionReports(ctx, request.GetString("connection", ""), request.GetBool("include_server", false))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(reports)
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
