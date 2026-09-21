package postgres

import (
	"testing"

	"dbmcp/spec"
)

func TestBuildURI(t *testing.T) {
	uri, err := buildURI(&spec.Connection{Host: "db.example", Database: "app", User: "u", Password: "p", SSLMode: "disable"})
	if err != nil {
		t.Fatal(err)
	}
	if uri != "postgres://u:p@db.example:5432/app?sslmode=disable" {
		t.Fatalf("uri=%s", uri)
	}
}
