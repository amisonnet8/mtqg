package journal

import "testing"

func TestNewID(t *testing.T) {
	seen := make(map[string]bool)
	for range 10000 {
		id, err := newID()
		if err != nil {
			t.Fatal(err)
		}
		if !isID(id) {
			t.Fatalf("newID() = %q, want 32 lowercase hex digits", id)
		}
		// UUID version 4, variant 10.
		if id[12] != '4' {
			t.Fatalf("newID() = %q: the version digit is %q, want 4", id, id[12])
		}
		if !contains("89ab", id[16]) {
			t.Fatalf("newID() = %q: the variant digit is %q, want one of 8 9 a b", id, id[16])
		}
		if seen[id] {
			t.Fatalf("newID() repeated %q", id)
		}
		seen[id] = true
	}
}

func TestIsID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"81e74ef5e8e24d949ed904759531985d", true},
		{"81e74ef5e8", false},                           // a short ID is not stored
		{"81E74EF5E8E24D949ED904759531985D", false},     // uppercase
		{"81e74ef5-e8e2-4d94-9ed9-04759531985d", false}, // hyphens
		{"81e74ef5e8e24d949ed904759531985g", false},     // not hex
		{"81e74ef5e8e24d949ed904759531985d0", false},    // too long
		{"", false},
	}
	for _, tt := range tests {
		if got := isID(tt.in); got != tt.want {
			t.Errorf("isID(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func contains(set string, c byte) bool {
	for i := 0; i < len(set); i++ {
		if set[i] == c {
			return true
		}
	}
	return false
}
