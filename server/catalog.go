package server

import (
	"sync"

	"dbmcp/access"
	"dbmcp/db"

	"github.com/mark3labs/mcp-go/server"
)

const (
	ToolListConnections = "list_connections"
	ToolListPermissions = "list_permissions"
	ToolPing            = "ping"
	ToolLandscape       = "landscape"

	ToolMongoFind            = "mongo_find"
	ToolMongoAggregate       = "mongo_aggregate"
	ToolMongoCount           = "mongo_count"
	ToolMongoListDatabases   = "mongo_list_databases"
	ToolMongoListCollections = "mongo_list_collections"
	ToolMongoIndexes         = "mongo_indexes"
	ToolMongoSchema          = "mongo_schema"
	ToolMongoStats           = "mongo_stats"
	ToolMongoInsert          = "mongo_insert"
	ToolMongoUpdate          = "mongo_update"
	ToolMongoDelete          = "mongo_delete"

	ToolPostgresQuery         = "postgres_query"
	ToolPostgresExecute       = "postgres_execute"
	ToolPostgresListDatabases = "postgres_list_databases"
	ToolPostgresListSchemas   = "postgres_list_schemas"
	ToolPostgresListTables    = "postgres_list_tables"
	ToolPostgresDescribeTable = "postgres_describe_table"
	ToolPostgresStats         = "postgres_stats"

	ToolRedisCommand = "redis_command"
	ToolRedisGet     = "redis_get"
	ToolRedisScan    = "redis_scan"
	ToolRedisInfo    = "redis_info"
	ToolRedisSet     = "redis_set"
	ToolRedisDelete  = "redis_delete"
)

type toolSpec struct {
	Name     string
	DBType   db.Type
	Op       access.Operation
	Register func(s *server.MCPServer, app *App)
	Note     func(adapter db.Adapter, allowed bool) string
}

var (
	engineToolsMu sync.Mutex
	engineTools   []toolSpec
)

// RegisterEngineTools adds MCP tools for a database kind. Call from init in
// that engine's tools file. Removing the file unregisters the features.
func RegisterEngineTools(tools ...toolSpec) {
	engineToolsMu.Lock()
	defer engineToolsMu.Unlock()
	engineTools = append(engineTools, tools...)
}

func commonTools() []toolSpec {
	return []toolSpec{
		{Name: ToolListConnections, Op: access.OpRead, Register: registerListConnections},
		{Name: ToolListPermissions, Op: access.OpRead, Register: registerListPermissions},
		{Name: ToolPing, Op: access.OpRead, Register: registerPing},
		{Name: ToolLandscape, Op: access.OpRead, Register: registerLandscape},
	}
}

func catalog() []toolSpec {
	engineToolsMu.Lock()
	defer engineToolsMu.Unlock()
	out := make([]toolSpec, 0, 4+len(engineTools))
	out = append(out, commonTools()...)
	out = append(out, engineTools...)
	return out
}

func ShouldRegister(tool toolSpec, app *App) bool {
	if !app.Spec.Tools.GloballyAllowed(tool.Name) {
		return false
	}
	if tool.DBType == "" {
		return true
	}
	return app.Registry.AnyAllows(tool.DBType, tool.Op, tool.Name)
}

func RegisterTools(s *server.MCPServer, app *App) {
	for _, tool := range catalog() {
		if !ShouldRegister(tool, app) {
			continue
		}
		tool.Register(s, app)
	}
}
