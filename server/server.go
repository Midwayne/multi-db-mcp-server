package server

import (
	"fmt"

	"dbmcp/db"
	"dbmcp/spec"

	"github.com/mark3labs/mcp-go/server"
)

type App struct {
	Spec     *spec.Spec
	Registry *db.Registry
}

func NewMCPServer(app *App) *server.MCPServer {
	instructions := fmt.Sprintf(
		"%s is a multi-database MCP server. Use list_connections to see configured databases, then call engine-specific tools with the connection name. Access modes (read_only, read_write, admin) are enforced per connection.",
		app.Spec.Server.Name,
	)
	s := server.NewMCPServer(
		app.Spec.Server.Name,
		app.Spec.Server.Version,
		server.WithInstructions(instructions),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)
	RegisterTools(s, app)
	RegisterResources(s, app)
	return s
}
