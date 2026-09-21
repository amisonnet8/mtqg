package cli

import (
	"strings"
	"unicode"

	"golang.org/x/text/width"
)

// runeWidth is the number of terminal columns a character takes: two for the
// characters the East Asian Width property calls wide or fullwidth (Japanese
// among them), none for marks that sit on the character before them and for
// control characters, and one for the rest.
//
// The width property comes from golang.org/x/text/width, which has no function
// for the width itself. What it does not settle: a sequence of emoji joined by a
// zero width joiner counts as the sum of its parts, and a character of
// ambiguous width counts as one column. Both can only make a table a few
// columns off; no data is affected.
func runeWidth(r rune) int {
	if unicode.IsControl(r) || unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
		return 0
	}
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	default:
		return 1
	}
}

// displayWidth is the number of columns a string takes.
func displayWidth(s string) int {
	n := 0
	for _, r := range s {
		n += runeWidth(r)
	}
	return n
}

// truncate cuts s to at most max columns. If it has to cut, it ends with "..."
// (which counts against max). Marks that belong to the last character kept stay
// with it. With a max too small for the dots, it cuts without them.
func truncate(s string, max int) string {
	if displayWidth(s) <= max {
		return s
	}
	const dots = "..."
	withDots := max > len(dots)
	limit := max
	if withDots {
		limit = max - len(dots)
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		w := runeWidth(r)
		if used+w > limit {
			break
		}
		b.WriteRune(r)
		used += w
	}
	if withDots {
		b.WriteString(dots)
	}
	return b.String()
}

// padRight adds spaces to s until it takes width columns.
func padRight(s string, width int) string {
	if n := width - displayWidth(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}
