package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"dbmcp/connect"
	"dbmcp/server"
	"dbmcp/spec"

	"github.com/100mslive/memongo/v2"
	"github.com/100mslive/memongo/v2/memongolog"
	"github.com/alicebob/miniredis/v2"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func skipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	shutdownSharedEngines()
	os.Exit(code)
}

func newLiveApp(t *testing.T, yamlSpec string) *server.App {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte(yamlSpec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	cfg, err := spec.Load(path)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	reg, err := connect.NewRegistry(context.Background(), cfg)
	if err != nil {
		t.Fatalf("connect registry: %v", err)
	}
	t.Cleanup(func() {
		_ = reg.Close(context.Background())
	})
	return &server.App{Spec: cfg, Registry: reg}
}

func newMCPClient(t *testing.T, app *server.App) *mcpclient.Client {
	t.Helper()
	cl, err := mcpclient.NewInProcessClient(server.NewMCPServer(app))
	if err != nil {
		t.Fatal(err)
	}
	if err := cl.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cl.Close() })
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "integration-client", Version: "1.0.0"}
	if _, err := cl.Initialize(context.Background(), initReq); err != nil {
		t.Fatal(err)
	}
	return cl
}

func callTool(t *testing.T, cl *mcpclient.Client, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	if args == nil {
		args = map[string]any{}
	}
	result, err := cl.CallTool(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return result
}

func callToolText(t *testing.T, cl *mcpclient.Client, name string, args map[string]any) string {
	t.Helper()
	result := callTool(t, cl, name, args)
	if result.IsError {
		t.Fatalf("%s returned error: %s", name, toolText(result))
	}
	text := toolText(result)
	if text == "" {
		t.Fatalf("%s returned no text", name)
	}
	return text
}

func callToolError(t *testing.T, cl *mcpclient.Client, name string, args map[string]any) string {
	t.Helper()
	result := callTool(t, cl, name, args)
	if !result.IsError {
		t.Fatalf("%s should have failed, got: %s", name, toolText(result))
	}
	return toolText(result)
}

func toolText(res *mcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		return ""
	}
	return text.Text
}

func decodeJSON(t *testing.T, raw string, dest any) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		t.Fatalf("json decode: %v\n%s", err, raw)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func startRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	return miniredis.RunT(t)
}

func redisURI(r *miniredis.Miniredis) string {
	return "redis://" + r.Addr() + "/0"
}

type permissionReport struct {
	Connection   string   `json:"connection"`
	Type         string   `json:"type"`
	Access       string   `json:"access"`
	CanRead      bool     `json:"can_read"`
	CanWrite     bool     `json:"can_write"`
	CanAdmin     bool     `json:"can_admin"`
	Database     string   `json:"database"`
	Ready        bool     `json:"ready"`
	AllowedTools []string `json:"allowed_tools"`
	DeniedTools  []string `json:"denied_tools"`
	Server       any      `json:"server"`
	ServerError  string   `json:"server_error"`
}

type queryResult struct {
	Columns   []string         `json:"columns"`
	Rows      []map[string]any `json:"rows"`
	Documents []any            `json:"documents"`
	RowCount  int              `json:"row_count"`
	Truncated bool             `json:"truncated"`
}

func rowValue(row map[string]any, col string) string {
	return fmt.Sprint(row[col])
}

func docsHave(docs []any, key, want string) bool {
	for _, doc := range docs {
		m, ok := doc.(map[string]any)
		if !ok {
			continue
		}
		if fmt.Sprint(m[key]) == want {
			return true
		}
	}
	return false
}

var (
	pgOnce    sync.Once
	pgInst    *embeddedpostgres.EmbeddedPostgres
	pgPort    uint32
	pgErr     error
	mongoOnce sync.Once
	mongoSrv  *memongo.Server
	mongoErr  error
)

func shutdownSharedEngines() {
	if pgInst != nil {
		_ = pgInst.Stop()
	}
	if mongoSrv != nil {
		mongoSrv.Stop()
	}
}

type postgresEnv struct {
	Port uint32
}

func pgURI(port uint32, database string) string {
	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/%s?sslmode=disable", port, database)
}

