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
// Before mtqg v1 the version was 0, meaning "not yet stable"; mtqg upgrade
// raises a repository to this version without touching any line
// (docs/reference/schema.md "Versioning").
const SupportedVersion = 1

// SchemaVersion is the mtqg release docs/reference/schema.md's content
// matches, as a marker comment near the end of that file also says
// (<!-- schema as of mtqg X.Y.Z -->). It is independent of SupportedVersion:
// schema.md's wording can change without the format itself changing. Raise
// this whenever schema.md's content changes, and update the marker to match
// in the same change; a test checks the two agree.
const SchemaVersion = "1.1.0"

// maxVersionDigits keeps the number far from overflowing an int.
const maxVersionDigits = 9

// readVersion reads .mtqg/version: one non-negative integer and a newline.
func readVersion(root *os.Root) (int, error) {
	path := filepath.Join(root.Name(), versionName)
	data, err := root.ReadFile(versionName)
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
