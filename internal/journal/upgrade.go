package journal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Upgrade raises the format version declared by .mtqg/version to
// SupportedVersion, and rewrites SCHEMA.md (schema is the text of the new
// .mtqg/SCHEMA.md, as passed to Init) whenever the format is raised or the
// existing SCHEMA.md's marker is missing or older than SchemaVersion
// (docs/reference/schema.md "Versioning"). journal.jsonl is never touched:
// existing lines keep the meaning their own v had when they were written.
// schemaUpdated reports whether SCHEMA.md was (or, with dryRun, would be)
// rewritten; it can be true even when from == to.
//
// dryRun reports what would happen without writing anything. Writing is
// refused while journal.jsonl holds conflict markers, the same as Append and
// Rewrite: a repository in that state should be resolved first. The check
// only runs when something would actually be written.
func (j *Journal) Upgrade(schema string, dryRun bool) (from, to int, schemaUpdated bool, err error) {
	from, to = j.version, SupportedVersion
	raise := from < to

	stale, err := j.schemaMarkerStale()
	if err != nil {
		return from, to, false, err
	}
	schemaUpdated = raise || stale
	if !schemaUpdated || dryRun {
		return from, to, schemaUpdated, nil
	}

	release, err := j.lock()
	if err != nil {
		return from, to, schemaUpdated, err
	}
	defer release()

	root, err := j.openRoot()
	if err != nil {
		return from, to, schemaUpdated, err
	}
	defer func() { _ = root.Close() }()

	data, err := root.ReadFile(journalName)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return from, to, schemaUpdated, fmt.Errorf("journal: %w", err)
	}
	if hasConflictMarkers(data) {
		return from, to, schemaUpdated, ErrConflictMarkers
	}

	if raise {
		if err := replaceRootFile(root, versionName, []byte(fmt.Sprintf("%d\n", to)), filePerm(root, versionName)); err != nil {
			return from, to, schemaUpdated, err
		}
	}
	if err := replaceRootFile(root, schemaName, []byte(schema), filePerm(root, schemaName)); err != nil {
		return from, to, schemaUpdated, err
	}
	if raise {
		j.version = to
	}
	return from, to, schemaUpdated, nil
}

// schemaMarkerStale reports whether the repository's existing SCHEMA.md (if
// any) is missing SchemaVersion's marker or carries an older one.
func (j *Journal) schemaMarkerStale() (bool, error) {
	root, err := j.openRoot()
	if err != nil {
		return false, err
	}
	defer func() { _ = root.Close() }()

	current, err := root.ReadFile(schemaName)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("schema: %w", err)
	}
	return schemaStale(current), nil
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
