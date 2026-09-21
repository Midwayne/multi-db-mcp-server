package server

import (
	"context"
	"testing"

	"dbmcp/access"
	"dbmcp/db"
	"dbmcp/spec"
)

func boolPtr(v bool) *bool { return &v }

type stubAdapter struct {
	db.Meta
	ready      bool
	connectErr error
}

func (s *stubAdapter) EnsureConnected(context.Context) error {
	if s.connectErr != nil {
		return s.connectErr
	}
	s.ready = true
	return nil
}
func (s *stubAdapter) Ping(context.Context) error { return s.EnsureConnected(context.Background()) }
func (s *stubAdapter) Close(context.Context) error {
	s.ready = false
	return nil
}
func (s *stubAdapter) Connected() bool { return s.ready }
func (s *stubAdapter) Landscape(context.Context) (any, error) {
	return map[string]string{"ok": "true"}, nil
}

func newTestApp(t *testing.T, conns []spec.Connection, tools spec.ToolsConfig) *App {
	t.Helper()
	for i := range conns {
		if conns[i].Enabled == nil {
			conns[i].Enabled = boolPtr(true)
			conns[i].EnabledVal = true
		}
		if conns[i].URI == "" {
			conns[i].URI = "stub://"
		}
	}
	sp := &spec.Spec{
		Server:      spec.ServerConfig{Name: "test-mcp", Version: "0.0.1"},
		Connections: conns,
		Tools:       tools,
	}
	reg, err := db.NewRegistry(context.Background(), sp, func(c spec.Connection) (db.Adapter, error) {
		return &stubAdapter{Meta: db.Meta{Conn: c}, connectErr: connectErrFrom(c)}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return &App{Spec: sp, Registry: reg}
}

func connectErrFrom(c spec.Connection) error {
	if c.Options == nil {
		return nil
	}
	if err, ok := c.Options["connect_error"].(error); ok {
		return err
	}
	return nil
}

func mixedConnections() []spec.Connection {
	return []spec.Connection{
		{Name: "mongo", TypeNormalized: "mongodb", AccessMode: access.ModeReadOnly, Database: "app"},
		{Name: "pg", TypeNormalized: "postgres", AccessMode: access.ModeReadWrite},
		{Name: "cache", TypeNormalized: "redis", AccessMode: access.ModeAdmin},
	}
}
