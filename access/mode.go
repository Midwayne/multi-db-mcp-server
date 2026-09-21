package access

import (
	"fmt"
	"strings"
)

// Mode is the access level granted to a database connection.
type Mode string

const (
	ModeReadOnly  Mode = "read_only"
	ModeReadWrite Mode = "read_write"
	ModeAdmin     Mode = "admin"
)

// Operation is the kind of work a tool performs.
type Operation string

const (
	OpRead  Operation = "read"
	OpWrite Operation = "write"
	OpAdmin Operation = "admin"
)

// ParseMode accepts canonical names and common aliases.
func ParseMode(raw string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(ModeReadOnly), "readonly", "read-only", "ro":
		return ModeReadOnly, nil
	case string(ModeReadWrite), "readwrite", "read-write", "rw":
		return ModeReadWrite, nil
	case string(ModeAdmin), "administration":
		return ModeAdmin, nil
	default:
		return "", fmt.Errorf("invalid access mode %q (want read_only, read_write, or admin)", raw)
	}
}

// Allows reports whether this mode may perform the given operation.
func (m Mode) Allows(op Operation) bool {
	switch m {
	case ModeReadOnly:
		return op == OpRead
	case ModeReadWrite:
		return op == OpRead || op == OpWrite
	case ModeAdmin:
		return op == OpRead || op == OpWrite || op == OpAdmin
	default:
		return false
	}
}

// DenyMessage is a stable error string for access violations.
func (m Mode) DenyMessage(op Operation) string {
	return fmt.Sprintf("connection access mode %q does not allow %s operations", m, op)
}

// Operations returns the operation kinds this mode permits, in a stable order.
func (m Mode) Operations() []Operation {
	all := []Operation{OpRead, OpWrite, OpAdmin}
	out := make([]Operation, 0, len(all))
	for _, op := range all {
		if m.Allows(op) {
			out = append(out, op)
		}
	}
	return out
}
