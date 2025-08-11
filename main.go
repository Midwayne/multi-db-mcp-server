package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"mongomcp/config"
	mongoServer "mongomcp/server"
	"mongomcp/services"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	connManager, err := config.NewConnectionManager(*cfg)
	if err != nil {
		fmt.Printf("Error initializing connection manager: %v\n", err)
		os.Exit(1)
	}

	landscapeCache := services.NewLandscapeCache(5*time.Minute, connManager)

	s := mongoServer.InitializeMCPServer()

	mongoServer.InitializeTools(s, connManager, cfg.Tools)

	mongoServer.InitializeMongoResource(s, landscapeCache)

	// To be removed in the near future
	s.AddTool(mcp.NewTool("landscape",
		mcp.WithDescription("Gets a view of the MongoDB landscape"),
	), server.ToolHandlerFunc(func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return services.GetMongoLandscape(landscapeCache, request)
	}))

	fmt.Printf("Starting Multi MongoDB MCP Server in %s mode...\n", cfg.ServeMode)
	address := fmt.Sprintf(":%s", cfg.Port)

	switch strings.ToLower(cfg.ServeMode) {
	case "stdio":
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error (stdio): %v", err)
		}
	case "http":
		fmt.Printf("Listening on http://localhost%s\n", address)
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(address); err != nil {
			log.Fatalf("Server error (http): %v", err)
		}
	case "sse":
		fmt.Printf("Listening on http://localhost%s\n", address)
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(address); err != nil {
			log.Fatalf("Server error (sse): %v", err)
		}
	default:
		log.Fatalf("Invalid SERVE_MODE: %s. Must be one of 'stdio', 'http', or 'sse'.", cfg.ServeMode)
	}
}
