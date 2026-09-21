package spec

import (
	"testing"
)

func TestToolAllowed(t *testing.T) {
	c := Connection{Tools: ToolsConfig{Include: []string{"mongo_find"}, Exclude: []string{"mongo_aggregate"}}}
	if !c.ToolAllowed("mongo_find") {
		t.Fatal("include should allow mongo_find")
	}
	if c.ToolAllowed("mongo_count") {
		t.Fatal("include list should deny unspecified tools")
	}
	c = Connection{Tools: ToolsConfig{Exclude: []string{"mongo_delete"}}}
	if !c.ToolAllowed("mongo_find") || c.ToolAllowed("mongo_delete") {
		t.Fatal("exclude list mismatch")
	}
}

func TestRegisterKindExtendsNormalizeAndURI(t *testing.T) {
	RegisterKind(Kind{
		Canonical: "testdb",
		Aliases:   []string{"td", "test-db"},
		BuildURI: func(c *Connection) (string, error) {
			return "testdb://" + JoinHostPort(c.Host, c.Port, 9999) + "/" + c.Database, nil
		},
	})

	got, err := NormalizeType("TD")
	if err != nil {
		t.Fatal(err)
	}
	if got != "testdb" {
		t.Fatalf("alias TD => %s", got)
	}

	c := Connection{Host: "localhost", Database: "app", TypeNormalized: "testdb"}
	uri, err := buildURI(&c)
	if err != nil {
		t.Fatal(err)
	}
	if uri != "testdb://localhost:9999/app" {
		t.Fatalf("uri=%s", uri)
	}
}
