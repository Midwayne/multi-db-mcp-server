package server

import (
	"dbmcp/access"
	"dbmcp/db"

	"github.com/mark3labs/mcp-go/server"
)

const (
	ToolListConnections = "list_connections"
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
}

func catalog() []toolSpec {
	return []toolSpec{
		{Name: ToolListConnections, Op: access.OpRead, Register: registerListConnections},
		{Name: ToolPing, Op: access.OpRead, Register: registerPing},
		{Name: ToolLandscape, Op: access.OpRead, Register: registerLandscape},

		{Name: ToolMongoFind, DBType: db.TypeMongoDB, Op: access.OpRead, Register: registerMongoFind},
		{Name: ToolMongoAggregate, DBType: db.TypeMongoDB, Op: access.OpRead, Register: registerMongoAggregate},
		{Name: ToolMongoCount, DBType: db.TypeMongoDB, Op: access.OpRead, Register: registerMongoCount},
		{Name: ToolMongoListDatabases, DBType: db.TypeMongoDB, Op: access.OpRead, Register: registerMongoListDatabases},
		{Name: ToolMongoListCollections, DBType: db.TypeMongoDB, Op: access.OpRead, Register: registerMongoListCollections},
		{Name: ToolMongoIndexes, DBType: db.TypeMongoDB, Op: access.OpRead, Register: registerMongoIndexes},
		{Name: ToolMongoSchema, DBType: db.TypeMongoDB, Op: access.OpRead, Register: registerMongoSchema},
		{Name: ToolMongoStats, DBType: db.TypeMongoDB, Op: access.OpRead, Register: registerMongoStats},
		{Name: ToolMongoInsert, DBType: db.TypeMongoDB, Op: access.OpWrite, Register: registerMongoInsert},
		{Name: ToolMongoUpdate, DBType: db.TypeMongoDB, Op: access.OpWrite, Register: registerMongoUpdate},
		{Name: ToolMongoDelete, DBType: db.TypeMongoDB, Op: access.OpWrite, Register: registerMongoDelete},

		{Name: ToolPostgresQuery, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresQuery},
		{Name: ToolPostgresExecute, DBType: db.TypePostgres, Op: access.OpWrite, Register: registerPostgresExecute},
		{Name: ToolPostgresListDatabases, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresListDatabases},
		{Name: ToolPostgresListSchemas, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresListSchemas},
		{Name: ToolPostgresListTables, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresListTables},
		{Name: ToolPostgresDescribeTable, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresDescribeTable},
		{Name: ToolPostgresStats, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresStats},

		{Name: ToolRedisCommand, DBType: db.TypeRedis, Op: access.OpRead, Register: registerRedisCommand},
		{Name: ToolRedisGet, DBType: db.TypeRedis, Op: access.OpRead, Register: registerRedisGet},
		{Name: ToolRedisScan, DBType: db.TypeRedis, Op: access.OpRead, Register: registerRedisScan},
		{Name: ToolRedisInfo, DBType: db.TypeRedis, Op: access.OpRead, Register: registerRedisInfo},
		{Name: ToolRedisSet, DBType: db.TypeRedis, Op: access.OpWrite, Register: registerRedisSet},
		{Name: ToolRedisDelete, DBType: db.TypeRedis, Op: access.OpWrite, Register: registerRedisDelete},
	}
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
