package mongodb

import (
	"net/url"

	"dbmcp/db"
	"dbmcp/spec"
)

func init() {
	spec.RegisterKind(spec.Kind{
		Canonical: string(db.TypeMongoDB),
		Aliases:   []string{"mongo", "mongodb"},
		BuildURI:  buildURI,
	})
	db.Register(db.Engine{
		Type: db.TypeMongoDB,
		Open: func(conn spec.Connection) (db.Adapter, error) {
			return New(conn)
		},
	})
}

func buildURI(c *spec.Connection) (string, error) {
	u := &url.URL{
		Scheme: "mongodb",
		Host:   spec.JoinHostPort(c.Host, c.Port, 27017),
	}
	if c.User != "" {
		if c.Password != "" {
			u.User = url.UserPassword(c.User, c.Password)
		} else {
			u.User = url.User(c.User)
		}
	}
	if c.Database != "" {
		u.Path = "/" + c.Database
	} else {
		u.Path = "/"
	}
	return u.String(), nil
}
