package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
	"github.com/amisonnet8/mtqg/internal/model"
)

// runArchive moves the items whose last event is in a range of dates out of
// journal.jsonl into .mtqg/archive/, or, with -n, only says what it would move. It
// is one of two commands that remove lines, so it always says how it read the
// range and what it did.
func runArchive(c *ctx) int {
	switch len(c.inv.words) {
	case 0:
		return c.usageFailure(msgMissingArgument(c.inv.cmd.usage))
	case 1:
	default:
		return c.usageFailure(msgTooManyArguments(c.inv.cmd.usage))
	}
	rng, err := parseRange(c.inv.words[0])
	if err != nil {
		return c.usageFailure(err.Error())
	}
	from, to := rng.bounds(c.env.Location)
	file := rng.fileName()

	// archive writes no event, so it needs no author. It reads first to warn about
	// what could not be read (those lines stay where they are).
	j, err := c.reader()
	if err != nil {
		return c.fail(err)
	}
	state, err := c.load(j)
	if err != nil {
		return c.fail(err)
	}

	plan := state.ArchiveTargets(from, to)
	if !c.inv.dryRun {
		// What moves is decided again under the lock, from the lines as they are
		// then, so that what is reported is what was moved.
		err = j.Archive(file, func(lines []journal.Line) (move, keep []journal.Line, err error) {
			plan = model.Build(eventsOf(lines)).ArchiveTargets(from, to)
			ids := make(map[string]struct{}, len(plan.Move))
			for _, r := range plan.Move {
				ids[r.ID] = struct{}{}
			}
			for _, l := range lines {
				if _, moves := ids[idOf(l)]; moves {
					move = append(move, l)
				} else {
					keep = append(keep, l)
				}
			}
			return move, keep, nil
		})
		if err != nil {
			return c.fail(err)
		}
	}

	archived, skipped := countArchived(plan.Move), countSkipped(plan.Skipped)
	if c.inv.json {
		return c.emit(jsonArchive{
			Command:  c.inv.cmd.label(),
			Range:    jsonArchiveRange{Start: rng.start.String(), End: rng.end.String()},
			File:     journal.ArchivePath(file),
			DryRun:   c.inv.dryRun,
			Archived: archived,
			Skipped:  skipped,
		})
	}

	c.println(msgArchiveRangeLine(rng.String(), c.inv.dryRun))
	if parts := archived.parts(); len(parts) == 0 {
		c.println(msgArchivedNothing)
	} else {
		c.println(msgArchivedLine(parts, journal.ArchivePath(file)))
	}
	if parts := skipped.parts(); len(parts) > 0 {
		c.println(msgSkippedLine(parts))
	}
	return exitOK
}

// idOf is the ID of the event of a line, or "" for a line that could not be read.
// No record has that ID, so such a line is never moved.
func idOf(l journal.Line) string {
	if l.Event == nil {
		return ""
	}
	return l.Event.ID
}

func countArchived(records []*model.Record) jsonArchiveCounts {
	n := jsonArchiveCounts{Records: len(records)}
	for _, r := range records {
		switch r.Kind() {
		case model.KindMemo:
			n.Memos++
		case model.KindTodo:
			n.Todos++
		case model.KindQuestion:
			n.Questions++
		case model.KindAnswer:
			n.Answers++
		case model.KindBug:
			n.Bugs++
		case model.KindReply:
			n.Replies++
		case model.KindGlossary:
			n.GlossaryEntries++
		case model.KindRule:
			n.Rules++
		}
	}
	return n
}

// countSkipped counts the records that stay although their last event is in the
// range. Only open todos, questions and bugs, glossary entries and rules can be
// here.
func countSkipped(records []*model.Record) jsonArchiveSkipped {
	n := jsonArchiveSkipped{Records: len(records)}
	for _, r := range records {
		switch r.Kind() {
		case model.KindTodo:
			n.OpenTodos++
		case model.KindQuestion:
			n.OpenQuestions++
		case model.KindBug:
			n.OpenBugs++
		case model.KindGlossary:
			n.GlossaryEntries++
		case model.KindRule:
			n.Rules++
		}
	}
	return n
}

// counted is a number of things that are said as "12 bugs" (and "1 bug").
type counted struct {
	n                int
	singular, plural string
}

// said is what the counts that are not zero say, in order.
func said(counts ...counted) []string {
	var parts []string
	for _, c := range counts {
		if c.n > 0 {
			parts = append(parts, msgCount(c.n, c.singular, c.plural))
		}
	}
	return parts
}

func (n jsonArchiveCounts) parts() []string {
	return said(
		counted{n.Memos, "memo", "memos"},
		counted{n.Todos, "todo", "todos"},
		counted{n.Questions, "question", "questions"},
		counted{n.Answers, "answer", "answers"},
		counted{n.Bugs, "bug", "bugs"},
		counted{n.Replies, "reply", "replies"},
		counted{n.GlossaryEntries, "glossary entry", "glossary entries"},
		counted{n.Rules, "rule", "rules"},
	)
}

