package journal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// initFile is a file that Init puts into a new .mtqg/.
type initFile struct {
	name    string
	content string
}

// Init makes .mtqg/ in the repository that start is in: at the root of the
// repository, next to .git, whichever directory it was called from. schema is
// the text of .mtqg/SCHEMA.md.
//
// It walks up like Find. A .mtqg/ on the way is an *AlreadyInitializedError,
// and outside a repository the error is ErrNotInRepository. Nothing is
// overwritten, and nothing else is touched: Init does not change git's
// configuration and does not commit (.claude/rules/git-integration.md).
//
// The files are journal.jsonl (empty), .gitattributes (union merge, so that two
// branches that both append keep the lines of both), .gitignore (for .local/),
// version and SCHEMA.md. If one of them cannot be made, the .mtqg/ that was
// made is removed again.
func Init(start, schema string) (Location, error) {
	root, hasMtqg, err := locate(start)
	if err != nil {
		return Location{}, err
	}
	if hasMtqg {
		return Location{}, &AlreadyInitializedError{Path: filepath.Join(root, mtqgDirName)}
	}
	return createMtqg(root, []initFile{
		{journalName, ""},
		{".gitattributes", "*.jsonl text eol=lf merge=union\n"},
		{".gitignore", localName + "/\n"},
		{versionName, fmt.Sprintf("%d\n", SupportedVersion)},
		{"SCHEMA.md", schema},
	})
}

// createMtqg makes .mtqg/ in root and puts files into it. The directory is for
// its owner and group; the files get the permissions any file made by a program
// gets (os.Root.Create: 0666 less the umask), because they are shared through
// git like other files of the project.
func createMtqg(root string, files []initFile) (Location, error) {
	dir := filepath.Join(root, mtqgDirName)
	if err := os.Mkdir(dir, 0o750); err != nil {
		if errors.Is(err, fs.ErrExist) {
			// Somebody else made it between the check and now.
			return Location{}, &AlreadyInitializedError{Path: dir}
		}
		return Location{}, fmt.Errorf("journal: %w", err)
	}

	fail := func(err error) (Location, error) {
		_ = os.RemoveAll(dir)
		return Location{}, fmt.Errorf("journal: %w", err)
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		return fail(err)
	}
	defer func() { _ = r.Close() }()
	for _, file := range files {
		f, err := r.Create(file.name)
		if err != nil {
			return fail(err)
		}
		if _, err := f.WriteString(file.content); err != nil {
			_ = f.Close()
			return fail(err)
		}
		if err := f.Close(); err != nil {
			return fail(err)
		}
	}
	return Location{Root: root, Dir: dir}, nil
}
