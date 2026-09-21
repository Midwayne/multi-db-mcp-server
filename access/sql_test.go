package access

import "testing"

func TestClassifySQL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		sql     string
		want    Operation
		wantErr bool
	}{
		{`SELECT * FROM users`, OpRead, false},
		{`  -- comment
SELECT 1`, OpRead, false},
		{`/* block */ EXPLAIN SELECT 1`, OpRead, false},
		{`WITH cte AS (SELECT 1) SELECT * FROM cte`, OpRead, false},
		{`WITH cte AS (SELECT 1) INSERT INTO t SELECT * FROM cte`, OpWrite, false},
		{`INSERT INTO t VALUES (1)`, OpWrite, false},
		{`UPDATE t SET a=1`, OpWrite, false},
		{`DELETE FROM t`, OpWrite, false},
		{`CREATE TABLE t (id int)`, OpAdmin, false},
		{`DROP TABLE t`, OpAdmin, false},
		{`SELECT 1; SELECT 2`, "", true},
		{`SELECT 'foo;bar'`, OpRead, false},
		{"", "", true},
		{`VACUUM`, OpAdmin, false},
	}

	for _, tc := range cases {
		got, err := ClassifySQL(tc.sql)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ClassifySQL(%q) expected error", tc.sql)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ClassifySQL(%q): %v", tc.sql, err)
		}
		if got != tc.want {
			t.Fatalf("ClassifySQL(%q)=%q want %q", tc.sql, got, tc.want)
		}
	}
}
