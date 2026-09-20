package journal

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestFind(t *testing.T) {
	t.Run("at the root of the repository", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", nil)
		got, err := Find(root)
		if err != nil {
			t.Fatal(err)
		}
		if got.Root != root || got.Dir != dir {
			t.Errorf("got %+v, want root %s dir %s", got, root, dir)
		}
	})

	t.Run("from a subdirectory", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", nil)
		sub := mkdir(t, filepath.Join(root, "a", "b", "c"))
		got, err := Find(sub)
		if err != nil {
			t.Fatal(err)
		}
		if got.Root != root || got.Dir != dir {
			t.Errorf("got %+v, want root %s dir %s", got, root, dir)
		}
	})

	t.Run("the repository has no .mtqg/ yet", func(t *testing.T) {
		root := newRepo(t)
		sub := mkdir(t, filepath.Join(root, "src"))
		_, err := Find(sub)
		var notInit *NotInitializedError
		if !errors.As(err, &notInit) || notInit.Root != root {
			t.Fatalf("err = %v, want a NotInitializedError for %s", err, root)
		}
	})

	t.Run("outside a git repository", func(t *testing.T) {
		isolateGit(t)
		dir := realPath(t, t.TempDir())
		if _, err := Find(dir); !errors.Is(err, ErrNotInRepository) {
			t.Fatalf("err = %v, want ErrNotInRepository", err)
		}
	})

	t.Run("does not look above the root of the repository", func(t *testing.T) {
		// A nested repository (a submodule has its own .git, too) must not pick
		// up the .mtqg/ of the repository around it.
		outer := newRepo(t)
		newMtqg(t, outer, "0\n", nil)
		inner := mkdir(t, filepath.Join(outer, "inner"))
		git(t, inner, "init", "-q", "-b", "main")

		_, err := Find(inner)
		var notInit *NotInitializedError
		if !errors.As(err, &notInit) || notInit.Root != inner {
			t.Fatalf("err = %v, want a NotInitializedError for %s", err, inner)
		}
	})

	t.Run(".git is a file in a worktree", func(t *testing.T) {
		root := newRepo(t)
		git(t, root, "commit", "-q", "--allow-empty", "-m", "first")
		worktree := filepath.Join(realPath(t, t.TempDir()), "wt")
		git(t, root, "worktree", "add", "-q", "-b", "feature", worktree)
		if info, err := os.Lstat(filepath.Join(worktree, ".git")); err != nil || info.IsDir() {
			t.Fatalf("expected .git to be a file in the worktree, got %v %v", info, err)
		}

		// No .mtqg/ in the worktree yet: the worktree is the boundary.
		_, err := Find(worktree)
		var notInit *NotInitializedError
		if !errors.As(err, &notInit) || notInit.Root != worktree {
			t.Fatalf("err = %v, want a NotInitializedError for %s", err, worktree)
		}

		dir := newMtqg(t, worktree, "0\n", nil)
		got, err := Find(worktree)
		if err != nil {
			t.Fatal(err)
		}
		if got.Root != worktree || got.Dir != dir {
			t.Errorf("got %+v, want root %s dir %s", got, worktree, dir)
		}
	})

	t.Run("through a symbolic link", func(t *testing.T) {
		root := newRepo(t)
		dir := newMtqg(t, root, "0\n", nil)
		sub := mkdir(t, filepath.Join(root, "sub"))
		link := filepath.Join(realPath(t, t.TempDir()), "link")
		if err := os.Symlink(sub, link); err != nil {
			t.Skipf("cannot create a symbolic link here: %v", err)
		}
		got, err := Find(link)
		if err != nil {
			t.Fatal(err)
		}
		if got.Root != root || got.Dir != dir {
			t.Errorf("got %+v, want root %s dir %s", got, root, dir)
		}
	})

	t.Run("start does not exist", func(t *testing.T) {
		isolateGit(t)
		_, err := Find(filepath.Join(t.TempDir(), "missing"))
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("err = %v, want a not-exist error", err)
		}
	})

	t.Run("start is a file", func(t *testing.T) {
		root := newRepo(t)
		file := filepath.Join(root, "file.txt")
		if err := os.WriteFile(file, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Find(file); err == nil {
			t.Fatal("Find accepted a file as the place to start")
		}
	})

	t.Run(".mtqg is a file", func(t *testing.T) {
		root := newRepo(t)
		if err := os.WriteFile(filepath.Join(root, mtqgDirName), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Find(root)
		var notInit *NotInitializedError
		if err == nil || errors.As(err, &notInit) {
			t.Fatalf("err = %v, want a plain error about .mtqg not being a directory", err)
		}
	})
}
