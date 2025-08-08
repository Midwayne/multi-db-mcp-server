package services

import (
	"context"
	"encoding/json"
	"fmt"
	"mongomcp/config"

	"github.com/mark3labs/mcp-go/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// CollectionSchema infers the schema of a collection by sampling one document.
func CollectionSchema(connManager *config.DBConnections, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	var result bson.M
	err = collection.FindOne(context.Background(), bson.D{}).Decode(&result)
	if err != nil {
		return mcp.NewToolResultError("Could not find a document to infer schema: " + err.Error()), nil
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}
