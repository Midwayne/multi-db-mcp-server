package server

import (
	"context"
	"encoding/json"
	"mongomcp/config"
	"mongomcp/models"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// cacheEntry holds the cached data and its timestamp.
type cacheEntry struct {
	data      []mcp.ResourceContents
	timestamp time.Time
}

// CachedResourceHandler provides a caching layer for a resource handler.
type CachedResourceHandler struct {
	cache       map[string]cacheEntry
	mutex       sync.RWMutex
	ttl         time.Duration
	fetchData   server.ResourceHandlerFunc
	connManager *config.DBConnections
}

// NewCachedResourceHandler creates a new cached handler with a specific TTL.
func NewCachedResourceHandler(ttl time.Duration, connManager *config.DBConnections) *CachedResourceHandler {
	handler := &CachedResourceHandler{
		cache:       make(map[string]cacheEntry),
		ttl:         ttl,
		connManager: connManager,
	}

	handler.fetchData = server.ResourceHandlerFunc(func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		landscape := make(map[string]map[string][]string)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for host, conn := range *handler.connManager {
			wg.Add(1)
			go func(host string, conn *models.DBConnection) {
				defer wg.Done()
				client := conn.Client
				dbNames, err := client.ListDatabaseNames(ctx, nil)
				if err != nil {
					return
				}
				hostData := make(map[string][]string)
				for _, dbName := range dbNames {
					collections, err := client.Database(dbName).ListCollectionNames(ctx, nil)
					if err != nil {
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

		jsonData, err := json.Marshal(landscape)
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		}, nil
	})
	return handler
}

// Handle checks the cache first, and if the data is stale or non-existent, fetches fresh data.
func (h *CachedResourceHandler) Handle(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	h.mutex.RLock()
	if entry, exists := h.cache[req.Params.URI]; exists {
		if time.Since(entry.timestamp) < h.ttl {
			h.mutex.RUnlock()
			return entry.data, nil
		}
	}
	h.mutex.RUnlock()

	// Fetch fresh data using the encapsulated fetch function
	data, err := h.fetchData(ctx, req)
	if err != nil {
		return nil, err
	}

	// Cache the result
	h.mutex.Lock()
	h.cache[req.Params.URI] = cacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
	h.mutex.Unlock()

	return data, nil
}

// InitializeMongoResourceWithCache adds the MongoDB resource definition to the MCP server.
func InitializeMongoResourceWithCache(s *server.MCPServer, connManager *config.DBConnections) {
	cachedHandler := NewCachedResourceHandler(5*time.Minute, connManager) // Currently hardcoded to 5 mins (TBD)

	s.AddResource(mcp.Resource{
		URI:         "config://db",
		Name:        "Mongo Landscape",
		Description: "A landscape view of all connected MongoDB hosts, their databases and collections.",
	}, server.ResourceHandlerFunc(cachedHandler.Handle))
}
