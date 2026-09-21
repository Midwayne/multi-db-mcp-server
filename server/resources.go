package server

import (
	"context"

	"dbmcp/db"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func RegisterResources(s *mcpserver.MCPServer, app *App) {
	s.AddResource(mcp.Resource{
		URI:         "config://db",
		Name:        "Database landscape",
		Description: "Configured connections and a namespace overview for each database.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		text, err := db.ToJSON(map[string]any{
			"connections": app.Registry.List(),
			"landscape":   collectLandscape(ctx, app.Registry),
		})
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     text,
			},
		}, nil
	})
}
