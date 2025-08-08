package services

import (
	"context"
	"encoding/json"
	"fmt"
	"mongomcp/config"

	"github.com/mark3labs/mcp-go/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// DBStats gets statistics for a database.
func DBStats(connManager *config.DBConnections, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	host, err := request.RequireString("host")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	dbName, err := request.RequireString("db_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := connManager.GetConnection(host)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	db := client.Database(dbName)
	var stats bson.M
	err = db.RunCommand(context.Background(), bson.D{{"dbStats", 1}}).Decode(&stats)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonData, err := json.Marshal(stats)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}
