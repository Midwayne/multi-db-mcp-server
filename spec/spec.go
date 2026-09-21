package spec

import (
	"fmt"
	"strings"
	"time"

	"dbmcp/access"
)

const (
	DefaultServeMode          = "stdio"
	DefaultPort               = 8080
	DefaultAccess             = "read_only"
	DefaultConnectTimeout     = 10 * time.Second
	DefaultInactivityTimeout  = 30 * time.Minute
	DefaultMaxRows            = 200
	DefaultServerName         = "Multi-Database MCP"
	DefaultServerVersion      = "0.1.0"
	DefaultFailOnConnectError = true
	DefaultConnectOnStart     = true
)

// Spec is the root configuration document loaded from YAML or JSON.
type Spec struct {
	Server      ServerConfig `yaml:"server" json:"server"`
	Defaults    Defaults     `yaml:"defaults" json:"defaults"`
	Connections []Connection `yaml:"connections" json:"connections"`
	Tools       ToolsConfig  `yaml:"tools" json:"tools"`
}

type ServerConfig struct {
	Name      string `yaml:"name" json:"name"`
	Version   string `yaml:"version" json:"version"`
	ServeMode string `yaml:"serve_mode" json:"serve_mode"`
	Port      int    `yaml:"port" json:"port"`
}

type Defaults struct {
	Access             string `yaml:"access" json:"access"`
	ConnectTimeout     string `yaml:"connect_timeout" json:"connect_timeout"`
	InactivityTimeout  string `yaml:"inactivity_timeout" json:"inactivity_timeout"`
	ConnectOnStart     *bool  `yaml:"connect_on_start" json:"connect_on_start"`
	FailOnConnectError *bool  `yaml:"fail_on_connect_error" json:"fail_on_connect_error"`
	MaxRows            int    `yaml:"max_rows" json:"max_rows"`
}

type ToolsConfig struct {
	Include []string `yaml:"include" json:"include"`
	Exclude []string `yaml:"exclude" json:"exclude"`
}

// Connection is a single named database target.
type Connection struct {
	Name               string         `yaml:"name" json:"name"`
	Type               string         `yaml:"type" json:"type"`
	URI                string         `yaml:"uri" json:"uri"`
	Host               string         `yaml:"host" json:"host"`
	Port               int            `yaml:"port" json:"port"`
	User               string         `yaml:"user" json:"user"`
	Password           string         `yaml:"password" json:"password"`
	Database           string         `yaml:"database" json:"database"`
	SSLMode            string         `yaml:"ssl_mode" json:"ssl_mode"`
	RedisDB            int            `yaml:"db" json:"db"`
	Access             string         `yaml:"access" json:"access"`
	Enabled            *bool          `yaml:"enabled" json:"enabled"`
	Tools              ToolsConfig    `yaml:"tools" json:"tools"`
	Options            map[string]any `yaml:"options" json:"options"`
	ConnectOnStart     *bool          `yaml:"connect_on_start" json:"connect_on_start"`
	ConnectTimeout     string         `yaml:"connect_timeout" json:"connect_timeout"`
	InactivityTimeout  string         `yaml:"inactivity_timeout" json:"inactivity_timeout"`
	MaxRows            int            `yaml:"max_rows" json:"max_rows"`
	FailOnConnectError *bool          `yaml:"fail_on_connect_error" json:"fail_on_connect_error"`

	// Normalized fields populated by Spec.Normalize.
	TypeNormalized        string        `yaml:"-" json:"-"`
	AccessMode            access.Mode   `yaml:"-" json:"-"`
	ConnectTimeoutDur     time.Duration `yaml:"-" json:"-"`
	InactivityTimeoutDur  time.Duration `yaml:"-" json:"-"`
	ConnectOnStartVal     bool          `yaml:"-" json:"-"`
	FailOnConnectErrorVal bool          `yaml:"-" json:"-"`
	EnabledVal            bool          `yaml:"-" json:"-"`
	MaxRowsVal            int           `yaml:"-" json:"-"`
}

func boolVal(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func durationOr(raw string, fallback time.Duration) (time.Duration, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", raw, err)
	}
	return d, nil
}

