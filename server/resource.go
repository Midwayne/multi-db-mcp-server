package server

import (
	"context"
	"mongomcp/services"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// InitializeMongoResource adds the MongoDB resource definition to the MCP server
func InitializeMongoResource(s *server.MCPServer, landscapeCache *services.LandscapeCache) {
	resourceHandler := server.ResourceHandlerFunc(func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		jsonData, err := landscapeCache.GetLandscapeAsJSON(ctx)
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     jsonData,
			},
		}, nil
	})

	s.AddResource(mcp.Resource{
		URI:         "config://db",
		Name:        "Mongo Landscape",
		Description: "A landscape view of all connected MongoDB hosts, their databases and collections.",
	}, resourceHandler)
}
