package journal

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// The speed of writing a note down comes first (CLAUDE.md), and every append
// reads the whole journal to look for conflict markers. These benchmarks show
// what that costs on journals of a size a project may reach.
var benchSizes = []int{10_000, 100_000}

// benchJournal creates a journal of n lines. A line is about 250 bytes, the
// size of a short note with its metadata.
func benchJournal(b *testing.B, n int) *Journal {
	b.Helper()
	root := newRepo(b)
	var sb strings.Builder
	for i := range n {
		line, err := encodeLine(Event{
			ID: fmt.Sprintf("%032x", i+1), Op: OpCreate, Type: TypeMemo,
			Text: fmt.Sprintf("Note %d: the parser now skips // comments at the end of a line and keeps the position", i),
			V:    0, TS: "2026-09-17T00:00:00Z", Author: yamada, TTY: "3e9a0b12",
		})
		if err != nil {
			b.Fatal(err)
		}
		sb.Write(line)
	}
	newMtqg(b, root, "0\n", str(sb.String()))
	j, err := Open(root, Options{Author: yamada, TTY: "3e9a0b12", LockTimeout: 30 * time.Second})
	if err != nil {
		b.Fatal(err)
	}
	return j
}

func BenchmarkAppend(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("%d lines", n), func(b *testing.B) {
			j := benchJournal(b, n)
			b.ResetTimer()
			for range b.N {
				if _, err := j.Append(Event{Op: OpCreate, Type: TypeMemo, Text: "a new note"}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkRead(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("%d lines", n), func(b *testing.B) {
			j := benchJournal(b, n)
			b.ResetTimer()
			for range b.N {
				result, err := j.Read()
				if err != nil {
					b.Fatal(err)
				}
				if len(result.Events) != n {
					b.Fatalf("got %d events, want %d", len(result.Events), n)
				}
			}
		})
	}
}

func BenchmarkRewrite(b *testing.B) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("%d lines", n), func(b *testing.B) {
			j := benchJournal(b, n)
			b.ResetTimer()
			for range b.N {
				if err := j.Rewrite(keepAll); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
