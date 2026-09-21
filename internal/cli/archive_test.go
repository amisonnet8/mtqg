package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// arid is an ID for the records of these tests.
func arid(n int) string { return fmt.Sprintf("a%031x", n) }

// archiveFixture is a journal that has something of every kind in 2023 and around
// it, in the order of its lines: the lines of one record are together.
func archiveFixture(h *harness) []string {
	lines := []string{
		// A todo done in 2023: moves.
		record(arid(1), "todo", "Add tests for comment handling", "yamada", "2023-02-01T09:00:00Z"),
		change(arid(1), "open", "done", "yamada", "2023-03-01T09:00:00Z"),
		// A todo that is still open, in 2023: stays, and is skipped.
		record(arid(2), "todo", "Support nested comments", "yamada", "2023-04-01T09:00:00Z"),
		// Memos in 2022, 2023 and 2024: only the one of 2023 moves.
		record(arid(3), "memo", "Policy of 2022", "yamada", "2022-06-01T09:00:00Z"),
		record(arid(4), "memo", "Policy of 2023", "yamada", "2023-06-01T09:00:00Z"),
		record(arid(5), "memo", "Policy of 2024", "yamada", "2024-06-01T09:00:00Z"),
		// A question closed in 2023 with two answers: the three lines and the close move.
		question(arid(6), "Should nested comments be supported?", "yamada", "2023-05-01T09:00:00Z"),
		answerLine(arid(7), arid(6), "Yes", "claude-code", "ai", "2023-05-02T09:00:00Z"),
		answerLine(arid(8), arid(6), "Not yet", "yamada", "human", "2023-05-03T09:00:00Z"),
		change(arid(6), "open", "done", "yamada", "2023-05-04T09:00:00Z"),
		// A bug closed in 2023 with a reply: moves.
		bugLine(arid(9), "Parser crashes on empty input", "yamada", "2023-07-01T09:00:00Z"),
		replyLine(arid(10), arid(9), "Fixed", "claude-code", "ai", "2023-07-02T09:00:00Z"),
		change(arid(9), "open", "done", "yamada", "2023-07-03T09:00:00Z"),
		// A bug still open: stays, and is skipped.
		bugLine(arid(11), "Slow on large files", "yamada", "2023-08-01T09:00:00Z"),
		// An entry of the glossary: stays, and is skipped. One that was deleted moves.
		entryLine(arid(12), "lexing", "Reading source and turning it into tokens", "yamada", "2023-09-01T09:00:00Z"),
		entryLine(arid(13), "parsing", "A wrong definition", "yamada", "2023-09-02T09:00:00Z"),
		deleteLine(arid(13), "yamada", "2023-09-03T09:00:00Z"),
		// A question closed in 2022 that got an answer in 2024: the thread belongs to
		// neither year, so nothing of it moves.
		question(arid(14), "Is a tab one column?", "yamada", "2022-03-01T09:00:00Z"),
		change(arid(14), "open", "done", "yamada", "2022-03-02T09:00:00Z"),
		answerLine(arid(15), arid(14), "Late answer", "yamada", "human", "2024-01-15T09:00:00Z"),
		// An answer to nothing: it is an item of its own, and moves.
		answerLine(arid(16), arid(99), "An answer to a question that is not here", "yamada", "human", "2023-10-01T09:00:00Z"),
	}
	h.setJournal(lines...)
	return lines
}

// linesOf splits a file into its lines.
func linesOf(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

func (h *harness) archiveDir() string { return filepath.Join(h.root, ".mtqg", "archive") }

func (h *harness) readArchive(name string) string {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.archiveDir(), name))
	if err != nil {
		h.t.Fatal(err)
	}
	return string(data)
}

func (h *harness) noArchiveDir() {
	h.t.Helper()
	if _, err := os.Stat(h.archiveDir()); !os.IsNotExist(err) {
		h.t.Errorf("archive/ exists (%v)", err)
	}
}

