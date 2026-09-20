package journal

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadVersion(t *testing.T) {
	tests := []struct {
		name    string
		content *string
		want    int
		wantErr bool
	}{
		{name: "zero with a newline", content: str("0\n"), want: 0},
		{name: "no newline", content: str("0"), want: 0},
		{name: "white space around", content: str(" 2 \r\n"), want: 2},
		{name: "a newer number", content: str("13\n"), want: 13},
		{name: "missing file", content: nil, wantErr: true},
		{name: "empty", content: str(""), wantErr: true},
		{name: "text", content: str("abc\n"), wantErr: true},
		{name: "negative", content: str("-1\n"), wantErr: true},
		{name: "a sign", content: str("+1\n"), wantErr: true},
		{name: "a fraction", content: str("1.5\n"), wantErr: true},
		{name: "two lines", content: str("0\n1\n"), wantErr: true},
		{name: "too many digits", content: str("12345678901\n"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.content != nil {
				if err := os.WriteFile(filepath.Join(dir, versionName), []byte(*tt.content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := readVersion(dir)
			if tt.wantErr {
				var verr *VersionFileError
				if !errors.As(err, &verr) || !errors.Is(err, ErrInvalidVersionFile) {
					t.Fatalf("err = %v, want a VersionFileError", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %d, %v; want %d", got, err, tt.want)
			}
		})
	}
}

func TestOpen(t *testing.T) {
	t.Run("a repository with format 0", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", nil)
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if j.Location().Dir != dir || j.Version() != 0 {
			t.Errorf("location %+v version %d", j.Location(), j.Version())
		}
		if j.opts.LockTimeout != DefaultLockTimeout {
			t.Errorf("lock timeout = %s, want the default %s", j.opts.LockTimeout, DefaultLockTimeout)
		}
	})

	t.Run("a format newer than this build is refused", func(t *testing.T) {
		root := newRepo(t)
		newMtqg(t, root, "1\n", str(lineOf(t, memo(idA, "x"))+"\n"))
		j, err := Open(root, Options{})
		var tooNew *FormatTooNewError
		if !errors.As(err, &tooNew) || tooNew.Found != 1 || tooNew.Supported != SupportedVersion {
			t.Fatalf("err = %v, want a FormatTooNewError for version 1", err)
		}
		if j != nil {
			t.Error("a Journal was returned for a format that is refused: it could be read or written")
		}
	})

	t.Run("a missing version file is an error", func(t *testing.T) {
		root := newRepo(t)
		mkdir(t, filepath.Join(root, mtqgDirName))
		if _, err := Open(root, Options{}); !errors.Is(err, ErrInvalidVersionFile) {
			t.Fatalf("err = %v, want ErrInvalidVersionFile", err)
		}
	})

	t.Run("without .mtqg/", func(t *testing.T) {
		root := newRepo(t)
		var notInit *NotInitializedError
		if _, err := Open(root, Options{}); !errors.As(err, &notInit) {
			t.Fatalf("err = %v, want a NotInitializedError", err)
		}
	})
}

func TestJournalRead(t *testing.T) {
	a := lineOf(t, memo(idA, "first"))
	b := lineOf(t, memo(idB, "second"))

	t.Run("events and warnings", func(t *testing.T) {
		root := newRepo(t)
		newMtqg(t, root, "0\n", str(a+"\n"+"broken\n"+b+"\n"+a+"\n"))
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		got, err := j.Read()
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Events) != 2 || got.Events[0].ID != idA || got.Events[1].ID != idB {
			t.Errorf("events = %+v, want the two distinct events in file order", got.Events)
		}
		if len(got.Warnings) != 1 || got.Warnings[0] != (Warning{Kind: WarnInvalidJSON, Line: 2}) {
			t.Errorf("warnings = %+v", got.Warnings)
		}
	})

	t.Run("a missing journal.jsonl is an empty journal", func(t *testing.T) {
		root := newRepo(t)
		newMtqg(t, root, "0\n", nil)
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		got, err := j.Read()
		if err != nil || len(got.Events) != 0 || len(got.Warnings) != 0 {
			t.Fatalf("got %+v, %v", got, err)
		}
	})

	t.Run("conflict markers warn but do not stop reading", func(t *testing.T) {
		root := newRepo(t)
		newMtqg(t, root, "0\n", str("<<<<<<< HEAD\n"+a+"\n=======\n"+b+"\n>>>>>>> other\n"))
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		got, err := j.Read()
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Events) != 2 {
			t.Errorf("both sides must be read, got %d events", len(got.Events))
		}
		markers := 0
		for _, w := range got.Warnings {
			if w.Kind == WarnConflictMarker {
				markers++
			}
		}
		if markers != 3 {
			t.Errorf("got %d conflict marker warnings, want 3 (%+v)", markers, got.Warnings)
		}
	})

	t.Run("resolving the conflict by removing the markers reads cleanly", func(t *testing.T) {
		root := newRepo(t)
		newMtqg(t, root, "0\n", str(a+"\n"+b+"\n"))
		j, err := Open(root, Options{})
		if err != nil {
			t.Fatal(err)
		}
		got, err := j.Read()
		if err != nil || len(got.Events) != 2 || len(got.Warnings) != 0 {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
}
