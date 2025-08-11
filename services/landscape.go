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
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// LandscapeData defines the structure for the landscape view.
type LandscapeData map[string]map[string][]string

// cacheEntry holds the cached data and its timestamp.
type cacheEntry struct {
	data      LandscapeData
	timestamp time.Time
}

// LandscapeCache provides a caching layer for landscape data.
type LandscapeCache struct {
	cache       cacheEntry
	mutex       sync.RWMutex
	ttl         time.Duration
	connManager *config.DBConnections
}

// NewLandscapeCache creates a new cache with a specific TTL.
func NewLandscapeCache(ttl time.Duration, connManager *config.DBConnections) *LandscapeCache {
	return &LandscapeCache{
		ttl:         ttl,
		connManager: connManager,
	}
}

// GetLandscape fetches the landscape data, using a cache to avoid redundant calls.
func (lc *LandscapeCache) GetLandscape(ctx context.Context) (LandscapeData, error) {
	lc.mutex.RLock()
	// Check if the cache is still valid
	if time.Since(lc.cache.timestamp) < lc.ttl {
		lc.mutex.RUnlock()
		return lc.cache.data, nil
	}
	lc.mutex.RUnlock()

	// If cache is stale or empty, fetch fresh data
	landscape, err := lc.fetchFreshLandscape(ctx)
	if err != nil {
		return nil, err
	}

	// Lock the cache for writing and update it
	lc.mutex.Lock()
	lc.cache = cacheEntry{
		data:      landscape,
		timestamp: time.Now(),
	}
	lc.mutex.Unlock()

	return landscape, nil
}

// GetLandscapeAsJSON fetches the landscape data and returns it as a JSON string.
func (lc *LandscapeCache) GetLandscapeAsJSON(ctx context.Context) (string, error) {
	landscape, err := lc.GetLandscape(ctx)
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(landscape)
	if err != nil {
		return "", fmt.Errorf("failed to marshal landscape data: %v", err)
	}

	return string(jsonData), nil
}

// fetchFreshLandscape contains the core logic to get data from MongoDB.
func (lc *LandscapeCache) fetchFreshLandscape(ctx context.Context) (LandscapeData, error) {
	landscape := make(LandscapeData)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for host, conn := range *lc.connManager {
		wg.Add(1)
		go func(host string, conn *models.DBConnection) {
			defer wg.Done()
			client := conn.Client
			dbNames, err := client.ListDatabaseNames(ctx, nil)
			if err != nil {
				// Fallback to parsing the database from the connection string if listing fails
				fmt.Printf("Could not list all databases for host %s: %v. Attempting to use database from connection string.\n", host, err)
				parsedURL, parseErr := url.Parse(conn.ConnString)
				if parseErr == nil {
					dbNameFromURI := strings.TrimPrefix(parsedURL.Path, "/")
					if dbNameFromURI != "" {
						dbNames = []string{dbNameFromURI}
					} else {
						return // Skip if no db in URI
					}
				} else {
					return // Skip if URI is unparseable
				}
			}

			hostData := make(map[string][]string)
			for _, dbName := range dbNames {
				collections, err := client.Database(dbName).ListCollectionNames(ctx, nil)
				if err != nil {
					// Add the database to the list even if collections can't be listed
					hostData[dbName] = []string{}
					continue
				}
				hostData[dbName] = collections
			}
			mu.Lock()
			landscape[host] = hostData
			mu.Unlock()
		}(host, conn)
	}
	wg.Wait()

	return landscape, nil
}

// GetMongoLandscape uses the cached landscape service to get the data.
// The request parameter is unused because the landscape is global, but it's
// kept to match the required ToolHandlerFunc signature.
func GetMongoLandscape(landscapeCache *LandscapeCache, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Using a background context as this is a self-contained tool call.
	ctx := context.Background()

	jsonData, err := landscapeCache.GetLandscapeAsJSON(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get landscape: %v", err)), nil
	}

	return mcp.NewToolResultText(jsonData), nil
}
