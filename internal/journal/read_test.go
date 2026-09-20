package journal

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func lineOf(t *testing.T, ev Event) string {
	t.Helper()
	b, err := encodeLine(ev)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSuffix(string(b), "\n")
}

func memo(id, text string) Event {
	return Event{ID: id, Op: OpCreate, Type: "memo", Text: text, V: 0, TS: "2026-09-17T00:00:00Z", Author: yamada}
}

func TestScan(t *testing.T) {
	a := lineOf(t, memo(idA, "first"))
	b := lineOf(t, memo(idB, "second"))

	tests := []struct {
		name      string
		input     string
		wantLines int
		wantEvent []bool // per returned line: read as an event?
		wantWarns []Warning
	}{
		{name: "empty input", input: "", wantLines: 0},
		{name: "two lines", input: a + "\n" + b + "\n", wantLines: 2, wantEvent: []bool{true, true}},
		{
			name:      "blank and white-space lines are skipped without a warning",
			input:     "\n" + a + "\n  \t\n\n" + b + "\n",
			wantLines: 2, wantEvent: []bool{true, true},
		},
		{
			name:      "not JSON",
			input:     a + "\nhello\n" + b + "\n",
			wantLines: 3, wantEvent: []bool{true, false, true},
			wantWarns: []Warning{{Kind: WarnInvalidJSON, Line: 2}},
		},
		{
			name:      "no id",
			input:     `{"op":"create"}` + "\n",
			wantLines: 1, wantEvent: []bool{false},
			wantWarns: []Warning{{Kind: WarnMissingField, Line: 1}},
		},
		{
			name:      "invalid UTF-8",
			input:     "{\"id\":\"" + idA + "\",\"op\":\"create\",\"text\":\"caf\xe9\"}\n",
			wantLines: 1, wantEvent: []bool{false},
			wantWarns: []Warning{{Kind: WarnInvalidUTF8, Line: 1}},
		},
		{
			name:      "conflict markers",
			input:     a + "\n<<<<<<< HEAD\n" + b + "\n=======\n" + lineOf(t, memo(idC, "third")) + "\n>>>>>>> feature\n",
			wantLines: 6, wantEvent: []bool{true, false, true, false, true, false},
			wantWarns: []Warning{
				{Kind: WarnConflictMarker, Line: 2},
				{Kind: WarnConflictMarker, Line: 4},
				{Kind: WarnConflictMarker, Line: 6},
			},
		},
		{
			name:      "the last line has no line feed",
			input:     a + "\n" + b,
			wantLines: 2, wantEvent: []bool{true, true},
			wantWarns: []Warning{{Kind: WarnNoTrailingNewline, Line: 2}},
		},
		{name: "CRLF line endings", input: a + "\r\n" + b + "\r\n", wantLines: 2, wantEvent: []bool{true, true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, warns, err := Scan(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if len(lines) != tt.wantLines {
				t.Fatalf("got %d lines, want %d", len(lines), tt.wantLines)
			}
			for i, line := range lines {
				if got := line.Event != nil; got != tt.wantEvent[i] {
					t.Errorf("line %d read as an event = %v, want %v", line.Number, got, tt.wantEvent[i])
				}
			}
			if !reflect.DeepEqual(warns, tt.wantWarns) {
				t.Errorf("warnings = %+v, want %+v", warns, tt.wantWarns)
			}
		})
	}
}

func TestScanKeepsRawBytesAndNumbers(t *testing.T) {
	// Formatting that mtqg would never write, an unknown field, a line that
	// cannot be read and one with invalid UTF-8: all must come back byte for byte.
	handMade := `{ "op" : "create", "id":"` + idA + `", "priority":"high" }`
	broken := "not json at all"
	bad := "{\"id\":\"" + idB + "\",\"op\":\"create\",\"text\":\"caf\xe9\"}"
	input := "\n" + handMade + "\r\n" + broken + "\n\n" + bad + "\n"

	lines, _, err := Scan(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		number int
		raw    string
	}{{2, handMade}, {3, broken}, {5, bad}}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(lines), len(want))
	}
	for i, w := range want {
		if lines[i].Number != w.number || string(lines[i].Raw) != w.raw {
			t.Errorf("line %d: got number %d raw %q, want number %d raw %q", i, lines[i].Number, lines[i].Raw, w.number, w.raw)
		}
	}
	if lines[0].Event == nil || lines[0].Event.ID != idA {
		t.Errorf("the hand-made line should read as an event, got %+v", lines[0].Event)
	}
}

func TestScanLongLine(t *testing.T) {
	long := lineOf(t, memo(idA, strings.Repeat("あ", 1<<19))) // about 1.5 MB
	lines, warns, err := Scan(strings.NewReader(long + "\n" + lineOf(t, memo(idB, "after")) + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || len(warns) != 0 || lines[0].Event == nil || lines[1].Event == nil {
		t.Fatalf("lines = %d, warnings = %+v", len(lines), warns)
	}
}

func TestScanReaderError(t *testing.T) {
	boom := errors.New("boom")
	r := io.MultiReader(strings.NewReader(lineOf(t, memo(idA, "x"))+"\n"), iotest.ErrReader(boom))
	if _, _, err := Scan(r); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the error of the reader", err)
	}
}

func TestIsConflictMarker(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"<<<<<<< HEAD", true},
		{"<<<<<<<", true},
		{"=======", true},
		{"======= ", true},
		{">>>>>>> feature/x", true},
		{"||||||| base", true},
		{"<<<<<<<<", false}, // eight, not a marker
		{"====", false},
		{`{"id":"x"}`, false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isConflictMarker([]byte(tt.line)); got != tt.want {
			t.Errorf("isConflictMarker(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestNewResultCountsIdenticalLinesOnce(t *testing.T) {
	a := lineOf(t, memo(idA, "first"))
	// The same event written with other spacing is a different line: only lines
	// whose content is exactly the same are one event.
	aSpaced := strings.Replace(a, `"op":"create"`, `"op": "create"`, 1)

	lines, warns, err := Scan(strings.NewReader(a + "\n" + a + "\n" + aSpaced + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	result := newResult(lines, warns)
	if len(result.Events) != 2 {
		t.Fatalf("got %d events, want 2 (identical lines count once)", len(result.Events))
	}
	if len(lines) != 3 {
		t.Errorf("Scan must return every line for a rewrite, got %d", len(lines))
	}
}
