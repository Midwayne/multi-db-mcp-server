package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"dbmcp/db"
	"dbmcp/spec"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Adapter struct {
	db.Meta
	mu     sync.Mutex
	client *mongo.Client
	ready  bool
}

func New(conn spec.Connection) (*Adapter, error) {
	if conn.URI == "" {
		return nil, fmt.Errorf("mongodb uri is required")
	}
	return &Adapter{Meta: db.Meta{Conn: conn}}, nil
}

func (a *Adapter) EnsureConnected(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ready && a.client != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, a.Conn.ConnectTimeoutDur)
	defer cancel()
	opts := options.Client().
		ApplyURI(a.Conn.URI).
		SetConnectTimeout(a.Conn.ConnectTimeoutDur).
		SetServerSelectionTimeout(a.Conn.ConnectTimeoutDur).
		SetMaxConnIdleTime(a.Conn.InactivityTimeoutDur)
	client, err := mongo.Connect(opts)
	if err != nil {
		return fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return fmt.Errorf("mongo ping: %w", err)
	}
	a.client = client
	a.ready = true
	return nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	if err := a.EnsureConnected(ctx); err != nil {
		return err
	}
	return a.client.Ping(ctx, nil)
}

func (a *Adapter) Close(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		a.ready = false
		return nil
	}
	err := a.client.Disconnect(ctx)
	a.client = nil
	a.ready = false
	return err
}

func (a *Adapter) Connected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ready
}

func (a *Adapter) Identity(ctx context.Context) (any, error) {
	if err := a.EnsureConnected(ctx); err != nil {
		return nil, err
	}
	var status bson.M
	err := a.client.Database("admin").RunCommand(ctx, bson.D{{Key: "connectionStatus", Value: 1}}).Decode(&status)
	if err != nil {
		return nil, err
	}
	return status, nil
}

func (a *Adapter) Client() *mongo.Client {
	return a.client
}

func (a *Adapter) ResolveDatabase(name string) (string, error) {
	if name != "" {
		return name, nil
	}
	if a.Conn.Database != "" {
		return a.Conn.Database, nil
	}
	return "", fmt.Errorf("database is required (set database on the connection or pass db_name)")
}

func (a *Adapter) collection(dbName, collName string) (*mongo.Collection, error) {
	resolved, err := a.ResolveDatabase(dbName)
	if err != nil {
		return nil, err
	}
	if collName == "" {
		return nil, fmt.Errorf("collection_name is required")
	}
	if a.client == nil {
		return nil, fmt.Errorf("mongodb client is not connected")
	}
	return a.client.Database(resolved).Collection(collName), nil
}

func (a *Adapter) Find(ctx context.Context, dbName, collName, filterJSON, projectionJSON, sortJSON string, limit, skip int) (*db.QueryResult, error) {
	coll, err := a.collection(dbName, collName)
	if err != nil {
		return nil, err
	}
	filter, err := parseMap(filterJSON, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}
	opts := options.Find()
	limit = clampLimit(limit, a.MaxRows())
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if skip > 0 {
		opts.SetSkip(int64(skip))
	}
	if projectionJSON != "" {
		proj, err := parseMap(projectionJSON, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid projection: %w", err)
		}
		opts.SetProjection(proj)
	}
	if sortJSON != "" {
		sort, err := parseMap(sortJSON, nil)
		if err != nil {
			return nil, fmt.Errorf("invalid sort: %w", err)
		}
		opts.SetSort(sort)
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	truncated := a.MaxRows() > 0 && len(docs) >= a.MaxRows() && limit == a.MaxRows()
	return &db.QueryResult{
		Documents: toAny(docs),
		RowCount:  len(docs),
		Truncated: truncated,
	}, nil
}

func (a *Adapter) Aggregate(ctx context.Context, dbName, collName, pipelineJSON string) (*db.QueryResult, error) {
	coll, err := a.collection(dbName, collName)
	if err != nil {
		return nil, err
	}
	var pipeline []bson.M
	if err := json.Unmarshal([]byte(pipelineJSON), &pipeline); err != nil {
		return nil, fmt.Errorf("invalid pipeline: %w", err)
	}
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	max := a.MaxRows()
	truncated := false
	if max > 0 && len(docs) > max {
		docs = docs[:max]
		truncated = true
	}
	return &db.QueryResult{Documents: toAny(docs), RowCount: len(docs), Truncated: truncated}, nil
}

func (a *Adapter) Count(ctx context.Context, dbName, collName, filterJSON string) (int64, error) {
	coll, err := a.collection(dbName, collName)
	if err != nil {
		return 0, err
	}
	filter, err := parseMap(filterJSON, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("invalid filter: %w", err)
	}
	return coll.CountDocuments(ctx, filter)
}

func (a *Adapter) Indexes(ctx context.Context, dbName, collName string) ([]bson.M, error) {
	coll, err := a.collection(dbName, collName)
	if err != nil {
		return nil, err
	}
	cursor, err := coll.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var indexes []bson.M
	if err := cursor.All(ctx, &indexes); err != nil {
		return nil, err
	}
	return indexes, nil
}

func (a *Adapter) Schema(ctx context.Context, dbName, collName string, sample int) (map[string]string, error) {
	coll, err := a.collection(dbName, collName)
	if err != nil {
		return nil, err
	}
	if sample <= 0 {
		sample = 10
	}
	opts := options.Find().SetLimit(int64(sample))
	cursor, err := coll.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("could not find a document to infer schema")
	}
	schema := make(map[string]string)
	for _, doc := range docs {
		for k, v := range doc {
			schema[k] = fmt.Sprintf("%T", v)
		}
	}
	return schema, nil
}

