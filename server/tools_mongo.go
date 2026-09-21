package server

import (
	"context"

	"dbmcp/access"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func mongoDBName() mcp.ToolOption {
	return mcp.WithString("db_name", mcp.Description("Database name. Defaults to the connection's database if set."))
}

func mongoCollection() mcp.ToolOption {
	return mcp.WithString("collection_name", mcp.Required(), mcp.Description("Collection name"))
}

func registerMongoFind(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoFind,
		mcp.WithDescription("Find documents in a MongoDB collection"),
		connectionOption(),
		mongoDBName(),
		mongoCollection(),
		mcp.WithString("filter", mcp.Description("JSON query filter. Defaults to {}.")),
		mcp.WithString("projection", mcp.Description("JSON projection")),
		mcp.WithString("sort", mcp.Description("JSON sort document")),
		mcp.WithNumber("limit", mcp.Description("Maximum documents to return")),
		mcp.WithNumber("skip", mcp.Description("Number of documents to skip")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoFind, access.OpRead)
		if res != nil {
			return res, nil
		}
		result, err := mongo.Find(ctx,
			request.GetString("db_name", ""),
			request.GetString("collection_name", ""),
			request.GetString("filter", ""),
			request.GetString("projection", ""),
			request.GetString("sort", ""),
			request.GetInt("limit", 0),
			request.GetInt("skip", 0),
		)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})
}

func registerMongoAggregate(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoAggregate,
		mcp.WithDescription("Run an aggregation pipeline on a MongoDB collection"),
		connectionOption(),
		mongoDBName(),
		mongoCollection(),
		mcp.WithString("pipeline", mcp.Required(), mcp.Description("JSON array aggregation pipeline")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoAggregate, access.OpRead)
		if res != nil {
			return res, nil
		}
		pipeline, err := request.RequireString("pipeline")
		if err != nil {
			return errResult(err)
		}
		result, err := mongo.Aggregate(ctx, request.GetString("db_name", ""), request.GetString("collection_name", ""), pipeline)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})
}

func registerMongoCount(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoCount,
		mcp.WithDescription("Count documents in a MongoDB collection"),
		connectionOption(),
		mongoDBName(),
		mongoCollection(),
		mcp.WithString("filter", mcp.Description("JSON query filter. Defaults to {}.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoCount, access.OpRead)
		if res != nil {
			return res, nil
		}
		count, err := mongo.Count(ctx, request.GetString("db_name", ""), request.GetString("collection_name", ""), request.GetString("filter", ""))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"count": count})
	})
}

func registerMongoListDatabases(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoListDatabases,
		mcp.WithDescription("List databases on a MongoDB connection"),
		connectionOption(),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoListDatabases, access.OpRead)
		if res != nil {
			return res, nil
		}
		names, err := mongo.ListDatabases(ctx)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(names)
	})
}

func registerMongoListCollections(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoListCollections,
		mcp.WithDescription("List collections in a MongoDB database"),
		connectionOption(),
		mongoDBName(),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoListCollections, access.OpRead)
		if res != nil {
			return res, nil
		}
		names, err := mongo.ListCollections(ctx, request.GetString("db_name", ""))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(names)
	})
}

func registerMongoIndexes(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoIndexes,
		mcp.WithDescription("List indexes on a MongoDB collection"),
		connectionOption(),
		mongoDBName(),
		mongoCollection(),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoIndexes, access.OpRead)
		if res != nil {
			return res, nil
		}
		indexes, err := mongo.Indexes(ctx, request.GetString("db_name", ""), request.GetString("collection_name", ""))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(indexes)
	})
}

func registerMongoSchema(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoSchema,
		mcp.WithDescription("Infer a MongoDB collection schema by sampling documents"),
		connectionOption(),
		mongoDBName(),
		mongoCollection(),
		mcp.WithNumber("sample", mcp.Description("Number of documents to sample. Defaults to 10.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoSchema, access.OpRead)
		if res != nil {
			return res, nil
		}
		schema, err := mongo.Schema(ctx, request.GetString("db_name", ""), request.GetString("collection_name", ""), request.GetInt("sample", 10))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(schema)
	})
}

func registerMongoStats(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoStats,
		mcp.WithDescription("Get MongoDB database statistics"),
		connectionOption(),
		mongoDBName(),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoStats, access.OpRead)
		if res != nil {
			return res, nil
		}
		stats, err := mongo.Stats(ctx, request.GetString("db_name", ""))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(stats)
	})
}

func registerMongoInsert(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoInsert,
		mcp.WithDescription("Insert one JSON document or an array of documents into a MongoDB collection"),
		connectionOption(),
		mongoDBName(),
		mongoCollection(),
		mcp.WithString("documents", mcp.Required(), mcp.Description("JSON object or array of objects to insert")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoInsert, access.OpWrite)
		if res != nil {
			return res, nil
		}
		docs, err := request.RequireString("documents")
		if err != nil {
			return errResult(err)
		}
		result, err := mongo.Insert(ctx, request.GetString("db_name", ""), request.GetString("collection_name", ""), docs)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})
}

func registerMongoUpdate(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoUpdate,
		mcp.WithDescription("Update documents in a MongoDB collection"),
		connectionOption(),
		mongoDBName(),
		mongoCollection(),
		mcp.WithString("filter", mcp.Required(), mcp.Description("JSON filter selecting documents to update")),
		mcp.WithString("update", mcp.Required(), mcp.Description("JSON update document, e.g. {\"$set\":{\"status\":\"done\"}}")),
		mcp.WithBoolean("many", mcp.Description("If true, update all matches. Defaults to false.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoUpdate, access.OpWrite)
		if res != nil {
			return res, nil
		}
		filter, err := request.RequireString("filter")
		if err != nil {
			return errResult(err)
		}
		update, err := request.RequireString("update")
		if err != nil {
			return errResult(err)
		}
		result, err := mongo.Update(ctx, request.GetString("db_name", ""), request.GetString("collection_name", ""), filter, update, request.GetBool("many", false))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})
}

func registerMongoDelete(s *mcpserver.MCPServer, app *App) {
	s.AddTool(mcp.NewTool(ToolMongoDelete,
		mcp.WithDescription("Delete documents from a MongoDB collection"),
		connectionOption(),
		mongoDBName(),
		mongoCollection(),
		mcp.WithString("filter", mcp.Required(), mcp.Description("JSON filter selecting documents to delete")),
		mcp.WithBoolean("many", mcp.Description("If true, delete all matches. Defaults to false.")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mongo, res := app.requireMongo(ctx, request, ToolMongoDelete, access.OpWrite)
		if res != nil {
			return res, nil
		}
		filter, err := request.RequireString("filter")
		if err != nil {
			return errResult(err)
		}
		result, err := mongo.Delete(ctx, request.GetString("db_name", ""), request.GetString("collection_name", ""), filter, request.GetBool("many", false))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(result)
	})
}
