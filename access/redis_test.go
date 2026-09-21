package access

import "testing"

func TestClassifyRedis(t *testing.T) {
	t.Parallel()

	cases := []struct {
		cmd  string
		want Operation
	}{
		{"GET", OpRead},
		{"get foo", OpRead},
		{"HGETALL", OpRead},
		{"SCAN", OpRead},
		{"SET", OpWrite},
		{"del", OpWrite},
		{"HSET", OpWrite},
		{"FLUSHALL", OpAdmin},
		{"CONFIG", OpAdmin},
		{"unknown-cmd", OpAdmin},
		{"", OpAdmin},
	}
	for _, tc := range cases {
		if got := ClassifyRedis(tc.cmd); got != tc.want {
			t.Fatalf("ClassifyRedis(%q)=%q want %q", tc.cmd, got, tc.want)
		}
	}
}