func (a *Adapter) Stats(ctx context.Context, dbName string) (bson.M, error) {
	resolved, err := a.ResolveDatabase(dbName)
	if err != nil {
		return nil, err
	}
	var stats bson.M
	err = a.client.Database(resolved).RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&stats)
	return stats, err
}

// DatabaseInfo is a MongoDB database and the collections it contains.
type DatabaseInfo struct {
	Name        string   `json:"name"`
	Collections []string `json:"collections"`
}

func (a *Adapter) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	return a.listDatabasesWithCollections(ctx)
}

func (a *Adapter) ListCollections(ctx context.Context, dbName string) ([]string, error) {
	resolved, err := a.ResolveDatabase(dbName)
	if err != nil {
		return nil, err
	}
	return a.collectionNames(ctx, resolved)
}

func (a *Adapter) listDatabasesWithCollections(ctx context.Context) ([]DatabaseInfo, error) {
	if a.client == nil {
		return nil, fmt.Errorf("mongodb client is not connected")
	}
	dbNames, err := a.client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	sort.Strings(dbNames)
	out := make([]DatabaseInfo, 0, len(dbNames))
	for _, name := range dbNames {
		cols, err := a.collectionNames(ctx, name)
		if err != nil {
			cols = []string{}
		}
		out = append(out, DatabaseInfo{Name: name, Collections: cols})
	}
	return out, nil
}

func (a *Adapter) collectionNames(ctx context.Context, dbName string) ([]string, error) {
	if a.client == nil {
		return nil, fmt.Errorf("mongodb client is not connected")
	}
	cols, err := a.client.Database(dbName).ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	if cols == nil {
		cols = []string{}
	}
	sort.Strings(cols)
	return cols, nil
}

func (a *Adapter) Insert(ctx context.Context, dbName, collName, documentsJSON string) (*db.ExecResult, error) {
	coll, err := a.collection(dbName, collName)
	if err != nil {
		return nil, err
	}
	docs, err := parseDocuments(documentsJSON)
	if err != nil {
		return nil, err
	}
	result, err := coll.InsertMany(ctx, docs)
	if err != nil {
		return nil, err
	}
	return &db.ExecResult{RowsAffected: int64(len(result.InsertedIDs)), InsertedIDs: result.InsertedIDs}, nil
}

func (a *Adapter) Update(ctx context.Context, dbName, collName, filterJSON, updateJSON string, many bool) (*db.ExecResult, error) {
	coll, err := a.collection(dbName, collName)
	if err != nil {
		return nil, err
	}
	filter, err := parseMap(filterJSON, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}
	update, err := parseMap(updateJSON, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid update: %w", err)
	}
	if filter == nil {
		return nil, fmt.Errorf("filter is required")
	}
	if update == nil {
		return nil, fmt.Errorf("update is required")
	}
	if many {
		result, err := coll.UpdateMany(ctx, filter, update)
		if err != nil {
			return nil, err
		}
		return &db.ExecResult{Matched: result.MatchedCount, Modified: result.ModifiedCount, RowsAffected: result.ModifiedCount}, nil
	}
	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	return &db.ExecResult{Matched: result.MatchedCount, Modified: result.ModifiedCount, RowsAffected: result.ModifiedCount}, nil
}

func (a *Adapter) Delete(ctx context.Context, dbName, collName, filterJSON string, many bool) (*db.ExecResult, error) {
	coll, err := a.collection(dbName, collName)
	if err != nil {
		return nil, err
	}
	filter, err := parseMap(filterJSON, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}
	if filter == nil {
		return nil, fmt.Errorf("filter is required")
	}
	if many {
		result, err := coll.DeleteMany(ctx, filter)
		if err != nil {
			return nil, err
		}
		return &db.ExecResult{Deleted: result.DeletedCount, RowsAffected: result.DeletedCount}, nil
	}
	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &db.ExecResult{Deleted: result.DeletedCount, RowsAffected: result.DeletedCount}, nil
}

func (a *Adapter) Landscape(ctx context.Context) (any, error) {
	if err := a.EnsureConnected(ctx); err != nil {
		return nil, err
	}
	dbs, err := a.listDatabasesWithCollections(ctx)
	if err != nil {
		if a.Conn.Database == "" {
			return nil, err
		}
		cols, colErr := a.ListCollections(ctx, a.Conn.Database)
		if colErr != nil {
			cols = []string{}
		}
		dbs = []DatabaseInfo{{Name: a.Conn.Database, Collections: cols}}
	}
	hostData := make(map[string][]string, len(dbs))
	for _, info := range dbs {
		hostData[info.Name] = info.Collections
	}
	return hostData, nil
}

func parseMap(raw string, empty bson.M) (bson.M, error) {
	if raw == "" {
		return empty, nil
	}
	var m bson.M
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func parseDocuments(raw string) ([]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("documents are required")
	}
	if raw[0] == '[' {
		var docs []any
		if err := json.Unmarshal([]byte(raw), &docs); err != nil {
			return nil, fmt.Errorf("invalid documents array: %w", err)
		}
		if len(docs) == 0 {
			return nil, fmt.Errorf("documents array is empty")
		}
		return docs, nil
	}
	var doc any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, fmt.Errorf("invalid document: %w", err)
	}
	return []any{doc}, nil
}

func toAny(docs []bson.M) []any {
	out := make([]any, len(docs))
	for i, d := range docs {
		out[i] = d
	}
	return out
}

func clampLimit(limit, max int) int {
	if max <= 0 {
		return limit
	}
	if limit <= 0 || limit > max {
		return max
	}
	return limit
}
