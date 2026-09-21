package mtqg_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// Go source must not hold characters that cannot be seen: the ones that turn
// text around, zero width ones, and control characters. A line that reads one
// way to a person and another to the compiler is how "Trojan Source" attacks
// work, and it is how a test that means "a JSON escape" quietly turns into a
// test of something else. Write such characters as escapes (U+2028, \xe2\x80\xa8)
// or, in code that is not a literal, as numbers.
//
// The ranges are written as numbers, for the same reason.
func TestSourceHasNoInvisibleCharacters(t *testing.T) {
	invisible := func(r rune) bool {
		switch {
		case r >= 0x200B && r <= 0x200F, // zero width space to right-to-left mark
			r >= 0x2028 && r <= 0x202E, // line and paragraph separators, embeddings, overrides
			r >= 0x2060 && r <= 0x2064, // word joiner and invisible operators
			r >= 0x2066 && r <= 0x206F, // isolates and deprecated format characters
			r == 0x061C, r == 0xFEFF, r == 0x00AD:
			return true
		}
		return unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r'
	}

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for n, line := range strings.Split(string(data), "\n") {
			for _, r := range line {
				if invisible(r) {
					t.Errorf("%s:%d has the invisible character U+%04X; write it as an escape", path, n+1, r)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
