package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/spec"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Adapter struct {
	db.Meta
	mu    sync.Mutex
	pool  *pgxpool.Pool
	ready bool
}

func New(conn spec.Connection) (*Adapter, error) {
	if conn.URI == "" {
		return nil, fmt.Errorf("postgres uri is required")
	}
	return &Adapter{Meta: db.Meta{Conn: conn}}, nil
}

func (a *Adapter) EnsureConnected(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ready && a.pool != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, a.Conn.ConnectTimeoutDur)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(a.Conn.URI)
	if err != nil {
		return fmt.Errorf("postgres parse uri: %w", err)
	}
	cfg.MaxConnIdleTime = a.Conn.InactivityTimeoutDur
	cfg.ConnConfig.ConnectTimeout = a.Conn.ConnectTimeoutDur
	if a.Access() == access.ModeReadOnly {
		cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, "SET default_transaction_read_only = on")
			return err
		}
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("postgres ping: %w", err)
	}
	a.pool = pool
	a.ready = true
	return nil
}

func (a *Adapter) Ping(ctx context.Context) error {
	if err := a.EnsureConnected(ctx); err != nil {
		return err
	}
	return a.pool.Ping(ctx)
}

func (a *Adapter) Close(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pool != nil {
		a.pool.Close()
		a.pool = nil
	}
	a.ready = false
	return nil
}

func (a *Adapter) Connected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ready
}

func (a *Adapter) Query(ctx context.Context, sqlText, paramsJSON string) (*db.QueryResult, error) {
	op, err := access.ClassifySQL(sqlText)
	if err != nil {
		return nil, err
	}
	if !a.Access().Allows(op) {
		return nil, fmt.Errorf("%s", a.Access().DenyMessage(op))
	}
	if op != access.OpRead {
		return nil, fmt.Errorf("postgres_query only allows read statements; use postgres_execute for writes")
	}
	args, err := parseParams(paramsJSON)
	if err != nil {
		return nil, err
	}
	rows, err := a.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectRows(rows, a.MaxRows())
}

func (a *Adapter) Execute(ctx context.Context, sqlText, paramsJSON string) (*db.ExecResult, error) {
	op, err := access.ClassifySQL(sqlText)
	if err != nil {
		return nil, err
	}
	if op == access.OpRead {
		return nil, fmt.Errorf("postgres_execute is for writes; use postgres_query for SELECT")
	}
	if !a.Access().Allows(op) {
		return nil, fmt.Errorf("%s", a.Access().DenyMessage(op))
	}
	args, err := parseParams(paramsJSON)
	if err != nil {
		return nil, err
	}
	tag, err := a.pool.Exec(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	return &db.ExecResult{RowsAffected: tag.RowsAffected()}, nil
}

func (a *Adapter) ListDatabases(ctx context.Context) ([]map[string]any, error) {
	return a.queryMaps(ctx, `SELECT datname AS name FROM pg_database WHERE datistemplate = false ORDER BY datname`)
}

func (a *Adapter) ListSchemas(ctx context.Context) ([]map[string]any, error) {
	return a.queryMaps(ctx, `
		SELECT schema_name AS name
		FROM information_schema.schemata
		ORDER BY schema_name`)
}

func (a *Adapter) ListTables(ctx context.Context, schema string) ([]map[string]any, error) {
	sqlText := `
		SELECT table_schema, table_name, table_type
		FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')`
	args := []any{}
	if schema != "" {
		sqlText += " AND table_schema = $1"
		args = append(args, schema)
	}
	sqlText += " ORDER BY table_schema, table_name"
	rows, err := a.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result, err := collectRows(rows, a.MaxRows())
	if err != nil {
		return nil, err
	}
	return result.Rows, nil
}

func (a *Adapter) DescribeTable(ctx context.Context, schema, table string) ([]map[string]any, error) {
	if table == "" {
		return nil, fmt.Errorf("table is required")
	}
	if schema == "" {
		schema = "public"
	}
	rows, err := a.pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable, column_default, ordinal_position
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result, err := collectRows(rows, a.MaxRows())
	if err != nil {
		return nil, err
	}
	return result.Rows, nil
}

func (a *Adapter) Stats(ctx context.Context) (map[string]any, error) {
	row := a.pool.QueryRow(ctx, `
		SELECT
			current_database() AS database,
			current_user AS user,
			version() AS version,
			pg_size_pretty(pg_database_size(current_database())) AS size`)
	var database, user, version, size string
	if err := row.Scan(&database, &user, &version, &size); err != nil {
		return nil, err
	}
	return map[string]any{
		"database": database,
		"user":     user,
		"version":  version,
		"size":     size,
	}, nil
}

func (a *Adapter) Landscape(ctx context.Context) (any, error) {
	if err := a.EnsureConnected(ctx); err != nil {
		return nil, err
	}
	tables, err := a.ListTables(ctx, "")
	if err != nil {
		return nil, err
	}
	grouped := map[string][]string{}
	for _, t := range tables {
		schema, _ := t["table_schema"].(string)
		name, _ := t["table_name"].(string)
		grouped[schema] = append(grouped[schema], name)
	}
	return grouped, nil
}

func (a *Adapter) queryMaps(ctx context.Context, sqlText string, args ...any) ([]map[string]any, error) {
	rows, err := a.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result, err := collectRows(rows, a.MaxRows())
	if err != nil {
		return nil, err
	}
	return result.Rows, nil
}

func collectRows(rows pgx.Rows, max int) (*db.QueryResult, error) {
	fields := rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, f := range fields {
		columns[i] = f.Name
	}
	var out []map[string]any
	truncated := false
	for rows.Next() {
		if max > 0 && len(out) >= max {
			truncated = true
			break
		}
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			row[col] = jsonFriendly(vals[i])
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &db.QueryResult{Columns: columns, Rows: out, RowCount: len(out), Truncated: truncated}, nil
}

func jsonFriendly(v any) any {
	switch t := v.(type) {
	case []byte:
		return string(t)
	case time.Time:
		return t.Format(time.RFC3339Nano)
	default:
		return v
	}
}

func parseParams(raw string) ([]any, error) {
	if raw == "" {
		return nil, nil
	}
	var params []any
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return nil, fmt.Errorf("invalid params JSON array: %w", err)
	}
	for i, v := range params {
		params[i] = normalizeParam(v)
	}
	return params, nil
}

func normalizeParam(v any) any {
	switch t := v.(type) {
	case float64:
		if t == float64(int64(t)) {
			return int64(t)
		}
		return t
	default:
		return v
	}
}
