package connect

import (
	"context"

	"dbmcp/db"
	"dbmcp/spec"

	_ "dbmcp/db/mongodb"
	_ "dbmcp/db/postgres"
	_ "dbmcp/db/redis"
)

// Open builds an adapter for a normalized spec connection using the engine
// registry. Blank-import a new db/<kind> package in this file to enable it.
func Open(conn spec.Connection) (db.Adapter, error) {
	return db.Open(conn)
}

func NewRegistry(ctx context.Context, sp *spec.Spec) (*db.Registry, error) {
	return db.NewRegistry(ctx, sp, db.Open)
}
