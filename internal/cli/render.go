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

// tableRow is one line of a table: one cell for each column.
type tableRow struct {
	cells []string
	// reply marks a line under another one (the latest answer under its question).
	// A "└" is put in the gap after the first column, so that the text of the
	// answer lines up with the text of the question.
	reply bool
	// dim draws the line faint: it is finished.
	dim bool
}

// formatTable lays rows out as columns, aligned by display width. The column flex
// is the one that is cut with "..." to fit termWidth; the column id is colored as
// an ID. When termWidth is 0, as for a pipe or a file, nothing is cut. A column
// that has nothing in it in any row is left out, gap and all. Lines have no
// trailing spaces.
func formatTable(rows []tableRow, flex, id, termWidth int, st style) []string {
	if len(rows) == 0 {
		return nil
	}
	widths := make([]int, len(rows[0].cells))
	for _, r := range rows {
		for c, cell := range r.cells {
			widths[c] = max(widths[c], displayWidth(cell))
		}
	}
	if termWidth > 0 {
		// The last column of the window is left free: a character written there
		// makes many terminals move to the next line.
		fixed, columns := 0, 0
		for c, w := range widths {
			if w > 0 && c != flex {
				fixed += w
				columns++
			}
		}
		fixed += columns * len(gap) // a gap before the text, and between the others
		widths[flex] = min(widths[flex], max(termWidth-1-fixed, minTextWidth))
	}

	lines := make([]string, len(rows))
	for i, r := range rows {
		var b strings.Builder
		shown := 0
		for c, w := range widths {
			if w == 0 {
				continue
			}
			if shown > 0 {
				if shown == 1 && r.reply {
					b.WriteString("\u2514 ") // U+2514, the corner that leads the eye to the answer
				} else {
					b.WriteString(gap)
				}
			}
			cell := r.cells[c]
			if c == flex {
				cell = truncate(cell, w)
			}
			cell = padRight(cell, w)
			if c == id {
				cell = st.id(cell)
			}
			b.WriteString(cell)
			shown++
		}
		line := strings.TrimRight(b.String(), " ")
		if r.dim {
			line = st.dim(line)
		}
		lines[i] = line
	}
	return lines
}

// listRow is one line of a list of todos, memos, questions or glossary entries.
type listRow struct {
	ID     string
	Word   string // a glossary entry's word, before its text
	Text   string // one line, already made safe
	Author string
	When   string
	// Tail ends the line: the state of a question. A finished item that has none
	// ends with "done".
	Tail  string
	Done  bool // finished: drawn faint
	Reply bool // the latest answer, under its question (ID and Tail are empty)
}

// formatList lays list rows out: ID, word, text, author, time and tail. When
// termWidth is more than 0 the text is cut to fit the window; when it is 0, as
// for a pipe or a file, the text is never cut.
func formatList(rows []listRow, termWidth int, st style) []string {
	table := make([]tableRow, len(rows))
	for i, r := range rows {
		tail := r.Tail
		if tail == "" && r.Done && !r.Reply {
			tail = "done"
		}
		table[i] = tableRow{
			cells: []string{r.ID, r.Word, r.Text, r.Author, r.When, tail},
			reply: r.Reply,
			dim:   r.Done,
		}
	}
	return formatTable(table, 2, 0, termWidth, st)
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
