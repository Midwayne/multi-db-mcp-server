package server

import (
	"context"
	"fmt"

	"dbmcp/access"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func registerRedisCommand(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolRedisCommand,
		mcp.WithDescription("Run a Redis command. Read-only connections may only run read commands; FLUSHALL and similar require admin."),
		connectionOption(),
		mcp.WithString("command", mcp.Required(), mcp.Description("Redis command name, e.g. GET, HGETALL, SET")),
		mcp.WithArray("args", mcp.Description("Command arguments"), mcp.WithStringItems()),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rdb, res := app.requireRedis(ctx, request, ToolRedisCommand, access.OpRead)
		if res != nil {
			return res, nil
		}
		command, err := request.RequireString("command")
		if err != nil {
			return errResult(err)
		}
		op := access.ClassifyRedis(command)
		if !rdb.Access().Allows(op) {
			return errResult(fmt.Errorf("%s", rdb.Access().DenyMessage(op)))
		}
		result, err := rdb.Command(ctx, command, request.GetStringSlice("args", nil))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"result": result})
	})
}

func registerRedisGet(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolRedisGet,
		mcp.WithDescription("GET a Redis string key"),
		connectionOption(),
		mcp.WithString("key", mcp.Required(), mcp.Description("Key name")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rdb, res := app.requireRedis(ctx, request, ToolRedisGet, access.OpRead)
		if res != nil {
			return res, nil
		}
		key, err := request.RequireString("key")
		if err != nil {
			return errResult(err)
		}
		val, err := rdb.Get(ctx, key)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"key": key, "value": val})
	})
}

func registerRedisScan(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolRedisScan,
		mcp.WithDescription("SCAN Redis keys matching a pattern"),
		connectionOption(),
		mcp.WithString("pattern", mcp.Description("MATCH pattern. Defaults to *.")),
		mcp.WithNumber("count", mcp.Description("HINT count per SCAN iteration")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rdb, res := app.requireRedis(ctx, request, ToolRedisScan, access.OpRead)
		if res != nil {
			return res, nil
		}
		result, err := rdb.Scan(ctx, request.GetString("pattern", "*"), int64(request.GetInt("count", 100)))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})
}

func registerRedisInfo(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolRedisInfo,
		mcp.WithDescription("Return Redis INFO output, optionally for one section"),
		connectionOption(),
		mcp.WithString("section", mcp.Description("Optional INFO section, e.g. server, memory, keyspace")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rdb, res := app.requireRedis(ctx, request, ToolRedisInfo, access.OpRead)
		if res != nil {
			return res, nil
		}
		info, err := rdb.Info(ctx, request.GetString("section", ""))
		if err != nil {
			return errResult(err)
		}
		return mcp.NewToolResultText(info), nil
	})
}

func registerRedisSet(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolRedisSet,
		mcp.WithDescription("SET a Redis string key"),
		connectionOption(),
		mcp.WithString("key", mcp.Required(), mcp.Description("Key name")),
		mcp.WithString("value", mcp.Required(), mcp.Description("Value to store")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rdb, res := app.requireRedis(ctx, request, ToolRedisSet, access.OpWrite)
		if res != nil {
			return res, nil
		}
		key, err := request.RequireString("key")
		if err != nil {
			return errResult(err)
		}
		value, err := request.RequireString("value")
		if err != nil {
			return errResult(err)
		}
		if err := rdb.Set(ctx, key, value); err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"ok": true, "key": key})
	})
}

func registerRedisDelete(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolRedisDelete,
		mcp.WithDescription("DEL one or more Redis keys"),
		connectionOption(),
		mcp.WithArray("keys", mcp.Required(), mcp.Description("Keys to delete"), mcp.WithStringItems()),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rdb, res := app.requireRedis(ctx, request, ToolRedisDelete, access.OpWrite)
		if res != nil {
			return res, nil
		}
		keys := request.GetStringSlice("keys", nil)
		n, err := rdb.Delete(ctx, keys)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"deleted": n})
	})
}
