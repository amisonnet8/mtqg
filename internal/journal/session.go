package journal

import (
	"crypto/sha256"
	jsonv2 "encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// sessionsDirName is where an agent hook's session files live, inside .local/
// (never committed, may be lost at any time - journal-format.md "ローカルの
// 状態"). Losing one only means a Stop hook may prompt again, or a
// SessionStart re-reads git state it already had: nothing that a lost record
// or a lost lock would mean.
const sessionsDirName = "sessions"

// SessionMaxAge bounds how long a session file is kept. SessionStart prunes
// files older than this every time it runs, so .local/sessions/ does not grow
// forever across many sessions.
const SessionMaxAge = 7 * 24 * time.Hour

// sessionIDPattern is what a session ID may look like to be used as a file name
// directly: ASCII letters, digits, - and _, not too long. Anything else is
// hashed instead, so a session ID that an agent makes up never decides a path
// (directory-structure.md "os.OpenRoot").
var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// Session is what a hook remembers about one agent session between its
// SessionStart and its Stop: what the repository looked like when the session
// began, and whether the agent has already been prompted once
// (docs/design/07-integrations.md §11.3).
type Session struct {
	StartedAt    time.Time `json:"started_at"`
	Head         string    `json:"head,omitzero"`
	StatusDigest string    `json:"status_digest,omitzero"`
	Prompted     bool      `json:"prompted,omitzero"`
}

func sessionFileName(id string) string {
	name := id
	if !sessionIDPattern.MatchString(name) {
		sum := sha256.Sum256([]byte(id))
		name = fmt.Sprintf("%x", sum)[:16]
	}
	return name + ".json"
}

// ReadSession returns what is remembered of a session, and whether there was
// anything. A missing file, one that cannot be read, or one that does not
// parse is "nothing" (false), never an error: a hook must not fail because its
// own bookkeeping went missing (.local/ may be deleted at any time).
func (j *Journal) ReadSession(id string) (Session, bool) {
	root, err := j.openRoot()
	if err != nil {
		return Session{}, false
	}
	defer func() { _ = root.Close() }()
	data, err := root.ReadFile(filepath.Join(localName, sessionsDirName, sessionFileName(id)))
	if err != nil {
		return Session{}, false
	}
	var s Session
	if err := jsonv2.Unmarshal(data, &s); err != nil {
		return Session{}, false
	}
	return s, true
}

// WriteSession saves what is remembered of a session, replacing what was there.
// The write goes to a temporary file first and is put in place with the same
// replace that Rewrite uses, so a reader never sees a half-written file; a
// session file lost to a crash just means the session is treated as new.
func (j *Journal) WriteSession(id string, s Session) error {
	root, err := j.openRoot()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	dir := filepath.Join(localName, sessionsDirName)
	if err := root.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("journal: %w", err)
	}
	data, err := jsonv2.Marshal(&s)
	if err != nil {
		return fmt.Errorf("journal: %w", err)
	}
	data = append(data, '\n')

	name := filepath.Join(dir, sessionFileName(id))
	tmp := name + ".tmp"
	_ = root.Remove(tmp) // a leftover from a write that crashed earlier

	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("journal: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = root.Remove(tmp)
		return fmt.Errorf("journal: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("journal: %w", err)
	}
	if err := replaceFile(root, tmp, name); err != nil {
		_ = root.Remove(tmp)
		return fmt.Errorf("journal: %w", err)
	}
	return nil
}

// PruneSessions removes session files whose last write is older than
// SessionMaxAge. It is best effort: a session file it cannot read or remove is
// left alone, and there is nothing to report (the same as everything else in
// .local/ - journal-format.md "いつ消えても困らないものだけを置く").
func (j *Journal) PruneSessions(now time.Time) {
	root, err := j.openRoot()
	if err != nil {
		return
	}
	defer func() { _ = root.Close() }()

	dir := filepath.Join(localName, sessionsDirName)
	f, err := root.Open(dir)
	if err != nil {
		return
	}
	entries, err := f.ReadDir(-1)
	_ = f.Close()
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > SessionMaxAge {
			_ = root.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
