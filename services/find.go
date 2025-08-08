package services

import (
	"context"
	"encoding/json"
	"fmt"
	"mongomcp/config"

	"github.com/mark3labs/mcp-go/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Find executes a find query on a collection.
func Find(connManager *config.DBConnections, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := connManager.GetConnection(host)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collection := client.Database(dbName).Collection(collectionName)

	var filter bson.M
	if err := json.Unmarshal([]byte(query), &filter); err != nil {
		return mcp.NewToolResultError("Invalid query format: " + err.Error()), nil
	}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer cursor.Close(context.Background())

	var results []bson.M
	if err := cursor.All(context.Background(), &results); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonData, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}
