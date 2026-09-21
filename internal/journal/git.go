package journal

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// gitTimeout is how long a call to git may take. git only reads here, so a
// longer wait means something is wrong (a lock, a slow file system).
const gitTimeout = 10 * time.Second

// The journal layer reads two things from git: who the user is, and what the
// last commit holds of journal.jsonl. It asks the real git command, so that
// git's own configuration, worktrees and submodules are honored, and it never
// writes anything to git (.claude/rules/git-integration.md).

// What the journal layer asks git. Each is a fixed command line: the only thing
// that comes from outside is the directory git runs in (cmd.Dir), so no
// argument is built from a path or a name.
type gitQuery int

const (
	queryUserName gitQuery = iota
	queryCommittedJournal
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