func startSharedPostgres() {
	pgOnce.Do(func() {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			pgErr = err
			return
		}
		port := uint32(l.Addr().(*net.TCPAddr).Port)
		_ = l.Close()
		runtimeDir, err := os.MkdirTemp("", "dbmcp-pg-")
		if err != nil {
			pgErr = err
			return
		}
		cfg := embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(port).
			Database("postgres").
			Username("postgres").
			Password("postgres").
			RuntimePath(runtimeDir).
			Logger(io.Discard).
			StartTimeout(2 * time.Minute)
		db := embeddedpostgres.NewDatabase(cfg)
		if err := db.Start(); err != nil {
			pgErr = err
			return
		}
		pgInst = db
		pgPort = port
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, pgURI(port, "postgres"))
		if err != nil {
			pgErr = err
			return
		}
		defer conn.Close(ctx)
		for _, name := range []string{"analytics", "app"} {
			if _, err := conn.Exec(ctx, "CREATE DATABASE "+name); err != nil {
				pgErr = fmt.Errorf("create database %s: %w", name, err)
				return
			}
		}
	})
}

func requirePostgres(t *testing.T) postgresEnv {
	t.Helper()
	skipIfShort(t)
	startSharedPostgres()
	if pgErr != nil {
		t.Skipf("embedded postgres unavailable: %v", pgErr)
	}
	return postgresEnv{Port: pgPort}
}

func optionalPostgres(t *testing.T) *postgresEnv {
	t.Helper()
	startSharedPostgres()
	if pgErr != nil {
		t.Logf("postgres skipped: %v", pgErr)
		return nil
	}
	env := postgresEnv{Port: pgPort}
	return &env
}

func seedPostgres(t *testing.T, port uint32) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	app, err := pgx.Connect(ctx, pgURI(port, "app"))
	if err != nil {
		t.Fatalf("seed app db: %v", err)
	}
	defer app.Close(ctx)
	if _, err := app.Exec(ctx, `CREATE TABLE IF NOT EXISTS items (id INT PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Exec(ctx, `INSERT INTO items (id, name) VALUES (1, 'widget') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`); err != nil {
		t.Fatal(err)
	}

	analytics, err := pgx.Connect(ctx, pgURI(port, "analytics"))
	if err != nil {
		t.Fatalf("seed analytics db: %v", err)
	}
	defer analytics.Close(ctx)
	if _, err := analytics.Exec(ctx, `CREATE TABLE IF NOT EXISTS metrics (id INT PRIMARY KEY, value INT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := analytics.Exec(ctx, `INSERT INTO metrics (id, value) VALUES (1, 42) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
}

func startSharedMongo() {
	mongoOnce.Do(func() {
		srv, err := memongo.StartWithOptions(&memongo.Options{
			MongoVersion:   "6.0.16",
			LogLevel:       memongolog.LogLevelSilent,
			Logger:         log.New(io.Discard, "", 0),
			StartupTimeout: 60 * time.Second,
		})
		if err != nil {
			mongoErr = err
			return
		}
		mongoSrv = srv
	})
}

func requireMongo(t *testing.T) *memongo.Server {
	t.Helper()
	skipIfShort(t)
	startSharedMongo()
	if mongoErr != nil {
		t.Skipf("memongo unavailable: %v", mongoErr)
	}
	return mongoSrv
}

func optionalMongo(t *testing.T) *memongo.Server {
	t.Helper()
	startSharedMongo()
	if mongoErr != nil {
		t.Logf("mongo skipped: %v", mongoErr)
		return nil
	}
	return mongoSrv
}

func seedMongo(t *testing.T, uri string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetServerSelectionTimeout(10 * time.Second))
	if err != nil {
		t.Fatalf("mongo seed connect: %v", err)
	}
	defer func() { _ = client.Disconnect(ctx) }()
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("mongo seed ping: %v", err)
	}
	inv := client.Database("inventory").Collection("products")
	_ = inv.Drop(ctx)
	if _, err := inv.InsertOne(ctx, map[string]any{"sku": "A-1", "name": "bolt"}); err != nil {
		t.Fatal(err)
	}
	orders := client.Database("orders").Collection("tickets")
	_ = orders.Drop(ctx)
	if _, err := orders.InsertOne(ctx, map[string]any{"sku": "T-9", "status": "open"}); err != nil {
		t.Fatal(err)
	}
}