// Normalize applies defaults, aliases, URI construction, and validation.
func (s *Spec) Normalize() error {
	if s.Server.Name == "" {
		s.Server.Name = DefaultServerName
	}
	if s.Server.Version == "" {
		s.Server.Version = DefaultServerVersion
	}
	if s.Server.ServeMode == "" {
		s.Server.ServeMode = DefaultServeMode
	}
	switch strings.ToLower(s.Server.ServeMode) {
	case "stdio", "http", "sse":
		s.Server.ServeMode = strings.ToLower(s.Server.ServeMode)
	default:
		return fmt.Errorf("invalid serve_mode %q (want stdio, http, or sse)", s.Server.ServeMode)
	}
	if s.Server.Port == 0 {
		s.Server.Port = DefaultPort
	}

	if s.Defaults.Access == "" {
		s.Defaults.Access = DefaultAccess
	}
	if _, err := access.ParseMode(s.Defaults.Access); err != nil {
		return fmt.Errorf("defaults.access: %w", err)
	}
	if s.Defaults.MaxRows == 0 {
		s.Defaults.MaxRows = DefaultMaxRows
	}

	defaultConnectTimeout, err := durationOr(s.Defaults.ConnectTimeout, DefaultConnectTimeout)
	if err != nil {
		return fmt.Errorf("defaults.connect_timeout: %w", err)
	}
	defaultInactivity, err := durationOr(s.Defaults.InactivityTimeout, DefaultInactivityTimeout)
	if err != nil {
		return fmt.Errorf("defaults.inactivity_timeout: %w", err)
	}
	defaultConnectOnStart := boolVal(s.Defaults.ConnectOnStart, DefaultConnectOnStart)
	defaultFailOnConnect := boolVal(s.Defaults.FailOnConnectError, DefaultFailOnConnectError)

	if len(s.Connections) == 0 {
		return fmt.Errorf("spec must declare at least one connection")
	}

	seen := make(map[string]struct{}, len(s.Connections))
	enabledCount := 0
	for i := range s.Connections {
		c := &s.Connections[i]
		if err := normalizeConnection(c, s.Defaults, defaultConnectTimeout, defaultInactivity, defaultConnectOnStart, defaultFailOnConnect); err != nil {
			return err
		}
		if _, ok := seen[c.Name]; ok {
			return fmt.Errorf("duplicate connection name %q", c.Name)
		}
		seen[c.Name] = struct{}{}
		if c.EnabledVal {
			enabledCount++
		}
	}
	if enabledCount == 0 {
		return fmt.Errorf("spec must declare at least one enabled connection")
	}
	return nil
}

func normalizeConnection(c *Connection, defaults Defaults, defaultConnectTimeout, defaultInactivity time.Duration, defaultConnectOnStart, defaultFailOnConnect bool) error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return fmt.Errorf("connection name is required")
	}

	typ, err := NormalizeType(c.Type)
	if err != nil {
		return fmt.Errorf("connection %q: %w", c.Name, err)
	}
	c.TypeNormalized = typ

	accessRaw := c.Access
	if accessRaw == "" {
		accessRaw = defaults.Access
	}
	mode, err := access.ParseMode(accessRaw)
	if err != nil {
		return fmt.Errorf("connection %q: %w", c.Name, err)
	}
	c.AccessMode = mode
	c.Access = string(mode)

	c.EnabledVal = boolVal(c.Enabled, true)
	c.ConnectOnStartVal = boolVal(c.ConnectOnStart, defaultConnectOnStart)
	c.FailOnConnectErrorVal = boolVal(c.FailOnConnectError, defaultFailOnConnect)

	timeout, err := durationOr(c.ConnectTimeout, defaultConnectTimeout)
	if err != nil {
		return fmt.Errorf("connection %q connect_timeout: %w", c.Name, err)
	}
	c.ConnectTimeoutDur = timeout

	inactivity, err := durationOr(c.InactivityTimeout, defaultInactivity)
	if err != nil {
		return fmt.Errorf("connection %q inactivity_timeout: %w", c.Name, err)
	}
	c.InactivityTimeoutDur = inactivity

	if c.MaxRows > 0 {
		c.MaxRowsVal = c.MaxRows
	} else {
		c.MaxRowsVal = defaults.MaxRows
	}

	if strings.TrimSpace(c.URI) == "" {
		uri, err := buildURI(c)
		if err != nil {
			return fmt.Errorf("connection %q: %w", c.Name, err)
		}
		c.URI = uri
	}
	return nil
}

// ToolAllowed reports whether a tool can run against this connection's allow/deny lists.
func (c Connection) ToolAllowed(tool string) bool {
	for _, name := range c.Tools.Exclude {
		if name == tool {
			return false
		}
	}
	if len(c.Tools.Include) == 0 {
		return true
	}
	for _, name := range c.Tools.Include {
		if name == tool {
			return true
		}
	}
	return false
}

// GloballyAllowed reports whether a tool survives the spec-level include/exclude lists.
func (t ToolsConfig) GloballyAllowed(tool string) bool {
	for _, name := range t.Exclude {
		if name == tool {
			return false
		}
	}
	if len(t.Include) == 0 {
		return true
	}
	for _, name := range t.Include {
		if name == tool {
			return true
		}
	}
	return false
}
