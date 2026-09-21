package postgres

import (
	"net/url"

	"dbmcp/db"
	"dbmcp/spec"
)

func init() {
	spec.RegisterKind(spec.Kind{
		Canonical: string(db.TypePostgres),
		Aliases:   []string{"postgres", "postgresql", "pg"},
		BuildURI:  buildURI,
	})
	db.Register(db.Engine{
		Type: db.TypePostgres,
		Open: func(conn spec.Connection) (db.Adapter, error) {
			return New(conn)
		},
	})
}

func buildURI(c *spec.Connection) (string, error) {
	u := &url.URL{
		Scheme: "postgres",
		Host:   spec.JoinHostPort(c.Host, c.Port, 5432),
		Path:   "/" + c.Database,
	}
	if c.User != "" {
		if c.Password != "" {
			u.User = url.UserPassword(c.User, c.Password)
		} else {
			u.User = url.User(c.User)
		}
	}
	q := url.Values{}
	if c.SSLMode != "" {
		q.Set("sslmode", c.SSLMode)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
