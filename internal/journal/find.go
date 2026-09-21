package journal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Names inside .mtqg/ (docs/reference/schema.md).
const (
	mtqgDirName = ".mtqg"
	journalName = "journal.jsonl"
	versionName = "version"
	localName   = ".local"
	lockName    = "lock"
)

// Location says where the .mtqg/ directory of a repository is.
type Location struct {
	// Root is the directory that holds .mtqg/ (the root of the repository).
	Root string
	// Dir is the .mtqg/ directory.
	Dir string
}

// Find looks for .mtqg/ the way git looks for .git: from start up to the parents,
// one directory at a time. In each directory it checks, in this order:
//
//  1. .mtqg/ exists: use it.
//  2. .git exists (a directory, or a file in a worktree or a submodule): this is
//     the root of the repository and it has no .mtqg/ yet. Find returns a
//     *NotInitializedError.
//
// It never looks above the root of the repository, so an unrelated .mtqg/ outside
// it is not picked up. Outside a repository it returns ErrNotInRepository.
//
// start is made physical first (symbolic links are resolved), like git does, so
// that the same .mtqg/ is found whichever path leads to it.
func Find(start string) (Location, error) {
	root, hasMtqg, err := locate(start)
	if err != nil {
		return Location{}, err
	}
	if !hasMtqg {
		return Location{}, &NotInitializedError{Root: root}
	}
	return Location{Root: root, Dir: filepath.Join(root, mtqgDirName)}, nil
}

// locate walks up from start the way Find describes. It returns the directory it
// stopped at, and whether that directory has a .mtqg/: true when it found one,
// false when it met the .git of a repository that has none. Find and Init share
// it, so that they always agree on where .mtqg/ is or would be.
func locate(start string) (root string, hasMtqg bool, err error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", false, fmt.Errorf("journal: %w", err)
	}
	dir, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", false, fmt.Errorf("journal: %w", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", false, fmt.Errorf("journal: %w", err)
	}
	if !info.IsDir() {
		return "", false, fmt.Errorf("journal: %s is not a directory", dir)
	}

	for {
		mtqg := filepath.Join(dir, mtqgDirName)
		switch info, err := os.Stat(mtqg); {
		case err == nil && info.IsDir():
			return dir, true, nil
		case err == nil:
			return "", false, fmt.Errorf("journal: %s is not a directory", mtqg)
		case !errors.Is(err, fs.ErrNotExist):
			return "", false, fmt.Errorf("journal: %w", err)
		}

		switch _, err := os.Lstat(filepath.Join(dir, ".git")); {
		case err == nil:
			return dir, false, nil
		case !errors.Is(err, fs.ErrNotExist):
			return "", false, fmt.Errorf("journal: %w", err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, ErrNotInRepository
		}
		dir = parent
	}
}
