package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"mongomcp/config"
	mongoServer "mongomcp/server"

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

	s := mongoServer.InitializeMCPServer()

	mongoServer.InitializeTools(s, connManager, cfg.Tools)

	mongoServer.InitializeMongoResourceWithCache(s, connManager)

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
