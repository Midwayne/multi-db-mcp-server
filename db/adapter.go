package db

import (
	"context"
	"encoding/json"

	"dbmcp/access"
)

const (
	TypeMongoDB  Type = "mongodb"
	TypePostgres Type = "postgres"
	TypeRedis    Type = "redis"
)

// Type is a canonical database engine name.
type Type string

// Adapter is the common contract every database engine implements.
type Adapter interface {
	Name() string
	Type() Type
	Access() access.Mode
	MaxRows() int
	DefaultDatabase() string
	ToolAllowed(tool string) bool

	EnsureConnected(ctx context.Context) error
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
	Connected() bool
	Landscape(ctx context.Context) (any, error)
}

// IdentityProvider is an optional adapter capability used by list_permissions
// when include_server is true. Engines that do not implement it are skipped.
type IdentityProvider interface {
	Identity(ctx context.Context) (any, error)
}

// Info is a JSON-serializable view of a configured connection.
type Info struct {
	Name     string      `json:"name"`
	Type     Type        `json:"type"`
	Access   access.Mode `json:"access"`
	CanRead  bool        `json:"can_read"`
	CanWrite bool        `json:"can_write"`
	CanAdmin bool        `json:"can_admin"`
	Database string      `json:"database,omitempty"`
	Ready    bool        `json:"ready"`
}

func ToJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type QueryResult struct {
	Columns   []string         `json:"columns,omitempty"`
	Rows      []map[string]any `json:"rows,omitempty"`
	Documents []any            `json:"documents,omitempty"`
	RowCount  int              `json:"row_count"`
	Truncated bool             `json:"truncated,omitempty"`
}

type ExecResult struct {
	RowsAffected int64 `json:"rows_affected"`
	InsertedIDs  []any `json:"inserted_ids,omitempty"`
	Matched      int64 `json:"matched,omitempty"`
	Modified     int64 `json:"modified,omitempty"`
	Deleted      int64 `json:"deleted,omitempty"`
}
