package model

import (
	"testing"
	"time"

	"github.com/amisonnet8/mtqg/internal/journal"
)

func TestShouldPrompt(t *testing.T) {
	start := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	repoA := RepoState{Head: "a", StatusDigest: "x"}
	repoB := RepoState{Head: "b", StatusDigest: "x"} // HEAD changed
	repoC := RepoState{Head: "a", StatusDigest: "y"} // working tree changed

	eventAt := func(ts string) journal.Event { return journal.Event{TS: ts} }

	tests := []struct {
		name     string
		prompted bool
		start    RepoState
		now      RepoState
		events   []journal.Event
		want     bool
	}{
		{name: "nothing changed", start: repoA, now: repoA, want: false},
		{name: "HEAD changed, no records", start: repoA, now: repoB, want: true},
		{name: "working tree changed, no records", start: repoA, now: repoC, want: true},
		{name: "already prompted", prompted: true, start: repoA, now: repoB, want: false},
		{
			name: "a record after the session started", start: repoA, now: repoB,
			events: []journal.Event{eventAt("2026-09-25T12:00:05Z")}, want: false,
		},
		{
			name:  "a record in the same second the session started (ts has only second precision)",
			start: repoA, now: repoB,
			events: []journal.Event{eventAt("2026-09-25T12:00:00Z")}, want: false,
		},
		{
			name: "only a record before the session started", start: repoA, now: repoB,
			events: []journal.Event{eventAt("2026-09-25T11:59:00Z")}, want: true,
		},
		{
			name: "an old record, then a new one", start: repoA, now: repoB,
			events: []journal.Event{eventAt("2026-09-25T11:59:00Z"), eventAt("2026-09-25T12:01:00Z")}, want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startedAt := start.Add(15 * time.Millisecond) // a session's clock is not exactly on the second
			got := ShouldPrompt(tt.prompted, startedAt, tt.start, tt.now, tt.events)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
