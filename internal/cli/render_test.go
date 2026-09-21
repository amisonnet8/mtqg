package cli

import (
	"strings"
	"testing"
	"time"
)

func TestRuneAndDisplayWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"ASCII", "block comment", 13},
		{"hiragana", "あいう", 6},
		{"kanji", "字句解析", 8},
		{"katakana", "トークン", 8},
		{"half-width katakana", "ﾄｰｸﾝ", 4},
		{"full-width Latin", "ＡＢＣ", 6},
		{"mixed", "Lexer（字句解析）", 5 + 2 + 8 + 2},
		{"a combining accent", "e\u0301", 1},
		{"a Japanese voiced mark", "か\u3099", 2},
		{"an emoji", "🙂", 2},
		{"a zero width joiner", "a\u200db", 2},
		{"a control character", "a\x1bb", 2},
		{"the replacement character", "\uFFFD", 1},
		{"empty", "", 0},
	}
	for _, tt := range tests {
		if got := displayWidth(tt.in); got != tt.want {
			t.Errorf("%s: displayWidth(%q) = %d, want %d", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"fits", "short", 10, "short"},
		{"fits exactly", "exact", 5, "exact"},
		{"cut with dots", "Skip block comments /* */", 12, "Skip bloc..."},
		{"Japanese cut by width, not by characters", "ブロックコメントの読み飛ばし", 11, "ブロック..."},
		{"a wide character that would cross the limit is left out", "あいうえお", 8, "あい..."},
		{"a mark stays with its character", "a\u0301bcdefgh", 6, "a\u0301bc..."},
		{"too narrow for dots", "abcdef", 3, "abc"},
		{"nothing fits", "abcdef", 0, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.in, tt.max)
		if got != tt.want {
			t.Errorf("%s: truncate(%q, %d) = %q, want %q", tt.name, tt.in, tt.max, got, tt.want)
		}
		if displayWidth(got) > tt.max {
			t.Errorf("%s: %q is %d columns, more than %d", tt.name, got, displayWidth(got), tt.max)
		}
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text is unchanged", "Skip block comments /* */", "Skip block comments /* */"},
		{"Japanese is unchanged", "エラーは英語で。", "エラーは英語で。"},
		{"the escape character", "a\x1b[2Jb", "a\uFFFD[2Jb"},
		{"a carriage return", "safe\rfake", "safe\uFFFDfake"},
		{"the bell and a null", "a\x07b\x00c", "a\uFFFDb\uFFFDc"},
		{"a C1 control character", "a\u009bb", "a\uFFFDb"},
		{"a tab becomes a space", "a\tb", "a b"},
		{"a line feed is kept", "a\nb", "a\nb"},
		{"right-to-left override", "a\u202eb", "a\uFFFDb"},
		{"isolates", "a\u2066b\u2069c", "a\uFFFDb\uFFFDc"},
		{"line separator", "a\u2028b", "a\uFFFDb"},
	}
	for _, tt := range tests {
		if got := sanitize(tt.in); got != tt.want {
			t.Errorf("%s: sanitize(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
	if got := oneLine("first\nsecond\x1b[0m"); got != "first" {
		t.Errorf("oneLine = %q", got)
	}
	if got := oneLine("a\x1b[31mred\nb"); got != "a\uFFFD[31mred" {
		t.Errorf("oneLine = %q", got)
	}
}

func TestFormatTime(t *testing.T) {
	utc := time.UTC
	tokyo := time.FixedZone("JST", 9*3600)
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, utc)

	tests := []struct {
		name string
		when time.Time
		loc  *time.Location
		want string
	}{
		{"today is the time", time.Date(2026, 9, 17, 10, 18, 0, 0, utc), utc, "10:18"},
		{"another day is the date", time.Date(2026, 9, 16, 23, 59, 0, 0, utc), utc, "2026-09-16"},
		{"another year", time.Date(2025, 9, 17, 12, 0, 0, 0, utc), utc, "2025-09-17"},
		{"the place decides what today is", time.Date(2026, 9, 17, 20, 0, 0, 0, utc), tokyo, "2026-09-18"},
		{"the place decides the hour", time.Date(2026, 9, 17, 1, 30, 0, 0, utc), tokyo, "10:30"},
		{"a time that could not be read", time.Time{}, utc, "-"},
	}
	for _, tt := range tests {
		if got := formatTime(tt.when, now, tt.loc); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestShortID(t *testing.T) {
	full := "81e74ef5e8e24d949ed904759531985d"
	if got := shortID(full, false); got != "81e74ef5e8" {
		t.Errorf("got %q", got)
	}
	if got := shortID(full, true); got != full {
		t.Errorf("got %q", got)
	}
	if got := shortID("abc", false); got != "abc" {
		t.Errorf("got %q", got)
	}
}

var listFixture = []listRow{
	{ID: "6cad4a268d", Text: "Skip block comments /* */", Author: "claude-code", When: "10:18"},
	{ID: "6513270e26", Text: "ブロックコメントの読み飛ばし", Author: "yamada", When: "2026-09-16"},
	{ID: "1e27a1c08a", Text: "Show error positions", Author: "yamada", When: "11:06", Done: true},
}

func TestFormatListAlignsByDisplayWidth(t *testing.T) {
	lines := formatList(listFixture, 0, style{})
	got := strings.Join(lines, "\n")
	// Every line puts the author at the same display column.
	col := -1
	for i, line := range lines {
		idx := strings.Index(line, []string{"claude-code", "yamada", "yamada"}[i])
		w := displayWidth(line[:idx])
		if col < 0 {
			col = w
		}
		if w != col {
			t.Errorf("line %d: the author starts at column %d, want %d:\n%s", i, w, col, got)
		}
	}
	if !strings.HasSuffix(lines[2], "done") || strings.Contains(lines[0], "done") || strings.Contains(lines[1], "done") {
		t.Errorf("only the finished item ends with done:\n%s", got)
	}
}

func TestFormatListCutsTheTextToTheWindow(t *testing.T) {
	long := []listRow{
		{ID: "6cad4a268d", Text: strings.Repeat("あ", 60), Author: "yamada", When: "10:18"},
		{ID: "6513270e26", Text: strings.Repeat("x", 100), Author: "claude-code", When: "10:52"},
	}
	for _, width := range []int{60, 80, 100} {
		for _, line := range formatList(long, width, style{}) {
			if displayWidth(line) > width-1 {
				t.Errorf("width %d: a line is %d columns: %q", width, displayWidth(line), line)
			}
			if !strings.Contains(line, "...") {
				t.Errorf("width %d: a cut text should end with dots: %q", width, line)
			}
		}
	}

	// Without a window (a pipe or a file) nothing is cut.
	for _, line := range formatList(long, 0, style{}) {
		if strings.Contains(line, "...") {
			t.Errorf("a text was cut although the output is not a terminal: %q", line)
		}
	}
	if got := formatList(long, 0, style{}); !strings.Contains(got[0], strings.Repeat("あ", 60)) || !strings.Contains(got[1], strings.Repeat("x", 100)) {
		t.Error("the whole text should be there")
	}
}

func TestFormatListNeverCutsBelowAReadableWidth(t *testing.T) {
	rows := []listRow{{ID: "6cad4a268d", Text: strings.Repeat("x", 80), Author: "someone-with-a-long-name", When: "2026-09-16"}}
	line := formatList(rows, 20, style{})[0] // far too narrow for the other columns
	if !strings.Contains(line, strings.Repeat("x", minTextWidth-3)) {
		t.Errorf("the text should keep a readable width: %q", line)
	}
}

func TestFormatListColor(t *testing.T) {
	plain := formatList(listFixture, 0, style{})
	colored := formatList(listFixture, 0, style{on: true})
	for i := range plain {
		if strings.Contains(plain[i], "\x1b") {
			t.Errorf("plain output has an escape: %q", plain[i])
		}
		if !strings.Contains(colored[i], "\x1b[") {
			t.Errorf("colored output has no color: %q", colored[i])
		}
	}
	// Color is decoration: with the escape sequences removed, it is the same.
	strip := func(s string) string {
		for _, code := range []string{"\x1b[33m", "\x1b[2m", "\x1b[1m", "\x1b[0m"} {
			s = strings.ReplaceAll(s, code, "")
		}
		return s
	}
	for i := range plain {
		if strip(colored[i]) != plain[i] {
			t.Errorf("line %d differs apart from color:\n%q\n%q", i, strip(colored[i]), plain[i])
		}
	}
}