func TestParseRange(t *testing.T) {
	ok := []struct{ in, want string }{
		{"2021-01-01..2024-09-18", "2021-01-01..2024-09-18"},
		{"20210101....20240918", "2021-01-01..2024-09-18"},
		{"2021-0101..202409-18", "2021-01-01..2024-09-18"},
		{"2021..2023", "2021-01-01..2023-12-31"},
		{"202404..2024-09", "2024-04-01..2024-09-30"},
		{"2023..2023", "2023-01-01..2023-12-31"},
		{"2024-02..2024-02", "2024-02-01..2024-02-29"},       // a leap year
		{"2023-02..2023-02", "2023-02-01..2023-02-28"},       // not a leap year
		{"2023-12..2023-12", "2023-12-01..2023-12-31"},       // the end of the year
		{"2024-02-29..2024-02-29", "2024-02-29..2024-02-29"}, // one day
		{"2023-01-01......2023-01-02", "2023-01-01..2023-01-02"},
	}
	for _, tt := range ok {
		got, err := parseRange(tt.in)
		if err != nil || got.String() != tt.want {
			t.Errorf("%q: got %v, %v; want %s", tt.in, got, err, tt.want)
		}
	}

	bad := []struct{ name, in, want string }{
		{"no dots", "2023", "is not a range"},
		{"one dot", "2021.2023", "is not a range"},
		{"one side is empty", "2021..", "is not a date, a month or a year"},
		{"the other side is empty", "..2023", "is not a date, a month or a year"},
		{"nothing", "", "is not a range"},
		{"5 digits", "20210..2023", "is not a date, a month or a year"},
		{"7 digits", "2021-0-11..2023-01-01", "is not a date, a month or a year"},
		{"9 digits", "2021-01-011..2023-01-01", "is not a date, a month or a year"},
		{"a year and a month", "2021..2024-09", "not the same kind"},
		{"a month and a day", "2021-01..2024-09-01", "not the same kind"},
		{"a day that does not exist", "2023-02-29..2023-03-01", "2023-02-29 is not a date"},
		{"the 31st of a month with 30 days", "2023-04-01..2023-04-31", "2023-04-31 is not a date"},
		{"a month that does not exist", "2023-13..2023-14", "2023-13 is not a date"},
		{"month 0", "2023-00..2023-01", "2023-00 is not a date"},
		{"day 0", "2023-01-00..2023-01-02", "2023-01-00 is not a date"},
		{"backwards", "2024..2023", "starts after it ends: 2024-01-01..2023-12-31"},
		{"backwards by a day", "2023-01-02..2023-01-01", "starts after it ends"},
		{"a letter", "2021..20x3", "only digits, - and . are allowed"},
		{"a slash", "2021/01..2021/02", "only digits, - and . are allowed"},
		{"a space", "2021 ..2023", "only digits, - and . are allowed"},
		{"a relative date", "2y..now", "only digits, - and . are allowed"},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRange(tt.in)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("%q: got %v, %v; want an error with %q", tt.in, got, err, tt.want)
			}
		})
	}
}

func TestRangeBoundsAreLocalDays(t *testing.T) {
	rng, err := parseRange("2023-12-31..2023-12-31")
	if err != nil {
		t.Fatal(err)
	}
	tokyo := time.FixedZone("JST", 9*3600)
	from, to := rng.bounds(tokyo)
	// The first instant of the day in Tokyo, and the first of the next day.
	if want := time.Date(2023, 12, 30, 15, 0, 0, 0, time.UTC); !from.Equal(want) {
		t.Errorf("from = %v, want %v", from, want)
	}
	if want := time.Date(2023, 12, 31, 15, 0, 0, 0, time.UTC); !to.Equal(want) {
		t.Errorf("to = %v, want %v", to, want)
	}
	// The last day of a year: the next day is in the next year.
	if _, to := (dateRange{date{2023, 1, 1}, date{2023, 12, 31}}).bounds(time.UTC); !to.Equal(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("to = %v, want the start of 2024", to)
	}
}

// hasRecord says whether a line of lines is about the record with this number.
func hasRecord(lines []string, n int) bool {
	for _, l := range lines {
		if strings.Contains(l, `"id":"`+arid(n)+`"`) {
			return true
		}
	}
	return false
}

