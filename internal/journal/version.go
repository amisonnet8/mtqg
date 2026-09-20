package journal

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SupportedVersion is the newest format version this build reads and writes.
// Format 0 means "not yet stable" (docs/reference/schema.md).
const SupportedVersion = 0

// maxVersionDigits keeps the number far from overflowing an int.
const maxVersionDigits = 9

// readVersion reads .mtqg/version: one non-negative integer and a newline.
func readVersion(dir string) (int, error) {
	path := filepath.Join(dir, versionName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, &VersionFileError{Path: path, Reason: "the file is missing"}
		}
		return 0, &VersionFileError{Path: path, Reason: err.Error()}
	}
	text := strings.TrimSpace(string(data))
	if text == "" || len(text) > maxVersionDigits || strings.Trim(text, "0123456789") != "" {
		return 0, &VersionFileError{Path: path, Reason: "it must hold one non-negative integer"}
	}
	version, err := strconv.Atoi(text)
	if err != nil {
		return 0, &VersionFileError{Path: path, Reason: "it must hold one non-negative integer"}
	}
	return version, nil
}
