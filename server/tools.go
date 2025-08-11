package server

import (
	"context"
	"mongomcp/config"
	"mongomcp/services"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func InitializeTools(s *server.MCPServer, connManager *config.DBConnections, enabledTools []string) {
	enabledToolsSet := make(map[string]struct{})
	for _, toolName := range enabledTools {
		enabledToolsSet[toolName] = struct{}{}
	}

	if _, ok := enabledToolsSet["find"]; ok {
		s.AddTool(mcp.NewTool("find",
			mcp.WithDescription("Finds documents in a collection"),
		), server.ToolHandlerFunc(func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return services.Find(connManager, request)
		}))
	}

	if _, ok := enabledToolsSet["aggregate"]; ok {
		s.AddTool(mcp.NewTool("aggregate",
			mcp.WithDescription("Runs aggregation queries"),
		), server.ToolHandlerFunc(func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return services.Aggregate(connManager, request)
		}))
	}

	if _, ok := enabledToolsSet["count"]; ok {
		s.AddTool(mcp.NewTool("count",
			mcp.WithDescription("Counts documents in a collection"),
		), server.ToolHandlerFunc(func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return services.Count(connManager, request)
		}))
	}

	if _, ok := enabledToolsSet["collection-indexes"]; ok {
		s.AddTool(mcp.NewTool("collection_indexes",
			mcp.WithDescription("Lists all indexes in a collection"),
		), server.ToolHandlerFunc(func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return services.CollectionIndexes(connManager, request)
		}))
	}

	if _, ok := enabledToolsSet["collection-schema"]; ok {
		s.AddTool(mcp.NewTool("collection_schema",
			mcp.WithDescription("Gets the schema of a collection"),
		), server.ToolHandlerFunc(func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return services.CollectionSchema(connManager, request)
		}))
	}

	if _, ok := enabledToolsSet["db-stats"]; ok {
		s.AddTool(mcp.NewTool("db_stats",
			mcp.WithDescription("Gets statistics for the database"),
		), server.ToolHandlerFunc(func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return services.DBStats(connManager, request)
		}))
	}
}
