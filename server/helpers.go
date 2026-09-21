package server

import (
	"context"
	"fmt"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/db/mongodb"
	"dbmcp/db/postgres"
	redisdb "dbmcp/db/redis"

	"github.com/mark3labs/mcp-go/mcp"
)

func jsonResult(v any) (*mcp.CallToolResult, error) {
	text, err := db.ToJSON(v)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(text), nil
}

func errResult(err error) (*mcp.CallToolResult, error) {
	if err == nil {
		return mcp.NewToolResultError("unknown error"), nil
	}
	return mcp.NewToolResultError(err.Error()), nil
}

func connectionOption() mcp.ToolOption {
	return mcp.WithString("connection",
		mcp.Required(),
		mcp.Description("Name of the connection as declared in the spec file"),
	)
}

func (a *App) requireAdapter(ctx context.Context, request mcp.CallToolRequest, tool string, expected db.Type, op access.Operation) (db.Adapter, *mcp.CallToolResult) {
	name, err := request.RequireString("connection")
	if err != nil {
		res, _ := errResult(err)
		return nil, res
	}
	adapter, err := a.Registry.Get(ctx, name)
	if err != nil {
		res, _ := errResult(err)
		return nil, res
	}
	if adapter.Type() != expected {
		res, _ := errResult(fmt.Errorf("connection %q is type %s, expected %s", name, adapter.Type(), expected))
		return nil, res
	}
	if !adapter.ToolAllowed(tool) {
		res, _ := errResult(fmt.Errorf("tool %s is not enabled for connection %q", tool, name))
		return nil, res
	}
	if !adapter.Access().Allows(op) {
		res, _ := errResult(fmt.Errorf("%s", adapter.Access().DenyMessage(op)))
		return nil, res
	}
	return adapter, nil
}

func (a *App) requireMongo(ctx context.Context, request mcp.CallToolRequest, tool string, op access.Operation) (*mongodb.Adapter, *mcp.CallToolResult) {
	adapter, res := a.requireAdapter(ctx, request, tool, db.TypeMongoDB, op)
	if res != nil {
		return nil, res
	}
	mongo, ok := adapter.(*mongodb.Adapter)
	if !ok {
		res, _ := errResult(fmt.Errorf("connection %q is not a mongodb adapter", adapter.Name()))
		return nil, res
	}
	return mongo, nil
}

func (a *App) requirePostgres(ctx context.Context, request mcp.CallToolRequest, tool string, op access.Operation) (*postgres.Adapter, *mcp.CallToolResult) {
	adapter, res := a.requireAdapter(ctx, request, tool, db.TypePostgres, op)
	if res != nil {
		return nil, res
	}
	pg, ok := adapter.(*postgres.Adapter)
	if !ok {
		res, _ := errResult(fmt.Errorf("connection %q is not a postgres adapter", adapter.Name()))
		return nil, res
	}
	return pg, nil
}

func (a *App) requireRedis(ctx context.Context, request mcp.CallToolRequest, tool string, op access.Operation) (*redisdb.Adapter, *mcp.CallToolResult) {
	adapter, res := a.requireAdapter(ctx, request, tool, db.TypeRedis, op)
	if res != nil {
		return nil, res
	}
	r, ok := adapter.(*redisdb.Adapter)
	if !ok {
		res, _ := errResult(fmt.Errorf("connection %q is not a redis adapter", adapter.Name()))
		return nil, res
	}
	return r, nil
}
