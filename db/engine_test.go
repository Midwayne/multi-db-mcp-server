package db

import (
	"testing"

	"dbmcp/access"
	"dbmcp/spec"
)

func TestOpenRegisteredEngine(t *testing.T) {
	t.Parallel()
	Register(Engine{
		Type: "fake",
		Open: func(c spec.Connection) (Adapter, error) {
			return &stubAdapter{Meta: Meta{Conn: c}}, nil
		},
	})

	adapter, err := Open(spec.Connection{
		Name:           "demo",
		TypeNormalized: "fake",
		AccessMode:     access.ModeReadOnly,
	})
	if err != nil {
		t.Fatal(err)
	}
	if adapter.Name() != "demo" || adapter.Type() != "fake" {
		t.Fatalf("opened %+v", adapter)
	}
}

func TestOpenUnknownEngine(t *testing.T) {
	t.Parallel()
	_, err := Open(spec.Connection{Name: "x", TypeNormalized: "no-such-engine"})
	if err == nil {
		t.Fatal("expected unsupported type error")
	}
}
