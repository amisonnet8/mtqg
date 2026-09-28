package mtqg_test

import (
	"strings"
	"testing"

	"github.com/amisonnet8/mtqg"
	"github.com/amisonnet8/mtqg/internal/journal"
)

func TestSchemaIsEmbedded(t *testing.T) {
	if !strings.HasPrefix(mtqg.Schema, "# mtqg journal format") {
		t.Errorf("Schema does not start with the title of schema.md: %.40q", mtqg.Schema)
	}
}

// TestSchemaMarkerMatchesSchemaVersion guards against the marker comment in
// docs/reference/schema.md (embedded as mtqg.Schema) drifting away from
// journal.SchemaVersion: both are hand-maintained, and nothing else keeps
// them in sync. See docs/reference/schema.md "Versioning".
func TestSchemaMarkerMatchesSchemaVersion(t *testing.T) {
	marker, ok := journal.SchemaMarker(mtqg.Schema)
	if !ok {
		t.Fatal("docs/reference/schema.md has no \"schema as of mtqg X.Y.Z\" marker")
	}
	if marker != journal.SchemaVersion {
		t.Errorf("schema.md's marker is %q, journal.SchemaVersion is %q; update both together", marker, journal.SchemaVersion)
	}
}
