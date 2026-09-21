package journal

import (
	"errors"
	"fmt"
	"time"
)

// The journal layer returns error kinds, not sentences. Callers test them with
// errors.Is and errors.As; the wording shown to people belongs to the CLI layer
// (.claude/rules/cli-output.md). The Error methods are for logs and tests.
var (
	// ErrNotInRepository means the search reached the top of the file system
	// without meeting a .mtqg/ directory or a .git entry.
	ErrNotInRepository = errors.New("journal: not inside a git repository")

	// ErrConflictMarkers means journal.jsonl still holds unresolved merge
	// conflict markers. Reading warns; writing refuses.
	ErrConflictMarkers = errors.New("journal: journal.jsonl has unresolved conflict markers")

	// ErrLockTimeout means the write lock could not be taken in time.
	ErrLockTimeout = errors.New("journal: timed out waiting for the write lock")

	// ErrInvalidEvent means an event cannot be written as a valid line.
	ErrInvalidEvent = errors.New("journal: invalid event")

	// ErrInvalidVersionFile means .mtqg/version is missing or unreadable.
	ErrInvalidVersionFile = errors.New("journal: invalid .mtqg/version")
)

// NotInitializedError means the search met the repository root (.git) before it
// met a .mtqg/ directory. Root is where mtqg init would create .mtqg/.
type NotInitializedError struct {
	Root string
}

func (e *NotInitializedError) Error() string {
	return fmt.Sprintf("journal: no .mtqg/ found; the repository root is %s", e.Root)
}

// FormatTooNewError means .mtqg/version declares a format newer than this
// build understands. Reading and writing are both refused.
type FormatTooNewError struct {
	Found     int
	Supported int
}

func (e *FormatTooNewError) Error() string {
	return fmt.Sprintf("journal: format version %d is newer than the supported version %d", e.Found, e.Supported)
}

// InvalidEventError names the field that made an event unwritable.
type InvalidEventError struct {
	Field  string
	Reason string
}

func (e *InvalidEventError) Error() string {
	return fmt.Sprintf("journal: invalid event: %s: %s", e.Field, e.Reason)
}

// Is makes errors.Is(err, ErrInvalidEvent) true.
func (e *InvalidEventError) Is(target error) bool { return target == ErrInvalidEvent }

// VersionFileError says why .mtqg/version could not be used.
type VersionFileError struct {
	Path   string
	Reason string
}

func (e *VersionFileError) Error() string {
	return fmt.Sprintf("journal: invalid version file %s: %s", e.Path, e.Reason)
}

// Is makes errors.Is(err, ErrInvalidVersionFile) true.
func (e *VersionFileError) Is(target error) bool { return target == ErrInvalidVersionFile }

// LockTimeoutError carries how long the lock was awaited.
type LockTimeoutError struct {
	Path   string
	Waited time.Duration
}

func (e *LockTimeoutError) Error() string {
	return fmt.Sprintf("journal: could not lock %s within %s", e.Path, e.Waited)
}

// Is makes errors.Is(err, ErrLockTimeout) true.
func (e *LockTimeoutError) Is(target error) bool { return target == ErrLockTimeout }

// ErrAlreadyInitialized means Init found a .mtqg/ where it was to make one.
var ErrAlreadyInitialized = errors.New("journal: .mtqg/ already exists")

// AlreadyInitializedError names the .mtqg/ that Init found.
type AlreadyInitializedError struct {
	Path string
}

func (e *AlreadyInitializedError) Error() string {
	return fmt.Sprintf("journal: .mtqg/ already exists: %s", e.Path)
}

// Is makes errors.Is(err, ErrAlreadyInitialized) true.
func (e *AlreadyInitializedError) Is(target error) bool { return target == ErrAlreadyInitialized }

// ErrNoUserName means git has no user.name for the repository.
var ErrNoUserName = errors.New("journal: git has no user.name")

// ErrGitUnavailable means the git command could not be run (it is not
// installed, or it did not answer in time).
var ErrGitUnavailable = errors.New("journal: git cannot be run")

// GitUnavailableError says why git could not be run.
type GitUnavailableError struct {
	Err error
}

func (e *GitUnavailableError) Error() string {
	return fmt.Sprintf("journal: git cannot be run: %v", e.Err)
}

func (e *GitUnavailableError) Unwrap() error { return e.Err }

// Is makes errors.Is(err, ErrGitUnavailable) true.
func (e *GitUnavailableError) Is(target error) bool { return target == ErrGitUnavailable }
