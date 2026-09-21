package server

import (
	"context"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/db/mongodb"
	"dbmcp/db/postgres"
	redisdb "dbmcp/db/redis"
)

type permissionReport struct {
	Connection   string             `json:"connection"`
	Type         db.Type            `json:"type"`
	Access       access.Mode        `json:"access"`
	Operations   []access.Operation `json:"operations"`
	CanRead      bool               `json:"can_read"`
	CanWrite     bool               `json:"can_write"`
	CanAdmin     bool               `json:"can_admin"`
	Database     string             `json:"database,omitempty"`
	Ready        bool               `json:"ready"`
	MaxRows      int                `json:"max_rows"`
	AllowedTools []string           `json:"allowed_tools"`
	DeniedTools  []string           `json:"denied_tools"`
	Notes        []string           `json:"notes,omitempty"`
	Server       any                `json:"server,omitempty"`
	ServerError  string             `json:"server_error,omitempty"`
}

func (a *App) permissionReports(ctx context.Context, connection string, includeServer bool) ([]permissionReport, error) {
	adapters := a.Registry.All()
	if connection != "" {
		adapter, err := a.Registry.Lookup(connection)
		if err != nil {
			return nil, err
		}
		adapters = []db.Adapter{adapter}
	}

	out := make([]permissionReport, 0, len(adapters))
	for _, adapter := range adapters {
		report := a.permissionReport(adapter)
		if includeServer {
			live, err := a.Registry.Get(ctx, adapter.Name())
			if err != nil {
				report.ServerError = err.Error()
			} else {
				report.Ready = live.Connected()
				ident, err := fetchServerIdentity(ctx, live)
				if err != nil {
					report.ServerError = err.Error()
				} else if ident != nil {
					report.Server = ident
				}
			}
		}
		out = append(out, report)
	}
	return out, nil
}

func (a *App) permissionReport(adapter db.Adapter) permissionReport {
	allowed := make([]string, 0)
	denied := make([]string, 0)
	for _, tool := range catalog() {
		if tool.DBType != "" && tool.DBType != adapter.Type() {
			continue
		}
		ok := a.Spec.Tools.GloballyAllowed(tool.Name) &&
			adapter.ToolAllowed(tool.Name) &&
			adapter.Access().Allows(tool.Op)
		if ok {
			allowed = append(allowed, tool.Name)
		} else {
			denied = append(denied, tool.Name)
		}
	}

	var notes []string
	if adapter.Type() == db.TypeRedis && adapter.ToolAllowed(ToolRedisCommand) && a.Spec.Tools.GloballyAllowed(ToolRedisCommand) {
		notes = append(notes, "redis_command is listed as a read tool; write and admin Redis commands are still blocked unless this connection's access mode allows them")
	}
	if adapter.Type() == db.TypePostgres && containsString(allowed, ToolPostgresExecute) && !adapter.Access().Allows(access.OpAdmin) {
		notes = append(notes, "postgres_execute allows DML (INSERT/UPDATE/DELETE); DDL such as CREATE/DROP requires admin access")
	}

	return permissionReport{
		Connection:   adapter.Name(),
		Type:         adapter.Type(),
		Access:       adapter.Access(),
		Operations:   adapter.Access().Operations(),
		CanRead:      adapter.Access().Allows(access.OpRead),
		CanWrite:     adapter.Access().Allows(access.OpWrite),
		CanAdmin:     adapter.Access().Allows(access.OpAdmin),
		Database:     adapter.DefaultDatabase(),
		Ready:        adapter.Connected(),
		MaxRows:      adapter.MaxRows(),
		AllowedTools: allowed,
		DeniedTools:  denied,
		Notes:        notes,
	}
}

func fetchServerIdentity(ctx context.Context, adapter db.Adapter) (any, error) {
	switch a := adapter.(type) {
	case *mongodb.Adapter:
		return a.Identity(ctx)
	case *postgres.Adapter:
		return a.Identity(ctx)
	case *redisdb.Adapter:
		return a.Identity(ctx)
	default:
		return nil, nil
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