func (n jsonArchiveSkipped) parts() []string {
	return said(
		counted{n.OpenTodos, "open todo", "open todos"},
		counted{n.OpenQuestions, "open question", "open questions"},
		counted{n.OpenBugs, "open bug", "open bugs"},
		counted{n.GlossaryEntries, "glossary entry", "glossary entries"},
		counted{n.Rules, "rule", "rules"},
	)
}

// The range of an archive.

// date is a day of the calendar, with no time of day and no zone.
type date struct {
	year  int
	month time.Month
	day   int
}

func (d date) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.year, int(d.month), d.day) }

func (d date) before(o date) bool {
	if d.year != o.year {
		return d.year < o.year
	}
	if d.month != o.month {
		return d.month < o.month
	}
	return d.day < o.day
}

// dateRange is the days from start to end, both included.
type dateRange struct{ start, end date }

func (r dateRange) String() string { return r.start.String() + ".." + r.end.String() }

// fileName is the name of the archive file, the same for every way of writing the
// same range.
func (r dateRange) fileName() string { return r.String() + ".jsonl" }

// bounds are the times the range covers in loc: from the start of its first day up
// to, and not including, the start of the day after its last.
func (r dateRange) bounds(loc *time.Location) (from, to time.Time) {
	from = time.Date(r.start.year, r.start.month, r.start.day, 0, 0, 0, 0, loc)
	to = time.Date(r.end.year, r.end.month, r.end.day+1, 0, 0, 0, 0, loc)
	return from, to
}

// parseRange reads "<start>..<end>" (docs/reference/cli.md "archive"): the two
// sides are cut at a run of two or more dots, every - is dropped, and the digits
// that are left are a day (8), a month (6) or a year (4); both sides must be the
// same. A month or a year starts on its first day and ends on its last.
func parseRange(arg string) (dateRange, error) {
	if strings.Trim(arg, "0123456789-.") != "" {
		return dateRange{}, &usageError{msgRangeBadCharacter(arg)}
	}
	i := strings.Index(arg, "..")
	if i < 0 {
		return dateRange{}, &usageError{msgRangeNotARange(arg)}
	}
	j := i
	for j < len(arg) && arg[j] == '.' {
		j++
	}
	left, right := arg[:i], arg[j:]

	l, err := parseSide(left)
	if err != nil {
		return dateRange{}, err
	}
	r, err := parseSide(right)
	if err != nil {
		return dateRange{}, err
	}
	if l.digits != r.digits {
		return dateRange{}, &usageError{msgRangeMixedUnits(left, l.unit(), right, r.unit())}
	}

	start, end := l.first(), r.last()
	for _, side := range []side{l, r} {
		if _, ok := side.valid(); !ok {
			return dateRange{}, &usageError{msgRangeNoSuchDate(side.written)}
		}
	}
	if end.before(start) {
		return dateRange{}, &usageError{msgRangeBackwards(start.String(), end.String())}
	}
	return dateRange{start: start, end: end}, nil
}

// side is one side of a range: a year, a month or a day, as far as it is given.
type side struct {
	written string // as it was written, for what is said about it
	digits  int    // 4, 6 or 8
	year    int
	month   int // 0 for a year
	day     int // 0 for a year or a month
}

func parseSide(written string) (side, error) {
	digits := strings.ReplaceAll(written, "-", "")
	if n := len(digits); n != 4 && n != 6 && n != 8 {
		return side{}, &usageError{msgRangeBadSide(written)}
	}
	s := side{written: written, digits: len(digits)}
	// The characters are digits (parseRange checked), so these cannot fail.
	s.year, _ = strconv.Atoi(digits[:4])
	if s.digits >= 6 {
		s.month, _ = strconv.Atoi(digits[4:6])
	}
	if s.digits == 8 {
		s.day, _ = strconv.Atoi(digits[6:8])
	}
	return s, nil
}

func (s side) unit() string {
	switch s.digits {
	case 4:
		return "year"
	case 6:
		return "month"
	default:
		return "day"
	}
}

// valid says whether the month exists, and the day in it if there is one.
func (s side) valid() (date, bool) {
	if s.digits == 4 {
		return date{s.year, time.January, 1}, true
	}
	if s.month < 1 || s.month > 12 {
		return date{}, false
	}
	if s.digits == 6 {
		return date{s.year, time.Month(s.month), 1}, true
	}
	d := time.Date(s.year, time.Month(s.month), s.day, 0, 0, 0, 0, time.UTC)
	if d.Day() != s.day || int(d.Month()) != s.month { // the day was moved into the next month
		return date{}, false
	}
	return date{s.year, time.Month(s.month), s.day}, true
}

// first is the first day the side stands for, and last the last. They are only
// meant for a side that is valid.
func (s side) first() date {
	d, _ := s.valid()
	return d
}

func (s side) last() date {
	d, _ := s.valid()
	switch s.digits {
	case 4:
		return date{s.year, time.December, 31}
	case 6:
		// Day 0 of the next month is the last day of this one.
		return date{s.year, time.Month(s.month), time.Date(s.year, time.Month(s.month)+1, 0, 0, 0, 0, 0, time.UTC).Day()}
	}
	return d
}
