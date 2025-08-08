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

	if _, ok := enabledToolsSet["aggregate"]; ok {
		s.AddTool(mcp.NewTool("aggregate",
			mcp.WithDescription("Runs aggregation queries"),
		), server.ToolHandlerFunc(func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return services.Aggregate(connManager, request)
		}))
	}
}
