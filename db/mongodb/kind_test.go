package mongodb

import (
	"testing"

	"dbmcp/spec"
)

func TestBuildURI(t *testing.T) {
	uri, err := buildURI(&spec.Connection{Host: "db.example", Port: 27018, User: "u", Password: "p", Database: "app"})
	if err != nil {
		t.Fatal(err)
	}
	if uri != "mongodb://u:p@db.example:27018/app" {
		t.Fatalf("uri=%s", uri)
	}
}
