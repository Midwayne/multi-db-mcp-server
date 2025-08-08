package services

import (
	"context"
	"encoding/json"
	"fmt"
	"mongomcp/config"

	"github.com/mark3labs/mcp-go/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Count counts the documents in a collection that match the query.
func Count(connManager *config.DBConnections, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	count, err := collection.CountDocuments(context.Background(), filter)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("%d", count)), nil
}
