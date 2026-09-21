package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"dbmcp/connect"
	"dbmcp/server"
	"dbmcp/spec"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	specPath := flag.String("spec", "", "Path to the YAML or JSON spec file (defaults to DBMCP_SPEC or spec.yaml)")
	flag.Parse()

	cfg, err := spec.Load(*specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading spec: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	registry, err := connect.NewRegistry(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing connections: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := registry.Close(context.Background()); err != nil {
			log.Printf("error closing connections: %v", err)
		}
	}()

	mcp := server.NewMCPServer(&server.App{Spec: cfg, Registry: registry})
	fmt.Fprintf(os.Stderr, "Starting %s (%s) in %s mode with %d connection(s)...\n",
		cfg.Server.Name, cfg.Server.Version, cfg.Server.ServeMode, len(registry.List()))

	if err := serve(mcp, cfg); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func serve(s *mcpserver.MCPServer, cfg *spec.Spec) error {
	address := fmt.Sprintf(":%d", cfg.Server.Port)
	switch strings.ToLower(cfg.Server.ServeMode) {
	case "stdio":
		return mcpserver.ServeStdio(s)
	case "http":
		fmt.Fprintf(os.Stderr, "Listening on http://localhost%s\n", address)
		return mcpserver.NewStreamableHTTPServer(s).Start(address)
	case "sse":
		fmt.Fprintf(os.Stderr, "Listening on http://localhost%s\n", address)
		return mcpserver.NewSSEServer(s).Start(address)
	default:
		return fmt.Errorf("invalid serve_mode %q", cfg.Server.ServeMode)
	}
}
