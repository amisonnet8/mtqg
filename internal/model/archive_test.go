package model

import (
	"strings"
	"testing"
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// on is a ts on a day of 2026 (month, day) at noon UTC.
func on(month, day int) string {
	return time.Date(2026, time.Month(month), day, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

// The range of most of the tests: from the start of April up to the end of June.
var (
	rangeFrom = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	rangeTo   = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
)

func made(id, typ, text, ts string) journal.Event {
	ev := create(id, typ, text, 0)
	ev.TS = ts
	return ev
}

func madeWord(id, word, text, ts string) journal.Event {
	ev := made(id, journal.TypeGlossary, text, ts)
	ev.Word = word
	return ev
}

func madeReply(id, typ, re, text, ts string) journal.Event {
	ev := made(id, typ, text, ts)
	ev.Re = re
	ev.Status = ""
	return ev
}

func changed(id, from, to, ts string) journal.Event {
	ev := status(id, from, to, 0, human)
	ev.TS = ts
	return ev
}

func deleted(id, ts string) journal.Event {
	return journal.Event{ID: id, Op: journal.OpDelete, TS: ts, Author: human}
}

func idList(records []*Record) string {
	var out []string
	for _, r := range records {
		out = append(out, r.ID)
	}
	return strings.Join(out, " ")
}

// Ids for the records of the tests.
const (
	i1 = "10000000000000000000000000000001"
	i2 = "10000000000000000000000000000002"
	i3 = "10000000000000000000000000000003"
	i4 = "10000000000000000000000000000004"
	i5 = "10000000000000000000000000000005"
	i6 = "10000000000000000000000000000006"
	i7 = "10000000000000000000000000000007"
	i8 = "10000000000000000000000000000008"
)

func TestArchiveTargetsByKind(t *testing.T) {
	events := []journal.Event{
		made(i1, journal.TypeTodo, "done todo", on(4, 1)), changed(i1, "open", "done", on(5, 1)),
		made(i2, journal.TypeTodo, "open todo", on(4, 2)),
		made(i3, journal.TypeMemo, "a memo", on(4, 3)),
		made(i4, journal.TypeQA, "closed question", on(4, 4)), changed(i4, "open", "done", on(5, 2)),
		made(i5, journal.TypeBug, "closed bug", on(4, 5)), changed(i5, "open", "done", on(5, 3)),
		madeWord(i6, "term", "a definition", on(4, 6)),
		made(i7, journal.TypeQA, "open question", on(4, 7)),
		made(i8, journal.TypeBug, "open bug", on(4, 8)),
	}
	plan := Build(events).ArchiveTargets(rangeFrom, rangeTo)

	if got, want := idList(plan.Move), strings.Join([]string{i1, i3, i4, i5}, " "); got != want {
		t.Errorf("Move = %s, want %s", got, want)
	}
	if got, want := idList(plan.Skipped), strings.Join([]string{i2, i6, i7, i8}, " "); got != want {
		t.Errorf("Skipped = %s, want %s", got, want)
	}
}

func TestArchiveTargetsAreDecidedByTheLastEvent(t *testing.T) {
	// Done in the range, but reopened after it: the last event is outside.
	events := []journal.Event{
		made(i1, journal.TypeTodo, "reopened later", on(4, 1)),
		changed(i1, "open", "done", on(5, 1)),
		changed(i1, "done", "open", on(8, 1)),
		// Written before the range and closed in it.
		made(i2, journal.TypeTodo, "closed in the range", on(1, 1)),
		changed(i2, "open", "done", on(5, 1)),
		// Closed before the range.
		made(i3, journal.TypeTodo, "closed before", on(1, 1)),
		changed(i3, "open", "done", on(2, 1)),
	}
	plan := Build(events).ArchiveTargets(rangeFrom, rangeTo)
	if got := idList(plan.Move); got != i2 {
		t.Errorf("Move = %s, want only %s", got, i2)
	}
	if len(plan.Skipped) != 0 {
		t.Errorf("Skipped = %s: a record whose last event is outside the range is not reported", idList(plan.Skipped))
	}
}

func TestArchiveTargetsIncludeTheFirstInstantAndExcludeTheEnd(t *testing.T) {
	mk := func(id string, at time.Time) journal.Event {
		return made(id, journal.TypeMemo, "m", at.Format(time.RFC3339))
	}
	plan := Build([]journal.Event{
		mk(i1, rangeFrom.Add(-time.Second)), // just before the range
		mk(i2, rangeFrom),                   // the first instant of it
		mk(i3, rangeTo.Add(-time.Second)),   // the last second of it
		mk(i4, rangeTo),                     // the first instant after it
	}).ArchiveTargets(rangeFrom, rangeTo)
	if got, want := idList(plan.Move), i2+" "+i3; got != want {
		t.Errorf("Move = %s, want %s", got, want)
	}
}

func TestArchiveTargetsTakeInTheAnswersOfAQuestion(t *testing.T) {
	// The question was closed in May, and answered in the range and out of it.
	build := func(answerAt string) *State {
		return Build([]journal.Event{
			made(i1, journal.TypeQA, "question", on(4, 1)),
			madeReply(i2, journal.TypeQA, i1, "answer", answerAt),
			changed(i1, "open", "done", on(5, 1)),
		})
	}

	t.Run("an answer inside the range moves with the question", func(t *testing.T) {
		plan := build(on(5, 2)).ArchiveTargets(rangeFrom, rangeTo)
		if got, want := idList(plan.Move), i1+" "+i2; got != want {
			t.Errorf("Move = %s, want %s", got, want)
		}
	})

	t.Run("an answer after the range keeps the whole thread", func(t *testing.T) {
		// The last event of the thread is the answer, after the range. Moving the
		// question and its answer would put a line of September in the archive for
		// the spring, and moving only the question would take it from its answer.
		plan := build(on(9, 1)).ArchiveTargets(rangeFrom, rangeTo)
		if len(plan.Move) != 0 || len(plan.Skipped) != 0 {
			t.Errorf("Move = %s, Skipped = %s, want nothing", idList(plan.Move), idList(plan.Skipped))
		}
	})

	t.Run("a range that holds the answer takes the question that was closed before it", func(t *testing.T) {
		later := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
		plan := build(on(9, 1)).ArchiveTargets(rangeTo, later)
		if got, want := idList(plan.Move), i1+" "+i2; got != want {
			t.Errorf("Move = %s, want %s", got, want)
		}
	})

	t.Run("an answer that is deleted still counts for the date", func(t *testing.T) {
		state := Build([]journal.Event{
			made(i1, journal.TypeQA, "question", on(4, 1)),
			madeReply(i2, journal.TypeQA, i1, "answer", on(4, 2)),
			deleted(i2, on(9, 1)),
			changed(i1, "open", "done", on(5, 1)),
		})
		if plan := state.ArchiveTargets(rangeFrom, rangeTo); len(plan.Move) != 0 {
			t.Errorf("Move = %s: the delete of the answer is after the range", idList(plan.Move))
		}
		later := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
		if plan := state.ArchiveTargets(rangeTo, later); idList(plan.Move) != i1+" "+i2 {
			t.Errorf("Move = %s, want the question and its deleted answer", idList(plan.Move))
		}
	})
}

func TestArchiveTargetsTakeInTheRepliesOfABug(t *testing.T) {
	plan := Build([]journal.Event{
		made(i1, journal.TypeBug, "bug", on(4, 1)),
		madeReply(i2, journal.TypeBug, i1, "reply", on(4, 2)),
		madeReply(i3, journal.TypeBug, i1, "another reply", on(4, 3)),
		changed(i1, "open", "done", on(5, 1)),
	}).ArchiveTargets(rangeFrom, rangeTo)
	if got, want := idList(plan.Move), i1+" "+i2+" "+i3; got != want {
		t.Errorf("Move = %s, want %s", got, want)
	}
}

func TestArchiveTargetsLeaveTheAnswersOfAQuestionThatStays(t *testing.T) {
	plan := Build([]journal.Event{
		made(i1, journal.TypeQA, "still open", on(4, 1)),
		madeReply(i2, journal.TypeQA, i1, "an answer", on(4, 2)),
	}).ArchiveTargets(rangeFrom, rangeTo)
	if len(plan.Move) != 0 {
		t.Errorf("Move = %s: an answer does not leave its question", idList(plan.Move))
	}
	// The question is counted, and the answer is not counted apart from it.
	if got := idList(plan.Skipped); got != i1 {
		t.Errorf("Skipped = %s, want only the question", got)
	}
}

func TestArchiveTargetsMoveWhatWasDeletedWhateverItIs(t *testing.T) {
	events := []journal.Event{
		made(i1, journal.TypeTodo, "open todo, deleted", on(4, 1)), deleted(i1, on(5, 1)),
		madeWord(i2, "term", "definition, deleted", on(4, 2)), deleted(i2, on(5, 2)),
		made(i3, journal.TypeQA, "open question, deleted", on(4, 3)),
		madeReply(i4, journal.TypeQA, i3, "its answer", on(4, 4)), deleted(i3, on(5, 3)),
		made(i5, journal.TypeTodo, "open todo, deleted after the range", on(4, 5)), deleted(i5, on(9, 1)),
	}
	plan := Build(events).ArchiveTargets(rangeFrom, rangeTo)
	if got, want := idList(plan.Move), strings.Join([]string{i1, i2, i3, i4}, " "); got != want {
		t.Errorf("Move = %s, want %s", got, want)
	}
	if len(plan.Skipped) != 0 {
		t.Errorf("Skipped = %s, want none: a deleted record is not skipped", idList(plan.Skipped))
	}
}

func TestArchiveTargetsTreatAnAnswerWithNoQuestionAsAMemo(t *testing.T) {
	plan := Build([]journal.Event{
		madeReply(i1, journal.TypeQA, "ffffffffffffffffffffffffffffffff", "answer to nothing", on(4, 1)),
		made(i2, journal.TypeMemo, "a memo", on(4, 2)),
		madeReply(i3, journal.TypeBug, i2, "a reply that names a memo", on(4, 3)),
		madeReply(i4, journal.TypeQA, i5, "answer to a bug", on(4, 4)),
		made(i5, journal.TypeBug, "a bug", on(4, 5)),
	}).ArchiveTargets(rangeFrom, rangeTo)
	if got, want := idList(plan.Move), strings.Join([]string{i1, i2, i3, i4}, " "); got != want {
		t.Errorf("Move = %s, want %s", got, want)
	}
	if got := idList(plan.Skipped); got != i5 {
		t.Errorf("Skipped = %s, want the open bug", got)
	}
}

func TestArchiveTargetsLeaveWhatHasNoTimeToJudge(t *testing.T) {
	events := []journal.Event{
		made(i1, journal.TypeMemo, "no time", "not a time"),
		made(i2, journal.TypeMemo, "no time at all", ""),
		made(i3, journal.TypeMemo, "a time", on(5, 1)),
	}
	state := Build(events)
	if got := idList(state.ArchiveTargets(rangeFrom, rangeTo).Move); got != i3 {
		t.Errorf("Move = %s, want only %s", got, i3)
	}
	// Not even a range that reaches back to the zero time takes what has no time.
	if got := idList(state.ArchiveTargets(time.Time{}, rangeTo).Move); got != i3 {
		t.Errorf("with no start, Move = %s, want only %s", got, i3)
	}
}

func TestArchiveTargetsIgnoreEventsWithNoRecord(t *testing.T) {
	// A change that came back from an old branch for a record that was archived.
	plan := Build([]journal.Event{
		changed(i1, "open", "done", on(5, 1)),
		made(i2, journal.TypeMemo, "a memo", on(5, 2)),
	}).ArchiveTargets(rangeFrom, rangeTo)
	if got := idList(plan.Move); got != i2 {
		t.Errorf("Move = %s, want only %s", got, i2)
	}
}

func TestArchiveTargetsAreOldestFirst(t *testing.T) {
	// Given newest first: the records are put in order of their time.
	plan := Build([]journal.Event{
		made(i3, journal.TypeMemo, "third", on(4, 3)),
		made(i1, journal.TypeMemo, "first", on(4, 1)),
		made(i2, journal.TypeMemo, "second", on(4, 2)),
	}).ArchiveTargets(rangeFrom, rangeTo)
	if got, want := idList(plan.Move), i1+" "+i2+" "+i3; got != want {
		t.Errorf("Move = %s, want %s", got, want)
	}
}

func TestArchiveTargetsOfAnEmptyJournal(t *testing.T) {
	plan := Build(nil).ArchiveTargets(rangeFrom, rangeTo)
	if plan.Move != nil || plan.Skipped != nil {
		t.Errorf("plan = %+v, want nothing", plan)
	}
}
