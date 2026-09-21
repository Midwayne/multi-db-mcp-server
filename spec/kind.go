package spec

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Kind describes a database type the spec file can name.
// Engines register themselves from init(); adding a kind does not require
// editing NormalizeType.
type Kind struct {
	Canonical string
	Aliases   []string
	BuildURI  func(*Connection) (string, error)
}

var (
	kindsMu  sync.RWMutex
	aliases  = map[string]string{}
	builders = map[string]func(*Connection) (string, error){}
)

// RegisterKind records type aliases and an optional host/port URI builder.
// Re-registering the same canonical name replaces the previous entry.
func RegisterKind(k Kind) {
	canonical := strings.ToLower(strings.TrimSpace(k.Canonical))
	if canonical == "" {
		panic("spec.RegisterKind: canonical type is required")
	}
	kindsMu.Lock()
	defer kindsMu.Unlock()
	aliases[canonical] = canonical
	for _, alias := range k.Aliases {
		alias = strings.ToLower(strings.TrimSpace(alias))
		if alias == "" {
			continue
		}
		aliases[alias] = canonical
	}
	if k.BuildURI != nil {
		builders[canonical] = k.BuildURI
	}
}

// NormalizeType maps type aliases onto canonical names using registered kinds.
func NormalizeType(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return "", fmt.Errorf("type is required")
	}
	kindsMu.RLock()
	defer kindsMu.RUnlock()
	if canonical, ok := aliases[key]; ok {
		return canonical, nil
	}
	return "", fmt.Errorf("unsupported database type %q (supported: %s)", raw, supportedLocked())
}

func buildURI(c *Connection) (string, error) {
	if strings.TrimSpace(c.Host) == "" {
		return "", fmt.Errorf("uri or host is required")
	}
	kindsMu.RLock()
	builder := builders[c.TypeNormalized]
	kindsMu.RUnlock()
	if builder == nil {
		return "", fmt.Errorf("no URI builder registered for type %q (set uri explicitly)", c.TypeNormalized)
	}
	return builder(c)
}

func supportedLocked() string {
	seen := make(map[string]struct{}, len(aliases))
	for _, canonical := range aliases {
		seen[canonical] = struct{}{}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "(none registered)"
	}
	return strings.Join(names, ", ")
}

// JoinHostPort formats host:port, using fallback when port is 0.
func JoinHostPort(host string, port, fallback int) string {
	if port == 0 {
		port = fallback
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]:" + strconv.Itoa(port)
	}
	return host + ":" + strconv.Itoa(port)
}
