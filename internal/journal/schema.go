package journal

import (
	"regexp"
	"strconv"
	"strings"
)

// schemaMarkerPattern matches the "schema as of mtqg X.Y.Z" marker comment
// near the end of docs/reference/schema.md (docs/reference/schema.md
// "Versioning").
var schemaMarkerPattern = regexp.MustCompile(`<!-- schema as of mtqg (\d+\.\d+\.\d+) -->`)

// SchemaMarker extracts the "schema as of mtqg X.Y.Z" marker from text (the
// embedded schema.md, or an existing .mtqg/SCHEMA.md). ok is false if no
// marker is present or it is not in the X.Y.Z form.
func SchemaMarker(text string) (version string, ok bool) {
	m := schemaMarkerPattern.FindStringSubmatch(text)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// schemaOlder reports whether a is older than b, comparing X.Y.Z numerically
// component by component (so "1.10.0" is newer than "1.9.0", unlike a plain
// string comparison). A version that cannot be parsed as X.Y.Z is treated as
// the oldest possible, so a missing or malformed marker always counts as
// older.
func schemaOlder(a, b string) bool {
	pa, oka := parseSchemaVersion(a)
	pb, okb := parseSchemaVersion(b)
	if !oka {
		return true
	}
	if !okb {
		return false
	}
	for i := range 3 {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return false
}

func parseSchemaVersion(v string) (parts [3]int, ok bool) {
	fields := strings.Split(v, ".")
	if len(fields) != 3 {
		return parts, false
	}
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil {
			return parts, false
		}
		parts[i] = n
	}
	return parts, true
}

// schemaStale reports whether current (the bytes of an existing
// .mtqg/SCHEMA.md, or nil if there is none) is missing SchemaVersion's
// marker, or carries an older one. A newer marker (written by a later mtqg)
// is not stale: an older mtqg running upgrade must not roll SCHEMA.md back.
func schemaStale(current []byte) bool {
	marker, ok := SchemaMarker(string(current))
	if !ok {
		return true
	}
	return schemaOlder(marker, SchemaVersion)
}
