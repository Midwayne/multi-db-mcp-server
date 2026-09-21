package access

import "testing"

func TestParseMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in      string
		want    Mode
		wantErr bool
	}{
		{"", ModeReadOnly, false},
		{"read_only", ModeReadOnly, false},
		{"RO", ModeReadOnly, false},
		{"read-write", ModeReadWrite, false},
		{"rw", ModeReadWrite, false},
		{"admin", ModeAdmin, false},
		{"nope", "", true},
	}
	for _, tc := range cases {
		got, err := ParseMode(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseMode(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseMode(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseMode(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestModeAllows(t *testing.T) {
	t.Parallel()

	if !ModeReadOnly.Allows(OpRead) || ModeReadOnly.Allows(OpWrite) || ModeReadOnly.Allows(OpAdmin) {
		t.Fatal("read_only should allow only read")
	}
	if !ModeReadWrite.Allows(OpRead) || !ModeReadWrite.Allows(OpWrite) || ModeReadWrite.Allows(OpAdmin) {
		t.Fatal("read_write should allow read and write")
	}
	if !ModeAdmin.Allows(OpRead) || !ModeAdmin.Allows(OpWrite) || !ModeAdmin.Allows(OpAdmin) {
		t.Fatal("admin should allow all operations")
	}
	if Mode("weird").Allows(OpRead) {
		t.Fatal("unknown mode should deny")
	}
}
