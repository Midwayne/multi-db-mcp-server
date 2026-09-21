package db

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"dbmcp/spec"
)

// Engine is a database kind that can open adapters from a spec connection.
// Packages register one from init(); connect blank-imports those packages.
type Engine struct {
	Type Type
	Open Opener
}

var (
	enginesMu sync.RWMutex
	engines   = map[Type]Engine{}
)

// Register adds or replaces a database engine.
func Register(e Engine) {
	if e.Type == "" {
		panic("db.Register: engine type is required")
	}
	if e.Open == nil {
		panic("db.Register: opener is required")
	}
	enginesMu.Lock()
	defer enginesMu.Unlock()
	engines[e.Type] = e
}

// Open constructs an adapter using the engine registered for conn.TypeNormalized.
func Open(conn spec.Connection) (Adapter, error) {
	typ := Type(conn.TypeNormalized)
	enginesMu.RLock()
	e, ok := engines[typ]
	enginesMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unsupported database type %q (registered: %s)", conn.TypeNormalized, RegisteredTypes())
	}
	return e.Open(conn)
}

// RegisteredTypes returns canonical engine names in sorted order.
func RegisteredTypes() string {
	enginesMu.RLock()
	defer enginesMu.RUnlock()
	names := make([]string, 0, len(engines))
	for typ := range engines {
		names = append(names, string(typ))
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "(none registered)"
	}
	return strings.Join(names, ", ")
}