func TestArchiveMovesTheItemsOfARange(t *testing.T) {
	h := initialized(t)
	all := archiveFixture(h)

	code, out, errOut := h.run("archive", "2023..2023")
	wantExit(t, code, 0, out, errOut)
	// The three answers are the two of the question and the one that has no question.
	want := "Range: 2023-01-01..2023-12-31\n" +
		"Archived: 1 memo, 1 todo, 1 question, 3 answers, 1 bug, 1 reply, 1 glossary entry -> .mtqg/archive/2023-01-01..2023-12-31.jsonl\n" +
		"Skipped: 1 open todo, 1 open bug, 1 glossary entry\n"
	if out != want || errOut != "" {
		t.Errorf("stdout:\n%s\nwant:\n%s\nstderr: %q", out, want, errOut)
	}

	// Every line is in one of the two files, once, and as it was.
	kept := linesOf(h.readJournal())
	moved := linesOf(h.readArchive("2023-01-01..2023-12-31.jsonl"))
	both := append(append([]string(nil), kept...), moved...)
	sort.Strings(both)
	sorted := append([]string(nil), all...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(both, sorted) {
		t.Fatalf("the two files hold\n%s\nwant the lines of the fixture\n%s", strings.Join(both, "\n"), strings.Join(sorted, "\n"))
	}

	// What stays and what moves, by record.
	for _, n := range []int{2, 3, 5, 11, 12, 14, 15} {
		if !hasRecord(kept, n) || hasRecord(moved, n) {
			t.Errorf("record %d should stay in journal.jsonl", n)
		}
	}
	for _, n := range []int{1, 4, 6, 7, 8, 9, 10, 13, 16} {
		if !hasRecord(moved, n) || hasRecord(kept, n) {
			t.Errorf("record %d should be moved to the archive", n)
		}
	}
	// The lines that stay are in the order they were in.
	var wantKept []string
	for _, l := range all {
		if !contains(moved, l) {
			wantKept = append(wantKept, l)
		}
	}
	if !reflect.DeepEqual(kept, wantKept) {
		t.Errorf("journal.jsonl holds\n%s\nwant\n%s", strings.Join(kept, "\n"), strings.Join(wantKept, "\n"))
	}

	// Archived IDs are not found: the commands read journal.jsonl only.
	code, out, errOut = h.run("show", arid(1))
	wantExit(t, code, 1, out, errOut)
	if !strings.Contains(errOut, "No record matches") {
		t.Errorf("show of an archived record: %q", errOut)
	}
	code, out, errOut = h.run("todo", "list", "--all")
	wantExit(t, code, 0, out, errOut)
	if strings.Contains(out, "Add tests for comment handling") || !strings.Contains(out, "Support nested comments") {
		t.Errorf("todo list --all:\n%s", out)
	}
}

func contains(lines []string, line string) bool {
	for _, l := range lines {
		if l == line {
			return true
		}
	}
	return false
}

func TestArchiveGivesBackWhatCatPutsBack(t *testing.T) {
	h := initialized(t)
	all := archiveFixture(h)
	before, _ := h.run2("status")

	h.mustRun("archive", "2023..2023")
	if after, _ := h.run2("status"); after == before {
		t.Fatalf("status did not change after archiving:\n%s", after)
	}

	// The way the spec gives to restore: append the file, delete it.
	name := filepath.Join(h.archiveDir(), "2023-01-01..2023-12-31.jsonl")
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(h.journalPath(), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}

	if got, _ := h.run2("status"); got != before {
		t.Errorf("status after restoring:\n%s\nbefore:\n%s", got, before)
	}
	restored := linesOf(h.readJournal())
	sort.Strings(restored)
	sorted := append([]string(nil), all...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(restored, sorted) {
		t.Errorf("the journal after restoring holds\n%s\nwant\n%s", strings.Join(restored, "\n"), strings.Join(sorted, "\n"))
	}
}

// run2 is run for a command that must succeed: the output, and the exit code.
func (h *harness) run2(args ...string) (string, int) {
	h.t.Helper()
	code, out, errOut := h.run(args...)
	if code != 0 {
		h.t.Fatalf("mtqg %s: exit %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, out, errOut)
	}
	return out, code
}

func (h *harness) mustRun(args ...string) string {
	h.t.Helper()
	out, _ := h.run2(args...)
	return out
}

func TestArchiveDryRunChangesNothing(t *testing.T) {
	for _, flag := range []string{"-n", "--dry-run"} {
		t.Run(flag, func(t *testing.T) {
			h := initialized(t)
			archiveFixture(h)
			before := h.readJournal()

			for _, args := range [][]string{{"archive", flag, "2023..2023"}, {"archive", "2023..2023", flag}, {flag, "archive", "2023..2023"}} {
				code, out, errOut := h.run(args...)
				wantExit(t, code, 0, out, errOut)
				want := "Range: 2023-01-01..2023-12-31 (dry run)\n" +
					"Archived: 1 memo, 1 todo, 1 question, 3 answers, 1 bug, 1 reply, 1 glossary entry -> .mtqg/archive/2023-01-01..2023-12-31.jsonl\n" +
					"Skipped: 1 open todo, 1 open bug, 1 glossary entry\n"
				if out != want {
					t.Errorf("%v: stdout:\n%s\nwant:\n%s", args, out, want)
				}
				if got := h.readJournal(); got != before {
					t.Fatalf("%v changed journal.jsonl", args)
				}
				h.noArchiveDir()
			}
		})
	}
}

func TestArchiveThatFindsNothingMakesNoFile(t *testing.T) {
	h := initialized(t)
	archiveFixture(h)
	before := h.readJournal()
	out := h.mustRun("archive", "1999..2000")
	if want := "Range: 1999-01-01..2000-12-31\nArchived: nothing\n"; out != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
	}
	if got := h.readJournal(); got != before {
		t.Error("journal.jsonl was changed")
	}
	h.noArchiveDir()

	// An empty journal, and one that is not there yet.
	h2 := initialized(t)
	if out := h2.mustRun("archive", "2023..2023"); out != "Range: 2023-01-01..2023-12-31\nArchived: nothing\n" {
		t.Errorf("stdout:\n%s", out)
	}
	h2.noArchiveDir()
}

func TestArchiveTheSameRangeAgainAppends(t *testing.T) {
	h := initialized(t)
	archiveFixture(h)
	h.mustRun("archive", "2023..2023")
	first := h.readArchive("2023-01-01..2023-12-31.jsonl")

	// Another item of 2023 turns up (a branch that was merged, say), and the same
	// range is archived again, written another way.
	extra := record(arid(20), "memo", "Turned up late", "yamada", "2023-11-11T09:00:00Z")
	f, err := os.OpenFile(h.journalPath(), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(extra + "\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	out := h.mustRun("archive", "20230101..20231231")
	want := "Range: 2023-01-01..2023-12-31\n" +
		"Archived: 1 memo -> .mtqg/archive/2023-01-01..2023-12-31.jsonl\n" +
		"Skipped: 1 open todo, 1 open bug, 1 glossary entry\n"
	if out != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
	}
	if got := h.readArchive("2023-01-01..2023-12-31.jsonl"); got != first+extra+"\n" {
		t.Errorf("the archive holds\n%s\nwant the first lines and then\n%s", got, extra)
	}
	entries, err := os.ReadDir(h.archiveDir())
	if err != nil || len(entries) != 1 {
		t.Errorf("archive/ holds %v (%v), want one file", entries, err)
	}
}

func TestArchiveNeedsNoAuthor(t *testing.T) {
	h := initialized(t)
	archiveFixture(h)
	git(t, h.root, "config", "--unset", "user.name")
	// Writing an event without an author is refused...
	code, out, errOut := h.run("m", "add", "no author")
	wantExit(t, code, 1, out, errOut)
	// ...but archive writes none.
	h.mustRun("archive", "2023..2023")
	h.mustRun("archive", "-n", "2022..2022")
}

func TestArchiveUsesTheLocalDayOfTheZone(t *testing.T) {
	// 20:00 UTC on New Year's Eve is already New Year's Day in Tokyo.
	line := record(arid(1), "memo", "Near midnight", "yamada", "2023-12-31T20:00:00Z")
	tokyo := time.FixedZone("JST", 9*3600)

	t.Run("in UTC it is 2023", func(t *testing.T) {
		h := initialized(t)
		h.setJournal(line)
		if out := h.mustRun("archive", "2023..2023"); !strings.Contains(out, "Archived: 1 memo") {
			t.Errorf("stdout:\n%s", out)
		}
	})
	t.Run("in Tokyo it is 2024", func(t *testing.T) {
		h := initialized(t)
		h.env.Location = tokyo
		h.setJournal(line)
		if out := h.mustRun("archive", "2023..2023"); !strings.Contains(out, "Archived: nothing") {
			t.Errorf("2023 in Tokyo:\n%s", out)
		}
		if out := h.mustRun("archive", "2024..2024"); !strings.Contains(out, "Archived: 1 memo") {
			t.Errorf("2024 in Tokyo:\n%s", out)
		}
	})
}

func TestArchiveLeavesWhatCannotBeReadWhereItIs(t *testing.T) {
	h := initialized(t)
	handMade := `{ "op" : "create", "id":"` + arid(30) + `", "type":"memo", "text":"unknown field", "ts":"2023-02-02T09:00:00Z", "v":0, "author":{"kind":"human","name":"yamada"}, "priority":"high" }`
	broken := "this is not json"
	noID := `{"op":"create","type":"memo","text":"no id","ts":"2023-02-03T09:00:00Z"}`
	dup := record(arid(31), "memo", "twice", "yamada", "2023-02-04T09:00:00Z")
	h.setJournal(handMade, broken, noID, dup, dup)

	code, out, errOut := h.run("archive", "2023..2023")
	wantExit(t, code, 0, out, errOut)
	if !strings.Contains(out, "Archived: 2 memos") {
		t.Errorf("stdout:\n%s", out)
	}
	// The lines that could not be read are warned about, and stay.
	if got := strings.Count(errOut, "warning"); got != 2 {
		t.Errorf("stderr has %d warnings, want 2:\n%s", got, errOut)
	}
	if got, want := h.readJournal(), broken+"\n"+noID+"\n"; got != want {
		t.Errorf("journal.jsonl holds\n%q\nwant\n%q", got, want)
	}
	// The line that is there twice is one event, and both copies move, as they were.
	if got, want := h.readArchive("2023-01-01..2023-12-31.jsonl"), handMade+"\n"+dup+"\n"+dup+"\n"; got != want {
		t.Errorf("the archive holds\n%q\nwant\n%q", got, want)
	}
}

func TestArchiveRefusesWhileConflictMarkersRemain(t *testing.T) {
	h := initialized(t)
	conflicted := "<<<<<<< HEAD\n" + record(arid(1), "memo", "a", "yamada", "2023-01-01T09:00:00Z") + "\n=======\n" +
		record(arid(2), "memo", "b", "yamada", "2023-01-02T09:00:00Z") + "\n>>>>>>> other\n"
	if err := os.WriteFile(h.journalPath(), []byte(conflicted), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := h.run("archive", "2023..2023")
	wantExit(t, code, 1, out, errOut)
	if got := h.readJournal(); got != conflicted {
		t.Errorf("journal.jsonl was changed: %q", got)
	}
	h.noArchiveDir()

	// A dry run only reads, so it says what it finds, and the warnings come with it.
	code, out, errOut = h.run("archive", "-n", "2023..2023")
	wantExit(t, code, 0, out, errOut)
}

func TestArchiveMistakesInTheCommandLine(t *testing.T) {
	h := initialized(t)
	archiveFixture(h)
	before := h.readJournal()
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"no range", []string{"archive"}, "Missing argument. Usage: mtqg archive <start>..<end> [-n]"},
		{"two ranges", []string{"archive", "2022..2022", "2023..2023"}, "Too many arguments"},
		{"no dots", []string{"archive", "2023"}, `"2023" is not a range`},
		{"backwards", []string{"archive", "2024..2023"}, "starts after it ends"},
		{"a relative date", []string{"archive", "2y..now"}, "only digits"},
		{"a dry run of another command", []string{"todo", "list", "-n"}, "Unknown option --dry-run"},
		{"a range that is not the first argument", []string{"archive", "--frob", "2023..2023"}, "Unknown option --frob"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errOut := h.run(tt.args...)
			wantExit(t, code, 2, out, errOut)
			if out != "" || !strings.Contains(errOut, tt.want) {
				t.Errorf("stdout %q, stderr %q, want %q", out, errOut, tt.want)
			}
			if got := h.readJournal(); got != before {
				t.Error("journal.jsonl was changed")
			}
			h.noArchiveDir()
		})
	}
}

func TestArchiveJSON(t *testing.T) {
	h := initialized(t)
	archiveFixture(h)

	obj := jsonObject(t, h.mustRun("--json", "archive", "-n", "2023..2023"))
	if obj["command"] != "archive" || field(t, obj, "dry_run") != true ||
		field(t, obj, "range", "start") != "2023-01-01" || field(t, obj, "range", "end") != "2023-12-31" ||
		field(t, obj, "file") != ".mtqg/archive/2023-01-01..2023-12-31.jsonl" {
		t.Errorf("%v", obj)
	}
	wantArchived := map[string]float64{"memos": 1, "todos": 1, "questions": 1, "answers": 3, "bugs": 1, "replies": 1, "glossary_entries": 1, "records": 9}
	for k, want := range wantArchived {
		if got := field(t, obj, "archived", k); got != want {
			t.Errorf("archived.%s = %v, want %v", k, got, want)
		}
	}
	wantSkipped := map[string]float64{"open_todos": 1, "open_questions": 0, "open_bugs": 1, "glossary_entries": 1, "records": 3}
	for k, want := range wantSkipped {
		if got := field(t, obj, "skipped", k); got != want {
			t.Errorf("skipped.%s = %v, want %v", k, got, want)
		}
	}
	h.noArchiveDir()

	// The same, done: only dry_run differs.
	obj = jsonObject(t, h.mustRun("--json", "archive", "2023..2023"))
	if field(t, obj, "dry_run") != false || field(t, obj, "archived", "records") != 9.0 {
		t.Errorf("%v", obj)
	}
	if got := linesOf(h.readArchive("2023-01-01..2023-12-31.jsonl")); len(got) != 13 {
		t.Errorf("the archive holds %d lines, want 13", len(got))
	}

	// Nothing moved: the counts are there, and zero.
	obj = jsonObject(t, h.mustRun("--json", "archive", "1999..1999"))
	for _, k := range []string{"memos", "todos", "questions", "answers", "bugs", "replies", "glossary_entries", "records"} {
		if got := field(t, obj, "archived", k); got != 0.0 {
			t.Errorf("archived.%s = %v, want 0", k, got)
		}
	}
	if field(t, obj, "file") != ".mtqg/archive/1999-01-01..1999-12-31.jsonl" {
		t.Errorf("file = %v: it names where the lines would go, though none went", field(t, obj, "file"))
	}

	// A mistake in the range is an error on standard error, as for any command.
	code, out, errOut := h.run("--json", "archive", "2024..2023")
	wantExit(t, code, 2, out, errOut)
	if e := field(t, oneLineOfJSON(t, errOut), "error").(map[string]any); e["kind"] != "usage" || out != "" {
		t.Errorf("error %v, stdout %q", e, out)
	}
}

func TestArchiveIsListedInHelp(t *testing.T) {
	h := newHarness(t)
	out := h.mustRun("help")
	if !strings.Contains(out, "mtqg archive <start>..<end> [-n]") {
		t.Errorf("help does not list archive:\n%s", out)
	}
	if got := h.mustRun("archive", "--help"); !strings.HasPrefix(got, "usage: mtqg archive <start>..<end> [-n]\n") {
		t.Errorf("archive --help:\n%s", got)
	}
}
