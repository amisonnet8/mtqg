package journal

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestUpgrade(t *testing.T) {
	t.Run("raises version 0 and rewrites SCHEMA.md, journal.jsonl untouched", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", str(lineOf(t, memo(idA, "x"))+"\n"))
		writeFile(t, filepath.Join(dir, schemaName), "old schema")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		journalBefore := readFile(t, filepath.Join(dir, journalName))

		from, to, err := j.Upgrade("new schema", false)
		if err != nil {
			t.Fatal(err)
		}
		if from != 0 || to != SupportedVersion {
			t.Fatalf("from, to = %d, %d, want 0, %d", from, to, SupportedVersion)
		}
		if got := readFile(t, filepath.Join(dir, versionName)); got != strconv.Itoa(SupportedVersion)+"\n" {
			t.Errorf("version file = %q, want %q", got, strconv.Itoa(SupportedVersion)+"\n")
		}
		if got := readFile(t, filepath.Join(dir, schemaName)); got != "new schema" {
			t.Errorf("SCHEMA.md = %q, want %q", got, "new schema")
		}
		if got := readFile(t, filepath.Join(dir, journalName)); got != journalBefore {
			t.Errorf("journal.jsonl changed: got %q, want %q", got, journalBefore)
		}
		if j.Version() != SupportedVersion {
			t.Errorf("j.Version() = %d, want %d", j.Version(), SupportedVersion)
		}
	})

	t.Run("already at the supported version: nothing is written", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, strconv.Itoa(SupportedVersion)+"\n", nil)
		writeFile(t, filepath.Join(dir, schemaName), "unchanged schema")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		from, to, err := j.Upgrade("new schema", false)
		if err != nil {
			t.Fatal(err)
		}
		if from != to || from != SupportedVersion {
			t.Fatalf("from, to = %d, %d, want both %d", from, to, SupportedVersion)
		}
		if got := readFile(t, filepath.Join(dir, schemaName)); got != "unchanged schema" {
			t.Errorf("SCHEMA.md was written: %q", got)
		}
	})

	t.Run("dry run: reports what would happen but writes nothing", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", nil)
		writeFile(t, filepath.Join(dir, schemaName), "old schema")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		from, to, err := j.Upgrade("new schema", true)
		if err != nil {
			t.Fatal(err)
		}
		if from != 0 || to != SupportedVersion {
			t.Fatalf("from, to = %d, %d, want 0, %d", from, to, SupportedVersion)
		}
		if got := readFile(t, filepath.Join(dir, versionName)); got != "0\n" {
			t.Errorf("version file was written: %q", got)
		}
		if got := readFile(t, filepath.Join(dir, schemaName)); got != "old schema" {
			t.Errorf("SCHEMA.md was written: %q", got)
		}
		if j.Version() != 0 {
			t.Errorf("j.Version() = %d, want 0 (unchanged)", j.Version())
		}
	})

	t.Run("conflict markers: refuses and writes nothing", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", str("<<<<<<< HEAD\n"+lineOf(t, memo(idA, "x"))+"\n=======\n>>>>>>> other\n"))
		writeFile(t, filepath.Join(dir, schemaName), "old schema")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = j.Upgrade("new schema", false)
		if !errors.Is(err, ErrConflictMarkers) {
			t.Fatalf("err = %v, want ErrConflictMarkers", err)
		}
		if got := readFile(t, filepath.Join(dir, versionName)); got != "0\n" {
			t.Errorf("version file was written: %q", got)
		}
		if got := readFile(t, filepath.Join(dir, schemaName)); got != "old schema" {
			t.Errorf("SCHEMA.md was written: %q", got)
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
