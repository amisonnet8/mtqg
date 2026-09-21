package cli

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// sanitize makes the text of a record safe to show on a terminal. A record can
// come from a repository somebody else made, and a terminal obeys the escape
// character and other control characters: they are replaced with U+FFFD, and so
// are the characters that turn text around (U+202E and the like), which can make
// one thing read as another. A tab becomes a space. A line feed is kept; cut the
// text to one line first (firstLine) where a line is wanted.
func sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			b.WriteRune(r)
		case r == '\t':
			b.WriteByte(' ')
		case unicode.IsControl(r), isBidiControl(r), r == lineSeparator, r == paragraphSeparator:
			b.WriteRune(utf8.RuneError)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// The characters that are written as numbers on purpose: written out, they would
// be invisible in this file (and gosec's G116 rightly objects to that).
const (
	lineSeparator      = 0x2028
	paragraphSeparator = 0x2029
)

// isBidiControl reports the characters that turn text around or set it apart:
// the embeddings and overrides (U+202A to U+202E), the isolates (U+2066 to
// U+2069), and the marks U+200E, U+200F and U+061C.
func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) ||
		r == 0x200E || r == 0x200F || r == 0x061C
}

// firstLine is the text up to its first line ending.
func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// oneLine is the first line of a text, made safe to show.
func oneLine(s string) string { return sanitize(firstLine(s)) }

// formatTime shows a time as HH:MM when it is on the same day as now, and as
// YYYY-MM-DD otherwise, in the given place. A time that could not be read is "-".
func formatTime(t, now time.Time, loc *time.Location) string {
	if t.IsZero() {
		return "-"
	}
	local, today := t.In(loc), now.In(loc)
	if local.Year() == today.Year() && local.YearDay() == today.YearDay() {
		return local.Format("15:04")
	}
	return local.Format("2006-01-02")
}

// style adds color. It is decoration only: nothing is said by color alone.
type style struct{ on bool }

func (s style) wrap(code, text string) string {
	if !s.on || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (s style) id(text string) string  { return s.wrap("33", text) }
func (s style) dim(text string) string { return s.wrap("2", text) }

// The gap between the columns of a table.
const gap = "  "

// minTextWidth is the narrowest a text column is cut to. A window narrower than
// the other columns plus this lets the lines wrap instead of cutting the text
// away.
const minTextWidth = 12

// listRow is one line of a list.
type listRow struct {
	ID     string
	Text   string // one line, already made safe
	Author string
	When   string
	Done   bool
}

// formatList lays rows out as columns: ID, text, author, time, and "done" for a
// finished item. The columns are aligned by display width. When termWidth is
// more than 0 the text is cut to fit the window; when it is 0, as for a pipe or a
// file, the text is never cut.
func formatList(rows []listRow, termWidth int, st style) []string {
	var idW, textW, authorW, whenW int
	anyDone := false
	for _, r := range rows {
		idW = max(idW, displayWidth(r.ID))
		textW = max(textW, displayWidth(r.Text))
		authorW = max(authorW, displayWidth(r.Author))
		whenW = max(whenW, displayWidth(r.When))
		anyDone = anyDone || r.Done
	}
	if termWidth > 0 {
		// The last column of the window is left free: a character written there
		// makes many terminals move to the next line.
		fixed := idW + len(gap) + len(gap) + authorW + len(gap) + whenW
		if anyDone {
			fixed += len(gap) + len("done")
		}
		textW = min(textW, max(termWidth-1-fixed, minTextWidth))
	}

	lines := make([]string, len(rows))
	for i, r := range rows {
		var b strings.Builder
		b.WriteString(st.id(padRight(r.ID, idW)))
		b.WriteString(gap)
		b.WriteString(padRight(truncate(r.Text, textW), textW))
		b.WriteString(gap)
		b.WriteString(padRight(r.Author, authorW))
		b.WriteString(gap)
		if anyDone {
			b.WriteString(padRight(r.When, whenW))
			if r.Done {
				b.WriteString(gap + "done")
			}
		} else {
			b.WriteString(r.When)
		}
		line := strings.TrimRight(b.String(), " ")
		if r.Done {
			line = st.dim(line)
		}
		lines[i] = line
	}
	return lines
}

// shortID is the first 10 digits of an ID, which is what mtqg shows, or the whole
// ID with --full-id.
func shortID(id string, full bool) string {
	const shown = 10
	if full || len(id) <= shown {
		return id
	}
	return id[:shown]
}
