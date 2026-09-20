// Package journal reads and writes .mtqg/journal.jsonl and the files around it.
//
// It is the lowest layer of the core (.claude/rules/directory-structure.md). It
// does not know what an event means: to it a line is an event, and a todo being
// done is just another line. Everything that writes to .mtqg/ lives here: the
// rules for how a line is written, the lock, and the rewriting used by undo and
// archive. The layers above never repeat them.
package journal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// DefaultLockTimeout is how long a write waits for the lock before it gives up.
// The lock is held for one append or one rewrite, so a longer wait means that
// some process is stuck while holding it.
const DefaultLockTimeout = 5 * time.Second

// Options are what the journal layer fills into the events it writes and how it
// behaves. The caller of Append gives only the meaning of an event.
type Options struct {
	// Author is written as the author of every event this Journal appends.
	Author Author

	// TTY identifies the terminal the events are written from. Empty when there
	// is none (a server, CI): the field is then left out.
	TTY string

	// LockTimeout is how long to wait for the write lock. Zero means
	// DefaultLockTimeout.
	LockTimeout time.Duration
}

// Journal is an open .mtqg/ directory. It keeps no file open: every operation
// opens what it needs and closes it again, because a short-lived command is the
// normal user and Windows cannot replace a file that is open.
type Journal struct {
	loc     Location
	version int
	opts    Options

	// now returns the time of an event. Tests replace it.
	now func() time.Time
}

// Open finds .mtqg/ from start (see Find) and checks its format version. A
// version newer than SupportedVersion is refused with a *FormatTooNewError, for
// reading as well as for writing.
func Open(start string, opts Options) (*Journal, error) {
	loc, err := Find(start)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(loc.Dir)
	if err != nil {
		return nil, fmt.Errorf("journal: %w", err)
	}
	defer func() { _ = root.Close() }()
	version, err := readVersion(root)
	if err != nil {
		return nil, err
	}
	if version > SupportedVersion {
		return nil, &FormatTooNewError{Found: version, Supported: SupportedVersion}
	}
	if opts.LockTimeout == 0 {
		opts.LockTimeout = DefaultLockTimeout
	}
	return &Journal{loc: loc, version: version, opts: opts, now: time.Now}, nil
}

// Location returns where the .mtqg/ directory is.
func (j *Journal) Location() Location { return j.loc }

// Version returns the format version declared by .mtqg/version.
func (j *Journal) Version() int { return j.version }

func (j *Journal) journalPath() string { return filepath.Join(j.loc.Dir, journalName) }

// openRoot opens .mtqg/ as a root. Everything the journal layer does inside
// .mtqg/ goes through it, and a root refuses to leave the directory: not with
// "..", and not through a symbolic link that leads outside. A repository that
// somebody else made can hold a .mtqg/journal.jsonl that is a link to another
// file, and mtqg is run inside repositories that were just cloned. Without the
// root, it would write to whatever the link points at.
//
// The caller closes the root when it is done; files opened through it stay open
// after that.
func (j *Journal) openRoot() (*os.Root, error) {
	root, err := os.OpenRoot(j.loc.Dir)
	if err != nil {
		return nil, fmt.Errorf("journal: %w", err)
	}
	return root, nil
}

// Read returns the events of journal.jsonl. It does not take the lock: an append
// is one write, and a rewrite replaces the file in one step, so a reader sees
// the file either before or after. A line that is still being written shows up
// as a warning.
//
// A missing journal.jsonl reads as an empty journal. Lines that cannot be read
// are skipped with a warning; conflict markers are among them. Reading never
// fails because of them.
func (j *Journal) Read() (Result, error) {
	lines, warnings, err := j.readLines()
	if err != nil {
		return Result{}, err
	}
	return newResult(lines, warnings), nil
}

// readLines returns every non-blank line of journal.jsonl. The file is closed
// before it returns.
func (j *Journal) readLines() ([]Line, []Warning, error) {
	root, err := j.openRoot()
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = root.Close() }()
	f, err := root.Open(journalName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("journal: %w", err)
	}
	// Nothing was written to f, so a failure to close it changes nothing.
	defer func() { _ = f.Close() }()
	lines, warnings, err := Scan(f)
	if err != nil {
		return nil, nil, fmt.Errorf("journal: read %s: %w", journalName, err)
	}
	return lines, warnings, nil
}
