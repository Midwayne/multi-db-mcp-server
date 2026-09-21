package access

import (
	"fmt"
	"strings"
	"unicode"
)

// ClassifySQL returns the operation class of a SQL statement.
// Multiple statements are rejected. Classification is conservative: unknown
// leading keywords are treated as admin.
func ClassifySQL(sql string) (Operation, error) {
	stripped, err := stripSQLComments(sql)
	if err != nil {
		return "", err
	}
	stripped = strings.TrimSpace(stripped)
	if stripped == "" {
		return "", fmt.Errorf("empty SQL statement")
	}
	if err := rejectMultipleStatements(stripped); err != nil {
		return "", err
	}

	keyword := leadingSQLKeyword(stripped)
	if keyword == "WITH" {
		main, err := keywordAfterCTEs(stripped)
		if err != nil {
			return "", err
		}
		keyword = main
	}

	switch keyword {
	case "SELECT", "SHOW", "EXPLAIN", "DESCRIBE", "DESC", "VALUES", "TABLE":
		return OpRead, nil
	case "INSERT", "UPDATE", "DELETE", "MERGE", "COPY", "REPLACE", "UPSERT":
		return OpWrite, nil
	case "CREATE", "DROP", "ALTER", "TRUNCATE", "GRANT", "REVOKE", "VACUUM",
		"REINDEX", "CLUSTER", "COMMENT", "SECURITY", "ANALYZE", "DISCARD",
		"LOAD", "CHECKPOINT", "REASSIGN", "REFRESH":
		return OpAdmin, nil
	case "":
		return "", fmt.Errorf("could not determine SQL statement type")
	default:
		return OpAdmin, nil
	}
}

func stripSQLComments(sql string) (string, error) {
	var b strings.Builder
	b.Grow(len(sql))
	inSingle, inDouble, inLineComment := false, false, false
	inBlockComment := false

	for i := 0; i < len(sql); i++ {
		c := sql[i]
		n := byte(0)
		if i+1 < len(sql) {
			n = sql[i+1]
		}

		if inLineComment {
			if c == '\n' {
				inLineComment = false
				b.WriteByte(c)
			}
			continue
		}
		if inBlockComment {
			if c == '*' && n == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if !inSingle && !inDouble {
			if c == '-' && n == '-' {
				inLineComment = true
				i++
				continue
			}
			if c == '/' && n == '*' {
				inBlockComment = true
				i++
				continue
			}
		}
		if c == '\'' && !inDouble {
			if inSingle && n == '\'' {
				b.WriteByte(c)
				b.WriteByte(n)
				i++
				continue
			}
			inSingle = !inSingle
		}
		if c == '"' && !inSingle {
			inDouble = !inDouble
		}
		b.WriteByte(c)
	}

	if inSingle || inDouble || inBlockComment {
		return "", fmt.Errorf("unterminated SQL string or comment")
	}
	return b.String(), nil
}

func rejectMultipleStatements(sql string) error {
	inSingle, inDouble, inDollar := false, false, false
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		if c == '\'' && !inDouble && !inDollar {
			if inSingle && i+1 < len(sql) && sql[i+1] == '\'' {
				i++
				continue
			}
			inSingle = !inSingle
			continue
		}
		if c == '"' && !inSingle && !inDollar {
			inDouble = !inDouble
			continue
		}
		if !inSingle && !inDouble && c == '$' {
			inDollar = !inDollar
			continue
		}
		if c == ';' && !inSingle && !inDouble && !inDollar {
			rest := strings.TrimSpace(sql[i+1:])
			if rest != "" {
				return fmt.Errorf("multiple SQL statements are not allowed")
			}
		}
	}
	return nil
}

func leadingSQLKeyword(sql string) string {
	i := 0
	for i < len(sql) && unicode.IsSpace(rune(sql[i])) {
		i++
	}
	start := i
	for i < len(sql) {
		r := rune(sql[i])
		if !unicode.IsLetter(r) {
			break
		}
		i++
	}
	return strings.ToUpper(sql[start:i])
}

func keywordAfterCTEs(sql string) (string, error) {
	upper := strings.ToUpper(sql)
	i := strings.Index(upper, "WITH")
	if i < 0 {
		return leadingSQLKeyword(sql), nil
	}
	i += 4
	depth := 0
	inSingle, inDouble := false, false
	sawParen := false

	for i < len(sql) {
		c := sql[i]
		if c == '\'' && !inDouble {
			if inSingle && i+1 < len(sql) && sql[i+1] == '\'' {
				i += 2
				continue
			}
			inSingle = !inSingle
			i++
			continue
		}
		if c == '"' && !inSingle {
			inDouble = !inDouble
			i++
			continue
		}
		if inSingle || inDouble {
			i++
			continue
		}
		if c == '(' {
			depth++
			sawParen = true
			i++
			continue
		}
		if c == ')' {
			if depth > 0 {
				depth--
			}
			i++
			continue
		}
		if depth == 0 && sawParen {
			if unicode.IsLetter(rune(c)) {
				return leadingSQLKeyword(sql[i:]), nil
			}
		}
		i++
	}
	return "", fmt.Errorf("could not parse WITH clause")
}
