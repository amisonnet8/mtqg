//go:build !windows

package journal

import "os"

// replaceFile puts oldname in place of newname in one step. On these systems a
// rename replaces the target even when another process has it open, and readers
// see the old file or the new one, never a mix.
func replaceFile(root *os.Root, oldname, newname string) error {
	return root.Rename(oldname, newname)
}
