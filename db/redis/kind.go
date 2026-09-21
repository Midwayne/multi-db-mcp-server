package redisdb

import (
	"net/url"
	"strconv"

	"dbmcp/db"
	"dbmcp/spec"
)

func init() {
	spec.RegisterKind(spec.Kind{
		Canonical: string(db.TypeRedis),
		Aliases:   []string{"redis"},
		BuildURI:  buildURI,
	})
	db.Register(db.Engine{
		Type: db.TypeRedis,
		Open: func(conn spec.Connection) (db.Adapter, error) {
			return New(conn)
		},
	})
}

func buildURI(c *spec.Connection) (string, error) {
	u := &url.URL{
		Scheme: "redis",
		Host:   spec.JoinHostPort(c.Host, c.Port, 6379),
		Path:   "/" + strconv.Itoa(c.RedisDB),
	}
	if c.Password != "" {
		if c.User != "" {
			u.User = url.UserPassword(c.User, c.Password)
		} else {
			u.User = url.UserPassword("", c.Password)
		}
	} else if c.User != "" {
		u.User = url.User(c.User)
	}
	return u.String(), nil
}
