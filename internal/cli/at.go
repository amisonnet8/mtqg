package cli

import (
	"strconv"
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// parseAt reads the value of --at: <path>, or <path>:<line>. The split is at
// the last colon, so a path that itself contains one (rare, but a colon is a
// legal path character) is not cut in the middle unless it also ends in
// digits. A line, when given, must be 1 or more.
func parseAt(value string) (*journal.At, error) {
	path := value
	if i := strings.LastIndexByte(value, ':'); i >= 0 {
		tail := value[i+1:]
		if tail != "" && isAllDigits(tail) {
			path = value[:i]
			line, err := strconv.Atoi(tail)
			if err != nil || line < 1 {
				return nil, &usageError{msgBadAt(value)}
			}
			if path == "" {
				return nil, &usageError{msgBadAt(value)}
			}
			return &journal.At{Path: path, Line: line}, nil
		}
	}
	if path == "" {
		return nil, &usageError{msgBadAt(value)}
	}
	return &journal.At{Path: path}, nil
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// atOption reads --at from the command line, or returns nil, nil when it was
// not given.
func (c *ctx) atOption() (*journal.At, error) {
	v, ok := c.inv.values["--at"]
	if !ok {
		return nil, nil
	}
	return parseAt(v)
}

// withHead fills in at.Head from the commit HEAD points to in root, if at is
// not nil. Git being unavailable, or there being no commit yet, is not an
// error here: the record is written either way, just without head (§5.5).
func withHead(at *journal.At, root string) *journal.At {
	if at == nil {
		return nil
	}
	head, _ := journal.GitShortHead(root)
	at.Head = head
	return at
}
