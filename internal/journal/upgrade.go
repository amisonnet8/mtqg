package journal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Upgrade raises the format version declared by .mtqg/version to
// SupportedVersion and rewrites SCHEMA.md to match (schema is the text of the
// new .mtqg/SCHEMA.md, as passed to Init). journal.jsonl is never touched:
// existing lines keep the meaning their own v had when they were written
// (docs/reference/schema.md "Versioning"). If the repository is already at
// SupportedVersion, from == to and nothing is written.
//
// dryRun reports what would happen without writing anything. Upgrading is
// refused while journal.jsonl holds conflict markers, the same as Append and
// Rewrite: a repository in that state should be resolved first.
func (j *Journal) Upgrade(schema string, dryRun bool) (from, to int, err error) {
	from, to = j.version, SupportedVersion
	if from >= to || dryRun {
		return from, to, nil
	}

	release, err := j.lock()
	if err != nil {
		return from, to, err
	}
	defer release()

	root, err := j.openRoot()
	if err != nil {
		return from, to, err
	}
	defer func() { _ = root.Close() }()

	data, err := root.ReadFile(journalName)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return from, to, fmt.Errorf("journal: %w", err)
	}
	if hasConflictMarkers(data) {
		return from, to, ErrConflictMarkers
	}

	if err := replaceRootFile(root, versionName, []byte(fmt.Sprintf("%d\n", to)), filePerm(root, versionName)); err != nil {
		return from, to, err
	}
	if err := replaceRootFile(root, schemaName, []byte(schema), filePerm(root, schemaName)); err != nil {
		return from, to, err
	}
	j.version = to
	return from, to, nil
}

// filePerm returns name's current permissions, or 0o600 if it cannot be
// statted (Init always creates both version and SCHEMA.md, so this is a
// fallback, not the common case).
func filePerm(root *os.Root, name string) fs.FileMode {
	if info, err := root.Stat(name); err == nil {
		return info.Mode().Perm()
	}
	return 0o600
}
