package db

import (
	"dbmcp/access"
	"dbmcp/spec"
)

// Meta is shared connection identity embedded by every adapter.
type Meta struct {
	Conn spec.Connection
}

func (m Meta) Name() string            { return m.Conn.Name }
func (m Meta) Type() Type              { return Type(m.Conn.TypeNormalized) }
func (m Meta) Access() access.Mode     { return m.Conn.AccessMode }
func (m Meta) MaxRows() int            { return m.Conn.MaxRowsVal }
func (m Meta) DefaultDatabase() string { return m.Conn.Database }
func (m Meta) ToolAllowed(tool string) bool {
	return m.Conn.ToolAllowed(tool)
}
