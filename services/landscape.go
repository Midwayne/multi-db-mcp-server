package services

import (
	"context"
	"encoding/json"
	"fmt"
	"mongomcp/config"
	"mongomcp/models"
	"net/url"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

// GetMongoLandscape fetches a view of all connected hosts, their databases and collections.
// This method was created to provide a comprehensive overview of the MongoDB landscape since many agentic tools do not support MCP Resources functionality yet.
func GetMongoLandscape(connManager *config.DBConnections, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	landscape := make(map[string]map[string][]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for host, conn := range *connManager {
		wg.Add(1)
		go func(host string, conn *models.DBConnection) {
			defer wg.Done()
			fmt.Println("Fetching landscape for host:", host)
			client := conn.Client
			// Using a background context as this is a self-contained tool call.
			ctx := context.Background()

			dbNames, err := client.ListDatabaseNames(ctx, nil)
			if err != nil {
				// This error often indicates a permissions issue. As a fallback,
				// try to extract the database name from the connection URI itself.
				fmt.Printf("Could not list all databases for host %s: %v. Attempting to use database from connection string.\n", host, err)
				parsedURL, parseErr := url.Parse(conn.ConnString)
				if parseErr == nil {
					dbNameFromURI := strings.TrimPrefix(parsedURL.Path, "/")
					if dbNameFromURI != "" {
						fmt.Printf("Found database in URI for host %s: %s\n", host, dbNameFromURI)
						// Use the database from the URI as the only one to check.
						dbNames = []string{dbNameFromURI}
					} else {
						fmt.Printf("No database found in URI for host %s. Skipping.\n", host)
						return
					}
				} else {
					fmt.Printf("Could not parse connection string for host %s. Skipping.\n", host)
					return
				}
			}

			fmt.Println("Found databases for host:", host, "Databases:", dbNames)

			hostData := make(map[string][]string)
			for _, dbName := range dbNames {
				collections, err := client.Database(dbName).ListCollectionNames(ctx, nil)
				if err != nil {
					fmt.Printf("Error listing collections for database %s on host %s: %v\n", dbName, host, err)
					hostData[dbName] = []string{}

					continue
				}
				hostData[dbName] = collections
			}

			fmt.Println("Found collections for host:", host, "Data:", hostData)

			mu.Lock()
			landscape[host] = hostData
			mu.Unlock()
		}(host, conn)
	}

	wg.Wait()

	jsonData, err := json.Marshal(landscape)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}
