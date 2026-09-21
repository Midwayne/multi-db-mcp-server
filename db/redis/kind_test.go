package redisdb

import (
	"testing"

	"dbmcp/spec"
)

func TestBuildURI(t *testing.T) {
	uri, err := buildURI(&spec.Connection{Host: "cache", Port: 6380, Password: "s3cret", RedisDB: 2})
	if err != nil {
		t.Fatal(err)
	}
	if uri != "redis://:s3cret@cache:6380/2" {
		t.Fatalf("uri=%s", uri)
	}
}
