package db

import (
	"context"
	"fmt"
	"log"
	"sync"

	"dbmcp/access"
	"dbmcp/spec"
)

// Opener constructs an adapter from a normalized connection spec.
type Opener func(conn spec.Connection) (Adapter, error)

// Registry holds named adapters created from a spec file.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
	order    []string
}

func NewRegistry(ctx context.Context, sp *spec.Spec, open Opener) (*Registry, error) {
	if open == nil {
		return nil, fmt.Errorf("database opener is required")
	}
	r := &Registry{adapters: make(map[string]Adapter, len(sp.Connections))}

	for _, conn := range sp.Connections {
		if !conn.EnabledVal {
			continue
		}
		adapter, err := open(conn)
		if err != nil {
			return nil, fmt.Errorf("connection %q: %w", conn.Name, err)
		}
		if conn.ConnectOnStartVal {
			if err := adapter.EnsureConnected(ctx); err != nil {
				_ = adapter.Close(ctx)
				if conn.FailOnConnectErrorVal {
					return nil, fmt.Errorf("connection %q: %w", conn.Name, err)
				}
				log.Printf("warning: connection %q failed to connect: %v", conn.Name, err)
			}
		}
		r.adapters[conn.Name] = adapter
		r.order = append(r.order, conn.Name)
	}

	if len(r.adapters) == 0 {
		return nil, fmt.Errorf("no database connections were created")
	}
	return r, nil
}

func (r *Registry) Get(ctx context.Context, name string) (Adapter, error) {
	adapter, err := r.Lookup(name)
	if err != nil {
		return nil, err
	}
	if err := adapter.EnsureConnected(ctx); err != nil {
		return nil, err
	}
	return adapter, nil
}

func (r *Registry) Lookup(name string) (Adapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[name]
	if !ok {
		return nil, fmt.Errorf("unknown connection %q", name)
	}
	return adapter, nil
}

func (r *Registry) List() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Info, 0, len(r.order))
	for _, name := range r.order {
		a := r.adapters[name]
		out = append(out, Info{
			Name:     a.Name(),
			Type:     a.Type(),
			Access:   a.Access(),
			Database: a.DefaultDatabase(),
			Ready:    a.Connected(),
		})
	}
	return out
}

func (r *Registry) All() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Adapter, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.adapters[name])
	}
	return out
}

func (r *Registry) HasType(t Type) bool {
	for _, a := range r.All() {
		if a.Type() == t {
			return true
		}
	}
	return false
}

func (r *Registry) AnyAllows(t Type, op access.Operation, tool string) bool {
	for _, a := range r.All() {
		if a.Type() != t {
			continue
		}
		if !a.ToolAllowed(tool) {
			continue
		}
		if a.Access().Allows(op) {
			return true
		}
	}
	return false
}

func (r *Registry) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var first error
	for _, name := range r.order {
		if err := r.adapters[name].Close(ctx); err != nil && first == nil {
			first = fmt.Errorf("%s: %w", name, err)
		}
	}
	return first
}
