package server

import (
	"context"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/db/postgres"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerPostgresQuery(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolPostgresQuery,
		mcp.WithDescription("Run a read-only SQL statement (SELECT/WITH/EXPLAIN/SHOW) against PostgreSQL"),
		connectionOption(),
		mcp.WithString("sql", mcp.Required(), mcp.Description("SQL statement")),
		mcp.WithString("params", mcp.Description("JSON array of bind parameters, e.g. [1, \"active\"]")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pg, res := app.requirePostgres(ctx, request, ToolPostgresQuery, access.OpRead)
		if res != nil {
			return res, nil
		}
		sqlText, err := request.RequireString("sql")
		if err != nil {
			return errResult(err)
		}
		result, err := pg.Query(ctx, sqlText, request.GetString("params", ""))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})
}

func registerPostgresExecute(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolPostgresExecute,
		mcp.WithDescription("Run a write SQL statement (INSERT/UPDATE/DELETE) against PostgreSQL. DDL requires admin access."),
		connectionOption(),
		mcp.WithString("sql", mcp.Required(), mcp.Description("SQL statement")),
		mcp.WithString("params", mcp.Description("JSON array of bind parameters")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pg, res := app.requirePostgres(ctx, request, ToolPostgresExecute, access.OpWrite)
		if res != nil {
			return res, nil
		}
		sqlText, err := request.RequireString("sql")
		if err != nil {
			return errResult(err)
		}
		result, err := pg.Execute(ctx, sqlText, request.GetString("params", ""))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})
}

func registerPostgresListDatabases(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolPostgresListDatabases,
		mcp.WithDescription("List PostgreSQL databases"),
		connectionOption(),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pg, res := app.requirePostgres(ctx, request, ToolPostgresListDatabases, access.OpRead)
		if res != nil {
			return res, nil
		}
		rows, err := pg.ListDatabases(ctx)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(rows)
	})
}

func registerPostgresListSchemas(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolPostgresListSchemas,
		mcp.WithDescription("List PostgreSQL schemas"),
		connectionOption(),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pg, res := app.requirePostgres(ctx, request, ToolPostgresListSchemas, access.OpRead)
		if res != nil {
			return res, nil
		}
		rows, err := pg.ListSchemas(ctx)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(rows)
	})
}

func registerPostgresListTables(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolPostgresListTables,
		mcp.WithDescription("List PostgreSQL tables, optionally filtered by schema"),
		connectionOption(),
		mcp.WithString("schema", mcp.Description("Schema name. If omitted, all non-system schemas are listed.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pg, res := app.requirePostgres(ctx, request, ToolPostgresListTables, access.OpRead)
		if res != nil {
			return res, nil
		}
		rows, err := pg.ListTables(ctx, request.GetString("schema", ""))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(rows)
	})
}

func registerPostgresDescribeTable(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolPostgresDescribeTable,
		mcp.WithDescription("Describe columns of a PostgreSQL table"),
		connectionOption(),
		mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		mcp.WithString("schema", mcp.Description("Schema name. Defaults to public.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pg, res := app.requirePostgres(ctx, request, ToolPostgresDescribeTable, access.OpRead)
		if res != nil {
			return res, nil
		}
		table, err := request.RequireString("table")
		if err != nil {
			return errResult(err)
		}
		rows, err := pg.DescribeTable(ctx, request.GetString("schema", ""), table)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(rows)
	})
}

func registerPostgresStats(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolPostgresStats,
		mcp.WithDescription("Get PostgreSQL database size, user, and version"),
		connectionOption(),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pg, res := app.requirePostgres(ctx, request, ToolPostgresStats, access.OpRead)
		if res != nil {
			return res, nil
		}
		stats, err := pg.Stats(ctx)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(stats)
	})
}

func (a *App) requirePostgres(ctx context.Context, request mcp.CallToolRequest, tool string, op access.Operation) (*postgres.Adapter, *mcp.CallToolResult) {
	return requireAs[*postgres.Adapter](a, ctx, request, tool, db.TypePostgres, op)
}

func postgresExecuteNote(adapter db.Adapter, allowed bool) string {
	if allowed && !adapter.Access().Allows(access.OpAdmin) {
		return "postgres_execute allows DML (INSERT/UPDATE/DELETE); DDL such as CREATE/DROP requires admin access"
	}
	return ""
}

func init() {
	RegisterEngineTools(
		toolSpec{Name: ToolPostgresQuery, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresQuery},
		toolSpec{Name: ToolPostgresExecute, DBType: db.TypePostgres, Op: access.OpWrite, Register: registerPostgresExecute, Note: postgresExecuteNote},
		toolSpec{Name: ToolPostgresListDatabases, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresListDatabases},
		toolSpec{Name: ToolPostgresListSchemas, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresListSchemas},
		toolSpec{Name: ToolPostgresListTables, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresListTables},
		toolSpec{Name: ToolPostgresDescribeTable, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresDescribeTable},
		toolSpec{Name: ToolPostgresStats, DBType: db.TypePostgres, Op: access.OpRead, Register: registerPostgresStats},
	)
}
