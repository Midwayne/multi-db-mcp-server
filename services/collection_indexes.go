package services

import (
	"context"
	"encoding/json"
	"fmt"
	"mongomcp/config"

	"github.com/mark3labs/mcp-go/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// CollectionIndexes lists all indexes for a collection.
func CollectionIndexes(connManager *config.DBConnections, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	host, err := request.RequireString("host")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	dbName, err := request.RequireString("db_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := connManager.GetConnection(host)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collection := client.Database(dbName).Collection(collectionName)
	cursor, err := collection.Indexes().List(context.Background())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer cursor.Close(context.Background())

	var indexes []bson.M
	if err := cursor.All(context.Background(), &indexes); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonData, err := json.Marshal(indexes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}
