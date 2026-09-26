package journal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// gitTimeout is how long a call to git may take. git only reads here, so a
// longer wait means something is wrong (a lock, a slow file system).
const gitTimeout = 10 * time.Second

// The journal layer reads what git says: who the user is, what the last commit
// holds of journal.jsonl, which branch is checked out, the commit HEAD points
// to, and a digest of the working tree (for the agent hooks, §11.3). It asks
// the real git command, so that git's own configuration, worktrees and
// submodules are honored, and it never writes anything to git
// (.claude/rules/git-integration.md).

// What the journal layer asks git. Each is a fixed command line: the only thing
// that comes from outside is the directory git runs in (cmd.Dir), so no
// argument is built from a path or a name.
type gitQuery int

const (
	queryUserName gitQuery = iota
	queryCommittedJournal
	queryBranch
	queryHead
	queryShortHead
	queryStatus
)

// committedJournal names journal.jsonl in the last commit. It starts with "./"
// so that the path is relative to the directory git runs in, not to the top of
// the repository.
const committedJournal = "HEAD:./" + mtqgDirName + "/" + journalName

// runGit runs a query in root and returns what it printed and the exit code. A
// git that ran and failed is not an error here: the caller reads the exit code.
// An error means git could not be run at all.
func runGit(root string, query gitQuery) (stdout []byte, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	var cmd *exec.Cmd
	switch query {
	case queryUserName:
		cmd = exec.CommandContext(ctx, "git", "--no-pager", "config", "user.name")
	case queryCommittedJournal:
		cmd = exec.CommandContext(ctx, "git", "--no-pager", "show", committedJournal)
	case queryBranch:
		cmd = exec.CommandContext(ctx, "git", "--no-pager", "branch", "--show-current")
	case queryHead:
		cmd = exec.CommandContext(ctx, "git", "--no-pager", "rev-parse", "HEAD")
	case queryShortHead:
		cmd = exec.CommandContext(ctx, "git", "--no-pager", "rev-parse", "--short", "HEAD")
	case queryStatus:
		cmd = exec.CommandContext(ctx, "git", "--no-pager", "status", "--porcelain", "-z")
	default:
		return nil, 0, &GitUnavailableError{Err: errors.New("unknown git query")}
	}
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return out.Bytes(), exit.ExitCode(), nil
		}
		return nil, 0, &GitUnavailableError{Err: err}
	}
	return out.Bytes(), 0, nil
}

// GitUserName returns user.name as git sees it in the repository at root, so a
// setting of that repository applies. It returns ErrNoUserName when there is
// none, and an error that is ErrGitUnavailable when git cannot be run.
func GitUserName(root string) (string, error) {
	out, code, err := runGit(root, queryUserName)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(out))
	if code != 0 || name == "" {
		return "", ErrNoUserName
	}
	return name, nil
}

// GitBranch returns the name of the branch that is checked out in the repository
// at root, or "" when there is none (a detached HEAD). A branch that has no commit
// yet has a name. If git cannot be run, the error is ErrGitUnavailable.
func GitBranch(root string) (string, error) {
	out, code, err := runGit(root, queryBranch)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// GitHead returns the commit HEAD points to in the repository at root, or ""
// when there is no commit yet (a fresh repository, or a detached HEAD that
// somehow fails). It is used to notice that a commit was made while an agent
// session was open (§11.3). If git cannot be run, the error is
// ErrGitUnavailable.
func GitHead(root string) (string, error) {
	out, code, err := runGit(root, queryHead)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// GitShortHead returns the abbreviated commit hash HEAD points to in the
// repository at root, the way "git show" would print it (following
// core.abbrev), or "" when there is no commit yet. It is used to record which
// commit was checked out when a record was written (§5.5, the "at" field). If
// git cannot be run, the error is ErrGitUnavailable.
func GitShortHead(root string) (string, error) {
	out, code, err := runGit(root, queryShortHead)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// GitStatusDigest is a hash of what "git status --porcelain -z" reports for the
// repository at root, leaving out .mtqg/ itself. It says only whether the
// working tree looks different from one call to the next, not what changed:
// it is used to notice that a session did some work (§11.3), not to read the
// status. If git cannot be run, the error is ErrGitUnavailable.
func GitStatusDigest(root string) (string, error) {
	out, code, err := runGit(root, queryStatus)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", nil
	}
	sum := sha256.Sum256(statusWithoutMtqgDir(out))
	return fmt.Sprintf("%x", sum), nil
}

// statusWithoutMtqgDir drops the entries of a "git status --porcelain -z"
// output that mention .mtqg/, so that mtqg's own writes (the journal, a
// session file) never look like "the working tree changed". It matches by
// whether an entry contains the directory name at all, which also catches the
// second, path-only token of a rename entry.
func statusWithoutMtqgDir(raw []byte) []byte {
	marker := []byte(mtqgDirName + "/")
	var kept [][]byte
	for _, entry := range bytes.Split(raw, []byte{0}) {
		if len(entry) == 0 || bytes.Contains(entry, marker) {
			continue
		}
		kept = append(kept, entry)
	}
	return bytes.Join(kept, []byte{0})
}

// UncommittedEvents returns the events of journal.jsonl that are not in the last
// commit (HEAD): the lines that were added since, whether they are staged or
// not. Lines that only the last commit has (removed since, by undo or archive)
// are not events of the present journal and are not returned. Lines whose
// content is exactly the same are one event.
//
// When there is no commit yet, or the last commit has no journal.jsonl, every
// event is uncommitted. If git cannot be run, the error is ErrGitUnavailable.
func (j *Journal) UncommittedEvents() ([]Event, error) {
	lines, _, err := j.readLines()
	if err != nil {
		return nil, err
	}
	committed, err := j.committedLines()
	if err != nil {
		return nil, err
	}

	var events []Event
	seen := make(map[string]struct{})
	for _, line := range lines {
		if line.Event == nil {
			continue
		}
		key := string(line.Raw)
		if _, ok := committed[key]; ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		events = append(events, *line.Event)
	}
	return events, nil
}

// committedLines returns the lines of journal.jsonl in the last commit, as a
// set.
func (j *Journal) committedLines() (map[string]struct{}, error) {
	out, code, err := runGit(j.loc.Root, queryCommittedJournal)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	if code != 0 {
		// No commit yet, or the commit has no such file: nothing is committed.
		return set, nil
	}
	lines, _, _ := Scan(bytes.NewReader(out))
	for _, line := range lines {
		set[string(line.Raw)] = struct{}{}
	}
	return set, nil
}
