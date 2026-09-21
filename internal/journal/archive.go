package journal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// archiveDirName is the directory of .mtqg/ that archive files are in
// (docs/reference/schema.md).
const archiveDirName = "archive"

// ArchivePath names an archive file the way a person is told where it is: relative
// to the repository, with slashes.
func ArchivePath(name string) string {
	return mtqgDirName + "/" + archiveDirName + "/" + name
}

// Archive moves lines of journal.jsonl to the archive file .mtqg/archive/<name>.
// Together with undo (Rewrite) it is the only operation that removes or moves
// lines.
//
// Under the write lock, fn is given every non-blank line of journal.jsonl and says
// which of them to move and which to keep. The lines are written as they are, byte
// for byte, so unknown fields, lines that could not be read and hand-made
// formatting survive on either side.
//
// The lines to move are first appended to the archive file and synced, and only
// then is journal.jsonl replaced by the lines to keep. A crash in between leaves the
// moved lines in both files, which reads as they were before (a line that is in a
// file twice is one event) and is put right by archiving again. The other order
// would lose them. The archive file is made when it is needed; one that is there
// already is appended to, and a line ending is added first if its last line has
// none (as Append does).
//
// Nothing is written if fn returns an error, if it moves nothing (the archive
// directory is not made either), or if the lines it moves and keeps are not
// together the lines it was given: moving is the one way a line can be lost, so a
// miscount is refused here and not left to the caller. Archiving is refused while
// journal.jsonl holds conflict markers. fn must not call Append or Rewrite: the
// lock is not re-entrant.
func (j *Journal) Archive(name string, fn func(lines []Line) (move, keep []Line, err error)) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("journal: %q is not the name of an archive file", name)
	}

	release, err := j.lock()
	if err != nil {
		return err
	}
	defer release()

	root, err := j.openRoot()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	lines, exists, err := readLocked(root)
	if err != nil {
		return err
	}
	move, keep, err := fn(lines)
	if err != nil {
		return err
	}
	if len(move)+len(keep) != len(lines) {
		return fmt.Errorf("journal: archive: %d lines to move and %d to keep are not the %d lines of %s",
			len(move), len(keep), len(lines), journalName)
	}
	if len(move) == 0 {
		return nil
	}
	// Both are rendered before anything is written, so that a line that cannot be
	// written stops the archive while nothing has changed.
	moved, err := renderLines(move)
	if err != nil {
		return err
	}
	kept, err := renderLines(keep)
	if err != nil {
		return err
	}

	if err := appendArchive(root, name, moved); err != nil {
		return err
	}
	return replaceJournal(root, kept, exists)
}

// appendArchive appends data to the archive file and syncs it before it returns.
// A file made here gets the permissions of journal.jsonl: it is shared through git
// like it.
func appendArchive(root *os.Root, name string, data []byte) error {
	path := archiveDirName + "/" + name
	fail := func(err error) error { return fmt.Errorf("journal: write %s: %w", path, err) }

	if err := root.MkdirAll(archiveDirName, 0o750); err != nil {
		return fail(err)
	}
	existing, err := root.ReadFile(path)
	created := errors.Is(err, fs.ErrNotExist)
	if err != nil && !created {
		return fail(err)
	}
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		data = append([]byte{'\n'}, data...)
	}

	f, err := root.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return fail(err)
	}
	if created {
		if info, err := root.Stat(journalName); err == nil {
			if err := f.Chmod(info.Mode().Perm()); err != nil {
				_ = f.Close()
				return fail(err)
			}
		}
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fail(err)
	}
	// The lines must be on disk before journal.jsonl stops holding them.
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fail(err)
	}
	if err := f.Close(); err != nil {
		return fail(err)
	}
	return nil
}
