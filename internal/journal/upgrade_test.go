package journal

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// currentMarker is a SCHEMA.md body whose marker already matches
// SchemaVersion, the same as a repository that is fully up to date.
var currentMarker = "schema text\n<!-- schema as of mtqg " + SchemaVersion + " -->\n"

func TestUpgrade(t *testing.T) {
	t.Run("raises the format version and rewrites SCHEMA.md, journal.jsonl untouched", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", str(lineOf(t, memo(idA, "x"))+"\n"))
		writeFile(t, filepath.Join(dir, schemaName), "old schema")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		journalBefore := readFile(t, filepath.Join(dir, journalName))

		from, to, schemaUpdated, err := j.Upgrade("new schema", false)
		if err != nil {
			t.Fatal(err)
		}
		if from != 0 || to != SupportedVersion {
			t.Fatalf("from, to = %d, %d, want 0, %d", from, to, SupportedVersion)
		}
		if !schemaUpdated {
			t.Error("schemaUpdated = false, want true")
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

	t.Run("already at the supported version and SCHEMA.md's marker matches: nothing is written", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, strconv.Itoa(SupportedVersion)+"\n", nil)
		writeFile(t, filepath.Join(dir, schemaName), currentMarker)
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		from, to, schemaUpdated, err := j.Upgrade("new schema", false)
		if err != nil {
			t.Fatal(err)
		}
		if from != to || from != SupportedVersion {
			t.Fatalf("from, to = %d, %d, want both %d", from, to, SupportedVersion)
		}
		if schemaUpdated {
			t.Error("schemaUpdated = true, want false")
		}
		if got := readFile(t, filepath.Join(dir, schemaName)); got != currentMarker {
			t.Errorf("SCHEMA.md was written: %q", got)
		}
	})

	t.Run("SCHEMA.md's marker is missing: rewritten even though the format is unchanged", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, strconv.Itoa(SupportedVersion)+"\n", nil)
		writeFile(t, filepath.Join(dir, schemaName), "schema text with no marker at all")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		from, to, schemaUpdated, err := j.Upgrade("new schema", false)
		if err != nil {
			t.Fatal(err)
		}
		if from != to || from != SupportedVersion {
			t.Fatalf("from, to = %d, %d, want both %d (format must not change)", from, to, SupportedVersion)
		}
		if !schemaUpdated {
			t.Error("schemaUpdated = false, want true")
		}
		if got := readFile(t, filepath.Join(dir, schemaName)); got != "new schema" {
			t.Errorf("SCHEMA.md = %q, want %q", got, "new schema")
		}
		if got := readFile(t, filepath.Join(dir, versionName)); got != strconv.Itoa(SupportedVersion)+"\n" {
			t.Errorf("version file changed: %q", got)
		}
	})

	t.Run("SCHEMA.md's marker is older than SchemaVersion: rewritten even though the format is unchanged", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, strconv.Itoa(SupportedVersion)+"\n", nil)
		writeFile(t, filepath.Join(dir, schemaName), "old\n<!-- schema as of mtqg 1.0.0 -->\n")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		from, to, schemaUpdated, err := j.Upgrade("new schema", false)
		if err != nil {
			t.Fatal(err)
		}
		if from != to {
			t.Fatalf("from, to = %d, %d, want equal (format must not change)", from, to)
		}
		if !schemaUpdated {
			t.Error("schemaUpdated = false, want true")
		}
		if got := readFile(t, filepath.Join(dir, schemaName)); got != "new schema" {
			t.Errorf("SCHEMA.md = %q, want %q", got, "new schema")
		}
	})

	t.Run("SCHEMA.md's marker is newer than SchemaVersion: left alone", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, strconv.Itoa(SupportedVersion)+"\n", nil)
		future := "from the future\n<!-- schema as of mtqg 9.9.9 -->\n"
		writeFile(t, filepath.Join(dir, schemaName), future)
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		from, to, schemaUpdated, err := j.Upgrade("new schema", false)
		if err != nil {
			t.Fatal(err)
		}
		if from != to {
			t.Fatalf("from, to = %d, %d, want equal", from, to)
		}
		if schemaUpdated {
			t.Error("schemaUpdated = true, want false: an older mtqg must not roll back a newer SCHEMA.md")
		}
		if got := readFile(t, filepath.Join(dir, schemaName)); got != future {
			t.Errorf("SCHEMA.md was rewritten: %q", got)
		}
	})

	t.Run("dry run when the format is behind: reports what would happen but writes nothing", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", nil)
		writeFile(t, filepath.Join(dir, schemaName), "old schema")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		from, to, schemaUpdated, err := j.Upgrade("new schema", true)
		if err != nil {
			t.Fatal(err)
		}
		if from != 0 || to != SupportedVersion {
			t.Fatalf("from, to = %d, %d, want 0, %d", from, to, SupportedVersion)
		}
		if !schemaUpdated {
			t.Error("schemaUpdated = false, want true")
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

	t.Run("dry run when only SCHEMA.md's marker is stale: reports it but writes nothing", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, strconv.Itoa(SupportedVersion)+"\n", nil)
		writeFile(t, filepath.Join(dir, schemaName), "no marker here")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		from, to, schemaUpdated, err := j.Upgrade("new schema", true)
		if err != nil {
			t.Fatal(err)
		}
		if from != to {
			t.Fatalf("from, to = %d, %d, want equal", from, to)
		}
		if !schemaUpdated {
			t.Error("schemaUpdated = false, want true")
		}
		if got := readFile(t, filepath.Join(dir, schemaName)); got != "no marker here" {
			t.Errorf("SCHEMA.md was written: %q", got)
		}
	})

	t.Run("conflict markers with the format behind: refuses and writes nothing", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", str("<<<<<<< HEAD\n"+lineOf(t, memo(idA, "x"))+"\n=======\n>>>>>>> other\n"))
		writeFile(t, filepath.Join(dir, schemaName), "old schema")
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, err = j.Upgrade("new schema", false)
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

	t.Run("conflict markers with nothing to write: not refused, since nothing would be written anyway", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, strconv.Itoa(SupportedVersion)+"\n", str("<<<<<<< HEAD\n"+lineOf(t, memo(idA, "x"))+"\n=======\n>>>>>>> other\n"))
		writeFile(t, filepath.Join(dir, schemaName), currentMarker)
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		from, to, schemaUpdated, err := j.Upgrade("new schema", false)
		if err != nil {
			t.Fatal(err)
		}
		if from != to || schemaUpdated {
			t.Fatalf("from, to, schemaUpdated = %d, %d, %v, want %d, %d, false", from, to, schemaUpdated, from, from)
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
