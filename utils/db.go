package utils

import (
	"errors"
	"strings"
)

func ExtractMongoHost(uri string) (string, error) {
	// Remove mongodb:// prefix
	uri = strings.TrimPrefix(uri, "mongodb://")

	// Get part after '@' if credentials are present
	if atIdx := strings.Index(uri, "@"); atIdx != -1 {
		uri = uri[atIdx+1:]
	}

	// Host is before the first '/'
	slashIdx := strings.Index(uri, "/")
	if slashIdx == -1 {
		return "", errors.New("invalid URI: missing '/' after host")
	}

	host := uri[:slashIdx]
	if host == "" {
		return "", errors.New("invalid URI: host is empty")
	}

	return host, nil
}
