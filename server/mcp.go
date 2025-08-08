package server

import (
	"github.com/mark3labs/mcp-go/server"
)

// InitialiseMCPServer creates and returns a new MCP server instance with the specified configurations.
// It sets the server name, version, and enables recovery.
func InitialiseMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"Multi MongoDB MCP",
		"0.0.1",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)

	return s
}
