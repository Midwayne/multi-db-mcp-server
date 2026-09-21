package connect

import (
	"context"
	"fmt"

	"dbmcp/db"
	"dbmcp/db/mongodb"
	"dbmcp/db/postgres"
	redisdb "dbmcp/db/redis"
	"dbmcp/spec"
)

func Open(conn spec.Connection) (db.Adapter, error) {
	switch db.Type(conn.TypeNormalized) {
	case db.TypeMongoDB:
		return mongodb.New(conn)
	case db.TypePostgres:
		return postgres.New(conn)
	case db.TypeRedis:
		return redisdb.New(conn)
	default:
		return nil, fmt.Errorf("unsupported database type %q", conn.TypeNormalized)
	}
}

func NewRegistry(ctx context.Context, sp *spec.Spec) (*db.Registry, error) {
	return db.NewRegistry(ctx, sp, Open)
}
